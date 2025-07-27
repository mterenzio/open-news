package workers

import (
	"context"
	"log"
	"time"

	"open-news/internal/services"
)

// FollowsRefreshWorker handles periodic refresh of user follows
type FollowsRefreshWorker struct {
	followsService *services.UserFollowsService
	config         services.RefreshConfig
	ticker         *time.Ticker
	stopChan       chan bool
}

// NewFollowsRefreshWorker creates a new follows refresh worker
func NewFollowsRefreshWorker(followsService *services.UserFollowsService, refreshInterval time.Duration) *FollowsRefreshWorker {
	return &FollowsRefreshWorker{
		followsService: followsService,
		config: services.RefreshConfig{
			RefreshInterval: refreshInterval,
			BatchSize:       10,
			RateLimit:       time.Second,
		},
		stopChan: make(chan bool),
	}
}

// NewFollowsRefreshWorkerWithConfig creates a worker with custom config
func NewFollowsRefreshWorkerWithConfig(followsService *services.UserFollowsService, config services.RefreshConfig) *FollowsRefreshWorker {
	return &FollowsRefreshWorker{
		followsService: followsService,
		config:         config,
		stopChan:       make(chan bool),
	}
}

// Start begins the periodic refresh process
func (w *FollowsRefreshWorker) Start(ctx context.Context) {
	// Run every hour to check for users that need refresh
	w.ticker = time.NewTicker(1 * time.Hour)
	
	log.Printf("🔄 Starting follows refresh worker (checking every hour)")
	log.Printf("   📅 Refresh interval: %v", w.config.RefreshInterval)
	log.Printf("   📦 Batch size: %d users", w.config.BatchSize)
	log.Printf("   ⏱️  Rate limit: %v between API calls", w.config.RateLimit)

	// Run an initial check immediately
	go func() {
		if err := w.followsService.RefreshBatch(w.config); err != nil {
			log.Printf("❌ Error in initial follows refresh: %v", err)
		}
	}()

	// Start the periodic ticker
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("🛑 Follows refresh worker stopping due to context cancellation")
				return
			case <-w.stopChan:
				log.Printf("🛑 Follows refresh worker stopping")
				return
			case <-w.ticker.C:
				if err := w.followsService.RefreshBatch(w.config); err != nil {
					log.Printf("❌ Error in periodic follows refresh: %v", err)
				}
			}
		}
	}()
}

// Stop stops the worker
func (w *FollowsRefreshWorker) Stop() {
	if w.ticker != nil {
		w.ticker.Stop()
	}
	close(w.stopChan)
	log.Printf("✅ Follows refresh worker stopped")
}

// GetStats returns statistics about follow refresh status
func (w *FollowsRefreshWorker) GetStats() (*FollowsStats, error) {
	users, err := w.followsService.GetUsersNeedingRefresh(w.config, 1000) // Check up to 1000 users
	if err != nil {
		return nil, err
	}

	stats := &FollowsStats{
		UsersNeedingRefresh: len(users),
		RefreshInterval:     w.config.RefreshInterval,
		LastCheck:          time.Now(),
	}

	return stats, nil
}

// FollowsStats holds statistics about follow refresh status
type FollowsStats struct {
	UsersNeedingRefresh int           `json:"users_needing_refresh"`
	RefreshInterval     time.Duration `json:"refresh_interval"`
	LastCheck           time.Time     `json:"last_check"`
}
