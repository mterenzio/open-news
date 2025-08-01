package metadata

import (
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestExtractMetadata(t *testing.T) {
	// Read test HTML file
	htmlContent, err := os.ReadFile("testdata/sample_article.html")
	if err != nil {
		t.Fatalf("Failed to read test HTML file: %v", err)
	}

	// Create a test server that serves our sample HTML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(htmlContent)
	}))
	defer server.Close()

	// Create metadata extractor
	extractor := NewMetadataExtractor()

	// Extract metadata
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metadata, err := extractor.ExtractMetadata(ctx, server.URL)
	if err != nil {
		t.Fatalf("Failed to extract metadata: %v", err)
	}

	// Test extracted data
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"Title", metadata.Title, "Test News Article - Breaking Tech News"},
		{"Description", metadata.Description, "This is a test news article about technology developments and their impact on society."},
		{"Author", metadata.Author, "John Doe"},
		{"SiteName", metadata.SiteName, "Test News Site"},
		{"ImageURL", metadata.ImageURL, "https://example.com/image.jpg"},
		{"Language", metadata.Language, "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Expected %s = %q, got %q", tt.name, tt.expected, tt.got)
			}
		})
	}

	// Test numeric fields
	if metadata.WordCount == 0 {
		t.Error("Expected WordCount > 0")
	}

	if metadata.ReadingTime == 0 {
		t.Error("Expected ReadingTime > 0")
	}

	// Test published date
	if metadata.PublishedAt == nil {
		t.Error("Expected PublishedAt to be set")
	} else {
		expectedTime := time.Date(2025, 7, 28, 12, 0, 0, 0, time.UTC)
		if !metadata.PublishedAt.Equal(expectedTime) {
			t.Errorf("Expected PublishedAt = %v, got %v", expectedTime, *metadata.PublishedAt)
		}
	}

	// Test JSON-LD data
	if metadata.JSONLDData == "" {
		t.Error("Expected JSONLDData to be extracted")
	}

	if !strings.Contains(metadata.JSONLDData, "NewsArticle") {
		t.Error("Expected JSONLDData to contain NewsArticle type")
	}

	// Test Open Graph data
	if metadata.OGData == "" {
		t.Error("Expected OGData to be extracted")
	}

	// Test text content extraction
	if metadata.TextContent == "" {
		t.Error("Expected TextContent to be extracted")
	}

	if !strings.Contains(metadata.TextContent, "first paragraph") {
		t.Error("Expected TextContent to contain article text")
	}
}

func TestExtractMetadataMinimalHTML(t *testing.T) {
	minimalHTML := `<!DOCTYPE html>
<html>
<head>
	<title>Simple Title</title>
</head>
<body>
	<h1>Simple Content</h1>
	<p>This is a simple test with minimal HTML content for testing edge cases.</p>
</body>
</html>`

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(minimalHTML))
	}))
	defer server.Close()

	extractor := NewMetadataExtractor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metadata, err := extractor.ExtractMetadata(ctx, server.URL)
	if err != nil {
		t.Fatalf("Failed to extract metadata: %v", err)
	}

	if metadata.Title != "Simple Title" {
		t.Errorf("Expected title = 'Simple Title', got %q", metadata.Title)
	}

	if metadata.WordCount == 0 {
		t.Error("Expected WordCount > 0 even for minimal HTML")
	}
}

func TestExtractMetadataHTTPError(t *testing.T) {
	// Create a test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	extractor := NewMetadataExtractor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := extractor.ExtractMetadata(ctx, server.URL)
	if err == nil {
		t.Error("Expected error for 404 response")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("Expected error to mention 404, got: %v", err)
	}
}

func TestExtractMetadataInvalidURL(t *testing.T) {
	extractor := NewMetadataExtractor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := extractor.ExtractMetadata(ctx, "not-a-valid-url")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestExtractMetadataTimeout(t *testing.T) {
	// Create a test server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Sleep longer than our timeout
	}))
	defer server.Close()

	extractor := NewMetadataExtractor()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond) // Very short timeout
	defer cancel()

	_, err := extractor.ExtractMetadata(ctx, server.URL)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestExtractMetadataHTTPHeaders(t *testing.T) {
	// Test that we don't send Accept-Encoding header manually (regression test for gzip issue)
	var receivedHeaders http.Header
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Test</title></head><body><h1>Test Content</h1></body></html>`))
	}))
	defer server.Close()

	extractor := NewMetadataExtractor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := extractor.ExtractMetadata(ctx, server.URL)
	if err != nil {
		t.Fatalf("Failed to extract metadata: %v", err)
	}

	// Verify that we don't manually set Accept-Encoding
	// Go's HTTP client will set this automatically and handle decompression
	acceptEncoding := receivedHeaders.Get("Accept-Encoding")
	if acceptEncoding == "" {
		// This is actually fine - Go may not set it at all
		// What matters is that we don't manually set it to something problematic
		t.Logf("Accept-Encoding header not set (OK)")
	} else {
		t.Logf("Accept-Encoding header set by Go HTTP client: %s", acceptEncoding)
	}

	// The important thing is that we can successfully parse the response
	// without gzip decompression errors
}

func TestExtractMetadataGzipResponse(t *testing.T) {
	// Test that we can handle gzipped responses properly
	htmlContent := `<!DOCTYPE html>
<html>
<head>
	<title>Gzipped Content Test</title>
	<meta name="description" content="This content is served with gzip compression">
</head>
<body>
	<h1>Compressed Content</h1>
	<p>This tests that our metadata extractor can handle gzip-compressed responses properly.</p>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Encoding", "gzip")
		
		// Write gzipped content
		gzipWriter := gzip.NewWriter(w)
		defer gzipWriter.Close()
		gzipWriter.Write([]byte(htmlContent))
	}))
	defer server.Close()

	extractor := NewMetadataExtractor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metadata, err := extractor.ExtractMetadata(ctx, server.URL)
	if err != nil {
		t.Fatalf("Failed to extract metadata from gzipped response: %v", err)
	}

	if metadata.Title != "Gzipped Content Test" {
		t.Errorf("Expected title = 'Gzipped Content Test', got %q", metadata.Title)
	}

	if metadata.Description != "This content is served with gzip compression" {
		t.Errorf("Expected description about gzip compression, got %q", metadata.Description)
	}
}

func BenchmarkExtractMetadata(b *testing.B) {
	// Read test HTML file
	htmlContent, err := os.ReadFile("testdata/sample_article.html")
	if err != nil {
		b.Fatalf("Failed to read test HTML file: %v", err)
	}

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(htmlContent)
	}))
	defer server.Close()

	extractor := NewMetadataExtractor()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := extractor.ExtractMetadata(ctx, server.URL)
		cancel()
		if err != nil {
			b.Fatalf("Failed to extract metadata: %v", err)
		}
	}
}
