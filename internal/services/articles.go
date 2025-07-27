package services

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"open-news/internal/bluesky"
	"open-news/internal/models"

	"gorm.io/gorm"
)

// ArticlesService handles article import and seeding
type ArticlesService struct {
	db            *gorm.DB
	blueskyClient *bluesky.Client
}

// NewArticlesService creates a new articles service
func NewArticlesService(db *gorm.DB, blueskyClient *bluesky.Client) *ArticlesService {
	return &ArticlesService{
		db:            db,
		blueskyClient: blueskyClient,
	}
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
	
	// Get active sources from database
	var sources []models.Source
	if err := as.db.Limit(config.SampleSources).Find(&sources).Error; err != nil {
		return fmt.Errorf("failed to fetch sources: %w", err)
	}
	
	if len(sources) == 0 {
		return fmt.Errorf("no sources found in database")
	}
	
	log.Printf("📚 Attempting to import articles from %d sources...", len(sources))
	
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
		return fmt.Errorf("no articles were imported from Bluesky")
	}
	
	log.Printf("✅ Successfully imported %d articles from Bluesky sources", articlesCreated)
	return nil
}

// importFromSource tries to import articles from a specific source
func (as *ArticlesService) importFromSource(source models.Source, config ArticleSeedConfig) error {
	// Note: This would use the Bluesky API to get recent posts from the source
	// For now, we'll return an error since we don't have authentication
	// In production, this would call as.blueskyClient.GetRecentPosts(source.BlueSkyDID)
	return fmt.Errorf("authentication required for Bluesky API")
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
		
		// Create the article
		article := models.Article{
			URL:           articleData.URL,
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
		
		// Check if article already exists
		var existing models.Article
		if err := as.db.Where("url = ?", article.URL).First(&existing).Error; err == nil {
			continue // Article already exists
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
			PostText:     fmt.Sprintf("%s %s", articleData.PostText, article.URL),
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
