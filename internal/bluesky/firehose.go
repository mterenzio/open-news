package bluesky

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"open-news/internal/models"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// FirehoseConsumer handles the Bluesky Jetstream connection and processing
type FirehoseConsumer struct {
	db     *gorm.DB
	client *Client
	dialer *websocket.Dialer
}

// NewFirehoseConsumer creates a new firehose consumer
func NewFirehoseConsumer(db *gorm.DB, client *Client) *FirehoseConsumer {
	return &FirehoseConsumer{
		db:     db,
		client: client,
		dialer: websocket.DefaultDialer,
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
		// Article doesn't exist, create it
		article = models.Article{
			URL:       canonicalURL,
			IsCached:  false,
			CreatedAt: time.Now(),
		}
		if err := fc.db.Create(&article).Error; err != nil {
			return fmt.Errorf("failed to create article: %w", err)
		}

		log.Printf("New article discovered: %s", canonicalURL)
	} else if err != nil {
		return fmt.Errorf("failed to query article: %w", err)
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
