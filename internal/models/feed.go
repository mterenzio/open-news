package models

import (
	"time"

	"github.com/google/uuid"
)

// Feed represents a custom feed (global or personalized)
type Feed struct {
	ID          uuid.UUID `json:"id" db:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name        string    `json:"name" db:"name" gorm:"not null"`
	Description string    `json:"description" db:"description"`
	FeedType    string    `json:"feed_type" db:"feed_type" gorm:"not null"` // "global" or "personalized"
	IsActive    bool      `json:"is_active" db:"is_active" gorm:"default:true"`
	
	// Feed configuration
	MaxItems      int     `json:"max_items" db:"max_items" gorm:"default:50"`
	RefreshRate   int     `json:"refresh_rate" db:"refresh_rate" gorm:"default:300"` // seconds
	QualityThreshold float64 `json:"quality_threshold" db:"quality_threshold" gorm:"default:0.0"`
	
	CreatedAt time.Time `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	FeedItems []FeedItem `json:"feed_items,omitempty" gorm:"foreignKey:FeedID"`
}

// FeedItem represents an article in a feed with its ranking
type FeedItem struct {
	ID           uuid.UUID `json:"id" db:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	FeedID       uuid.UUID `json:"feed_id" db:"feed_id" gorm:"not null;index"`
	ArticleID    uuid.UUID `json:"article_id" db:"article_id" gorm:"not null;index"`
	UserID       *uuid.UUID `json:"user_id" db:"user_id" gorm:"index"` // NULL for global feed
	
	// Ranking and scoring
	Position     int     `json:"position" db:"position" gorm:"not null"`
	Score        float64 `json:"score" db:"score" gorm:"default:0.0"`
	Relevance    float64 `json:"relevance" db:"relevance" gorm:"default:0.0"` // For personalized feeds
	
	// Timestamps
	AddedAt      time.Time `json:"added_at" db:"added_at" gorm:"autoCreateTime"`
	LastShownAt  *time.Time `json:"last_shown_at" db:"last_shown_at"`
	
	CreatedAt time.Time `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	Feed    Feed    `json:"feed,omitempty" gorm:"foreignKey:FeedID;references:ID"`
	Article Article `json:"article,omitempty" gorm:"foreignKey:ArticleID;references:ID"`
	User    *User   `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID"`
}

// UserFeedPreference represents user preferences for personalized feeds
type UserFeedPreference struct {
	ID     uuid.UUID `json:"id" db:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID uuid.UUID `json:"user_id" db:"user_id" gorm:"not null;uniqueIndex"`
	
	// Preference weights
	RecencyWeight    float64 `json:"recency_weight" db:"recency_weight" gorm:"default:0.3"`
	QualityWeight    float64 `json:"quality_weight" db:"quality_weight" gorm:"default:0.4"`
	EngagementWeight float64 `json:"engagement_weight" db:"engagement_weight" gorm:"default:0.3"`
	
	// Content preferences
	PreferredTopics   []string `json:"preferred_topics" db:"preferred_topics" gorm:"type:text[]"`
	BlockedSources    []uuid.UUID `json:"blocked_sources" db:"blocked_sources" gorm:"type:uuid[]"`
	PreferredSources  []uuid.UUID `json:"preferred_sources" db:"preferred_sources" gorm:"type:uuid[]"`
	
	CreatedAt time.Time `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	User User `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID"`
}

// TableName methods
func (Feed) TableName() string {
	return "feeds"
}

func (FeedItem) TableName() string {
	return "feed_items"
}

func (UserFeedPreference) TableName() string {
	return "user_feed_preferences"
}
