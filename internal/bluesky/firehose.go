package bluesky

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"open-news/internal/metadata"
	"open-news/internal/models"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// FirehoseConsumer handles the Bluesky Jetstream connection and processing
type FirehoseConsumer struct {
	db                *gorm.DB
	client            *Client
	dialer            *websocket.Dialer
	metadataExtractor *metadata.MetadataExtractor
}

// NewFirehoseConsumer creates a new firehose consumer
func NewFirehoseConsumer(db *gorm.DB, client *Client) *FirehoseConsumer {
	return &FirehoseConsumer{
		db:                db,
		client:            client,
		dialer:            websocket.DefaultDialer,
		metadataExtractor: metadata.NewMetadataExtractor(),
	}
}

// JetstreamEvent represents an event from the Bluesky Jetstream
type JetstreamEvent struct {
	DID      string             `json:"did"`
	TimeUS   int64              `json:"time_us"`
	Kind     string             `json:"kind"`
	Commit   *JetstreamCommit   `json:"commit,omitempty"`
	Account  *JetstreamAccount  `json:"account,omitempty"`
	Identity *JetstreamIdentity `json:"identity,omitempty"`
}

// JetstreamCommit represents a commit event
type JetstreamCommit struct {
	Rev        string                 `json:"rev"`
	Operation  string                 `json:"operation"`
	Collection string                 `json:"collection"`
	RKey       string                 `json:"rkey"`
	Record     map[string]interface{} `json:"record,omitempty"`
	CID        string                 `json:"cid"`
}

// JetstreamAccount represents an account event
type JetstreamAccount struct {
	Active bool      `json:"active"`
	DID    string    `json:"did"`
	Seq    int64     `json:"seq"`
	Time   time.Time `json:"time"`
}

// JetstreamIdentity represents an identity event
type JetstreamIdentity struct {
	DID    string    `json:"did"`
	Handle string    `json:"handle"`
	Seq    int64     `json:"seq"`
	Time   time.Time `json:"time"`
}

// PostRecord represents a post record from Jetstream
type PostRecord struct {
	Type      string    `json:"$type"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	Facets    []Facet   `json:"facets,omitempty"`
	Embed     *Embed    `json:"embed,omitempty"`
	Reply     *Reply    `json:"reply,omitempty"`
	Langs     []string  `json:"langs,omitempty"`
}

// Reply represents reply information
type Reply struct {
	Root   RecordRef `json:"root"`
	Parent RecordRef `json:"parent"`
}

// RecordRef represents a reference to another record
type RecordRef struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

// StartConsuming starts consuming the Bluesky Jetstream
func (fc *FirehoseConsumer) StartConsuming(ctx context.Context) error {
	// Use Jetstream endpoint instead of raw firehose
	jetstreamURL := "wss://jetstream2.us-east.bsky.network/subscribe?wantedCollections=app.bsky.feed.post"

	log.Printf("Connecting to Bluesky Jetstream: %s", jetstreamURL)

	// Retry logic for connection
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := fc.connectAndConsume(ctx, jetstreamURL); err != nil {
				log.Printf("Jetstream connection error: %v. Reconnecting in 10 seconds...", err)

				// Wait before reconnecting
				select {
				case <-time.After(10 * time.Second):
					continue
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}

// connectAndConsume handles a single connection to Jetstream
func (fc *FirehoseConsumer) connectAndConsume(ctx context.Context, jetstreamURL string) error {
	conn, _, err := fc.dialer.DialContext(ctx, jetstreamURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Jetstream: %w", err)
	}
	defer conn.Close()

	log.Println("Successfully connected to Bluesky Jetstream")

	// Set up ping/pong handler to keep connection alive
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Ping goroutine
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("Failed to send ping: %v", err)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Message reading loop
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			_, message, err := conn.ReadMessage()
			if err != nil {
				return fmt.Errorf("failed to read message: %w", err)
			}

			if err := fc.processJetstreamMessage(message); err != nil {
				log.Printf("Error processing Jetstream message: %v", err)
				// Continue processing other messages even if one fails
			}
		}
	}
}

// processJetstreamMessage processes a single message from Jetstream
func (fc *FirehoseConsumer) processJetstreamMessage(data []byte) error {
	var event JetstreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal Jetstream event: %w", err)
	}

	// Only process commit events for posts
	if event.Kind == "commit" && event.Commit != nil &&
		event.Commit.Collection == "app.bsky.feed.post" &&
		event.Commit.Operation == "create" {

		return fc.processPostCommit(&event)
	}

	return nil
}

// processPostCommit processes a post creation commit
func (fc *FirehoseConsumer) processPostCommit(event *JetstreamEvent) error {
	// Parse the record as a post
	recordBytes, err := json.Marshal(event.Commit.Record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	var postRecord PostRecord
	if err := json.Unmarshal(recordBytes, &postRecord); err != nil {
		return fmt.Errorf("failed to unmarshal post record: %w", err)
	}

	// Extract links from the post
	links := fc.extractLinksFromPost(&postRecord)
	if len(links) == 0 {
		return nil // No links to process
	}

	// Check if this DID belongs to a source we're following
	var source models.Source
	result := fc.db.Where("blue_sky_d_id = ?", event.DID).First(&source)
	if result.Error != nil {
		// This source is not in our database, skip
		return nil
	}

	log.Printf("Found post with links from followed source %s: %v", source.Handle, links)

	// Process each link in the post
	for _, link := range links {
		if err := fc.processLink(link, &source, &postRecord, event); err != nil {
			log.Printf("Error processing link %s: %v", link, err)
		}
	}

	return nil
}

// extractLinksFromPost extracts URLs from a post's text, facets, and embeds
func (fc *FirehoseConsumer) extractLinksFromPost(post *PostRecord) []string {
	var links []string

	// Extract from facets (explicit links)
	for _, facet := range post.Facets {
		for _, feature := range facet.Features {
			if feature.Type == "app.bsky.richtext.facet#link" && feature.URI != "" {
				links = append(links, feature.URI)
			}
		}
	}

	// Extract from external embeds
	if post.Embed != nil && post.Embed.External != nil {
		links = append(links, post.Embed.External.URI)
	}

	// Simple URL extraction from text as fallback
	words := strings.Fields(post.Text)
	for _, word := range words {
		// Clean up common trailing punctuation
		word = strings.TrimRight(word, ".,!?;:")

		if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
			// Validate URL
			if _, err := url.Parse(word); err == nil {
				links = append(links, word)
			}
		}
	}

	// Remove duplicates
	uniqueLinks := make([]string, 0, len(links))
	seen := make(map[string]bool)
	for _, link := range links {
		if !seen[link] {
			seen[link] = true
			uniqueLinks = append(uniqueLinks, link)
		}
	}

	return uniqueLinks
}

// processLink processes a single article link from a post
func (fc *FirehoseConsumer) processLink(linkURL string, source *models.Source, post *PostRecord, event *JetstreamEvent) error {
	// Validate and normalize URL
	parsedURL, err := url.Parse(linkURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Skip non-HTTP(S) URLs
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil
	}

	canonicalURL := parsedURL.String()

	// Check if article already exists
	var article models.Article
	err = fc.db.Where("url = ?", canonicalURL).First(&article).Error

	if err == gorm.ErrRecordNotFound {
		// Article doesn't exist, first check if it's a NewsArticle
		log.Printf("New article discovered, checking if it's a NewsArticle: %s", canonicalURL)
		
		// Create context for NewsArticle validation
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		
		// Check if the URL contains NewsArticle schema
		isNewsArticle, validationErr := fc.checkIfNewsArticle(ctx, canonicalURL)
		
		// Handle different types of errors
		if validationErr != nil {
			log.Printf("Error checking NewsArticle schema for %s: %v", canonicalURL, validationErr)
			
			// Check if this is a reachability issue vs content issue
			if fc.isReachabilityError(validationErr) {
				log.Printf("Reachability issue detected, storing article for later validation: %s", canonicalURL)
				// Store the article but mark it as unreachable for background processing
				article = models.Article{
					URL:            canonicalURL,
					Title:          "", // Will be populated by background worker
					IsReachable:    false,
					FetchError:     validationErr.Error(),
					FetchRetries:   1,
					LastFetchError: &[]time.Time{time.Now()}[0],
					LastFetchAt:    &[]time.Time{time.Now()}[0],
				}
				
				if err := fc.db.Create(&article).Error; err != nil {
					return fmt.Errorf("failed to create unreachable article: %w", err)
				}
				
				log.Printf("Stored unreachable article for background processing: %s", canonicalURL)
			} else {
				log.Printf("Content validation failed (likely not a news article), skipping: %s", canonicalURL)
				return nil // Skip this article - it's not a valid news article
			}
		} else if !isNewsArticle {
			log.Printf("Skipping URL (not a NewsArticle): %s", canonicalURL)
			return nil // Skip this article
		} else {
			log.Printf("Confirmed as NewsArticle, extracting metadata: %s", canonicalURL)
			
			// Create context for metadata extraction
			ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel2()
			
			// Extract metadata from the URL
			metadata, err := fc.metadataExtractor.ExtractMetadata(ctx2, canonicalURL)
			now := time.Now()
			
			if err != nil {
				log.Printf("Failed to extract metadata for %s: %v", canonicalURL, err)
				// Create article with basic data and mark as unreachable
				article = models.Article{
					URL:            canonicalURL,
					IsCached:       false,
					IsReachable:    false,
					FetchError:     err.Error(),
					FetchRetries:   1,
					LastFetchError: &now,
					LastFetchAt:    &now,
					CreatedAt:      time.Now(),
				}
			} else {
				// Create article with extracted metadata
				article = models.Article{
					URL:          canonicalURL,
					Title:        metadata.Title,
					Description:  metadata.Description,
					Author:       metadata.Author,
					SiteName:     metadata.SiteName,
					ImageURL:     metadata.ImageURL,
					PublishedAt:  metadata.PublishedAt,
					JSONLDData:   metadata.JSONLDData,
					OGData:       metadata.OGData,
					HTMLContent:  metadata.HTMLContent,
					TextContent:  metadata.TextContent,
					WordCount:    int(metadata.WordCount),
					ReadingTime:  int(metadata.ReadingTime),
					Language:     metadata.Language,
					IsCached:     true,
					IsReachable:  true,
					CachedAt:     &now,
					LastFetchAt:  &now,
					CreatedAt:    time.Now(),
				}
			}
			
			if err := fc.db.Create(&article).Error; err != nil {
				return fmt.Errorf("failed to create article: %w", err)
			}

			log.Printf("New NewsArticle created with metadata: %s (title: %s)", canonicalURL, article.Title)
		}
	} else if err != nil {
		return fmt.Errorf("failed to query article: %w", err)
	} else {
		// Article exists - check if we should refresh metadata for unreachable articles
		// or articles that haven't been fetched recently
		shouldRefresh := false
		now := time.Now()
		
		// Refresh if article is marked as unreachable (to check if it's become reachable)
		if !article.IsReachable {
			shouldRefresh = true
		}
		
		// Refresh if it's been more than 24 hours since last fetch attempt
		if article.LastFetchAt != nil && time.Since(*article.LastFetchAt) > 24*time.Hour {
			shouldRefresh = true
		}
		
		// Refresh if article has never been fetched
		if article.LastFetchAt == nil {
			shouldRefresh = true
		}
		
		if shouldRefresh {
			log.Printf("Refreshing metadata for existing article: %s", canonicalURL)
			
			// Create context for metadata extraction
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			// Extract metadata from the URL
			metadata, err := fc.metadataExtractor.ExtractMetadata(ctx, canonicalURL)
			
			if err != nil {
				log.Printf("Failed to refresh metadata for %s: %v", canonicalURL, err)
				// Update article to mark as unreachable
				article.IsReachable = false
				article.FetchError = err.Error()
				article.FetchRetries++
				article.LastFetchError = &now
				article.LastFetchAt = &now
			} else {
				// Update article with refreshed metadata
				article.Title = metadata.Title
				article.Description = metadata.Description
				article.Author = metadata.Author
				article.SiteName = metadata.SiteName
				article.ImageURL = metadata.ImageURL
				article.PublishedAt = metadata.PublishedAt
				article.JSONLDData = metadata.JSONLDData
				article.OGData = metadata.OGData
				article.HTMLContent = metadata.HTMLContent
				article.TextContent = metadata.TextContent
				article.WordCount = int(metadata.WordCount)
				article.ReadingTime = int(metadata.ReadingTime)
				article.Language = metadata.Language
				article.IsCached = true
				article.IsReachable = true
				article.FetchError = "" // Clear any previous error
				article.CachedAt = &now
				article.LastFetchAt = &now
			}
			
			// Save the updated article
			if err := fc.db.Save(&article).Error; err != nil {
				log.Printf("Failed to update article %s: %v", canonicalURL, err)
			} else {
				log.Printf("Updated article metadata: %s (reachable: %v)", canonicalURL, article.IsReachable)
			}
		}
	}

	// Create post URI from Jetstream data
	postURI := fmt.Sprintf("at://%s/app.bsky.feed.post/%s", event.DID, event.Commit.RKey)

	// Check if this source article already exists (avoid duplicates)
	var existing models.SourceArticle
	err = fc.db.Where("source_id = ? AND article_id = ? AND post_uri = ?",
		source.ID, article.ID, postURI).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Create new source article record
		sourceArticle := models.SourceArticle{
			SourceID:     source.ID,
			ArticleID:    article.ID,
			PostURI:      postURI,
			PostCID:      event.Commit.CID,
			PostText:     post.Text,
			IsRepost:     fc.isRepost(post),
			PostedAt:     post.CreatedAt,
			LikesCount:   0, // Will be updated by engagement tracking
			RepostsCount: 0, // Will be updated by engagement tracking
			RepliesCount: 0, // Will be updated by engagement tracking
		}

		if err := fc.db.Create(&sourceArticle).Error; err != nil {
			return fmt.Errorf("failed to create source article: %w", err)
		}

		log.Printf("New share tracked: %s shared %s", source.Handle, canonicalURL)

		// TODO: Trigger article content fetching and feed updates
		// This could be done via a message queue or channel

	} else if err != nil {
		return fmt.Errorf("failed to query existing source article: %w", err)
	}

	return nil
}

// isRepost determines if a post is a repost
func (fc *FirehoseConsumer) isRepost(post *PostRecord) bool {
	// A post is a repost if it has a reply parent or if it's very short and contains a link
	return post.Reply != nil || (len(strings.TrimSpace(post.Text)) < 50 && len(post.Facets) > 0)
}

// checkIfNewsArticle validates if a URL contains NewsArticle JSON-LD schema
func (fc *FirehoseConsumer) checkIfNewsArticle(ctx context.Context, articleURL string) (bool, error) {
	// Create a temporary ArticlesService-like client for validation
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", articleURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// Set a reasonable User-Agent
	req.Header.Set("User-Agent", "OpenNews/1.0 (+https://opennews.social)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %w", err)
	}

	htmlContent := string(body)
	
	// Parse HTML and extract JSON-LD
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return false, fmt.Errorf("failed to parse HTML: %w", err)
	}

	jsonldData := fc.extractJSONLD(doc)
	return fc.isNewsArticle(jsonldData), nil
}

// extractJSONLD extracts JSON-LD structured data from HTML
func (fc *FirehoseConsumer) extractJSONLD(n *html.Node) string {
	var jsonldContent string
	fc.findJSONLDScripts(n, &jsonldContent)
	return jsonldContent
}

// findJSONLDScripts recursively searches for script tags with JSON-LD content
func (fc *FirehoseConsumer) findJSONLDScripts(n *html.Node, jsonldData *string) {
	if n.Type == html.ElementNode && n.Data == "script" {
		typeAttr := fc.getAttributeValue(n, "type")
		if typeAttr == "application/ld+json" {
			jsonContent := fc.getTextContent(n)
			if jsonContent != "" && *jsonldData == "" {
				*jsonldData = jsonContent
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		fc.findJSONLDScripts(c, jsonldData)
	}
}

// isNewsArticle checks if the JSON-LD data contains a NewsArticle schema type
func (fc *FirehoseConsumer) isNewsArticle(jsonldData string) bool {
	if jsonldData == "" {
		return false
	}

	// Parse JSON-LD data
	var jsonLD interface{}
	if err := json.Unmarshal([]byte(jsonldData), &jsonLD); err != nil {
		return false
	}

	// Check if it's an array of JSON-LD objects
	if jsonArray, ok := jsonLD.([]interface{}); ok {
		for _, item := range jsonArray {
			if fc.checkForNewsArticleType(item) {
				return true
			}
		}
		return false
	}

	// Check if it's a single JSON-LD object
	return fc.checkForNewsArticleType(jsonLD)
}

// checkForNewsArticleType checks if a JSON-LD object has @type of NewsArticle
func (fc *FirehoseConsumer) checkForNewsArticleType(obj interface{}) bool {
	jsonObj, ok := obj.(map[string]interface{})
	if !ok {
		return false
	}

	// Check @type field
	typeField, exists := jsonObj["@type"]
	if !exists {
		return false
	}

	// @type can be a string or array of strings
	switch t := typeField.(type) {
	case string:
		return t == "NewsArticle"
	case []interface{}:
		for _, typeName := range t {
			if typeStr, ok := typeName.(string); ok && typeStr == "NewsArticle" {
				return true
			}
		}
	}

	return false
}

// getTextContent recursively extracts text content from HTML nodes
func (fc *FirehoseConsumer) getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(fc.getTextContent(c))
	}

	return text.String()
}

// getAttributeValue gets the value of an HTML attribute
func (fc *FirehoseConsumer) getAttributeValue(n *html.Node, attrName string) string {
	for _, attr := range n.Attr {
		if attr.Key == attrName {
			return attr.Val
		}
	}
	return ""
}

// isReachabilityError determines if an error is due to network/reachability issues
// rather than content validation issues
func (fc *FirehoseConsumer) isReachabilityError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	
	// Network/connectivity issues
	if strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "connection timeout") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "network unreachable") ||
		strings.Contains(errStr, "temporary failure") {
		return true
	}
	
	// HTTP errors that suggest temporary issues
	if strings.Contains(errStr, "HTTP 5") || // 5xx server errors
		strings.Contains(errStr, "HTTP 429") || // rate limiting
		strings.Contains(errStr, "HTTP 408") || // request timeout
		strings.Contains(errStr, "HTTP 502") || // bad gateway
		strings.Contains(errStr, "HTTP 503") || // service unavailable
		strings.Contains(errStr, "HTTP 504") { // gateway timeout
		return true
	}
	
	return false
}
