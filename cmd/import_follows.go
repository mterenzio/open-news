package main

import (
	"log"
	"time"

	"open-news/internal/bluesky"
	"open-news/internal/database"
	"open-news/internal/models"
	"open-news/internal/services"

	"github.com/joho/godotenv"
)

// This utility manually imports follows for the configured Bluesky user
func main() {
	log.Printf("🔄 Importing real follows from Bluesky...")
	
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

	// Initialize Bluesky client
	client := bluesky.NewClient("https://bsky.social")
	
	// Authenticate with Bluesky
	identifier := "librenews.bsky.social"
	password := "q6f7-pper-ls6q-jyim"
	
	log.Printf("🔐 Authenticating with Bluesky as %s...", identifier)
	if err := client.CreateSession(identifier, password); err != nil {
		log.Fatal("Failed to authenticate with Bluesky:", err)
	}
	log.Printf("✅ Successfully authenticated with Bluesky")
	
	// Initialize UserFollowsService
	userFollowsService := services.NewUserFollowsService(database.DB, client)
	
	// Find the librenews.bsky.social user
	var user models.User
	if err := database.DB.Where("handle = ?", "librenews.bsky.social").First(&user).Error; err != nil {
		log.Fatal("User librenews.bsky.social not found in database:", err)
	}
	
	log.Printf("📥 Found user: %s (DID: %s)", user.Handle, user.BlueSkyDID)
	
	// Configure for real import with authentication
	config := services.RefreshConfig{
		RefreshInterval: 24 * time.Hour,
		BatchSize:       50,  // Larger batch for manual import
		RateLimit:       200 * time.Millisecond, // Be respectful to API
	}
	
	// Import follows
	if err := userFollowsService.ImportUserFollows(&user, config); err != nil {
		log.Fatal("Failed to import follows:", err)
	}
	
	log.Printf("✅ Successfully imported follows for %s", user.Handle)
}