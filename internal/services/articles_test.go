package services

import (
	"os"
	"testing"
	"time"

	"open-news/internal/database"
	"open-news/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// Set test environment variables
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "mterenzi")
	os.Setenv("DB_PASSWORD", "")
	os.Setenv("DB_NAME", "open_news_test")
	os.Setenv("DB_SSLMODE", "disable")

	// Load test database configuration
	config := database.LoadConfig()
	
	// Connect to test database
	err := database.Connect(config)
	if err != nil {
		t.Skipf("Skipping test - PostgreSQL test database not available: %v", err)
	}

	db := database.DB

	// Run migrations to ensure schema is up to date
	err = db.AutoMigrate(
		&models.User{},
		&models.Source{},
		&models.Article{},
		&models.SourceArticle{},
		&models.Feed{},
		&models.ArticleFact{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	// Clean up any existing test data
	db.Exec("TRUNCATE TABLE source_articles, article_facts, articles, user_sources, sources, users, feeds RESTART IDENTITY CASCADE")

	return db
}

func TestCreateMockArticles(t *testing.T) {
	db := setupTestDB(t)
	
	// Create test source
	source := &models.Source{
		ID:          uuid.New(),
		Handle:      "testnews.bsky.social",
		BlueSkyDID:  "did:plc:test123456789",
		DisplayName: "Test News",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	db.Create(source)

	// Create articles service (without bluesky client for this test)
	service := &ArticlesService{
		db:            db,
		blueskyClient: nil,
	}

	// Test creating mock articles with config
	config := ArticleSeedConfig{
		MaxArticles: 5,
		TimeWindow:  24 * time.Hour,
		RateLimit:   time.Second,
		SampleSources: 1,
	}
	
	err := service.CreateMockArticles(config)
	if err != nil {
		t.Fatalf("CreateMockArticles failed: %v", err)
	}

	// Verify articles were created
	var articles []models.Article
	db.Find(&articles)

	if len(articles) == 0 {
		t.Error("Expected mock articles to be created")
	}

	// Verify source articles were created
	var sourceArticles []models.SourceArticle
	db.Find(&sourceArticles)

	if len(sourceArticles) == 0 {
		t.Error("Expected source articles to be created")
	}

	// Verify articles have proper metadata
	for _, article := range articles {
		if article.URL == "" {
			t.Error("Expected article URL to be set")
		}
		if article.Title == "" {
			t.Error("Expected article title to be set")
		}
		if article.Description == "" {
			t.Error("Expected article description to be set")
		}
		if article.WordCount <= 0 {
			t.Error("Expected article word count to be > 0")
		}
		if article.ReadingTime <= 0 {
			t.Error("Expected article reading time to be > 0")
		}
	}
}

func TestCalculateReadingTime(t *testing.T) {
	tests := []struct {
		name      string
		wordCount int
		expected  int
	}{
		{"Short article", 100, 1},   // < 225 words = 1 min minimum
		{"Medium article", 400, 1},  // 400/225 = 1.7, rounds down to 1, minimum is 1
		{"Long article", 1000, 4},   // 1000/225 = 4.4, rounds down to 4
		{"Very long article", 2000, 8}, // 2000/225 = 8.8, rounds down to 8
		{"Empty article", 0, 1},     // minimum reading time is 1
		{"Exactly 225 words", 225, 1}, // 225/225 = 1
		{"450 words", 450, 2},       // 450/225 = 2
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateReadingTime(tt.wordCount)
			if result != tt.expected {
				t.Errorf("Expected reading time %d, got %d for %d words", tt.expected, result, tt.wordCount)
			}
		})
	}
}
