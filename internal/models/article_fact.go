package models

import (
	"time"

	"github.com/google/uuid"
)

// ArticleFact represents extracted facts from articles with OpenAI embeddings
type ArticleFact struct {
	ID        uuid.UUID `json:"id" db:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	ArticleID uuid.UUID `json:"article_id" db:"article_id" gorm:"not null;index"`
	
	// Fact content
	FactText    string  `json:"fact_text" db:"fact_text" gorm:"type:text;not null"`
	FactType    string  `json:"fact_type" db:"fact_type"`        // e.g., "claim", "statistic", "quote", "date"
	Context     string  `json:"context" db:"context" gorm:"type:text"`     // Surrounding context
	Confidence  float64 `json:"confidence" db:"confidence" gorm:"default:0.0"` // AI confidence score
	
	// Location in article
	StartPosition int `json:"start_position" db:"start_position"` // Character position in text
	EndPosition   int `json:"end_position" db:"end_position"`     // Character position in text
	
	// OpenAI embeddings (stored as JSON for now, can be upgraded to pgvector later)
	Embedding string `json:"embedding" db:"embedding" gorm:"type:text"` // JSON-encoded float64 array
	
	// Metadata
	ExtractedBy   string `json:"extracted_by" db:"extracted_by"`     // AI model used for extraction
	ExtractedAt   time.Time `json:"extracted_at" db:"extracted_at"`
	
	CreatedAt time.Time `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	Article Article `json:"article,omitempty" gorm:"foreignKey:ArticleID;references:ID"`
}

// TableName sets the table name for the ArticleFact model
func (ArticleFact) TableName() string {
	return "article_facts"
}
