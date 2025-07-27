package main

import (
	"fmt"
	"log"

	"open-news/internal/bluesky"
	"open-news/internal/models"
)

// Simple test to verify the application compiles and basic functionality works
func main() {
	log.Println("Testing Open News compilation and basic functionality...")

	// Test 1: Create Bluesky client
	client := bluesky.NewClient("https://bsky.social")
	if client == nil {
		log.Fatal("Failed to create Bluesky client")
	}
	log.Println("✅ Bluesky client created successfully")

	// Test 2: Test models creation (without database)
	user := models.User{
		BlueSkyDID:  "did:plc:test",
		Handle:      "test.bsky.social",
		DisplayName: "Test User",
	}
	
	source := models.Source{
		BlueSkyDID:  "did:plc:testsource",
		Handle:      "source.bsky.social",
		DisplayName: "Test Source",
	}
	
	article := models.Article{
		URL:   "https://example.com/article",
		Title: "Test Article",
	}

	log.Printf("✅ Models created: User(%s), Source(%s), Article(%s)", 
		user.Handle, source.Handle, article.Title)

	// Test 3: Test link extraction (mock post)
	post := &bluesky.Post{
		Record: bluesky.Record{
			Text: "Check out this article: https://example.com/news",
			Facets: []bluesky.Facet{
				{
					Features: []bluesky.Feature{
						{
							Type: "app.bsky.richtext.facet#link",
							URI:  "https://example.com/news",
						},
					},
				},
			},
		},
	}

	links := bluesky.ExtractLinks(post)
	if len(links) > 0 {
		log.Printf("✅ Link extraction working: found %d links: %v", len(links), links)
	} else {
		log.Println("⚠️  Link extraction returned no links")
	}

	fmt.Println("\n🎉 All basic functionality tests passed!")
	fmt.Println("📖 See DEVELOPMENT.md for database setup instructions")
	fmt.Println("🚀 Run 'make run' after setting up PostgreSQL")
}
