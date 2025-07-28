package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func main() {
	// Test URL that should work
	testURL := "https://example.com"
	
	fmt.Printf("Testing simple HTTP GET for: %s\n", testURL)
	
	req, err := http.NewRequestWithContext(context.Background(), "GET", testURL, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}
	
	// Use the same headers as the metadata extractor, but without Accept-Encoding
	req.Header.Set("User-Agent", "OpenNews/1.0 (+https://opennews.social)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()
	
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("Content-Encoding: %s\n", resp.Header.Get("Content-Encoding"))
	fmt.Printf("Content-Length: %s\n", resp.Header.Get("Content-Length"))
	
	// Read first 500 bytes to check if it's HTML
	buf := make([]byte, 500)
	n, err := resp.Body.Read(buf)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}
	
	content := string(buf[:n])
	fmt.Printf("\nFirst 500 bytes of content:\n%s\n", content)
	
	// Check if it looks like HTML
	if strings.Contains(strings.ToLower(content), "<html") || strings.Contains(strings.ToLower(content), "<!doctype") {
		fmt.Println("\n✅ Content looks like HTML")
		
		// Look for title
		if strings.Contains(strings.ToLower(content), "<title>") {
			fmt.Println("✅ Found <title> tag")
		} else {
			fmt.Println("❌ No <title> tag found in first 500 bytes")
		}
	} else {
		fmt.Println("\n❌ Content does not look like HTML")
		
		// Check if it's binary/compressed
		hasNullBytes := false
		for _, b := range buf[:n] {
			if b == 0 {
				hasNullBytes = true
				break
			}
		}
		
		if hasNullBytes {
			fmt.Println("❌ Content contains null bytes (likely binary/compressed)")
		}
	}
}
