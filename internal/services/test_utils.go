package services

import (
	"os"
	"testing"

	"open-news/internal/database"
	"open-news/internal/models"

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
		&models.UserSource{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Clean up any existing test data
	db.Exec("DELETE FROM user_sources")
	db.Exec("DELETE FROM source_articles")
	db.Exec("DELETE FROM article_facts")
	db.Exec("DELETE FROM articles")
	db.Exec("DELETE FROM sources WHERE blue_sky_d_id LIKE 'did:plc:test%'")
	db.Exec("DELETE FROM users WHERE blue_sky_d_id LIKE 'did:plc:test%'")

	return db
}
