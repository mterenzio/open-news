package services

import (
	"testing"
	"time"

	"open-news/internal/bluesky"
	"open-news/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBlueskyClient is a mock implementation of the Bluesky client
type MockBlueskyClient struct {
	mock.Mock
}

func (m *MockBlueskyClient) GetFollows(actor string, limit int, cursor string) (*bluesky.FollowsResponse, error) {
	args := m.Called(actor, limit, cursor)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*bluesky.FollowsResponse), args.Error(1)
}

func TestUserFollowsService_ShouldRefreshFollows(t *testing.T) {
	service := &UserFollowsService{}
	config := DefaultRefreshConfig()

	t.Run("should refresh when never refreshed", func(t *testing.T) {
		user := &models.User{
			FollowsLastRefreshed: nil,
		}
		assert.True(t, service.ShouldRefreshFollows(user, config))
	})

	t.Run("should refresh when last refresh is old", func(t *testing.T) {
		oldTime := time.Now().Add(-25 * time.Hour)
		user := &models.User{
			FollowsLastRefreshed: &oldTime,
		}
		assert.True(t, service.ShouldRefreshFollows(user, config))
	})

	t.Run("should not refresh when recently refreshed", func(t *testing.T) {
		recentTime := time.Now().Add(-1 * time.Hour)
		user := &models.User{
			FollowsLastRefreshed: &recentTime,
		}
		assert.False(t, service.ShouldRefreshFollows(user, config))
	})
}

func TestUserFollowsService_GetUsersNeedingRefresh(t *testing.T) {
	db := setupTestDB(t)

	service := &UserFollowsService{db: db}
	config := DefaultRefreshConfig()

	// Create test users
	now := time.Now()
	oldTime := now.Add(-25 * time.Hour)

	users := []models.User{
		{
			ID:         uuid.New(),
			BlueSkyDID: "did:plc:user1",
			Handle:     "user1.bsky.social",
			IsActive:   true,
			FollowsLastRefreshed: nil, // Never refreshed
		},
		{
			ID:         uuid.New(),
			BlueSkyDID: "did:plc:user2",
			Handle:     "user2.bsky.social",
			IsActive:   true,
			FollowsLastRefreshed: &oldTime, // Old refresh
		},
		{
			ID:         uuid.New(),
			BlueSkyDID: "did:plc:user3",
			Handle:     "user3.bsky.social",
			IsActive:   true,
			FollowsLastRefreshed: &now, // Recent refresh
		},
		{
			ID:         uuid.New(),
			BlueSkyDID: "did:plc:user4",
			Handle:     "user4.bsky.social",
			IsActive:   false, // Inactive user
			FollowsLastRefreshed: nil,
		},
	}

	for _, user := range users {
		db.Create(&user)
	}

	// Test getting users needing refresh
	needRefresh, err := service.GetUsersNeedingRefresh(config, 10)
	assert.NoError(t, err)
	
	// Should return users 1 and 2 (never refreshed or old refresh), but not user 3 (recent) or user 4 (inactive)
	assert.Len(t, needRefresh, 2)
	
	handles := make([]string, len(needRefresh))
	for i, user := range needRefresh {
		handles[i] = user.Handle
	}
	assert.Contains(t, handles, "user1.bsky.social")
	assert.Contains(t, handles, "user2.bsky.social")
}

func TestUserFollowsService_ImportUserFollows(t *testing.T) {
	db := setupTestDB(t)
	mockClient := &MockBlueskyClient{}

	service := &UserFollowsService{
		db:            db,
		blueskyClient: mockClient,
	}

	// Create test user
	user := &models.User{
		ID:          uuid.New(),
		BlueSkyDID:  "did:plc:testuser",
		Handle:      "testuser.bsky.social",
		DisplayName: "Test User",
		IsActive:    true,
	}
	db.Create(user)

	// Setup mock responses
	follows := &bluesky.FollowsResponse{
		Follows: []bluesky.Author{
			{
				DID:         "did:plc:follow1",
				Handle:      "follow1.bsky.social",
				DisplayName: "Follow 1",
				Avatar:      "https://example.com/avatar1.jpg",
			},
			{
				DID:         "did:plc:follow2",
				Handle:      "follow2.bsky.social",
				DisplayName: "Follow 2",
				Avatar:      "https://example.com/avatar2.jpg",
			},
		},
		Cursor: "", // No more pages
	}

	mockClient.On("GetFollows", "did:plc:testuser", 100, "").Return(follows, nil)

	config := DefaultRefreshConfig()

	// Test importing follows
	err := service.ImportUserFollows(user, config)
	assert.NoError(t, err)

	// Verify sources were created
	var sources []models.Source
	db.Find(&sources)
	assert.Len(t, sources, 2)

	// Verify user-source relationships were created
	var userSources []models.UserSource
	db.Where("user_id = ?", user.ID).Find(&userSources)
	assert.Len(t, userSources, 2)

	// Verify user's follows_last_refreshed was updated
	db.First(user, user.ID)
	assert.NotNil(t, user.FollowsLastRefreshed)
	assert.WithinDuration(t, time.Now(), *user.FollowsLastRefreshed, time.Minute)

	mockClient.AssertExpectations(t)
}

func TestUserFollowsService_ImportUserFollows_UpdateExistingSource(t *testing.T) {
	db := setupTestDB(t)
	mockClient := &MockBlueskyClient{}

	service := &UserFollowsService{
		db:            db,
		blueskyClient: mockClient,
	}

	// Create test user
	user := &models.User{
		ID:          uuid.New(),
		BlueSkyDID:  "did:plc:testuser2",
		Handle:      "testuser2.bsky.social",
		DisplayName: "Test User 2",
		IsActive:    true,
	}
	db.Create(user)

	// Create existing source with old info
	existingSource := &models.Source{
		ID:          uuid.New(),
		BlueSkyDID:  "did:plc:follow1",
		Handle:      "oldhandle.bsky.social",
		DisplayName: "Old Display Name",
		Avatar:      "https://example.com/old-avatar.jpg",
	}
	db.Create(existingSource)

	// Setup mock response with updated source info
	follows := &bluesky.FollowsResponse{
		Follows: []bluesky.Author{
			{
				DID:         "did:plc:follow1",
				Handle:      "newhandle.bsky.social",    // Updated handle
				DisplayName: "New Display Name",          // Updated display name
				Avatar:      "https://example.com/new-avatar.jpg", // Updated avatar
			},
		},
		Cursor: "",
	}

	mockClient.On("GetFollows", "did:plc:testuser2", 100, "").Return(follows, nil)

	config := DefaultRefreshConfig()

	// Test importing follows
	err := service.ImportUserFollows(user, config)
	assert.NoError(t, err)

	// Verify source was updated
	var updatedSource models.Source
	db.Where("blue_sky_d_id = ?", "did:plc:follow1").First(&updatedSource)
	assert.Equal(t, "newhandle.bsky.social", updatedSource.Handle)
	assert.Equal(t, "New Display Name", updatedSource.DisplayName)
	assert.Equal(t, "https://example.com/new-avatar.jpg", updatedSource.Avatar)

	mockClient.AssertExpectations(t)
}

func TestDefaultRefreshConfig(t *testing.T) {
	config := DefaultRefreshConfig()
	
	assert.Equal(t, 24*time.Hour, config.RefreshInterval)
	assert.Equal(t, 10, config.BatchSize)
	assert.Equal(t, 100*time.Millisecond, config.RateLimit)
}
