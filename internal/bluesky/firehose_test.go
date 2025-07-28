package bluesky

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"open-news/internal/database"
	"open-news/internal/metadata"
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

func createTestSource(t *testing.T, db *gorm.DB) *models.Source {
	source := &models.Source{
		ID:          uuid.New(),
		Handle:      "testnews.bsky.social",
		BlueSkyDID:  "did:plc:test123456789",
		DisplayName: "Test News",
		IsVerified:  true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := db.Create(source).Error; err != nil {
		t.Fatalf("Failed to create test source: %v", err)
	}

	return source
}

func TestProcessJetstreamMessage(t *testing.T) {
	db := setupTestDB(t)
	source := createTestSource(t, db)

	// Create firehose consumer
	consumer := &FirehoseConsumer{
		db:                db,
		client:            nil, // Not needed for this test
		metadataExtractor: metadata.NewMetadataExtractor(), // Create real metadata extractor
	}

	// Create a test Jetstream event
	event := JetstreamEvent{
		DID:    source.BlueSkyDID,
		TimeUS: time.Now().UnixMicro(),
		Kind:   "commit",
		Commit: &JetstreamCommit{
			Collection: "app.bsky.feed.post",
			Operation:  "create",
			RKey:       "test123",
			CID:        "bafytest123",
			Record: map[string]interface{}{
				"$type":     "app.bsky.feed.post",
				"text":      "Check out this great article: https://nonexistent-domain-12345.com/test-article and this one too: https://invalid-url-67890.com/another-article",
				"createdAt": time.Now().Format(time.RFC3339),
			},
		},
	}

	// Convert to JSON and back to simulate real message processing
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal test event: %v", err)
	}

	err = consumer.processJetstreamMessage(data)
	if err != nil {
		t.Errorf("processJetstreamMessage failed: %v", err)
	}

	// Verify articles were created with proper fetch status tracking
	var articles []models.Article
	db.Find(&articles)

	expectedURLs := []string{
		"https://nonexistent-domain-12345.com/test-article",
		"https://invalid-url-67890.com/another-article",
	}

	if len(articles) != len(expectedURLs) {
		t.Errorf("Expected %d articles, got %d", len(expectedURLs), len(articles))
	}

	// Verify fetch status fields are properly set
	for _, article := range articles {
		// Since the metadata extractor will try to fetch real URLs, they should fail
		// and be marked as unreachable
		if article.IsReachable {
			t.Errorf("Expected article %s to be marked as unreachable due to network failure", article.URL)
		}
		
		if article.FetchError == "" {
			t.Errorf("Expected article %s to have a fetch error", article.URL)
		}
		
		if article.FetchRetries != 1 {
			t.Errorf("Expected article %s to have 1 fetch retry, got %d", article.URL, article.FetchRetries)
		}
		
		if article.LastFetchAt == nil {
			t.Errorf("Expected article %s to have LastFetchAt set", article.URL)
		}
		
		if article.LastFetchError == nil {
			t.Errorf("Expected article %s to have LastFetchError set", article.URL)
		}
	}

	// Verify source articles were created
	var sourceArticles []models.SourceArticle
	db.Find(&sourceArticles)

	if len(sourceArticles) != len(expectedURLs) {
		t.Errorf("Expected %d source articles, got %d", len(expectedURLs), len(sourceArticles))
	}

	// Verify the source article relationships
	for _, sa := range sourceArticles {
		if sa.SourceID != source.ID {
			t.Errorf("Expected source ID %v, got %v", source.ID, sa.SourceID)
		}

		if sa.PostURI == "" {
			t.Error("Expected PostURI to be set")
		}
	}
}

func TestExtractLinksFromPost(t *testing.T) {
	consumer := &FirehoseConsumer{}

	tests := []struct {
		name     string
		post     *PostRecord
		expected []string
	}{
		{
			name: "Single URL in text",
			post: &PostRecord{
				Text: "Check out this article: https://example.com/article",
			},
			expected: []string{"https://example.com/article"},
		},
		{
			name: "Multiple URLs in text",
			post: &PostRecord{
				Text: "Read https://news.com/story1 and https://tech.com/story2",
			},
			expected: []string{"https://news.com/story1", "https://tech.com/story2"},
		},
		{
			name: "URLs with punctuation",
			post: &PostRecord{
				Text: "Great article: https://example.com/story! Also see https://news.com/update.",
			},
			expected: []string{"https://example.com/story", "https://news.com/update"},
		},
		{
			name: "No URLs",
			post: &PostRecord{
				Text: "This is just a regular post without any links",
			},
			expected: []string{},
		},
		{
			name: "HTTP and HTTPS URLs",
			post: &PostRecord{
				Text: "Old site: http://old.example.com and new site: https://new.example.com",
			},
			expected: []string{"http://old.example.com", "https://new.example.com"},
		},
		{
			name: "Duplicate URLs",
			post: &PostRecord{
				Text: "Same link twice: https://example.com/article https://example.com/article",
			},
			expected: []string{"https://example.com/article"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links := consumer.extractLinksFromPost(tt.post)
			
			if len(links) != len(tt.expected) {
				t.Errorf("Expected %d links, got %d: %v", len(tt.expected), len(links), links)
				return
			}

			for i, link := range links {
				if link != tt.expected[i] {
					t.Errorf("Expected link %d to be %q, got %q", i, tt.expected[i], link)
				}
			}
		})
	}
}

func TestProcessLinkDuplicateArticle(t *testing.T) {
	db := setupTestDB(t)
	source := createTestSource(t, db)

	consumer := &FirehoseConsumer{
		db:                db,
		client:            nil,
		metadataExtractor: nil,
	}

	// Create an existing article
	existingArticle := &models.Article{
		ID:        uuid.New(),
		URL:       "https://example.com/existing-article",
		Title:     "Existing Article",
		IsCached:  true,
		CreatedAt: time.Now(),
	}
	db.Create(existingArticle)

	// Create test event
	event := &JetstreamEvent{
		DID: source.BlueSkyDID,
		Commit: &JetstreamCommit{
			RKey: "test123",
			CID:  "bafytest123",
		},
	}

	post := &PostRecord{
		Text:      "Check this out",
		CreatedAt: time.Now(),
	}

	// Process the same URL again
	err := consumer.processLink("https://example.com/existing-article", source, post, event)
	if err != nil {
		t.Errorf("processLink failed: %v", err)
	}

	// Verify only one article exists
	var articles []models.Article
	db.Find(&articles)
	if len(articles) != 1 {
		t.Errorf("Expected 1 article, got %d", len(articles))
	}

	// Verify source article was still created
	var sourceArticles []models.SourceArticle
	db.Find(&sourceArticles)
	if len(sourceArticles) != 1 {
		t.Errorf("Expected 1 source article, got %d", len(sourceArticles))
	}
}

func TestIsRepost(t *testing.T) {
	consumer := &FirehoseConsumer{}

	tests := []struct {
		name     string
		post     *PostRecord
		expected bool
	}{
		{
			name: "Regular post",
			post: &PostRecord{
				Text:  "This is a regular post with some content that is longer than 50 characters to test the repost detection",
				Reply: nil,
			},
			expected: false,
		},
		{
			name: "Short post with link (likely repost)",
			post: &PostRecord{
				Text: "Check this out: https://example.com",
				Facets: []Facet{
					{}, // Mock facet to indicate links
				},
			},
			expected: true,
		},
		{
			name: "Reply post",
			post: &PostRecord{
				Text: "This is a reply to another post",
				Reply: &Reply{
					Root: RecordRef{URI: "at://did:plc:test/app.bsky.feed.post/abc"},
				},
			},
			expected: true,
		},
		{
			name: "Long post with link",
			post: &PostRecord{
				Text: "This is a longer post that provides substantial commentary and analysis about the article being shared, which makes it more than just a simple repost: https://example.com",
				Facets: []Facet{
					{}, // Mock facet
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := consumer.isRepost(tt.post)
			if result != tt.expected {
				t.Errorf("Expected isRepost = %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFetchStatusTracking(t *testing.T) {
	db := setupTestDB(t)

	// Create test source
	source := &models.Source{
		ID:          uuid.New(),
		Handle:      "test.bsky.social",
		BlueSkyDID:  "did:plc:test123456789",
		DisplayName: "Test Source",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	db.Create(source)

	// Create firehose consumer with real metadata extractor
	consumer := &FirehoseConsumer{
		db:                db,
		client:            nil, // Not needed for this test
		metadataExtractor: metadata.NewMetadataExtractor(),
	}

	// Create test event with invalid URL
	event := JetstreamEvent{
		DID:    source.BlueSkyDID,
		TimeUS: time.Now().UnixMicro(),
		Kind:   "commit",
		Commit: &JetstreamCommit{
			Collection: "app.bsky.feed.post",
			Operation:  "create",
			RKey:       "test123",
			CID:        "bafytest123",
			Record: map[string]interface{}{
				"$type":     "app.bsky.feed.post",
				"text":      "Check out this invalid URL: https://invalid.example.com/nonexistent",
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

	// Verify the article was created but marked as unreachable
	var article models.Article
	err = db.Where("url = ?", "https://invalid.example.com/nonexistent").First(&article).Error
	if err != nil {
		t.Fatalf("Failed to find created article: %v", err)
	}

	// Check fetch status fields
	if article.IsReachable {
		t.Error("Expected article to be marked as unreachable")
	}

	if article.FetchError == "" {
		t.Error("Expected article to have a fetch error message")
	}

	if article.FetchRetries != 1 {
		t.Errorf("Expected 1 fetch retry, got %d", article.FetchRetries)
	}

	if article.LastFetchAt == nil {
		t.Error("Expected LastFetchAt to be set")
	}

	if article.LastFetchError == nil {
		t.Error("Expected LastFetchError to be set")
	}

	// Verify the article has minimal metadata (just URL)
	if article.Title != "" {
		t.Errorf("Expected empty title for failed fetch, got: %s", article.Title)
	}

	if article.Description != "" {
		t.Errorf("Expected empty description for failed fetch, got: %s", article.Description)
	}

	if article.IsCached {
		t.Error("Expected article to not be marked as cached when fetch fails")
	}
}
