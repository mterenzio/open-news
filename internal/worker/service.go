package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"open-news/internal/bluesky"
	"open-news/internal/database"
	"open-news/internal/services"
	"open-news/internal/workers"
)

// WorkerService manages background workers for the application
type WorkerService struct {
	firehoseConsumer  *bluesky.FirehoseConsumer
	blueskyClient     *bluesky.Client
	followsWorker     *workers.FollowsRefreshWorker
	userFollowsService *services.UserFollowsService
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	running           bool
	mu                sync.RWMutex
}

// NewWorkerService creates a new worker service
func NewWorkerService() *WorkerService {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Initialize Bluesky client
	blueskyClient := bluesky.NewClient("https://bsky.social")
	
	// Initialize firehose consumer
	firehoseConsumer := bluesky.NewFirehoseConsumer(database.DB, blueskyClient)
	
	// Initialize user follows service
	userFollowsService := services.NewUserFollowsService(database.DB, blueskyClient)
	
	// Initialize follows refresh worker with 1 hour refresh interval
	followsWorker := workers.NewFollowsRefreshWorker(userFollowsService, time.Hour)
	
	return &WorkerService{
		firehoseConsumer:   firehoseConsumer,
		blueskyClient:      blueskyClient,
		followsWorker:      followsWorker,
		userFollowsService: userFollowsService,
		ctx:                ctx,
		cancel:             cancel,
		running:            false,
	}
}

// Start starts all background workers
func (ws *WorkerService) Start() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	
	if ws.running {
		return nil // Already running
	}
	
	log.Println("Starting background workers...")
	
	// Start firehose consumer
	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		ws.runFirehoseConsumer()
	}()
	
	// Start follows refresh worker
	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		ws.runFollowsRefreshWorker()
	}()
	
	// Start other workers here (article fetcher, feed generator, etc.)
	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		ws.runPeriodicTasks()
	}()
	
	ws.running = true
	log.Println("Background workers started successfully")
	
	return nil
}

// Stop stops all background workers
func (ws *WorkerService) Stop() {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	
	if !ws.running {
		return // Not running
	}
	
	log.Println("Stopping background workers...")
	
	// Cancel context to signal all workers to stop
	ws.cancel()
	
	// Wait for all workers to finish
	ws.wg.Wait()
	
	ws.running = false
	log.Println("Background workers stopped")
}

// IsRunning returns whether the worker service is currently running
func (ws *WorkerService) IsRunning() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.running
}

// runFirehoseConsumer runs the Bluesky firehose consumer
func (ws *WorkerService) runFirehoseConsumer() {
	log.Println("Starting Bluesky firehose consumer...")
	
	// Run with retry logic
	for {
		select {
		case <-ws.ctx.Done():
			log.Println("Firehose consumer stopped")
			return
		default:
			if err := ws.firehoseConsumer.StartConsuming(ws.ctx); err != nil {
				if ws.ctx.Err() != nil {
					// Context was cancelled, this is expected
					return
				}
				
				log.Printf("Firehose consumer error: %v. Restarting in 30 seconds...", err)
				
				// Wait before restarting
				select {
				case <-time.After(30 * time.Second):
					continue
				case <-ws.ctx.Done():
					return
				}
			}
		}
	}
}

// runFollowsRefreshWorker runs the follows refresh worker
func (ws *WorkerService) runFollowsRefreshWorker() {
	log.Println("Starting follows refresh worker...")
	
	ws.followsWorker.Start(ws.ctx)
	
	// Wait for context cancellation
	<-ws.ctx.Done()
	
	log.Println("Stopping follows refresh worker...")
	ws.followsWorker.Stop()
	log.Println("Follows refresh worker stopped")
}

// runPeriodicTasks runs periodic maintenance tasks
func (ws *WorkerService) runPeriodicTasks() {
	log.Println("Starting periodic tasks worker...")
	
	// Create tickers for different tasks
	feedUpdateTicker := time.NewTicker(5 * time.Minute)   // Update feeds every 5 minutes
	cleanupTicker := time.NewTicker(1 * time.Hour)       // Cleanup tasks every hour
	metricsTicker := time.NewTicker(15 * time.Minute)    // Update metrics every 15 minutes
	
	defer feedUpdateTicker.Stop()
	defer cleanupTicker.Stop()
	defer metricsTicker.Stop()
	
	for {
		select {
		case <-ws.ctx.Done():
			log.Println("Periodic tasks worker stopped")
			return
			
		case <-feedUpdateTicker.C:
			ws.updateFeeds()
			
		case <-cleanupTicker.C:
			ws.runCleanupTasks()
			
		case <-metricsTicker.C:
			ws.updateMetrics()
		}
	}
}

// updateFeeds triggers feed generation and updates
func (ws *WorkerService) updateFeeds() {
	log.Println("Running feed update task...")
	
	// TODO: Implement feed generation logic
	// This would:
	// 1. Calculate trending scores for articles
	// 2. Update global feed rankings
	// 3. Update personalized feeds for active users
	// 4. Clean up old feed items
	
	log.Println("Feed update task completed")
}

// runCleanupTasks performs various cleanup operations
func (ws *WorkerService) runCleanupTasks() {
	log.Println("Running cleanup tasks...")
	
	// TODO: Implement cleanup logic
	// This would:
	// 1. Remove old feed items beyond retention period
	// 2. Clean up cached article content that's too old
	// 3. Update source quality scores
	// 4. Archive old engagement data
	
	log.Println("Cleanup tasks completed")
}

// updateMetrics updates various application metrics
func (ws *WorkerService) updateMetrics() {
	log.Println("Updating metrics...")
	
	// Initialize quality score service
	qualityService := services.NewQualityScoreService(database.DB)
	
	// Update all quality scores
	if err := qualityService.UpdateAllQualityScores(); err != nil {
		log.Printf("Failed to update quality scores: %v", err)
	}
	
	log.Println("Metrics update completed")
}

// Graceful shutdown helpers
func (ws *WorkerService) Shutdown() {
	ws.Stop()
}

// GetUserFollowsService returns the user follows service for external use
func (ws *WorkerService) GetUserFollowsService() *services.UserFollowsService {
	return ws.userFollowsService
}

// GetStatus returns the current status of the worker service
func (ws *WorkerService) GetStatus() map[string]interface{} {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	
	status := map[string]interface{}{
		"running":           ws.running,
		"firehose_enabled":  true,
		"periodic_tasks":    true,
		"uptime":           time.Since(time.Now()), // This would be tracked properly in a real implementation
	}
	
	// Add follows worker statistics if available
	if ws.followsWorker != nil {
		followsStats, err := ws.followsWorker.GetStats()
		if err != nil {
			log.Printf("Failed to get follows worker stats: %v", err)
		} else {
			status["follows_worker"] = followsStats
		}
	}
	
	return status
}
