package worker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"open-news/internal/models"

	"golang.org/x/net/html"
	"gorm.io/gorm"
)

// ArticleFetcher handles fetching and caching article content
type ArticleFetcher struct {
	db         *gorm.DB
	httpClient *http.Client
}

// NewArticleFetcher creates a new article fetcher
func NewArticleFetcher(db *gorm.DB) *ArticleFetcher {
	return &ArticleFetcher{
		db: db,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Follow up to 10 redirects
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
		},
	}
}

// FetchAndCacheArticle fetches article content and metadata, then caches it
func (af *ArticleFetcher) FetchAndCacheArticle(ctx context.Context, articleID string) error {
	// Get the article from database
	var article models.Article
	if err := af.db.Where("id = ?", articleID).First(&article).Error; err != nil {
		return fmt.Errorf("failed to find article: %w", err)
	}

	// Skip if already cached
	if article.IsCached {
		return nil
	}

	// Fetch the article content
	content, err := af.fetchArticleContent(ctx, article.URL)
	if err != nil {
		return fmt.Errorf("failed to fetch article content: %w", err)
	}

	// Extract metadata from HTML
	metadata := af.extractMetadata(content.HTML)

	// Update article with fetched content and metadata
	now := time.Now()
	updateData := map[string]interface{}{
		"title":         coalesce(metadata.Title, article.Title),
		"description":   coalesce(metadata.Description, article.Description),
		"author":        coalesce(metadata.Author, article.Author),
		"site_name":     coalesce(metadata.SiteName, article.SiteName),
		"image_url":     coalesce(metadata.ImageURL, article.ImageURL),
		"published_at":  metadata.PublishedAt,
		"html_content":  content.HTML,
		"text_content":  content.Text,
		"word_count":    content.WordCount,
		"reading_time":  calculateReadingTime(content.WordCount),
		"language":      metadata.Language,
		"og_data":       metadata.OpenGraphJSON,
		"jsonld_data":   metadata.JSONLDDATA,
		"is_cached":     true,
		"cached_at":     &now,
		"last_fetch_at": &now,
		"updated_at":    now,
	}

	if err := af.db.Model(&article).Updates(updateData).Error; err != nil {
		return fmt.Errorf("failed to update article: %w", err)
	}

	return nil
}

// ArticleContent represents the fetched content of an article
type ArticleContent struct {
	HTML      string
	Text      string
	WordCount int
}

// ArticleMetadata represents extracted metadata from an article
type ArticleMetadata struct {
	Title           string
	Description     string
	Author          string
	SiteName        string
	ImageURL        string
	PublishedAt     *time.Time
	Language        string
	OpenGraphJSON   string
	JSONLDDATA      string
}

// fetchArticleContent fetches the HTML content of an article
func (af *ArticleFetcher) fetchArticleContent(ctx context.Context, articleURL string) (*ArticleContent, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", articleURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set a reasonable User-Agent
	req.Header.Set("User-Agent", "OpenNews/1.0 (+https://opennews.social)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := af.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	htmlContent := string(body)
	textContent := af.extractTextFromHTML(htmlContent)
	wordCount := af.countWords(textContent)

	return &ArticleContent{
		HTML:      htmlContent,
		Text:      textContent,
		WordCount: wordCount,
	}, nil
}

// extractMetadata extracts metadata from HTML content
func (af *ArticleFetcher) extractMetadata(htmlContent string) *ArticleMetadata {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return &ArticleMetadata{}
	}

	metadata := &ArticleMetadata{}
	
	// Extract basic metadata
	af.extractBasicMetadata(doc, metadata)
	
	// Extract Open Graph metadata
	af.extractOpenGraphMetadata(doc, metadata)
	
	// Extract JSON-LD data
	af.extractJSONLD(doc, metadata)

	return metadata
}

// extractBasicMetadata extracts basic HTML metadata
func (af *ArticleFetcher) extractBasicMetadata(n *html.Node, metadata *ArticleMetadata) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "title":
			if metadata.Title == "" {
				metadata.Title = af.getTextContent(n)
			}
		case "meta":
			name := af.getAttributeValue(n, "name")
			property := af.getAttributeValue(n, "property")
			content := af.getAttributeValue(n, "content")
			
			switch {
			case name == "description":
				metadata.Description = content
			case name == "author":
				metadata.Author = content
			case name == "language" || name == "lang":
				metadata.Language = content
			case property == "og:title" && metadata.Title == "":
				metadata.Title = content
			case property == "og:description" && metadata.Description == "":
				metadata.Description = content
			case property == "og:image":
				metadata.ImageURL = content
			case property == "og:site_name":
				metadata.SiteName = content
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		af.extractBasicMetadata(c, metadata)
	}
}

// extractOpenGraphMetadata extracts Open Graph metadata
func (af *ArticleFetcher) extractOpenGraphMetadata(n *html.Node, metadata *ArticleMetadata) {
	// This would be implemented to extract and serialize Open Graph data as JSON
	// For now, keeping it simple
}

// extractJSONLD extracts JSON-LD structured data
func (af *ArticleFetcher) extractJSONLD(n *html.Node, metadata *ArticleMetadata) {
	// This would be implemented to extract JSON-LD structured data
	// For now, keeping it simple
}

// extractTextFromHTML extracts plain text from HTML
func (af *ArticleFetcher) extractTextFromHTML(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return ""
	}

	return af.getTextContent(doc)
}

// getTextContent recursively extracts text content from HTML nodes
func (af *ArticleFetcher) getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(af.getTextContent(c))
	}

	return text.String()
}

// getAttributeValue gets the value of an HTML attribute
func (af *ArticleFetcher) getAttributeValue(n *html.Node, attrName string) string {
	for _, attr := range n.Attr {
		if attr.Key == attrName {
			return attr.Val
		}
	}
	return ""
}

// countWords counts words in text
func (af *ArticleFetcher) countWords(text string) int {
	words := strings.Fields(strings.TrimSpace(text))
	return len(words)
}

// calculateReadingTime calculates reading time in minutes based on word count
func calculateReadingTime(wordCount int) int {
	// Average reading speed is about 200-250 words per minute
	// Using 225 as a middle ground
	if wordCount <= 0 {
		return 0
	}
	return (wordCount + 224) / 225 // Round up
}

// coalesce returns the first non-empty string
func coalesce(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
