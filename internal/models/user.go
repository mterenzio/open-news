package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a Bluesky user that signs up by visiting a custom feed
type User struct {
	ID          uuid.UUID `json:"id" db:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	BlueSkyDID  string    `json:"bluesky_did" db:"bluesky_did" gorm:"uniqueIndex;not null"`
	Handle      string    `json:"handle" db:"handle" gorm:"uniqueIndex;not null"`
	DisplayName string    `json:"display_name" db:"display_name"`
	Avatar      string    `json:"avatar" db:"avatar"`
	Bio         string    `json:"bio" db:"bio"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`
	LastSeenAt           time.Time  `json:"last_seen_at" db:"last_seen_at"`
	FollowsLastRefreshed *time.Time `json:"follows_last_refreshed" db:"follows_last_refreshed"`
	IsActive             bool       `json:"is_active" db:"is_active" gorm:"default:true"`

	// Relationships
	UserSources []UserSource `json:"user_sources,omitempty" gorm:"foreignKey:UserID"`
}

// TableName sets the table name for the User model
func (User) TableName() string {
	return "users"
}
