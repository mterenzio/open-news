package models

import (
	"time"

	"github.com/google/uuid"
)

// UserSource represents the relationship between users and the sources they follow
type UserSource struct {
	ID        uuid.UUID `json:"id" db:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    uuid.UUID `json:"user_id" db:"user_id" gorm:"not null;index"`
	SourceID  uuid.UUID `json:"source_id" db:"source_id" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	User   User   `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID"`
	Source Source `json:"source,omitempty" gorm:"foreignKey:SourceID;references:ID"`
}

// TableName sets the table name for the UserSource model
func (UserSource) TableName() string {
	return "user_sources"
}
