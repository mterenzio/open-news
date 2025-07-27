package handlers

import (
	"net/http"
	"strconv"

	"open-news/internal/feeds"
	"open-news/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FeedHandler handles HTTP requests for feeds
type FeedHandler struct {
	feedService   *feeds.FeedService
	workerService *worker.WorkerService
}

// NewFeedHandler creates a new feed handler
func NewFeedHandler(db *gorm.DB, workerService *worker.WorkerService) *FeedHandler {
	return &FeedHandler{
		feedService:   feeds.NewFeedService(db),
		workerService: workerService,
	}
}

// GetGlobalFeed handles GET /api/feeds/global
func (h *FeedHandler) GetGlobalFeed(c *gin.Context) {
	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}
	
	offset := (page - 1) * limit

	// Get the global feed
	feedResponse, err := h.feedService.GetGlobalFeed(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve global feed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, feedResponse)
}

// GetPersonalizedFeed handles GET /api/feeds/personalized
func (h *FeedHandler) GetPersonalizedFeed(c *gin.Context) {
	// Get user ID from context (would be set by auth middleware)
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User authentication required",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}
	
	offset := (page - 1) * limit

	// Get the personalized feed
	feedResponse, err := h.feedService.GetPersonalizedFeed(userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve personalized feed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, feedResponse)
}

// HealthCheck handles GET /health
func (h *FeedHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "open-news",
	})
}

// WorkerStatus handles GET /api/worker/status
func (h *FeedHandler) WorkerStatus(c *gin.Context) {
	status := h.workerService.GetStatus()
	c.JSON(http.StatusOK, gin.H{
		"worker_status": status,
	})
}
