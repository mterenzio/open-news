package services

import (
	"testing"
	"time"

	"open-news/internal/models"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate all models
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
		{"Short article", 100, 1},
		{"Medium article", 400, 2}, 
		{"Long article", 1000, 5},
		{"Very long article", 2000, 10},
		{"Empty article", 0, 0},
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
