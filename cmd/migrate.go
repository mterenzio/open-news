package main

import (
	"log"

	"open-news/internal/database"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Connect to database
	dbConfig := database.LoadConfig()
	if err := database.Connect(dbConfig); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	log.Println("🔄 Running database migrations...")

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	log.Println("✅ Database migrations completed successfully")
}
