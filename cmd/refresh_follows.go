package main

import (
	"flag"
	"log"
	"time"

	"open-news/internal/bluesky"
	"open-news/internal/database"
	"open-news/internal/models"
	"open-news/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	// Command line flags
	userDID := flag.String("user", "", "User DID to refresh follows for (optional, refreshes all if not specified)")
	flag.Parse()

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load database configuration
	dbConfig := database.LoadConfig()

	// Connect to database
	if err := database.Connect(dbConfig); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Initialize Bluesky client
	blueskyClient := bluesky.NewClient("https://bsky.social")

	// Initialize user follows service
	userFollowsService := services.NewUserFollowsService(database.DB, blueskyClient)

	// Force refresh config (ignore time limits)
	config := services.RefreshConfig{
		RefreshInterval: 0, // Force immediate refresh
		BatchSize:       50,
		RateLimit:       100 * time.Millisecond,
	}

	if *userDID != "" {
		// Refresh specific user
		log.Printf("🔄 Refreshing follows for user: %s", *userDID)
		
		var user models.User
		if err := database.DB.Where("blue_sky_d_id = ?", *userDID).First(&user).Error; err != nil {
			log.Fatalf("❌ User not found: %v", err)
		}

		if err := userFollowsService.ImportUserFollows(&user, config); err != nil {
			log.Fatalf("❌ Failed to refresh follows: %v", err)
		}

		log.Printf("✅ Successfully refreshed follows for user %s", user.Handle)
	} else {
		// Refresh all users
		log.Println("🔄 Refreshing follows for all users...")
		
		if err := userFollowsService.RefreshBatch(config); err != nil {
			log.Fatalf("❌ Failed to refresh follows: %v", err)
		}

		log.Println("✅ Successfully refreshed follows for all users")
	}
}
