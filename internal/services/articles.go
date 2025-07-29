package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"open-news/internal/bluesky"
	"open-news/internal/models"

	"github.com/google/uuid"
	"golang.org/x/net/html"
	"gorm.io/gorm"
)

// canonicalizeURL removes tracking parameters and other noise to create a canonical URL
func canonicalizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL // Return original if parsing fails
	}
	
	// Remove common tracking and variant parameters
	query := parsed.Query()
	
	// List of parameters to remove for canonicalization
	paramsToRemove := []string{
		"variant", "utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
		"fbclid", "gclid", "msclkid", "ref", "source", "campaign",
		"_ga", "_gl", "mc_cid", "mc_eid", "yclid",
	}
	
	for _, param := range paramsToRemove {
		query.Del(param)
	}
	
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// CheckIfNewsArticle fetches a URL and checks if it contains NewsArticle JSON-LD schema
func (as *ArticlesService) CheckIfNewsArticle(ctx context.Context, articleURL string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", articleURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// Set a reasonable User-Agent
	req.Header.Set("User-Agent", "OpenNews/1.0 (+https://opennews.social)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := as.httpClient.Do(req)
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

	jsonldData := as.extractJSONLD(doc)
	return as.isNewsArticle(jsonldData), nil
}

// extractJSONLD extracts JSON-LD structured data from HTML
func (as *ArticlesService) extractJSONLD(n *html.Node) string {
	var jsonldContent string
	as.findJSONLDScripts(n, &jsonldContent)
	return jsonldContent
}

// findJSONLDScripts recursively searches for script tags with JSON-LD content
func (as *ArticlesService) findJSONLDScripts(n *html.Node, jsonldData *string) {
	if n.Type == html.ElementNode && n.Data == "script" {
		typeAttr := as.getAttributeValue(n, "type")
		if typeAttr == "application/ld+json" {
			jsonContent := as.getTextContent(n)
			if jsonContent != "" && *jsonldData == "" {
				*jsonldData = jsonContent
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		as.findJSONLDScripts(c, jsonldData)
	}
}

// isNewsArticle checks if the JSON-LD data contains a NewsArticle schema type
func (as *ArticlesService) isNewsArticle(jsonldData string) bool {
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
			if as.checkForNewsArticleType(item) {
				return true
			}
		}
		return false
	}

	// Check if it's a single JSON-LD object
	return as.checkForNewsArticleType(jsonLD)
}

// checkForNewsArticleType checks if a JSON-LD object has @type of NewsArticle
func (as *ArticlesService) checkForNewsArticleType(obj interface{}) bool {
	jsonObj, ok := obj.(map[string]interface{})
	if !ok {
		return false
	}

	// Check for @graph structure (common in JSON-LD)
	if graphField, hasGraph := jsonObj["@graph"]; hasGraph {
		if graphArray, isArray := graphField.([]interface{}); isArray {
			for _, graphItem := range graphArray {
				if as.checkForNewsArticleType(graphItem) {
					return true
				}
			}
		}
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
func (as *ArticlesService) getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(as.getTextContent(c))
	}

	return text.String()
}

// getAttributeValue gets the value of an HTML attribute
func (as *ArticlesService) getAttributeValue(n *html.Node, attrName string) string {
	for _, attr := range n.Attr {
		if attr.Key == attrName {
			return attr.Val
		}
	}
	return ""
}

// ArticlesService handles article import and seeding
type ArticlesService struct {
	db            *gorm.DB
	blueskyClient *bluesky.Client
	httpClient    *http.Client
}

// NewArticlesService creates a new articles service
func NewArticlesService(db *gorm.DB, blueskyClient *bluesky.Client) *ArticlesService {
	return &ArticlesService{
		db:            db,
		blueskyClient: blueskyClient,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Follow up to 5 redirects for checking
				if len(via) >= 5 {
					return fmt.Errorf("stopped after 5 redirects")
				}
				return nil
			},
		},
	}
}

// ArticleMetadata holds extracted metadata from an article
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

// ExtractArticleMetadata fetches and extracts full metadata from an article URL
func (as *ArticlesService) ExtractArticleMetadata(ctx context.Context, articleURL string) (*ArticleMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", articleURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set a reasonable User-Agent
	req.Header.Set("User-Agent", "OpenNews/1.0 (+https://opennews.social)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := as.httpClient.Do(req)
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
	
	// Parse HTML
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	metadata := &ArticleMetadata{
		HTMLContent: htmlContent,
		JSONLDData:  as.extractJSONLD(doc),
		OGData:      as.extractOGData(doc),
	}

	// Extract basic metadata
	metadata.Title = as.extractTitle(doc)
	metadata.Description = as.extractDescription(doc)
	metadata.Author = as.extractAuthor(doc)
	metadata.SiteName = as.extractSiteName(doc)
	metadata.ImageURL = as.extractImageURL(doc)
	metadata.PublishedAt = as.extractPublishedDate(doc, metadata.JSONLDData)
	
	// Extract text content
	metadata.TextContent = as.extractTextContent(doc)
	metadata.WordCount = int64(len(strings.Fields(metadata.TextContent)))
	metadata.ReadingTime = metadata.WordCount / 200 // Assume 200 words per minute
	metadata.Language = as.extractLanguage(doc)

	return metadata, nil
}

// ArticleSeedConfig contains configuration for article seeding
type ArticleSeedConfig struct {
	MaxArticles     int           // Maximum number of articles to create
	TimeWindow      time.Duration // How far back to look for posts
	RateLimit       time.Duration // Rate limiting between API calls
	SampleSources   int           // Number of sources to sample posts from
}

// ImportArticlesFromSources attempts to import recent articles from Bluesky sources
func (as *ArticlesService) ImportArticlesFromSources(config ArticleSeedConfig) error {
	log.Printf("🔄 Starting article import from Bluesky sources...")
	
	// Get sources that users actually follow (from user_sources table)
	var sources []models.Source
	query := `
		SELECT DISTINCT s.* FROM sources s 
		INNER JOIN user_sources us ON s.id = us.source_id 
		LIMIT ?
	`
	if err := as.db.Raw(query, config.SampleSources).Scan(&sources).Error; err != nil {
		return fmt.Errorf("failed to fetch user-followed sources: %w", err)
	}
	
	if len(sources) == 0 {
		return fmt.Errorf("no user-followed sources found in database")
	}
	
	log.Printf("📚 Attempting to import articles from %d user-followed sources...", len(sources))
	
	articlesCreated := 0
	for _, source := range sources {
		if articlesCreated >= config.MaxArticles {
			break
		}
		
		// Try to get recent posts from this source
		if err := as.importFromSource(source, config); err != nil {
			log.Printf("⚠️  Failed to import from %s: %v", source.Handle, err)
			continue
		}
		
		// Rate limiting
		time.Sleep(config.RateLimit)
		
		// Check how many articles we've created so far
		var count int64
		as.db.Model(&models.Article{}).Count(&count)
		articlesCreated = int(count)
	}
	
	if articlesCreated == 0 {
		log.Printf("ℹ️  No articles found in recent posts from followed sources (this is normal)")
		return nil // This is not an error - just no content found
	}
	
	log.Printf("✅ Successfully imported %d articles from Bluesky sources", articlesCreated)
	return nil
}

// importFromSource tries to import articles from a specific source
func (as *ArticlesService) importFromSource(source models.Source, config ArticleSeedConfig) error {
	if as.blueskyClient == nil {
		return fmt.Errorf("authentication required for Bluesky API")
	}

	log.Printf("📡 Importing articles from %s (%s)...", source.DisplayName, source.Handle)
	log.Printf("🔍 Getting posts from DID: %s", source.BlueSkyDID)
	
	// Get recent posts from this author
	posts, err := as.blueskyClient.GetAuthorFeed(source.BlueSkyDID, 20, "")
	if err != nil {
		log.Printf("❌ Failed to get posts from %s: %v", source.Handle, err)
		return fmt.Errorf("failed to get posts from %s: %w", source.Handle, err)
	}

	log.Printf("📊 Retrieved %d posts from %s", len(posts), source.Handle)

	articlesCreated := 0
	for i, post := range posts {
		log.Printf("🔍 Processing post %d: %s", i+1, post.URI)
		
		// Extract links from the post
		links := as.blueskyClient.ExtractLinksFromPost(post)
		log.Printf("🔗 Found %d links in post: %v", len(links), links)
		
		for _, link := range links {
			log.Printf("📰 Checking article for link: %s", link)
			
			canonicalURL := canonicalizeURL(link)
			
			// Check if article already exists
			var existingArticle models.Article
			if err := as.db.Where("url = ?", canonicalURL).First(&existingArticle).Error; err == nil {
				log.Printf("📚 Article already exists for URL: %s", canonicalURL)
				
				// Create source article linking this post to the existing article
				sourceArticle := models.SourceArticle{
					SourceID:  source.ID,
					ArticleID: existingArticle.ID,
					PostURI:   post.URI,
					PostCID:   post.CID,
					PostText:  post.Record.Text,
					PostedAt:  post.Record.CreatedAt,
				}

				if err := as.db.Create(&sourceArticle).Error; err != nil {
					log.Printf("⚠️ Failed to create source article for existing article %s: %v", canonicalURL, err)
				} else {
					log.Printf("✅ Linked existing article to post: %s", existingArticle.Title)
					articlesCreated++
				}
				continue
			}
			
			// Check if the URL contains a NewsArticle schema
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			isNewsArticle, err := as.CheckIfNewsArticle(ctx, canonicalURL)
			cancel()
			
			if err != nil {
				log.Printf("⚠️ Failed to check NewsArticle schema for %s: %v", canonicalURL, err)
				continue
			}
			
			if !isNewsArticle {
				log.Printf("⏭️ Skipping URL (not a NewsArticle): %s", canonicalURL)
				continue
			}
			
			log.Printf("✅ Found NewsArticle schema, extracting metadata for: %s", canonicalURL)
			
			// Extract full metadata from the HTML page
			ctx2, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
			metadata, err := as.ExtractArticleMetadata(ctx2, canonicalURL)
			cancel2()
			
			if err != nil {
				log.Printf("⚠️ Failed to extract metadata for %s: %v", canonicalURL, err)
				continue
			}
			
			// Create article with extracted metadata
			article := models.Article{
				Title:        metadata.Title,
				URL:          canonicalURL,
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
			}

			// Create the article
			if err := as.db.Create(&article).Error; err != nil {
				log.Printf("⚠️ Failed to create article %s: %v", article.URL, err)
				continue
			}

			// Create source article linking this post to the article
			sourceArticle := models.SourceArticle{
				SourceID:  source.ID,
				ArticleID: article.ID,
				PostURI:   post.URI,
				PostCID:   post.CID,
				PostText:  post.Record.Text,
				PostedAt:  post.Record.CreatedAt,
			}

			if err := as.db.Create(&sourceArticle).Error; err != nil {
				log.Printf("⚠️ Failed to create source article for %s: %v", article.URL, err)
				continue
			}

			log.Printf("✅ Successfully created NewsArticle: %s", article.Title)
			articlesCreated++
			if articlesCreated >= config.MaxArticles {
				break
			}
		}
		
		if articlesCreated >= config.MaxArticles {
			break
		}
	}

	log.Printf("✅ Imported %d articles from %s", articlesCreated, source.DisplayName)
	return nil
}

// CreateMockArticles creates realistic mock articles for development/testing
func (as *ArticlesService) CreateMockArticles(config ArticleSeedConfig) error {
	log.Printf("🔄 Creating mock articles for development...")
	
	// Get sources to attribute articles to
	var sources []models.Source
	if err := as.db.Find(&sources).Error; err != nil {
		return fmt.Errorf("failed to fetch sources: %w", err)
	}
	
	if len(sources) == 0 {
		return fmt.Errorf("no sources found - please seed sources first")
	}
	
	mockArticles := as.generateMockArticlesData(config.MaxArticles)
	
	articlesCreated := 0
	for i, articleData := range mockArticles {
		if i >= config.MaxArticles {
			break
		}
		
		// Select a source for this article (round-robin)
		source := sources[i%len(sources)]
		
		// Canonicalize the URL
		canonicalURL := canonicalizeURL(articleData.URL)
		
		// Check if article already exists (using canonical URL)
		var existing models.Article
		if err := as.db.Where("url = ?", canonicalURL).First(&existing).Error; err == nil {
			// Article exists, create source article link with existing article
			sourceArticle := models.SourceArticle{
				SourceID:     source.ID,
				ArticleID:    existing.ID,
				PostURI:      fmt.Sprintf("at://%s/app.bsky.feed.post/mock-%d", source.BlueSkyDID, time.Now().Unix()+int64(i)),
				PostCID:      fmt.Sprintf("bafyrei-mock-%d", time.Now().Unix()+int64(i)),
				PostText:     fmt.Sprintf("%s %s", articleData.PostText, articleData.URL), // Use original URL in post text
				IsRepost:     articleData.IsRepost,
				PostedAt:     articleData.PublishedAt.Add(-time.Duration(i) * time.Hour),
				LikesCount:   articleData.LikesCount,
				RepostsCount: articleData.RepostsCount,
				RepliesCount: articleData.RepliesCount,
				ShareScore:   articleData.ShareScore,
			}
			
			if err := as.db.Create(&sourceArticle).Error; err != nil {
				log.Printf("❌ Failed to create source article: %v", err)
			}
			continue // Skip creating new article, but we created the source article link
		}
		
		// Create the article with canonical URL
		article := models.Article{
			URL:           canonicalURL,
			Title:         articleData.Title,
			Description:   articleData.Description,
			Author:        articleData.Author,
			SiteName:      articleData.SiteName,
			ImageURL:      articleData.ImageURL,
			PublishedAt:   &articleData.PublishedAt,
			WordCount:     articleData.WordCount,
			ReadingTime:   calculateReadingTime(articleData.WordCount),
			Language:      "en",
			QualityScore:  articleData.QualityScore,
			TrendingScore: articleData.TrendingScore,
			IsCached:      false, // Will be cached by workers if needed
		}
		
		if err := as.db.Create(&article).Error; err != nil {
			log.Printf("❌ Failed to create article: %v", err)
			continue
		}
		
		// Create a source article record (the post that shared this article)
		sourceArticle := models.SourceArticle{
			SourceID:     source.ID,
			ArticleID:    article.ID,
			PostURI:      fmt.Sprintf("at://%s/app.bsky.feed.post/mock-%d", source.BlueSkyDID, time.Now().Unix()+int64(i)),
			PostCID:      fmt.Sprintf("bafyrei-mock-%d", time.Now().Unix()+int64(i)),
			PostText:     fmt.Sprintf("%s %s", articleData.PostText, articleData.URL), // Use original URL in post text
			IsRepost:     articleData.IsRepost,
			PostedAt:     articleData.PublishedAt.Add(-time.Duration(i) * time.Hour), // Stagger posting times
			LikesCount:   articleData.LikesCount,
			RepostsCount: articleData.RepostsCount,
			RepliesCount: articleData.RepliesCount,
			ShareScore:   articleData.ShareScore,
		}
		
		if err := as.db.Create(&sourceArticle).Error; err != nil {
			log.Printf("❌ Failed to create source article: %v", err)
			continue
		}
		
		articlesCreated++
	}
	
	log.Printf("✅ Created %d mock articles for testing", articlesCreated)
	return nil
}

// MockArticleData represents mock article data for seeding
type MockArticleData struct {
	URL           string
	Title         string
	Description   string
	Author        string
	SiteName      string
	ImageURL      string
	PublishedAt   time.Time
	WordCount     int
	QualityScore  float64
	TrendingScore float64
	PostText      string
	IsRepost      bool
	LikesCount    int
	RepostsCount  int
	RepliesCount  int
	ShareScore    float64
}

// generateMockArticlesData creates realistic mock article data
func (as *ArticlesService) generateMockArticlesData(maxArticles int) []MockArticleData {
	now := time.Now()
	articles := []MockArticleData{
		{
			URL:           "https://techcrunch.com/2025/01/15/ai-breakthrough-language-models",
			Title:         "Major AI Breakthrough: New Language Models Show Unprecedented Understanding",
			Description:   "Researchers have developed a new class of language models that demonstrate remarkable improvements in reasoning and factual accuracy.",
			Author:        "Sarah Johnson",
			SiteName:      "TechCrunch",
			ImageURL:      "https://picsum.photos/400/300?random=1",
			PublishedAt:   now.Add(-2 * time.Hour),
			WordCount:     1200,
			QualityScore:  0.92,
			TrendingScore: 0.88,
			PostText:      "This is huge for the AI field! 🚀",
			IsRepost:      false,
			LikesCount:    156,
			RepostsCount:  89,
			RepliesCount:  23,
			ShareScore:    0.85,
		},
		{
			URL:           "https://www.reuters.com/technology/space-exploration-milestone-2025-01-15",
			Title:         "SpaceX Achieves New Milestone in Deep Space Exploration",
			Description:   "The company's latest mission marks a significant step forward in making deep space travel more accessible and cost-effective.",
			Author:        "Michael Chen",
			SiteName:      "Reuters",
			ImageURL:      "https://picsum.photos/400/300?random=2",
			PublishedAt:   now.Add(-4 * time.Hour),
			WordCount:     950,
			QualityScore:  0.95,
			TrendingScore: 0.91,
			PostText:      "Space exploration continues to amaze! 🚀✨",
			IsRepost:      false,
			LikesCount:    234,
			RepostsCount:  145,
			RepliesCount:  67,
			ShareScore:    0.93,
		},
		{
			URL:           "https://www.nature.com/articles/climate-science-2025",
			Title:         "Climate Scientists Report Unexpected Positive Trend in Ocean Recovery",
			Description:   "New data suggests that marine ecosystems may be recovering faster than previously predicted due to recent conservation efforts.",
			Author:        "Dr. Emily Rodriguez",
			SiteName:      "Nature",
			ImageURL:      "https://picsum.photos/400/300?random=3",
			PublishedAt:   now.Add(-6 * time.Hour),
			WordCount:     1800,
			QualityScore:  0.98,
			TrendingScore: 0.75,
			PostText:      "Finally some good climate news! 🌊🐠",
			IsRepost:      false,
			LikesCount:    89,
			RepostsCount:  56,
			RepliesCount:  34,
			ShareScore:    0.78,
		},
		{
			URL:           "https://www.economist.com/finance/crypto-regulation-2025",
			Title:         "Global Cryptocurrency Regulation Reaches Historic Agreement",
			Description:   "Major economies have reached consensus on a unified framework for cryptocurrency regulation, providing clarity for the industry.",
			Author:        "James Patterson",
			SiteName:      "The Economist",
			ImageURL:      "https://picsum.photos/400/300?random=4",
			PublishedAt:   now.Add(-8 * time.Hour),
			WordCount:     1350,
			QualityScore:  0.89,
			TrendingScore: 0.82,
			PostText:      "This could change everything for crypto 💱",
			IsRepost:      true,
			LikesCount:    178,
			RepostsCount:  92,
			RepliesCount:  45,
			ShareScore:    0.81,
		},
		{
			URL:           "https://arxiv.org/abs/2025.01234",
			Title:         "Quantum Computing Breakthrough: Error Correction Reaches Practical Threshold",
			Description:   "Researchers demonstrate quantum error correction at a scale that makes practical quantum computing applications feasible.",
			Author:        "Prof. Lisa Zhang et al.",
			SiteName:      "arXiv",
			ImageURL:      "https://picsum.photos/400/300?random=5",
			PublishedAt:   now.Add(-12 * time.Hour),
			WordCount:     2200,
			QualityScore:  0.94,
			TrendingScore: 0.69,
			PostText:      "Quantum computing is getting real! ⚛️",
			IsRepost:      false,
			LikesCount:    67,
			RepostsCount:  34,
			RepliesCount:  12,
			ShareScore:    0.71,
		},
		{
			URL:           "https://www.bbc.com/news/health-medical-breakthrough-2025",
			Title:         "New Gene Therapy Shows Promise for Treating Rare Childhood Diseases",
			Description:   "Clinical trials reveal significant improvements in patients with previously incurable genetic conditions.",
			Author:        "Dr. Robert Thompson",
			SiteName:      "BBC News",
			ImageURL:      "https://picsum.photos/400/300?random=6",
			PublishedAt:   now.Add(-16 * time.Hour),
			WordCount:     1100,
			QualityScore:  0.91,
			TrendingScore: 0.77,
			PostText:      "Hope for families dealing with rare diseases 💙",
			IsRepost:      false,
			LikesCount:    145,
			RepostsCount:  78,
			RepliesCount:  29,
			ShareScore:    0.79,
		},
		{
			URL:           "https://www.wired.com/story/renewable-energy-storage-breakthrough",
			Title:         "Revolutionary Battery Technology Could Solve Renewable Energy Storage",
			Description:   "New solid-state batteries demonstrate unprecedented energy density and durability for grid-scale storage applications.",
			Author:        "Alex Murphy",
			SiteName:      "WIRED",
			ImageURL:      "https://picsum.photos/400/300?random=7",
			PublishedAt:   now.Add(-20 * time.Hour),
			WordCount:     1450,
			QualityScore:  0.87,
			TrendingScore: 0.84,
			PostText:      "This could accelerate the green energy transition ⚡🔋",
			IsRepost:      false,
			LikesCount:    203,
			RepostsCount:  112,
			RepliesCount:  56,
			ShareScore:    0.86,
		},
		{
			URL:           "https://www.theguardian.com/technology/ai-ethics-2025",
			Title:         "Tech Giants Announce Joint AI Ethics Initiative",
			Description:   "Major technology companies commit to new standards for responsible AI development and deployment.",
			Author:        "Maria Gonzalez",
			SiteName:      "The Guardian",
			ImageURL:      "https://picsum.photos/400/300?random=8",
			PublishedAt:   now.Add(-24 * time.Hour),
			WordCount:     1000,
			QualityScore:  0.85,
			TrendingScore: 0.73,
			PostText:      "About time we see more responsible AI development 🤖",
			IsRepost:      true,
			LikesCount:    98,
			RepostsCount:  52,
			RepliesCount:  31,
			ShareScore:    0.75,
		},
	}
	
	// If we need more articles, duplicate and modify the existing ones
	if maxArticles > len(articles) {
		for i := len(articles); i < maxArticles; i++ {
			// Take a base article and modify it
			base := articles[i%len(articles)]
			modified := base
			modified.URL = fmt.Sprintf("%s?variant=%d", base.URL, i)
			modified.Title = fmt.Sprintf("%s (Part %d)", base.Title, (i/len(articles))+1)
			modified.PublishedAt = now.Add(-time.Duration(i*2) * time.Hour)
			modified.LikesCount = base.LikesCount + (i * 5)
			modified.RepostsCount = base.RepostsCount + (i * 2)
			modified.QualityScore = base.QualityScore * (0.95 + float64(i%10)*0.01)
			articles = append(articles, modified)
		}
	}
	
	return articles[:maxArticles]
}

// extractOGData extracts Open Graph metadata from HTML
func (as *ArticlesService) extractOGData(doc *html.Node) string {
	var ogData strings.Builder
	ogData.WriteString("{")
	
	as.findMetaTags(doc, func(name, content string) {
		if strings.HasPrefix(name, "og:") {
			if ogData.Len() > 1 {
				ogData.WriteString(",")
			}
			ogData.WriteString(fmt.Sprintf(`"%s":"%s"`, name, strings.ReplaceAll(content, `"`, `\"`)))
		}
	})
	
	ogData.WriteString("}")
	return ogData.String()
}

// extractTitle extracts the title from HTML
func (as *ArticlesService) extractTitle(doc *html.Node) string {
	// Try OG title first
	if title := as.extractMetaContent(doc, "og:title"); title != "" {
		return title
	}
	
	// Try JSON-LD title
	if title := as.extractJSONLDField(doc, "headline"); title != "" {
		return title
	}
	
	// Fall back to HTML title tag
	return as.extractHTMLTitle(doc)
}

// extractDescription extracts the description from HTML
func (as *ArticlesService) extractDescription(doc *html.Node) string {
	// Try OG description first
	if desc := as.extractMetaContent(doc, "og:description"); desc != "" {
		return desc
	}
	
	// Try meta description
	if desc := as.extractMetaContent(doc, "description"); desc != "" {
		return desc
	}
	
	// Try JSON-LD description
	return as.extractJSONLDField(doc, "description")
}

// extractAuthor extracts the author from HTML
func (as *ArticlesService) extractAuthor(doc *html.Node) string {
	// Try JSON-LD author
	if author := as.extractJSONLDField(doc, "author"); author != "" {
		return author
	}
	
	// Try meta author
	return as.extractMetaContent(doc, "author")
}

// extractSiteName extracts the site name from HTML
func (as *ArticlesService) extractSiteName(doc *html.Node) string {
	// Try OG site name
	if siteName := as.extractMetaContent(doc, "og:site_name"); siteName != "" {
		return siteName
	}
	
	// Try JSON-LD publisher
	return as.extractJSONLDField(doc, "publisher")
}

// extractImageURL extracts the main image URL from HTML
func (as *ArticlesService) extractImageURL(doc *html.Node) string {
	// Try OG image
	if image := as.extractMetaContent(doc, "og:image"); image != "" {
		return image
	}
	
	// Try JSON-LD image
	return as.extractJSONLDField(doc, "image")
}

// extractPublishedDate extracts the published date from HTML
func (as *ArticlesService) extractPublishedDate(doc *html.Node, jsonldData string) *time.Time {
	// Try JSON-LD datePublished
	if dateStr := as.extractJSONLDField(doc, "datePublished"); dateStr != "" {
		if date, err := time.Parse(time.RFC3339, dateStr); err == nil {
			return &date
		}
	}
	
	// Try meta article:published_time
	if dateStr := as.extractMetaContent(doc, "article:published_time"); dateStr != "" {
		if date, err := time.Parse(time.RFC3339, dateStr); err == nil {
			return &date
		}
	}
	
	return nil
}

// extractTextContent extracts clean text content from HTML
func (as *ArticlesService) extractTextContent(doc *html.Node) string {
	// Find the main content area (article, main, or body)
	content := as.findMainContent(doc)
	if content == nil {
		content = doc
	}
	
	return strings.TrimSpace(as.getTextContent(content))
}

// extractLanguage extracts the language from HTML
func (as *ArticlesService) extractLanguage(doc *html.Node) string {
	// Try html lang attribute
	if doc.Type == html.ElementNode && doc.Data == "html" {
		return as.getAttributeValue(doc, "lang")
	}
	
	// Search for html tag
	var lang string
	as.findElementRecursive(doc, "html", func(n *html.Node) {
		lang = as.getAttributeValue(n, "lang")
	})
	
	return lang
}

// Helper functions for metadata extraction

// findMetaTags finds all meta tags and calls the callback for each
func (as *ArticlesService) findMetaTags(n *html.Node, callback func(name, content string)) {
	if n.Type == html.ElementNode && n.Data == "meta" {
		name := as.getAttributeValue(n, "name")
		property := as.getAttributeValue(n, "property")
		content := as.getAttributeValue(n, "content")
		
		if name != "" && content != "" {
			callback(name, content)
		}
		if property != "" && content != "" {
			callback(property, content)
		}
	}
	
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		as.findMetaTags(c, callback)
	}
}

// extractMetaContent extracts content from a meta tag by name or property
func (as *ArticlesService) extractMetaContent(doc *html.Node, nameOrProperty string) string {
	var content string
	as.findMetaTags(doc, func(name, value string) {
		if name == nameOrProperty && content == "" {
			content = value
		}
	})
	return content
}

// extractHTMLTitle extracts the title from HTML title tag
func (as *ArticlesService) extractHTMLTitle(doc *html.Node) string {
	var title string
	as.findElementRecursive(doc, "title", func(n *html.Node) {
		if title == "" {
			title = as.getTextContent(n)
		}
	})
	return title
}

// extractJSONLDField extracts a field from JSON-LD data
func (as *ArticlesService) extractJSONLDField(doc *html.Node, field string) string {
	jsonldData := as.extractJSONLD(doc)
	if jsonldData == "" {
		return ""
	}
	
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonldData), &data); err != nil {
		return ""
	}
	
	if value, exists := data[field]; exists {
		if str, ok := value.(string); ok {
			return str
		}
		// Handle nested objects (like author)
		if obj, ok := value.(map[string]interface{}); ok {
			if name, exists := obj["name"]; exists {
				if nameStr, ok := name.(string); ok {
					return nameStr
				}
			}
		}
	}
	
	return ""
}

// findMainContent finds the main content area in HTML
func (as *ArticlesService) findMainContent(doc *html.Node) *html.Node {
	// Look for article, main, or content divs
	var content *html.Node
	
	as.findElementRecursive(doc, "article", func(n *html.Node) {
		if content == nil {
			content = n
		}
	})
	
	if content != nil {
		return content
	}
	
	as.findElementRecursive(doc, "main", func(n *html.Node) {
		if content == nil {
			content = n
		}
	})
	
	return content
}

// findElementRecursive finds elements by tag name
func (as *ArticlesService) findElementRecursive(n *html.Node, tagName string, callback func(*html.Node)) {
	if n.Type == html.ElementNode && n.Data == tagName {
		callback(n)
	}
	
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		as.findElementRecursive(c, tagName, callback)
	}
}

// calculateReadingTime estimates reading time based on word count
func calculateReadingTime(wordCount int) int {
	// Average reading speed is about 200-250 words per minute
	// We'll use 225 as a middle ground
	readingTime := wordCount / 225
	if readingTime < 1 {
		readingTime = 1
	}
	return readingTime
}

// isValidURL checks if a string is a valid URL
func isValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// extractDomainFromURL extracts the domain from a URL
func extractDomainFromURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(u.Host, "www.")
}

// extractTitleFromPost extracts a title from a post text and link
func extractTitleFromPost(postText, link string) string {
	// For now, use the domain name and post text as title
	// In a real implementation, this might fetch the page and extract the actual title
	domain := extractDomainFromURL(link)
	truncated := truncateText(postText, 100)
	if truncated != "" {
		return fmt.Sprintf("%s - %s", domain, truncated)
	}
	return domain
}

// truncateText truncates text to a maximum length, adding ellipsis if needed
func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength-3] + "..."
}

// ValidateAndCleanupExistingArticles validates existing articles and removes those without proper NewsArticle schema
func (as *ArticlesService) ValidateAndCleanupExistingArticles(dryRun bool) error {
	log.Printf("🔍 Starting validation of existing articles (dry run: %v)...", dryRun)
	
	var articles []models.Article
	if err := as.db.Find(&articles).Error; err != nil {
		return fmt.Errorf("failed to fetch articles: %w", err)
	}

	log.Printf("📊 Found %d articles to validate", len(articles))
	
	invalidCount := 0
	validCount := 0
	errorCount := 0

	for i, article := range articles {
		log.Printf("🔍 Validating article %d/%d: %s", i+1, len(articles), article.URL)
		
		// Check if article has JSON-LD data with NewsArticle type
		if article.JSONLDData == "" {
			log.Printf("❌ Article has no JSON-LD data: %s", article.URL)
			invalidCount++
			
			if !dryRun {
				if err := as.deleteArticleAndReferences(article.ID); err != nil {
					log.Printf("⚠️ Failed to delete article %s: %v", article.URL, err)
					errorCount++
				} else {
					log.Printf("🗑️ Deleted invalid article: %s", article.URL)
				}
			}
			continue
		}

		// Parse and validate JSON-LD
		if !as.isNewsArticle(article.JSONLDData) {
			log.Printf("❌ Article JSON-LD is not NewsArticle type: %s", article.URL)
			invalidCount++
			
			if !dryRun {
				if err := as.deleteArticleAndReferences(article.ID); err != nil {
					log.Printf("⚠️ Failed to delete article %s: %v", article.URL, err)
					errorCount++
				} else {
					log.Printf("🗑️ Deleted invalid article: %s", article.URL)
				}
			}
			continue
		}

		validCount++
		log.Printf("✅ Article validated as NewsArticle: %s", article.URL)
	}

	log.Printf("📊 Validation complete:")
	log.Printf("   ✅ Valid articles: %d", validCount)
	log.Printf("   ❌ Invalid articles: %d", invalidCount)
	log.Printf("   ⚠️ Errors: %d", errorCount)
	
	if dryRun {
		log.Printf("🔍 This was a dry run - no articles were deleted")
		log.Printf("💡 Run with dryRun=false to actually remove invalid articles")
	}

	return nil
}

// deleteArticleAndReferences deletes an article and all its related data
func (as *ArticlesService) deleteArticleAndReferences(articleID uuid.UUID) error {
	// Delete in reverse order of foreign key dependencies
	
	// Delete article facts
	if err := as.db.Where("article_id = ?", articleID).Delete(&models.ArticleFact{}).Error; err != nil {
		return fmt.Errorf("failed to delete article facts: %w", err)
	}
	
	// Delete source articles
	if err := as.db.Where("article_id = ?", articleID).Delete(&models.SourceArticle{}).Error; err != nil {
		return fmt.Errorf("failed to delete source articles: %w", err)
	}
	
	// Finally delete the article itself
	if err := as.db.Delete(&models.Article{}, articleID).Error; err != nil {
		return fmt.Errorf("failed to delete article: %w", err)
	}

	return nil
}
