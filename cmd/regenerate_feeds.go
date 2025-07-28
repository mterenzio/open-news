package main

import (
	"log"

	"open-news/internal/database"
	"open-news/internal/feeds"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	log.Println("🔄 Starting feed regeneration...")

	// Load database configuration and connect
	dbConfig := database.LoadConfig()
	if err := database.Connect(dbConfig); err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize feed service
	feedService := feeds.NewFeedService(database.DB)

	// Regenerate global feed
	log.Println("🌐 Regenerating global feed...")
	if err := feedService.RegenerateGlobalFeed(); err != nil {
		log.Fatalf("❌ Failed to regenerate global feed: %v", err)
	}

	log.Println("✅ Feed regeneration completed successfully!")
}
