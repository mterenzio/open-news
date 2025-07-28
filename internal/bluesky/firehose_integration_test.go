// +build integration

package bluesky

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"open-news/internal/database"
	"open-news/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func setupIntegrationDB(t *testing.T) *gorm.DB {
	// Load environment variables from test file
	loadTestEnv(t)
	
	// Initialize database connection
	config := database.LoadConfig()
	err := database.Connect(config)
	if err != nil {
		t.Fatalf("Failed to connect to integration test database: %v", err)
	}

	// Run migrations to ensure schema exists
	err = database.Migrate()
	if err != nil {
		t.Fatalf("Failed to run migrations on test database: %v", err)
	}

	// Clean up any existing test data
	cleanupTestData(t, database.DB)

	return database.DB
}

func loadTestEnv(t *testing.T) {
	// Set environment variables for test database
	os.Setenv("DB_NAME", "open_news_test")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "mterenzi")
	os.Setenv("DB_PASSWORD", "")
	os.Setenv("DB_SSLMODE", "disable")
}

func cleanupTestData(t *testing.T, db *gorm.DB) {
	// Delete test data in reverse dependency order
	db.Where("post_uri LIKE ?", "%integration-test%").Delete(&models.SourceArticle{})
	db.Where("url LIKE ?", "%integration-test%").Delete(&models.Article{})
	db.Where("handle LIKE ?", "%integration-test%").Delete(&models.Source{})
}

func createIntegrationTestSource(t *testing.T, db *gorm.DB) *models.Source {
	source := &models.Source{
		ID:          uuid.New(),
		Handle:      "integration-test-news.bsky.social",
		BlueSkyDID:  "did:plc:integration-test-123",
		DisplayName: "Integration Test News",
		IsVerified:  true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := db.Create(source).Error; err != nil {
		t.Fatalf("Failed to create integration test source: %v", err)
	}

	return source
}

func TestIntegrationProcessJetstreamMessage(t *testing.T) {
	db := setupIntegrationDB(t)
	defer cleanupTestData(t, db)
	
	source := createIntegrationTestSource(t, db)

	// Create firehose consumer with real metadata extractor
	consumer := NewFirehoseConsumer(db, nil) // nil client is OK for this test

	// Create a test Jetstream event
	event := JetstreamEvent{
		DID:    source.BlueSkyDID,
		TimeUS: time.Now().UnixMicro(),
		Kind:   "commit",
		Commit: &JetstreamCommit{
			Collection: "app.bsky.feed.post",
			Operation:  "create",
			RKey:       "integration-test-123",
			CID:        "bafyintegrationtest123",
			Record: map[string]interface{}{
				"$type":     "app.bsky.feed.post",
				"text":      "Integration test article: https://example.com/integration-test-article",
				"createdAt": time.Now().Format(time.RFC3339),
			},
		},
	}

	// Convert to JSON and process
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal test event: %v", err)
	}

	err = consumer.processJetstreamMessage(data)
	if err != nil {
		t.Errorf("processJetstreamMessage failed: %v", err)
	}

	// Verify article was created with metadata extraction
	var articles []models.Article
	db.Where("url LIKE ?", "%integration-test%").Find(&articles)

	if len(articles) != 1 {
		t.Errorf("Expected 1 article, got %d", len(articles))
	}

	article := articles[0]
	
	// Check that metadata extraction was attempted
	// Note: This might fail if the URL doesn't exist, but we should see the URL was processed
	if article.URL != "https://example.com/integration-test-article" {
		t.Errorf("Expected URL to be processed correctly, got: %s", article.URL)
	}

	// Verify source article was created
	var sourceArticles []models.SourceArticle
	db.Where("post_uri LIKE ?", "%integration-test%").Find(&sourceArticles)

	if len(sourceArticles) != 1 {
		t.Errorf("Expected 1 source article, got %d", len(sourceArticles))
	}

	sourceArticle := sourceArticles[0]
	if sourceArticle.SourceID != source.ID {
		t.Errorf("Expected source ID %v, got %v", source.ID, sourceArticle.SourceID)
	}

	if sourceArticle.ArticleID != article.ID {
		t.Errorf("Expected article ID %v, got %v", article.ID, sourceArticle.ArticleID)
	}
}

func TestIntegrationArticleMetadataExtraction(t *testing.T) {
	db := setupIntegrationDB(t)
	defer cleanupTestData(t, db)
	
	source := createIntegrationTestSource(t, db)
	consumer := NewFirehoseConsumer(db, nil)

	// Test with a URL that should have good metadata (BBC News homepage)
	testURL := "https://www.bbc.com/news"
	
	event := &JetstreamEvent{
		DID: source.BlueSkyDID,
		Commit: &JetstreamCommit{
			RKey: "integration-test-metadata",
			CID:  "bafymetadatatest",
		},
	}

	post := &PostRecord{
		Text:      "Breaking news from BBC",
		CreatedAt: time.Now(),
	}

	// Process the link directly
	err := consumer.processLink(testURL, source, post, event)
	if err != nil {
		t.Errorf("processLink failed: %v", err)
	}

	// Verify article was created
	var articles []models.Article
	db.Where("url = ?", testURL).Find(&articles)

	if len(articles) != 1 {
		t.Errorf("Expected 1 article, got %d", len(articles))
		return
	}

	article := articles[0]
	
	// Verify metadata was extracted (BBC should have good metadata)
	if article.Title == "" {
		t.Log("Warning: No title extracted - this might indicate metadata extraction issues")
	}
	
	if article.Description == "" {
		t.Log("Warning: No description extracted")
	}
	
	if article.SiteName == "" {
		t.Log("Warning: No site name extracted")
	}

	// These should be populated regardless
	if article.IsCached != true {
		t.Error("Expected article to be marked as cached after metadata extraction")
	}

	if article.CachedAt == nil {
		t.Error("Expected CachedAt to be set after metadata extraction")
	}

	if article.LastFetchAt == nil {
		t.Error("Expected LastFetchAt to be set after metadata extraction")
	}
}

func TestIntegrationDuplicateArticleHandling(t *testing.T) {
	db := setupIntegrationDB(t)
	defer cleanupTestData(t, db)
	
	source := createIntegrationTestSource(t, db)
	consumer := NewFirehoseConsumer(db, nil)

	testURL := "https://example.com/integration-test-duplicate"

	// Create first event
	event1 := &JetstreamEvent{
		DID: source.BlueSkyDID,
		Commit: &JetstreamCommit{
			RKey: "integration-test-duplicate-1",
			CID:  "bafyduplicate1",
		},
	}

	post1 := &PostRecord{
		Text:      "First post about this article",
		CreatedAt: time.Now(),
	}

	// Create second event (same URL, different post)
	event2 := &JetstreamEvent{
		DID: source.BlueSkyDID,
		Commit: &JetstreamCommit{
			RKey: "integration-test-duplicate-2",
			CID:  "bafyduplicate2",
		},
	}

	post2 := &PostRecord{
		Text:      "Second post about the same article",
		CreatedAt: time.Now().Add(1 * time.Hour),
	}

	// Process both links
	err1 := consumer.processLink(testURL, source, post1, event1)
	if err1 != nil {
		t.Errorf("First processLink failed: %v", err1)
	}

	err2 := consumer.processLink(testURL, source, post2, event2)
	if err2 != nil {
		t.Errorf("Second processLink failed: %v", err2)
	}

	// Verify only one article exists
	var articles []models.Article
	db.Where("url = ?", testURL).Find(&articles)

	if len(articles) != 1 {
		t.Errorf("Expected 1 article, got %d", len(articles))
	}

	// Verify two source articles exist (different posts sharing same article)
	var sourceArticles []models.SourceArticle
	db.Where("article_id = ?", articles[0].ID).Find(&sourceArticles)

	if len(sourceArticles) != 2 {
		t.Errorf("Expected 2 source articles, got %d", len(sourceArticles))
	}

	// Verify they have different post URIs
	if sourceArticles[0].PostURI == sourceArticles[1].PostURI {
		t.Error("Expected different PostURIs for different posts")
	}
}
