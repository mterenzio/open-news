package main

import (
	"context"
	"fmt"
	"log"

	"open-news/internal/database"
	"open-news/internal/services"
	"open-news/internal/bluesky"
)

func main() {
	// Connect to database
	dbConfig := database.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "mterenzi",
		Password: "",
		Database: "open_news",
		SSLMode:  "disable",
	}

	db, err := database.Connect(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Create bluesky client (not needed for this test)
	blueskyClient := &bluesky.Client{}
	
	// Create articles service
	articlesService := services.NewArticlesService(db, blueskyClient)

	// Test URLs - some news articles, some non-news
	testURLs := []string{
		"https://www.nytimes.com/2025/01/15/technology/artificial-intelligence.html",
		"https://www.reuters.com/technology/ai-breakthrough-2025-01-15/",
		"https://techcrunch.com/2025/01/15/startup-funding-round/",
		"https://github.com/golang/go",              // Not a news article
		"https://stackoverflow.com/questions/12345", // Not a news article
		"https://www.bbc.com/news/technology-67890123",
	}

	fmt.Println("🧪 Testing NewsArticle schema detection...")
	fmt.Println("==========================================")

	for _, url := range testURLs {
		fmt.Printf("\n🔍 Testing URL: %s\n", url)
		
		ctx := context.Background()
		isNews, err := articlesService.CheckIfNewsArticle(ctx, url)
		
		if err != nil {
			fmt.Printf("❌ Error checking URL: %v\n", err)
		} else {
			if isNews {
				fmt.Printf("✅ Detected as NewsArticle\n")
			} else {
				fmt.Printf("⏭️  Not a NewsArticle (or no JSON-LD found)\n")
			}
		}
	}
}
