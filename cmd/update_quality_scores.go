package main

import (
	"log"
	"open-news/internal/database"
	"open-news/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	log.Println("🔄 Starting quality score calculation...")

	// Load database configuration and connect
	dbConfig := database.LoadConfig()
	if err := database.Connect(dbConfig); err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize quality score service
	qualityService := services.NewQualityScoreService(database.DB)

	// Update all quality scores
	if err := qualityService.UpdateAllQualityScores(); err != nil {
		log.Fatalf("❌ Failed to update quality scores: %v", err)
	}

	log.Println("✅ Quality score calculation completed successfully!")
}
