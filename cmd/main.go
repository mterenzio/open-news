package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"open-news/internal/bluesky"
	"open-news/internal/database"
	"open-news/internal/handlers"
	"open-news/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
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

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Initialize and start background workers
	workerService := worker.NewWorkerService()
	if err := workerService.Start(); err != nil {
		log.Fatal("Failed to start background workers:", err)
	}

	// Setup graceful shutdown
	setupGracefulShutdown(workerService)

	// Setup HTTP server
	setupServer(workerService)
}

func setupGracefulShutdown(workerService *worker.WorkerService) {
	// Setup signal handling for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Received shutdown signal, gracefully shutting down...")
		
		// Stop background workers
		workerService.Stop()
		
		// Close database connection
		database.Close()
		
		log.Println("Shutdown complete")
		os.Exit(0)
	}()
}

func setupServer(workerService *worker.WorkerService) {
	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Initialize handlers
	feedHandler := handlers.NewFeedHandler(database.DB, workerService)
	feedPageHandler := handlers.NewFeedPageHandler(database.DB)
	adminHandler := handlers.NewAdminHandler(database.DB, workerService.GetUserFollowsService())
	docsHandler := handlers.NewDocsHandler()
	
	// Initialize Bluesky client for custom feeds
	blueskyBaseURL := os.Getenv("BLUESKY_BASE_URL")
	if blueskyBaseURL == "" {
		blueskyBaseURL = "https://bsky.social"
	}
	blueskyClient := bluesky.NewClient(blueskyBaseURL)
	blueskyFeedHandler := handlers.NewBlueSkyFeedHandler(database.DB, blueskyClient)

	// Health check
	r.GET("/health", feedHandler.HealthCheck)

	// Serve static files for DID document
	r.Static("/.well-known", "./static/.well-known")
	r.Static("/static", "./static")
	
	// Serve documentation and home page
	r.Static("/docs", "./static/docs")
	r.StaticFile("/", "./static/index.html")
	r.StaticFile("/index.html", "./static/index.html")
	r.StaticFile("/widget-examples.html", "./static/widget-examples.html")
	
	// Feed web interface
	r.GET("/feeds", feedPageHandler.ServeMainFeedPage)
	r.GET("/feed/global", feedPageHandler.ServeGlobalFeedHTML)
	r.GET("/feed/personal", feedPageHandler.ServePersonalFeedHTML)
	
	// Embeddable widgets
	r.GET("/widget/global", feedPageHandler.ServeGlobalWidget)
	r.GET("/widget/personal", feedPageHandler.ServePersonalWidget)
	
	// Serve Markdown documentation as HTML
	r.GET("/doc/:doc", docsHandler.ServeMarkdownAsHTML)

	// AT Protocol custom feed endpoints
	xrpc := r.Group("/xrpc")
	{
		xrpc.GET("/app.bsky.feed.getFeedSkeleton", func(c *gin.Context) {
			feedParam := c.Query("feed")
			if strings.Contains(feedParam, "open-news-global") {
				blueskyFeedHandler.GetGlobalFeed(c)
			} else if strings.Contains(feedParam, "open-news-personal") {
				blueskyFeedHandler.GetPersonalizedFeed(c)
			} else {
				c.JSON(http.StatusNotFound, gin.H{
					"error": map[string]interface{}{
						"message": "Feed not found",
					},
				})
			}
		})
		
		xrpc.GET("/app.bsky.feed.describeFeedGenerator", blueskyFeedHandler.GetFeedInfo)
	}

	// API routes
	api := r.Group("/api")
	{
		feeds := api.Group("/feeds")
		{
			feeds.GET("/global", feedHandler.GetGlobalFeed)
			feeds.GET("/personalized", feedHandler.GetPersonalizedFeed)
		}
		
		worker := api.Group("/worker")
		{
			worker.GET("/status", feedHandler.WorkerStatus)
		}
	}

	// Admin routes (password protected)
	admin := r.Group("/admin", adminHandler.AdminAuth())
	{
		admin.GET("/", adminHandler.ServeAdminDashboard)
		admin.GET("/users", adminHandler.ServeUsersPage)
		admin.GET("/sources", adminHandler.ServeSourcesPage)
		admin.GET("/articles", adminHandler.ServeArticlesPage)
		admin.GET("/articles/:id", adminHandler.ServeArticleInspection)
		admin.POST("/refresh-follows", adminHandler.RefreshAllUserFollows)
		admin.POST("/refresh-follows/:user", adminHandler.RefreshUserFollows)
	}

	// Get port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
