package services

import (
	"fmt"
	"log"
	"time"

	"open-news/internal/bluesky"
	"open-news/internal/models"
	"gorm.io/gorm"
)

// UserFollowsService handles importing and updating user follows from Bluesky
type UserFollowsService struct {
	db            *gorm.DB
	blueskyClient *bluesky.Client
}

// NewUserFollowsService creates a new UserFollowsService
func NewUserFollowsService(db *gorm.DB, blueskyClient *bluesky.Client) *UserFollowsService {
	return &UserFollowsService{
		db:            db,
		blueskyClient: blueskyClient,
	}
}

// RefreshConfig holds configuration for follow refresh behavior
type RefreshConfig struct {
	RefreshInterval time.Duration // How often to refresh follows (default: 24 hours)
	BatchSize       int           // How many users to process at once (default: 10)
	RateLimit       time.Duration // Delay between API calls (default: 100ms)
}

// DefaultRefreshConfig returns default configuration for follow refresh
func DefaultRefreshConfig() RefreshConfig {
	return RefreshConfig{
		RefreshInterval: 24 * time.Hour,
		BatchSize:       10,
		RateLimit:       100 * time.Millisecond,
	}
}

// ShouldRefreshFollows determines if a user's follows need refreshing
func (s *UserFollowsService) ShouldRefreshFollows(user *models.User, config RefreshConfig) bool {
	if user.FollowsLastRefreshed == nil {
		return true
	}
	return time.Since(*user.FollowsLastRefreshed) > config.RefreshInterval
}

// ImportUserFollows imports or updates a user's follows from Bluesky
func (s *UserFollowsService) ImportUserFollows(user *models.User, config RefreshConfig) error {
	log.Printf("🔄 Importing follows for user %s (%s)", user.Handle, user.BlueSkyDID)
	
	limit := 100
	cursor := ""
	followsCount := 0
	sourcesCreated := 0
	sourcesUpdated := 0
	relationshipsCreated := 0

	for {
		follows, err := s.blueskyClient.GetFollows(user.BlueSkyDID, limit, cursor)
		if err != nil {
			return fmt.Errorf("failed to get follows from Bluesky: %w", err)
		}

		// Process each follow
		for _, follow := range follows.Follows {
			followsCount++

			// Create or update source record
			var source models.Source
			err := s.db.Where("blue_sky_d_id = ?", follow.DID).First(&source).Error

			if err == gorm.ErrRecordNotFound {
				// Create new source
				source = models.Source{
					BlueSkyDID:   follow.DID,
					Handle:       follow.Handle,
					DisplayName:  follow.DisplayName,
					Avatar:       follow.Avatar,
					QualityScore: 0.5, // Default quality score
				}

				if err := s.db.Create(&source).Error; err != nil {
					log.Printf("❌ Failed to create source for %s: %v", follow.Handle, err)
					continue
				}

				sourcesCreated++
				log.Printf("✅ Created source: %s (%s)", follow.Handle, follow.DID)
			} else if err != nil {
				log.Printf("❌ Failed to query source %s: %v", follow.Handle, err)
				continue
			} else {
				// Update existing source with latest profile info
				updated := false
				if source.Handle != follow.Handle {
					source.Handle = follow.Handle
					updated = true
				}
				if source.DisplayName != follow.DisplayName {
					source.DisplayName = follow.DisplayName
					updated = true
				}
				if source.Avatar != follow.Avatar {
					source.Avatar = follow.Avatar
					updated = true
				}

				if updated {
					if err := s.db.Save(&source).Error; err != nil {
						log.Printf("❌ Failed to update source %s: %v", follow.Handle, err)
					} else {
						sourcesUpdated++
					}
				}
			}

			// Create user-source relationship if it doesn't exist
			var userSource models.UserSource
			err = s.db.Where("user_id = ? AND source_id = ?", user.ID, source.ID).First(&userSource).Error

			if err == gorm.ErrRecordNotFound {
				userSource = models.UserSource{
					UserID:   user.ID,
					SourceID: source.ID,
				}

				if err := s.db.Create(&userSource).Error; err != nil {
					log.Printf("❌ Failed to create user-source relationship for %s: %v", follow.Handle, err)
				} else {
					relationshipsCreated++
				}
			} else if err != nil {
				log.Printf("❌ Failed to query user-source relationship for %s: %v", follow.Handle, err)
			}
		}

		// Check if there are more follows to fetch
		if follows.Cursor == "" || len(follows.Follows) < limit {
			break
		}
		cursor = follows.Cursor

		// Rate limiting
		time.Sleep(config.RateLimit)
	}

	// Update user's follows_last_refreshed timestamp
	now := time.Now()
	user.FollowsLastRefreshed = &now
	if err := s.db.Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user follows timestamp: %w", err)
	}

	log.Printf("✅ Successfully imported %d follows for user %s", followsCount, user.Handle)
	log.Printf("   📊 Stats: %d new sources, %d updated sources, %d new relationships", 
		sourcesCreated, sourcesUpdated, relationshipsCreated)

	return nil
}

// GetUsersNeedingRefresh gets users whose follows need refreshing
func (s *UserFollowsService) GetUsersNeedingRefresh(config RefreshConfig, limit int) ([]models.User, error) {
	var users []models.User
	
	cutoffTime := time.Now().Add(-config.RefreshInterval)
	
	err := s.db.Where("follows_last_refreshed IS NULL OR follows_last_refreshed < ?", cutoffTime).
		Where("is_active = ?", true).
		Limit(limit).
		Find(&users).Error
	
	return users, err
}

// RefreshBatch processes a batch of users for follow refresh
func (s *UserFollowsService) RefreshBatch(config RefreshConfig) error {
	users, err := s.GetUsersNeedingRefresh(config, config.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to get users needing refresh: %w", err)
	}

	if len(users) == 0 {
		log.Printf("📋 No users need follow refresh at this time")
		return nil
	}

	log.Printf("🔄 Processing follow refresh for %d users", len(users))

	for _, user := range users {
		if err := s.ImportUserFollows(&user, config); err != nil {
			log.Printf("⚠️  Failed to refresh follows for user %s: %v", user.Handle, err)
			// Continue with other users even if one fails
		}
		
		// Small delay between users
		time.Sleep(config.RateLimit)
	}

	return nil
}

// EnsureUserExistsWithFollows creates user and imports their follows (for use in feed handlers)
func (s *UserFollowsService) EnsureUserExistsWithFollows(did string, config RefreshConfig) (*models.User, error) {
	var user models.User
	err := s.db.Where("blue_sky_d_id = ?", did).First(&user).Error

	isNewUser := false
	if err == gorm.ErrRecordNotFound {
		isNewUser = true
		// Create basic user record first
		if err := s.createBasicUser(did); err != nil {
			return nil, err
		}

		// Get the created user
		if err := s.db.Where("blue_sky_d_id = ?", did).First(&user).Error; err != nil {
			return nil, fmt.Errorf("failed to retrieve created user: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// If user is new or hasn't had follows imported recently, import them
	if isNewUser || s.ShouldRefreshFollows(&user, config) {
		if err := s.ImportUserFollows(&user, config); err != nil {
			log.Printf("⚠️  Failed to import follows for user %s: %v", user.Handle, err)
			// Don't fail the request if follow import fails
		}
	}

	return &user, nil
}

// createBasicUser creates a basic user record with DID (minimal profile fetch)
func (s *UserFollowsService) createBasicUser(did string) error {
	// Try to get basic profile info
	// For now, create with minimal info - this could be enhanced to fetch profile
	user := models.User{
		BlueSkyDID:  did,
		Handle:      did, // Will be updated when profile is fetched
		DisplayName: did, // Will be updated when profile is fetched
		IsActive:    true,
	}

	return s.db.Create(&user).Error
}
