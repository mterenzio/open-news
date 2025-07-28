package main

import (
	"context"
	"fmt"
	"log"

	"open-news/internal/metadata"
)

func main() {
	extractor := metadata.NewMetadataExtractor()
	
	// Test with a simple URL
	testURL := "https://www.nytimes.com/2025/07/27/arts/music/tom-lehrer-dead.html"
	
	fmt.Printf("Testing metadata extraction for: %s\n", testURL)
	
	ctx := context.Background()
	meta, err := extractor.ExtractMetadata(ctx, testURL)
	if err != nil {
		log.Fatalf("Error extracting metadata: %v", err)
	}
	
	fmt.Printf("Title: %s\n", meta.Title)
	fmt.Printf("Description: %s\n", meta.Description)
	fmt.Printf("Author: %s\n", meta.Author)
	fmt.Printf("Site Name: %s\n", meta.SiteName)
	fmt.Printf("HTML Content length: %d\n", len(meta.HTMLContent))
	fmt.Printf("Text Content length: %d\n", len(meta.TextContent))
	fmt.Printf("Word Count: %d\n", meta.WordCount)
	
	// Check if HTML content looks like binary (contains null bytes or weird chars)
	hasNullBytes := false
	for _, b := range []byte(meta.HTMLContent) {
		if b == 0 {
			hasNullBytes = true
			break
		}
	}
	
	if hasNullBytes {
		fmt.Println("WARNING: HTML content contains null bytes (likely binary data)")
	} else {
		fmt.Println("HTML content looks good (no null bytes)")
	}
}
