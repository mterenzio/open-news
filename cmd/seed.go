package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"open-news/internal/bluesky"
	"open-news/internal/database"
	"open-news/internal/models"
	"open-news/internal/services"

	"github.com/joho/godotenv"
)

// This is a simple utility script to seed the database with some initial sources
// In a production system, this would be done through the API or admin interface

func main() {
	// Parse command line flags
	var userHandle = flag.String("handle", "", "Bluesky handle to seed as user (leave empty for mock data only)")
	var userDID = flag.String("did", "did:plc:z72i7hdynmk6r22z27h6tvur", "DID of the test user (optional)")
	var articlesOnly = flag.Bool("articles-only", false, "Only seed articles, skip users and sources")
	flag.Parse()
	
	log.Printf("🌱 Open News Database Seeder")
	log.Printf("============================")
	if *userHandle != "" {
		log.Printf("User: %s", *userHandle)
	} else {
		log.Printf("Mode: Mock data only")
	}
	
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

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Initialize Bluesky client for potential authentication
	var authenticatedClient *bluesky.Client
	identifier := os.Getenv("BLUESKY_IDENTIFIER")
	password := os.Getenv("BLUESKY_PASSWORD")
	
	if identifier != "" && password != "" {
		client := bluesky.NewClient("https://bsky.social")
		log.Printf("🔐 Authenticating with Bluesky as %s...", identifier)
		if err := client.CreateSession(identifier, password); err != nil {
			log.Printf("⚠️  Authentication failed: %v", err)
			log.Printf("💡 Will use mock data instead")
		} else {
			log.Printf("✅ Successfully authenticated with Bluesky")
			authenticatedClient = client
		}
	}

	if *articlesOnly {
		// Only seed articles
		log.Printf("📰 Articles-only seeding mode")
		seedArticles(authenticatedClient, *userHandle)
	} else {
		// Full seeding: users, sources, and articles
		// Seed a test user with a real Bluesky handle
		// This user's follows will be automatically imported when they access their personalized feed
		seedTestUser(*userHandle, *userDID)
		
		// Seed articles for testing
		seedArticles(authenticatedClient, *userHandle)
	}

	log.Println("✅ Database seeding completed")
	log.Println("")
	log.Println("🌐 Developer Dashboard:")
	log.Println("======================")
	log.Println("Visit http://localhost:8080 for:")
	log.Println("• 📚 Complete documentation")
	log.Println("• 🧪 Copy-paste testing commands") 
	log.Println("• 🔗 Live API endpoint links")
	log.Println("• ⚡ Quick start guide")
	log.Println("")
	log.Println("🧪 Quick Test Commands:")
	log.Println("=======================")
	log.Println("1. Start/restart the server:")
	log.Println("   ./dev restart")
	log.Println("")
	log.Println("2. Test endpoints:")
	log.Println("   ./dev test")
	log.Println("   curl http://localhost:8080/health")
	log.Println("")
	log.Println("3. Use the web interface at http://localhost:8080")
	log.Println("   • Beautiful documentation with proper styling")
	log.Println("   • Copy-paste development commands")
	log.Println("   • Live API endpoint testing")
}

func seedTestUser(handle, did string) {
	if handle == "" {
		log.Printf("🌱 No handle provided - seeding mock data only")
		// Create mock sources without a user
		seedPopularSources()
		return
	}
	
	log.Printf("🌱 Seeding user: %s", handle)
	
	// First, validate that this is a real Bluesky handle
	client := bluesky.NewClient("https://bsky.social")
	
	// Check if we have credentials for authentication
	identifier := os.Getenv("BLUESKY_IDENTIFIER")
	password := os.Getenv("BLUESKY_PASSWORD")
	
	if identifier == "" || password == "" {
		log.Printf("❌ No Bluesky credentials found in environment")
		log.Printf("💡 Set BLUESKY_IDENTIFIER and BLUESKY_PASSWORD to validate real handles")
		log.Printf("💡 Or run without -handle flag to create mock data only")
		log.Fatal("Cannot validate real handle without authentication")
	}
	
	log.Printf("� Authenticating with Bluesky to validate handle...")
	if err := client.CreateSession(identifier, password); err != nil {
		log.Printf("❌ Failed to authenticate with Bluesky: %v", err)
		log.Fatal("Cannot validate handle without authentication")
	}
	
	log.Printf("🔍 Validating handle: %s", handle)
	realDID, err := client.ResolveHandle(handle)
	if err != nil {
		log.Printf("❌ Handle '%s' is not a valid Bluesky account: %v", handle, err)
		log.Printf("💡 To seed mock data instead, run: go run cmd/seed.go (without -handle flag)")
		log.Fatalf("Invalid Bluesky handle: %s", handle)
	}
	
	log.Printf("✅ Validated handle: %s (DID: %s)", handle, realDID)
	
	// Check if user already exists by DID or handle
	var existingUser models.User
	if err := database.DB.Where("blue_sky_d_id = ? OR handle = ?", realDID, handle).First(&existingUser).Error; err != nil {
		// User doesn't exist, create them with the real DID
		testUser := models.User{
			BlueSkyDID:  realDID,
			Handle:      handle,
			DisplayName: handle,
			Bio:         "Real Bluesky user for local development",
			IsActive:    true,
		}
		
		if err := database.DB.Create(&testUser).Error; err != nil {
			log.Printf("❌ Failed to create user: %v", err)
			return
		}
		
		log.Printf("✅ Created user: %s (DID: %s)", testUser.Handle, testUser.BlueSkyDID)
		
		// Import their follows automatically
		importTestUserFollows(testUser)
		
	} else {
		log.Printf("✅ User already exists: %s", existingUser.Handle)
		
		// Check if they have follows imported
		var followCount int64
		database.DB.Model(&models.UserSource{}).Where("user_id = ?", existingUser.ID).Count(&followCount)
		
		if followCount == 0 {
			log.Printf("📥 No follows found for user, attempting to import...")
			importTestUserFollows(existingUser)
		} else {
			log.Printf("✅ User has %d follows already imported", followCount)
		}
	}
}

func createMockUserSourceRelationships(user models.User) {
	log.Printf("💡 Creating mock user-source relationships for %s...", user.Handle)
	
	// Get all sources
	var sources []models.Source
	if err := database.DB.Find(&sources).Error; err != nil {
		log.Printf("❌ Error fetching sources: %v", err)
		return
	}
	
	if len(sources) == 0 {
		log.Printf("⚠️  No sources found to create relationships with")
		return
	}
	
	created := 0
	for _, source := range sources {
		// Check if relationship already exists
		var existing models.UserSource
		if err := database.DB.Where("user_id = ? AND source_id = ?", user.ID, source.ID).First(&existing).Error; err != nil {
			// Relationship doesn't exist, create it
			userSource := models.UserSource{
				UserID:   user.ID,
				SourceID: source.ID,
			}
			
			if err := database.DB.Create(&userSource).Error; err != nil {
				log.Printf("❌ Failed to create user-source relationship for %s: %v", source.Handle, err)
				continue
			}
			created++
		}
	}
	
	log.Printf("✅ Created %d user-source relationships for %s", created, user.Handle)
}

func importTestUserFollows(user models.User) {
	log.Printf("📥 Attempting to import follows for %s using UserFollowsService...", user.Handle)
	
	// Initialize Bluesky client
	client := bluesky.NewClient("https://bsky.social")
	
	// Check if we have credentials for authentication
	identifier := os.Getenv("BLUESKY_IDENTIFIER")
	password := os.Getenv("BLUESKY_PASSWORD")
	
	if identifier != "" && password != "" {
		log.Printf("🔐 Found Bluesky credentials, authenticating for real follow import...")
		if err := client.CreateSession(identifier, password); err != nil {
			log.Printf("❌ Failed to authenticate with Bluesky: %v", err)
			log.Printf("💡 Falling back to mock data...")
			fallbackToMockSources(user)
			return
		}
		
		// Resolve real DID if user has a test DID
		if strings.Contains(user.BlueSkyDID, "test-") {
			log.Printf("🔍 Resolving real DID for %s...", user.Handle)
			realDID, err := client.ResolveHandle(user.Handle)
			if err != nil {
				log.Printf("❌ Failed to resolve real DID: %v", err)
				fallbackToMockSources(user)
				return
			}
			
			log.Printf("✅ Resolved real DID: %s", realDID)
			
			// Update user with real DID
			if err := database.DB.Model(&user).Update("blue_sky_d_id", realDID).Error; err != nil {
				log.Printf("❌ Failed to update user DID: %v", err)
				fallbackToMockSources(user)
				return
			}
			user.BlueSkyDID = realDID
		}
		
		log.Printf("✅ Successfully authenticated, importing real follows...")
	} else {
		log.Printf("💡 No Bluesky credentials found, creating mock sources for testing...")
		fallbackToMockSources(user)
		return
	}
	
	userFollowsService := services.NewUserFollowsService(database.DB, client)
	
	// Create configuration for follow import with authentication
	config := services.RefreshConfig{
		RefreshInterval: 24 * time.Hour,
		BatchSize:       100, // Larger batch for seeding
		RateLimit:       200 * time.Millisecond, // Faster for seeding
	}
	
	// Use the systematic follow refresh service
	if err := userFollowsService.ImportUserFollows(&user, config); err != nil {
		log.Printf("⚠️  Could not import follows with UserFollowsService: %v", err)
		log.Printf("💡 Creating mock sources for testing...")
		fallbackToMockSources(user)
		return
	}
	
	// Check results
	var followCount int64
	database.DB.Model(&models.UserSource{}).Where("user_id = ?", user.ID).Count(&followCount)
	log.Printf("✅ Successfully imported %d real follows for %s", followCount, user.Handle)
}

func fallbackToMockSources(user models.User) {
	log.Printf("💡 Creating mock sources and relationships for %s...", user.Handle)
	
	// Only create mock sources if we don't have real authentication
	seedPopularSources()
	createMockUserSourceRelationships(user)
}

func seedPopularSources() {
	log.Println("🌱 Seeding popular Bluesky accounts as sources...")
	
	// These are real, popular Bluesky accounts that often share interesting content
	popularSources := []models.Source{
		{
			BlueSkyDID:   "did:plc:z72i7hdynmk6r22z27h6tvur",
			Handle:       "bsky.app",
			DisplayName:  "Bluesky",
			Bio:          "Official Bluesky account",
			IsVerified:   true,
			QualityScore: 1.0,
		},
		{
			BlueSkyDID:   "did:plc:ewvi7nxzyoun6zhxrhs64oiz",
			Handle:       "atproto.com",
			DisplayName:  "AT Protocol",
			Bio:          "The AT Protocol team",
			IsVerified:   true,
			QualityScore: 0.95,
		},
		{
			BlueSkyDID:   "did:plc:vpkhqolt662uhesyj6nxm7ys",
			Handle:       "jay.bsky.team",
			DisplayName:  "Jay Graber",
			Bio:          "CEO of Bluesky",
			IsVerified:   true,
			QualityScore: 0.9,
		},
		{
			BlueSkyDID:   "did:plc:test-news-source-1",
			Handle:       "techcrunch.bsky.social",
			DisplayName:  "TechCrunch",
			Bio:          "Technology news and startup coverage",
			IsVerified:   true,
			QualityScore: 0.85,
		},
		{
			BlueSkyDID:   "did:plc:test-news-source-2",
			Handle:       "reuters.bsky.social",
			DisplayName:  "Reuters",
			Bio:          "Breaking news and global coverage",
			IsVerified:   true,
			QualityScore: 0.95,
		},
		{
			BlueSkyDID:   "did:plc:test-news-source-3",
			Handle:       "theeconomist.bsky.social",
			DisplayName:  "The Economist",
			Bio:          "International news and analysis",
			IsVerified:   true,
			QualityScore: 0.9,
		},
		{
			BlueSkyDID:   "did:plc:test-news-source-4",
			Handle:       "nature.bsky.social",
			DisplayName:  "Nature",
			Bio:          "Scientific research and discoveries",
			IsVerified:   true,
			QualityScore: 0.92,
		},
		{
			BlueSkyDID:   "did:plc:test-news-source-5",
			Handle:       "arxiv.bsky.social",
			DisplayName:  "arXiv",
			Bio:          "Preprint repository for scientific papers",
			IsVerified:   false,
			QualityScore: 0.88,
		},
		{
			BlueSkyDID:   "did:plc:ewvi7nxzyoun6zhxrhs64oiz",
			Handle:       "atproto.com",
			DisplayName:  "AT Protocol",
			Bio:          "The AT Protocol team",
			IsVerified:   true,
			QualityScore: 0.95,
		},
		{
			BlueSkyDID:   "did:plc:vpkhqolt662uhesyj6nxm7ys",
			Handle:       "jay.bsky.team",
			DisplayName:  "Jay Graber",
			Bio:          "CEO of Bluesky",
			IsVerified:   true,
			QualityScore: 0.9,
		},
	}

	for _, source := range popularSources {
		var existing models.Source
		if err := database.DB.Where("blue_sky_d_id = ?", source.BlueSkyDID).First(&existing).Error; err != nil {
			if err := database.DB.Create(&source).Error; err != nil {
				log.Printf("❌ Failed to create source %s: %v", source.Handle, err)
			} else {
				log.Printf("✅ Created source: %s", source.Handle)
			}
		} else {
			log.Printf("✅ Source already exists: %s", source.Handle)
		}
	}
}

// seedArticles seeds the database with test articles
func seedArticles(authenticatedClient *bluesky.Client, handle string) {
	log.Printf("📰 Seeding articles...")
	
	// Check if we already have articles
	var articleCount int64
	database.DB.Model(&models.Article{}).Count(&articleCount)
	
	if articleCount > 0 {
		log.Printf("✅ Database already has %d articles, skipping article seeding", articleCount)
		return
	}
	
	// Use authenticated client if available, otherwise create a new one
	var client *bluesky.Client
	if authenticatedClient != nil {
		client = authenticatedClient
		log.Printf("🔐 Using authenticated Bluesky client for article import")
	} else {
		client = bluesky.NewClient("https://bsky.social")
		log.Printf("⚠️  No authenticated client available, using unauthenticated client")
	}
	
	articlesService := services.NewArticlesService(database.DB, client)
	
	// Configure article seeding
	config := services.ArticleSeedConfig{
		MaxArticles:   20,                     // Create 20 test articles
		TimeWindow:    24 * time.Hour,         // Look back 24 hours
		RateLimit:     100 * time.Millisecond, // Fast for seeding
		SampleSources: 10,                     // Sample from 10 sources
	}
	
	// Try to import real articles from Bluesky first
	log.Printf("🔄 Attempting to import recent articles from Bluesky...")
	if err := articlesService.ImportArticlesFromSources(config); err != nil {
		log.Printf("⚠️  Technical error importing articles from Bluesky: %v", err)
		log.Printf("💡 Creating mock articles for testing...")
		
		// Fall back to creating mock articles for development only on technical errors
		if err := articlesService.CreateMockArticles(config); err != nil {
			log.Printf("❌ Failed to create mock articles: %v", err)
		}
	} else {
		// Check if we actually got any articles
		var articleCount int64
		database.DB.Model(&models.Article{}).Count(&articleCount)
		if articleCount == 0 {
			log.Printf("📰 No articles found in recent posts from followed sources")
			if handle == "" {
				log.Printf("💡 Creating mock articles for UI testing (mock mode)...")
				
				// Create mock articles only when in mock mode (no handle provided)
				mockConfig := config
				mockConfig.MaxArticles = 10 // More articles for UI testing in mock mode
				if err := articlesService.CreateMockArticles(mockConfig); err != nil {
					log.Printf("❌ Failed to create mock articles: %v", err)
				}
			} else {
				log.Printf("ℹ️  No mock articles created with real handle - this ensures data integrity")
				log.Printf("💡 To test the UI with sample data, run: go run cmd/seed.go (without -handle)")
			}
		} else {
			log.Printf("✅ Found %d real articles from followed sources", articleCount)
		}
	}
}
