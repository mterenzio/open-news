// Package models contains all data models for the open-news application
package models

import (
	"gorm.io/gorm"
)

// AllModels returns a slice of all model types for database migrations
func AllModels() []interface{} {
	return []interface{}{
		&User{},
		&Source{},
		&UserSource{},
		&Article{},
		&SourceArticle{},
		&ArticleFact{},
		&Feed{},
		&FeedItem{},
		&UserFeedPreference{},
	}
}

// AutoMigrate runs automatic migrations for all models
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(AllModels()...)
}
