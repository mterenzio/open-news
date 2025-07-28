package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Article represents the canonical URL, metadata, and cached HTML of an article
type Article struct {
	ID          uuid.UUID      `json:"id" db:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	URL         string         `json:"url" db:"url" gorm:"uniqueIndex;not null"` // Canonical URL
	Title       string         `json:"title" db:"title"`
	Description string         `json:"description" db:"description"`
	Author      string         `json:"author" db:"author"`
	SiteName    string         `json:"site_name" db:"site_name"`
	ImageURL    string         `json:"image_url" db:"image_url"`
	PublishedAt *time.Time     `json:"published_at" db:"published_at"`
	
	// JSON-LD and Open Graph metadata
	JSONLDData  string `json:"jsonld_data" db:"jsonld_data" gorm:"type:text"`  // Raw JSON-LD data
	OGData      string `json:"og_data" db:"og_data" gorm:"type:text"`       // Open Graph metadata as JSON
	
	// Cached HTML content
	HTMLContent string `json:"html_content" db:"html_content" gorm:"type:text"` // Full HTML cache
	TextContent string `json:"text_content" db:"text_content" gorm:"type:text"` // Extracted text content
	
	// Article metadata
	WordCount    int            `json:"word_count" db:"word_count" gorm:"default:0"`
	ReadingTime  int            `json:"reading_time" db:"reading_time" gorm:"default:0"` // in minutes
	Language     string         `json:"language" db:"language"`
	Tags         pq.StringArray `json:"tags" db:"tags" gorm:"type:text[]"`
	
	// Engagement metrics
	SharesCount  int `json:"shares_count" db:"shares_count" gorm:"default:0"`
	LikesCount   int `json:"likes_count" db:"likes_count" gorm:"default:0"`
	RepostsCount int `json:"reposts_count" db:"reposts_count" gorm:"default:0"`
	
	// Quality and ranking metrics
	QualityScore float64 `json:"quality_score" db:"quality_score" gorm:"default:0.0"`
	TrendingScore float64 `json:"trending_score" db:"trending_score" gorm:"default:0.0"`
	
	// Cache status
	IsCached     bool      `json:"is_cached" db:"is_cached" gorm:"default:false"`
	CachedAt     *time.Time `json:"cached_at" db:"cached_at"`
	LastFetchAt  *time.Time `json:"last_fetch_at" db:"last_fetch_at"`
	
	// Fetch status tracking
	IsReachable    bool   `json:"is_reachable" db:"is_reachable" gorm:"default:false"`
	FetchError     string `json:"fetch_error" db:"fetch_error"`              // Last error message
	FetchRetries   int    `json:"fetch_retries" db:"fetch_retries" gorm:"default:0"` // Number of failed attempts
	LastFetchError *time.Time `json:"last_fetch_error" db:"last_fetch_error"` // When the last error occurred
	
	CreatedAt time.Time `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	SourceArticles []SourceArticle `json:"source_articles,omitempty" gorm:"foreignKey:ArticleID"`
	Facts          []ArticleFact   `json:"facts,omitempty" gorm:"foreignKey:ArticleID"`
}

// TableName sets the table name for the Article model
func (Article) TableName() string {
	return "articles"
}
