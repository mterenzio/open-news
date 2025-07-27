package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"open-news/internal/auth"
	"open-news/internal/bluesky"
	"open-news/internal/feeds"
	"open-news/internal/models"
	"open-news/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BlueSkyFeedHandler handles custom Bluesky feed requests
type BlueSkyFeedHandler struct {
	db                 *gorm.DB
	feedService        *feeds.FeedService
	blueskyClient      *bluesky.Client
	userFollowsService *services.UserFollowsService
	jwtVerifier        interface {
		ValidateToken(authHeader string) (string, bool)
		ExtractDIDFromToken(tokenString string) (string, error)
	}
}

// NewBlueSkyFeedHandler creates a new Bluesky feed handler
func NewBlueSkyFeedHandler(db *gorm.DB, blueskyClient *bluesky.Client) *BlueSkyFeedHandler {
	var jwtVerifier interface {
		ValidateToken(authHeader string) (string, bool)
		ExtractDIDFromToken(tokenString string) (string, error)
	}
	
	// Use real JWT verification in production
	if os.Getenv("GIN_MODE") == "release" {
		log.Println("Initializing production JWT verifier")
		jwtVerifier = auth.NewJWTVerifier()
	} else {
		log.Println("Initializing mock JWT verifier for development")
		jwtVerifier = auth.NewMockJWTVerifier()
	}
	
	return &BlueSkyFeedHandler{
		db:                 db,
		feedService:        feeds.NewFeedService(db),
		blueskyClient:      blueskyClient,
		userFollowsService: services.NewUserFollowsService(db, blueskyClient),
		jwtVerifier:        jwtVerifier,
	}
}

// ATProtoFeedResponse represents the AT Protocol feed format
type ATProtoFeedResponse struct {
	Feed   []ATProtoFeedItem `json:"feed"`
	Cursor *string           `json:"cursor,omitempty"`
}

// ATProtoFeedItem represents a single item in the AT Protocol feed
type ATProtoFeedItem struct {
	Post   ATProtoPost   `json:"post"`
	Reason *ATProtoReason `json:"reason,omitempty"`
}

// ATProtoPost represents a post in the feed
type ATProtoPost struct {
	URI       string         `json:"uri"`
	CID       string         `json:"cid"`
	Author    ATProtoAuthor  `json:"author"`
	Record    ATProtoRecord  `json:"record"`
	IndexedAt time.Time      `json:"indexedAt"`
}

// ATProtoAuthor represents the author of a post
type ATProtoAuthor struct {
	DID         string  `json:"did"`
	Handle      string  `json:"handle"`
	DisplayName *string `json:"displayName,omitempty"`
	Avatar      *string `json:"avatar,omitempty"`
}

// ATProtoRecord represents the post content
type ATProtoRecord struct {
	Type      string                 `json:"$type"`
	Text      string                 `json:"text"`
	CreatedAt time.Time              `json:"createdAt"`
	Embed     *ATProtoEmbed          `json:"embed,omitempty"`
	Facets    []map[string]interface{} `json:"facets,omitempty"`
}

// ATProtoEmbed represents embedded content
type ATProtoEmbed struct {
	Type     string                 `json:"$type"`
	External *ATProtoExternalEmbed  `json:"external,omitempty"`
}

// ATProtoExternalEmbed represents external link embed
type ATProtoExternalEmbed struct {
	URI         string  `json:"uri"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Thumb       *string `json:"thumb,omitempty"`
}

// ATProtoReason represents why this post is in the feed
type ATProtoReason struct {
	Type string `json:"$type"`
	By   *ATProtoAuthor `json:"by,omitempty"`
}

// GetGlobalFeed handles custom Bluesky feed requests for global feed
// GET /xrpc/app.bsky.feed.getFeedSkeleton?feed=at://did:plc:example/app.bsky.feed.generator/open-news-global
func (h *BlueSkyFeedHandler) GetGlobalFeed(c *gin.Context) {
	// Extract authorization header to get requesting user's DID
	authHeader := c.GetHeader("Authorization")
	userDID := h.extractDIDFromAuth(authHeader)
	
	// If we have a user DID, ensure they exist in our system
	if userDID != "" {
		if err := h.ensureUserExists(userDID); err != nil {
			log.Printf("Failed to ensure user exists for DID %s: %v", userDID, err)
		}
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	// cursor := c.DefaultQuery("cursor", "") // TODO: Implement cursor-based pagination
	
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 30
	}

	// Get our internal global feed
	feedResponse, err := h.feedService.GetGlobalFeed(limit, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"message": "Failed to retrieve global feed",
			},
		})
		return
	}

	// Convert to AT Protocol format
	atProtoFeed := h.convertToATProtoFeed(feedResponse.Items)
	
	response := ATProtoFeedResponse{
		Feed: atProtoFeed,
	}
	
	// Add cursor for pagination (simplified - using timestamp)
	if len(atProtoFeed) > 0 {
		nextCursor := fmt.Sprintf("%d", time.Now().Unix())
		response.Cursor = &nextCursor
	}

	c.JSON(http.StatusOK, response)
}

// GetPersonalizedFeed handles custom Bluesky feed requests for personalized feed
// GET /xrpc/app.bsky.feed.getFeedSkeleton?feed=at://did:plc:example/app.bsky.feed.generator/open-news-personal
func (h *BlueSkyFeedHandler) GetPersonalizedFeed(c *gin.Context) {
	// Extract authorization header to get requesting user's DID
	authHeader := c.GetHeader("Authorization")
	userDID := h.extractDIDFromAuth(authHeader)
	
	if userDID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": map[string]interface{}{
				"message": "Authentication required",
			},
		})
		return
	}

	// Ensure user exists and is set up with their follows
	user, err := h.ensureUserExistsWithFollows(userDID)
	if err != nil {
		log.Printf("Failed to ensure user exists with follows for DID %s: %v", userDID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"message": "Failed to setup user account",
			},
		})
		return
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	// cursor := c.DefaultQuery("cursor", "") // TODO: Implement cursor-based pagination
	
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 30
	}

	// Get personalized feed for this user
	feedResponse, err := h.feedService.GetPersonalizedFeed(user.ID, limit, 0)
	if err != nil {
		// If no personalized feed exists, fall back to global feed filtered by user's sources
		feedResponse, err = h.getFilteredGlobalFeed(user.ID, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": map[string]interface{}{
					"message": "Failed to retrieve personalized feed",
				},
			})
			return
		}
	}

	// Convert to AT Protocol format
	atProtoFeed := h.convertToATProtoFeed(feedResponse.Items)
	
	response := ATProtoFeedResponse{
		Feed: atProtoFeed,
	}
	
	// Add cursor for pagination
	if len(atProtoFeed) > 0 {
		nextCursor := fmt.Sprintf("%d", time.Now().Unix())
		response.Cursor = &nextCursor
	}

	c.JSON(http.StatusOK, response)
}

// extractDIDFromAuth extracts the DID from the Authorization header
func (h *BlueSkyFeedHandler) extractDIDFromAuth(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	
	// Remove "Bearer " prefix
	if strings.HasPrefix(authHeader, "Bearer ") {
		// Use the JWT verifier to validate and extract DID
		did, valid := h.jwtVerifier.ValidateToken(authHeader)
		if valid {
			return did
		}
	}
	
	return ""
}

// ensureUserExists creates a user record if it doesn't exist
func (h *BlueSkyFeedHandler) ensureUserExists(did string) error {
	var user models.User
	err := h.db.Where("blue_sky_d_id = ?", did).First(&user).Error
	
	if err == gorm.ErrRecordNotFound {
		// User doesn't exist, create them
		// Get profile from Bluesky
		profile, err := h.blueskyClient.GetProfile(did)
		if err != nil {
			// If we can't get profile, create with minimal info
			user = models.User{
				BlueSkyDID:  did,
				Handle:      did, // Use DID as handle fallback
				DisplayName: "",
				IsActive:    true,
			}
		} else {
			user = models.User{
				BlueSkyDID:  did,
				Handle:      profile.Handle,
				DisplayName: profile.DisplayName,
				Avatar:      profile.Avatar,
				IsActive:    true,
			}
		}
		
		if err := h.db.Create(&user).Error; err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		
		log.Printf("Created new user from DID: %s (%s)", did, user.Handle)
	} else if err != nil {
		return fmt.Errorf("failed to query user: %w", err)
	}
	
	return nil
}

// ensureUserExistsWithFollows creates user and imports their follows as sources
func (h *BlueSkyFeedHandler) ensureUserExistsWithFollows(did string) (*models.User, error) {
	var user models.User
	err := h.db.Where("blue_sky_d_id = ?", did).First(&user).Error
	
	isNewUser := false
	if err == gorm.ErrRecordNotFound {
		isNewUser = true
		// Create user first
		if err := h.ensureUserExists(did); err != nil {
			return nil, err
		}
		
		// Get the created user
		if err := h.db.Where("blue_sky_d_id = ?", did).First(&user).Error; err != nil {
			return nil, fmt.Errorf("failed to retrieve created user: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	
	// Use the new UserFollowsService to handle follow import
	defaultConfig := services.RefreshConfig{
		RefreshInterval: 24 * time.Hour,
		BatchSize:       10,
		RateLimit:       100 * time.Millisecond,
	}
	
	if isNewUser || h.userFollowsService.ShouldRefreshFollows(&user, defaultConfig) {
		if err := h.userFollowsService.ImportUserFollows(&user, defaultConfig); err != nil {
			log.Printf("Failed to import follows for user %s: %v", user.Handle, err)
			// Don't fail the request if follow import fails
		}
	}
	
	return &user, nil
}

// getFilteredGlobalFeed gets global feed filtered by user's sources
func (h *BlueSkyFeedHandler) getFilteredGlobalFeed(userID uuid.UUID, limit int) (*feeds.FeedResponse, error) {
	// Get global feed but filter by articles from user's sources
	var feedItems []models.FeedItem
	
	query := h.db.Table("feed_items").
		Select("feed_items.*").
		Joins("JOIN feeds ON feeds.id = feed_items.feed_id").
		Joins("JOIN articles ON articles.id = feed_items.article_id").
		Joins("JOIN source_articles ON source_articles.article_id = articles.id").
		Joins("JOIN sources ON sources.id = source_articles.source_id").
		Joins("JOIN user_sources ON user_sources.source_id = sources.id").
		Where("feeds.feed_type = ? AND feeds.name = ?", "global", "Top Stories").
		Where("user_sources.user_id = ?", userID).
		Preload("Article").
		Preload("Article.SourceArticles.Source").
		Order("feed_items.position ASC").
		Limit(limit)
	
	if err := query.Find(&feedItems).Error; err != nil {
		return nil, err
	}
	
	// Get the global feed for metadata
	var globalFeed models.Feed
	if err := h.db.Where("feed_type = ? AND name = ?", "global", "Top Stories").First(&globalFeed).Error; err != nil {
		return nil, err
	}
	
	// Transform to response format (similar to feeds service)
	items := make([]feeds.FeedItemDetails, len(feedItems))
	for i, item := range feedItems {
		var source feeds.Source
		if len(item.Article.SourceArticles) > 0 {
			src := item.Article.SourceArticles[0].Source
			source = feeds.Source{
				ID:           src.ID,
				Handle:       src.Handle,
				DisplayName:  src.DisplayName,
				Avatar:       src.Avatar,
				QualityScore: src.QualityScore,
			}
		}
		
		items[i] = feeds.FeedItemDetails{
			FeedItem: item,
			Article: feeds.Article{
				ID:           item.Article.ID,
				URL:          item.Article.URL,
				Title:        item.Article.Title,
				Description:  item.Article.Description,
				ImageURL:     item.Article.ImageURL,
				PublishedAt:  item.Article.PublishedAt,
				SiteName:     item.Article.SiteName,
				QualityScore: item.Article.QualityScore,
			},
			Source: source,
		}
	}
	
	return &feeds.FeedResponse{
		Feed:  globalFeed,
		Items: items,
		Meta: feeds.FeedMeta{
			TotalItems:    len(items),
			Page:          1,
			PerPage:       limit,
			LastUpdatedAt: globalFeed.UpdatedAt,
		},
	}, nil
}

// convertToATProtoFeed converts internal feed items to AT Protocol format
func (h *BlueSkyFeedHandler) convertToATProtoFeed(items []feeds.FeedItemDetails) []ATProtoFeedItem {
	atProtoItems := make([]ATProtoFeedItem, 0, len(items))
	
	for _, item := range items {
		// Create a synthetic post URI (in real implementation, you'd use actual post URIs)
		postURI := fmt.Sprintf("at://%s/app.bsky.feed.post/%s", 
			item.Source.Handle, item.Article.ID.String())
		
		// Create external embed for the article
		var embed *ATProtoEmbed
		if item.Article.URL != "" {
			embed = &ATProtoEmbed{
				Type: "app.bsky.embed.external",
				External: &ATProtoExternalEmbed{
					URI:         item.Article.URL,
					Title:       item.Article.Title,
					Description: item.Article.Description,
				},
			}
			
			// Add thumbnail if available
			if item.Article.ImageURL != "" {
				embed.External.Thumb = &item.Article.ImageURL
			}
		}
		
		// Create post text
		postText := item.Article.Title
		if len(postText) > 280 { // Bluesky character limit
			postText = postText[:277] + "..."
		}
		
		// Use published date or fallback to when added to feed
		createdAt := item.FeedItem.AddedAt.UTC()
		if item.Article.PublishedAt != nil {
			createdAt = item.Article.PublishedAt.UTC()
		}
		
		// Create the AT Protocol post
		atProtoPost := ATProtoPost{
			URI: postURI,
			CID: fmt.Sprintf("bafyrei%s", item.Article.ID.String()[:20]), // Synthetic CID
			Author: ATProtoAuthor{
				DID:    fmt.Sprintf("did:plc:%s", item.Source.Handle),
				Handle: item.Source.Handle,
			},
			Record: ATProtoRecord{
				Type:      "app.bsky.feed.post",
				Text:      postText,
				CreatedAt: createdAt,
				Embed:     embed,
			},
			IndexedAt: item.FeedItem.AddedAt.UTC(),
		}
		
		// Add display name and avatar if available
		if item.Source.DisplayName != "" {
			atProtoPost.Author.DisplayName = &item.Source.DisplayName
		}
		if item.Source.Avatar != "" {
			atProtoPost.Author.Avatar = &item.Source.Avatar
		}
		
		atProtoItems = append(atProtoItems, ATProtoFeedItem{
			Post: atProtoPost,
		})
	}
	
	return atProtoItems
}

// GetFeedInfo returns information about the custom feeds
func (h *BlueSkyFeedHandler) GetFeedInfo(c *gin.Context) {
	feedURI := c.Query("feed")
	
	var feedInfo map[string]interface{}
	
	if strings.Contains(feedURI, "open-news-global") {
		feedInfo = map[string]interface{}{
			"uri":         feedURI,
			"displayName": "Open News - Global",
			"description": "Top stories from across the Bluesky network, ranked by engagement and quality.",
			"avatar":      "", // Add your feed avatar URL here
			"createdBy":   "did:plc:your-feed-generator-did", // Your feed generator's DID
		}
	} else if strings.Contains(feedURI, "open-news-personal") {
		feedInfo = map[string]interface{}{
			"uri":         feedURI,
			"displayName": "Open News - Personal",
			"description": "Personalized news feed based on accounts you follow on Bluesky.",
			"avatar":      "", // Add your feed avatar URL here
			"createdBy":   "did:plc:your-feed-generator-did", // Your feed generator's DID
		}
	} else {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]interface{}{
				"message": "Feed not found",
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, feedInfo)
}
