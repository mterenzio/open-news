package feeds

import (
	"open-news/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FeedService handles feed operations
type FeedService struct {
	db *gorm.DB
}

// NewFeedService creates a new feed service
func NewFeedService(db *gorm.DB) *FeedService {
	return &FeedService{db: db}
}

// FeedResponse represents the structure returned by feed endpoints
type FeedResponse struct {
	Feed  models.Feed       `json:"feed"`
	Items []FeedItemDetails `json:"items"`
	Meta  FeedMeta          `json:"meta"`
}

// FeedItemDetails includes article and source information for feed items
type FeedItemDetails struct {
	models.FeedItem
	Article Article `json:"article"`
	Source  Source  `json:"source"`
}

// Article represents simplified article data for feed responses
type Article struct {
	ID          uuid.UUID  `json:"id"`
	URL         string     `json:"url"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	ImageURL    string     `json:"image_url"`
	PublishedAt *time.Time `json:"published_at"`
	SiteName    string     `json:"site_name"`
	QualityScore float64   `json:"quality_score"`
}

// Source represents simplified source data for feed responses
type Source struct {
	ID           uuid.UUID `json:"id"`
	Handle       string    `json:"handle"`
	DisplayName  string    `json:"display_name"`
	Avatar       string    `json:"avatar"`
	QualityScore float64   `json:"quality_score"`
}

// FeedMeta contains metadata about the feed
type FeedMeta struct {
	TotalItems    int       `json:"total_items"`
	Page          int       `json:"page"`
	PerPage       int       `json:"per_page"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
}

// GetGlobalFeed returns the global top stories feed
func (fs *FeedService) GetGlobalFeed(limit, offset int) (*FeedResponse, error) {
	// Get or create global feed
	var globalFeed models.Feed
	err := fs.db.Where("feed_type = ? AND name = ?", "global", "Top Stories").
		First(&globalFeed).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create global feed if it doesn't exist
		globalFeed = models.Feed{
			Name:        "Top Stories",
			Description: "Global top stories from all sources",
			FeedType:    "global",
			MaxItems:    100,
			RefreshRate: 300,
		}
		if err := fs.db.Create(&globalFeed).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// Get feed items with articles and sources
	var feedItems []models.FeedItem
	err = fs.db.Preload("Article").
		Preload("Article.SourceArticles.Source").
		Where("feed_id = ?", globalFeed.ID).
		Order("position ASC").
		Limit(limit).
		Offset(offset).
		Find(&feedItems).Error
	
	if err != nil {
		return nil, err
	}

	// Transform to response format
	items := make([]FeedItemDetails, len(feedItems))
	for i, item := range feedItems {
		// Get the primary source for this article (first one for now)
		var source Source
		if len(item.Article.SourceArticles) > 0 {
			src := item.Article.SourceArticles[0].Source
			source = Source{
				ID:           src.ID,
				Handle:       src.Handle,
				DisplayName:  src.DisplayName,
				Avatar:       src.Avatar,
				QualityScore: src.QualityScore,
			}
		}

		items[i] = FeedItemDetails{
			FeedItem: item,
			Article: Article{
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

	// Get total count
	var totalCount int64
	fs.db.Model(&models.FeedItem{}).Where("feed_id = ?", globalFeed.ID).Count(&totalCount)

	return &FeedResponse{
		Feed:  globalFeed,
		Items: items,
		Meta: FeedMeta{
			TotalItems:    int(totalCount),
			Page:          offset/limit + 1,
			PerPage:       limit,
			LastUpdatedAt: globalFeed.UpdatedAt,
		},
	}, nil
}

// GetPersonalizedFeed returns a personalized feed for a specific user
func (fs *FeedService) GetPersonalizedFeed(userID uuid.UUID, limit, offset int) (*FeedResponse, error) {
	// Get or create personalized feed for user
	var personalizedFeed models.Feed
	err := fs.db.Where("feed_type = ? AND name = ?", "personalized", "Personal Feed").
		First(&personalizedFeed).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create personalized feed if it doesn't exist
		personalizedFeed = models.Feed{
			Name:        "Personal Feed",
			Description: "Personalized feed based on your interests",
			FeedType:    "personalized",
			MaxItems:    100,
			RefreshRate: 300,
		}
		if err := fs.db.Create(&personalizedFeed).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// Get feed items for this user
	var feedItems []models.FeedItem
	err = fs.db.Preload("Article").
		Preload("Article.SourceArticles.Source").
		Where("feed_id = ? AND user_id = ?", personalizedFeed.ID, userID).
		Order("position ASC").
		Limit(limit).
		Offset(offset).
		Find(&feedItems).Error
	
	if err != nil {
		return nil, err
	}

	// Transform to response format (same as global feed)
	items := make([]FeedItemDetails, len(feedItems))
	for i, item := range feedItems {
		var source Source
		if len(item.Article.SourceArticles) > 0 {
			src := item.Article.SourceArticles[0].Source
			source = Source{
				ID:           src.ID,
				Handle:       src.Handle,
				DisplayName:  src.DisplayName,
				Avatar:       src.Avatar,
				QualityScore: src.QualityScore,
			}
		}

		items[i] = FeedItemDetails{
			FeedItem: item,
			Article: Article{
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

	// Get total count
	var totalCount int64
	fs.db.Model(&models.FeedItem{}).
		Where("feed_id = ? AND user_id = ?", personalizedFeed.ID, userID).
		Count(&totalCount)

	return &FeedResponse{
		Feed:  personalizedFeed,
		Items: items,
		Meta: FeedMeta{
			TotalItems:    int(totalCount),
			Page:          offset/limit + 1,
			PerPage:       limit,
			LastUpdatedAt: personalizedFeed.UpdatedAt,
		},
	}, nil
}

// RegenerateGlobalFeed regenerates the global feed by creating feed items from top articles
func (fs *FeedService) RegenerateGlobalFeed() error {
	// Get or create global feed
	var globalFeed models.Feed
	err := fs.db.Where("feed_type = ? AND name = ?", "global", "Top Stories").
		First(&globalFeed).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create global feed if it doesn't exist
		globalFeed = models.Feed{
			Name:        "Top Stories",
			Description: "Global top stories from all sources",
			FeedType:    "global",
			MaxItems:    100,
			RefreshRate: 300,
		}
		if err := fs.db.Create(&globalFeed).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Clear existing feed items for this feed
	if err := fs.db.Where("feed_id = ?", globalFeed.ID).Delete(&models.FeedItem{}).Error; err != nil {
		return err
	}

	// Get top articles from the last 7 days with quality scores > 0
	cutoffDate := time.Now().AddDate(0, 0, -7)
	var articles []models.Article
	
	err = fs.db.Where("created_at > ? AND quality_score > 0", cutoffDate).
		Order("quality_score DESC, trending_score DESC, created_at DESC").
		Limit(100).
		Find(&articles).Error
	
	if err != nil {
		return err
	}

	// Create feed items for each article
	var feedItems []models.FeedItem
	for i, article := range articles {
		// Calculate position-based score (higher for earlier positions)
		positionBonus := float64(len(articles)-i) / float64(len(articles)) * 0.1
		
		// Combine article scores
		finalScore := article.QualityScore + (article.TrendingScore * 0.3) + positionBonus

		feedItem := models.FeedItem{
			ID:        uuid.New(),
			FeedID:    globalFeed.ID,
			ArticleID: article.ID,
			Position:  i + 1,
			Score:     finalScore,
			Relevance: article.QualityScore,
			AddedAt:   time.Now(),
		}
		
		feedItems = append(feedItems, feedItem)
	}

	// Batch insert feed items
	if len(feedItems) > 0 {
		if err := fs.db.CreateInBatches(feedItems, 50).Error; err != nil {
			return err
		}
	}

	// Update feed timestamp
	globalFeed.UpdatedAt = time.Now()
	if err := fs.db.Save(&globalFeed).Error; err != nil {
		return err
	}

	return nil
}
