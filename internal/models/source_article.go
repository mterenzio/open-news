package models

import (
	"time"

	"github.com/google/uuid"
)

// SourceArticle represents a source's post or repost that contains an article
type SourceArticle struct {
	ID         uuid.UUID `json:"id" db:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SourceID   uuid.UUID `json:"source_id" db:"source_id" gorm:"not null;index"`
	ArticleID  uuid.UUID `json:"article_id" db:"article_id" gorm:"not null;index"`
	
	// Bluesky post information
	PostURI    string `json:"post_uri" db:"post_uri" gorm:"uniqueIndex;not null"` // Bluesky post AT URI
	PostCID    string `json:"post_cid" db:"post_cid"`                             // Content identifier
	PostText   string `json:"post_text" db:"post_text" gorm:"type:text"`          // Post content
	
	// Post metadata
	IsRepost     bool      `json:"is_repost" db:"is_repost" gorm:"default:false"`
	OriginalURI  string    `json:"original_uri" db:"original_uri"`      // If repost, original post URI
	PostedAt     time.Time `json:"posted_at" db:"posted_at"`            // When posted on Bluesky
	
	// Engagement metrics from Bluesky
	LikesCount   int `json:"likes_count" db:"likes_count" gorm:"default:0"`
	RepostsCount int `json:"reposts_count" db:"reposts_count" gorm:"default:0"`
	RepliesCount int `json:"replies_count" db:"replies_count" gorm:"default:0"`
	
	// Local metrics
	ViewsCount   int     `json:"views_count" db:"views_count" gorm:"default:0"`
	ClicksCount  int     `json:"clicks_count" db:"clicks_count" gorm:"default:0"`
	ShareScore   float64 `json:"share_score" db:"share_score" gorm:"default:0.0"` // Calculated engagement score
	
	CreatedAt time.Time `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	Source  Source  `json:"source,omitempty" gorm:"foreignKey:SourceID;references:ID"`
	Article Article `json:"article,omitempty" gorm:"foreignKey:ArticleID;references:ID"`
}

// TableName sets the table name for the SourceArticle model
func (SourceArticle) TableName() string {
	return "source_articles"
}
