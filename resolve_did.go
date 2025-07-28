package main

import (
	"fmt"
	"log"

	"open-news/internal/bluesky"
	"open-news/internal/database"
	"open-news/internal/models"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	dbConfig := database.LoadConfig()
	if err := database.Connect(dbConfig); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	client := bluesky.NewClient("https://bsky.social")
	
	realDID, err := client.ResolveHandle("librenews.bsky.social")
	if err != nil {
		log.Fatal("Failed to resolve handle:", err)
	}
	
	fmt.Printf("Real DID for librenews.bsky.social: %s\n", realDID)
	
	// Update user record
	result := database.DB.Model(&models.User{}).
		Where("handle = ?", "librenews.bsky.social").
		Update("blue_sky_d_id", realDID)
		
	if result.Error != nil {
		log.Fatal("Failed to update user:", result.Error)
	}
	
	fmt.Printf("Updated user record with real DID\n")
}
