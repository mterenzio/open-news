package models

import (
	"time"

	"github.com/google/uuid"
)

// Source represents users that share links (content creators)
type Source struct {
	ID          uuid.UUID `json:"id" db:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	BlueSkyDID  string    `json:"bluesky_did" db:"bluesky_did" gorm:"uniqueIndex;not null"`
	Handle      string    `json:"handle" db:"handle" gorm:"uniqueIndex;not null"`
	DisplayName string    `json:"display_name" db:"display_name"`
	Avatar      string    `json:"avatar" db:"avatar"`
	Bio         string    `json:"bio" db:"bio"`
	FollowersCount int    `json:"followers_count" db:"followers_count" gorm:"default:0"`
	IsVerified     bool   `json:"is_verified" db:"is_verified" gorm:"default:false"`
	QualityScore   float64 `json:"quality_score" db:"quality_score" gorm:"default:0.0"` // Algorithm score for source quality
	CreatedAt      time.Time `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	SourceArticles []SourceArticle `json:"source_articles,omitempty" gorm:"foreignKey:SourceID"`
	UserSources    []UserSource    `json:"user_sources,omitempty" gorm:"foreignKey:SourceID"`
}

// TableName sets the table name for the Source model
func (Source) TableName() string {
	return "sources"
}
