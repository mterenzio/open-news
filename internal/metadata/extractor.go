package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// ArticleMetadata represents extracted metadata from an article
type ArticleMetadata struct {
	Title       string
	Description string
	Author      string
	SiteName    string
	ImageURL    string
	PublishedAt *time.Time
	JSONLDData  string
	OGData      string
	HTMLContent string
	TextContent string
	WordCount   int64
	ReadingTime int64
	Language    string
}

// MetadataExtractor handles extracting metadata from web articles
type MetadataExtractor struct {
	httpClient *http.Client
}

// NewMetadataExtractor creates a new metadata extractor
func NewMetadataExtractor() *MetadataExtractor {
	return &MetadataExtractor{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
		},
	}
}

// ExtractMetadata fetches and extracts full metadata from an article URL
func (me *MetadataExtractor) ExtractMetadata(ctx context.Context, articleURL string) (*ArticleMetadata, error) {
	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", articleURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set appropriate headers
	req.Header.Set("User-Agent", "OpenNews/1.0 (+https://opennews.social)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	// Remove Accept-Encoding to let Go's HTTP client handle compression automatically
	req.Header.Set("Connection", "keep-alive")

	// Make HTTP request
	resp, err := me.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	htmlContent := string(body)

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract metadata
	metadata := &ArticleMetadata{
		HTMLContent: htmlContent,
	}

	me.extractOGData(doc, metadata)
	me.extractJSONLD(doc, metadata)
	me.extractTitle(doc, metadata)
	me.extractDescription(doc, metadata)
	me.extractAuthor(doc, metadata)
	me.extractSiteName(doc, metadata)
	me.extractImageURL(doc, metadata)
	me.extractPublishedDate(doc, metadata)
	me.extractTextContent(doc, metadata)
	me.extractLanguage(doc, metadata)

	// Calculate reading time (average 200 words per minute)
	if metadata.WordCount > 0 {
		metadata.ReadingTime = int64((float64(metadata.WordCount) / 200.0) + 0.5)
		if metadata.ReadingTime < 1 {
			metadata.ReadingTime = 1
		}
	}

	return metadata, nil
}

func (me *MetadataExtractor) extractOGData(doc *html.Node, metadata *ArticleMetadata) {
	ogData := make(map[string]string)
	
	var findMeta func(*html.Node)
	findMeta = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var property, content string
			for _, attr := range n.Attr {
				if attr.Key == "property" && strings.HasPrefix(attr.Val, "og:") {
					property = attr.Val
				} else if attr.Key == "content" {
					content = attr.Val
				}
			}
			if property != "" && content != "" {
				ogData[property] = content
				
				// Extract specific fields
				switch property {
				case "og:title":
					if metadata.Title == "" {
						metadata.Title = content
					}
				case "og:description":
					if metadata.Description == "" {
						metadata.Description = content
					}
				case "og:image":
					if metadata.ImageURL == "" {
						metadata.ImageURL = content
					}
				case "og:site_name":
					if metadata.SiteName == "" {
						metadata.SiteName = content
					}
				}
			}
		}
		
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findMeta(c)
		}
	}
	
	findMeta(doc)
	
	if len(ogData) > 0 {
		if jsonData, err := json.Marshal(ogData); err == nil {
			metadata.OGData = string(jsonData)
		}
	}
}

func (me *MetadataExtractor) extractJSONLD(doc *html.Node, metadata *ArticleMetadata) {
	var findJSONLD func(*html.Node)
	findJSONLD = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			for _, attr := range n.Attr {
				if attr.Key == "type" && attr.Val == "application/ld+json" {
					if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
						jsonldText := strings.TrimSpace(n.FirstChild.Data)
						if jsonldText != "" {
							metadata.JSONLDData = jsonldText
							me.extractFromJSONLD(jsonldText, metadata)
						}
					}
					return
				}
			}
		}
		
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findJSONLD(c)
		}
	}
	
	findJSONLD(doc)
}

func (me *MetadataExtractor) extractFromJSONLD(jsonldText string, metadata *ArticleMetadata) {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonldText), &data); err != nil {
		return
	}
	
	var processItem func(interface{})
	processItem = func(item interface{}) {
		if obj, ok := item.(map[string]interface{}); ok {
			if typeVal, exists := obj["@type"]; exists {
				if typeStr, ok := typeVal.(string); ok && (typeStr == "NewsArticle" || typeStr == "Article") {
					// Extract article data
					if headline, ok := obj["headline"].(string); ok && metadata.Title == "" {
						metadata.Title = headline
					}
					if description, ok := obj["description"].(string); ok && metadata.Description == "" {
						metadata.Description = description
					}
					if author, ok := obj["author"]; ok {
						if authorObj, ok := author.(map[string]interface{}); ok {
							if name, ok := authorObj["name"].(string); ok && metadata.Author == "" {
								metadata.Author = name
							}
						}
					}
					if publisher, ok := obj["publisher"]; ok {
						if pubObj, ok := publisher.(map[string]interface{}); ok {
							if name, ok := pubObj["name"].(string); ok && metadata.SiteName == "" {
								metadata.SiteName = name
							}
						}
					}
					if image, ok := obj["image"]; ok {
						if imageStr, ok := image.(string); ok && metadata.ImageURL == "" {
							metadata.ImageURL = imageStr
						} else if imageArr, ok := image.([]interface{}); ok && len(imageArr) > 0 {
							if imageObj, ok := imageArr[0].(map[string]interface{}); ok {
								if url, ok := imageObj["url"].(string); ok && metadata.ImageURL == "" {
									metadata.ImageURL = url
								}
							}
						}
					}
					if datePublished, ok := obj["datePublished"].(string); ok && metadata.PublishedAt == nil {
						if parsedTime, err := time.Parse(time.RFC3339, datePublished); err == nil {
							metadata.PublishedAt = &parsedTime
						}
					}
				}
			}
		} else if arr, ok := item.([]interface{}); ok {
			for _, subItem := range arr {
				processItem(subItem)
			}
		}
	}
	
	processItem(data)
}

func (me *MetadataExtractor) extractTitle(doc *html.Node, metadata *ArticleMetadata) {
	if metadata.Title != "" {
		return
	}
	
	var findTitle func(*html.Node) string
	findTitle = func(n *html.Node) string {
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				return strings.TrimSpace(n.FirstChild.Data)
			}
		}
		
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if title := findTitle(c); title != "" {
				return title
			}
		}
		return ""
	}
	
	metadata.Title = findTitle(doc)
}

func (me *MetadataExtractor) extractDescription(doc *html.Node, metadata *ArticleMetadata) {
	if metadata.Description != "" {
		return
	}
	
	var findMeta func(*html.Node)
	findMeta = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var name, content string
			for _, attr := range n.Attr {
				if attr.Key == "name" && (attr.Val == "description" || attr.Val == "twitter:description") {
					name = attr.Val
				} else if attr.Key == "content" {
					content = attr.Val
				}
			}
			if name != "" && content != "" && metadata.Description == "" {
				metadata.Description = content
			}
		}
		
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findMeta(c)
		}
	}
	
	findMeta(doc)
}

func (me *MetadataExtractor) extractAuthor(doc *html.Node, metadata *ArticleMetadata) {
	if metadata.Author != "" {
		return
	}
	
	var findMeta func(*html.Node)
	findMeta = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var name, content string
			for _, attr := range n.Attr {
				if attr.Key == "name" && (attr.Val == "author" || attr.Val == "article:author") {
					name = attr.Val
				} else if attr.Key == "content" {
					content = attr.Val
				}
			}
			if name != "" && content != "" && metadata.Author == "" {
				metadata.Author = content
			}
		}
		
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findMeta(c)
		}
	}
	
	findMeta(doc)
}

func (me *MetadataExtractor) extractSiteName(doc *html.Node, metadata *ArticleMetadata) {
	if metadata.SiteName != "" {
		return
	}
	
	var findMeta func(*html.Node)
	findMeta = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var property, content string
			for _, attr := range n.Attr {
				if attr.Key == "property" && attr.Val == "og:site_name" {
					property = attr.Val
				} else if attr.Key == "content" {
					content = attr.Val
				}
			}
			if property != "" && content != "" && metadata.SiteName == "" {
				metadata.SiteName = content
			}
		}
		
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findMeta(c)
		}
	}
	
	findMeta(doc)
}

func (me *MetadataExtractor) extractImageURL(doc *html.Node, metadata *ArticleMetadata) {
	if metadata.ImageURL != "" {
		return
	}
	
	var findMeta func(*html.Node)
	findMeta = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var property, content string
			for _, attr := range n.Attr {
				if attr.Key == "property" && (attr.Val == "og:image" || attr.Val == "twitter:image") {
					property = attr.Val
				} else if attr.Key == "content" {
					content = attr.Val
				}
			}
			if property != "" && content != "" && metadata.ImageURL == "" {
				metadata.ImageURL = content
			}
		}
		
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findMeta(c)
		}
	}
	
	findMeta(doc)
}

func (me *MetadataExtractor) extractPublishedDate(doc *html.Node, metadata *ArticleMetadata) {
	if metadata.PublishedAt != nil {
		return
	}
	
	var findMeta func(*html.Node)
	findMeta = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var property, content string
			for _, attr := range n.Attr {
				if attr.Key == "property" && (attr.Val == "article:published_time" || attr.Val == "article:published") {
					property = attr.Val
				} else if attr.Key == "content" {
					content = attr.Val
				}
			}
			if property != "" && content != "" && metadata.PublishedAt == nil {
				if parsedTime, err := time.Parse(time.RFC3339, content); err == nil {
					metadata.PublishedAt = &parsedTime
				}
			}
		}
		
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findMeta(c)
		}
	}
	
	findMeta(doc)
}

func (me *MetadataExtractor) extractTextContent(doc *html.Node, metadata *ArticleMetadata) {
	var extractText func(*html.Node) string
	extractText = func(n *html.Node) string {
		// Skip script and style elements
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return ""
		}
		
		var text strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode {
				text.WriteString(c.Data)
			} else if c.Type == html.ElementNode {
				childText := extractText(c)
				if childText != "" {
					if text.Len() > 0 {
						text.WriteString(" ")
					}
					text.WriteString(childText)
				}
			}
		}
		return text.String()
	}
	
	rawText := extractText(doc)
	
	// Clean up the text
	re := regexp.MustCompile(`\s+`)
	cleanText := re.ReplaceAllString(strings.TrimSpace(rawText), " ")
	
	metadata.TextContent = cleanText
	
	// Count words
	if cleanText != "" {
		words := strings.Fields(cleanText)
		metadata.WordCount = int64(len(words))
	}
}

func (me *MetadataExtractor) extractLanguage(doc *html.Node, metadata *ArticleMetadata) {
	var findLang func(*html.Node) string
	findLang = func(n *html.Node) string {
		if n.Type == html.ElementNode && n.Data == "html" {
			for _, attr := range n.Attr {
				if attr.Key == "lang" {
					return attr.Val
				}
			}
		}
		
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if lang := findLang(c); lang != "" {
				return lang
			}
		}
		return ""
	}
	
	metadata.Language = findLang(doc)
}
