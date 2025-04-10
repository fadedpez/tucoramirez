package scheduler

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/repositories/game"
)

// ElasticsearchMaintenanceScheduler manages scheduled maintenance tasks for Elasticsearch
type ElasticsearchMaintenanceScheduler struct {
	scheduler *Scheduler
	repo      *game.ElasticsearchRepository
}

// NewElasticsearchMaintenanceScheduler creates a new scheduler for Elasticsearch maintenance tasks
func NewElasticsearchMaintenanceScheduler(repo *game.ElasticsearchRepository) *ElasticsearchMaintenanceScheduler {
	return &ElasticsearchMaintenanceScheduler{
		scheduler: NewScheduler(),
		repo:      repo,
	}
}

// Start initializes and starts the maintenance scheduler
func (s *ElasticsearchMaintenanceScheduler) Start(ctx context.Context) {
	// Configure maintenance tasks based on repository configuration
	config := s.repo.GetConfig()

	// Schedule index rotation - default to daily if not specified
	rotationInterval := config.RotationPeriod
	if rotationInterval <= 0 {
		rotationInterval = 24 * time.Hour
	}
	s.scheduler.AddTask("index_rotation", rotationInterval, s.rotateIndices)

	// Schedule index pruning - default to weekly if not specified
	pruneInterval := 7 * 24 * time.Hour // Default to weekly
	s.scheduler.AddTask("index_pruning", pruneInterval, s.pruneOldIndices)

	// Schedule backups if enabled
	if config.BackupEnabled {
		// Default to daily backups
		backupInterval := 24 * time.Hour
		
		// Try to parse the backup schedule if it's a simple hourly interval
		if config.BackupSchedule != "" {
			// If it's a simple number, interpret as hours
			if hours, err := strconv.Atoi(config.BackupSchedule); err == nil && hours > 0 {
				backupInterval = time.Duration(hours) * time.Hour
			} else if strings.Contains(config.BackupSchedule, "h") {
				// Try to parse as a Go duration string (e.g., "12h")
				if duration, err := time.ParseDuration(config.BackupSchedule); err == nil && duration > 0 {
					backupInterval = duration
				}
			}
			// Note: For more complex cron-like schedules, we would need a cron parser
			// For now, we'll just use simple duration-based scheduling
		}
		
		s.scheduler.AddTask("index_backup", backupInterval, s.backupIndices)
	}

	// Start the scheduler
	s.scheduler.Start(ctx)
	log.Println("Elasticsearch maintenance scheduler started")
}

// Stop stops the maintenance scheduler
func (s *ElasticsearchMaintenanceScheduler) Stop() {
	s.scheduler.Stop()
	log.Println("Elasticsearch maintenance scheduler stopped")
}

// rotateIndices rotates indices based on the configured rotation period
func (s *ElasticsearchMaintenanceScheduler) rotateIndices(ctx context.Context) error {
	log.Println("Running scheduled index rotation task")
	return s.repo.RotateIndices(ctx)
}

// pruneOldIndices prunes old indices based on the configured retention period
func (s *ElasticsearchMaintenanceScheduler) pruneOldIndices(ctx context.Context) error {
	log.Println("Running scheduled index pruning task")
	return s.repo.PruneOldIndices(ctx)
}

// backupIndices creates backups of all indices
func (s *ElasticsearchMaintenanceScheduler) backupIndices(ctx context.Context) error {
	log.Println("Running scheduled index backup task")
	
	// Get all indices that match the pattern
	pattern := s.repo.GetIndexPrefix() + "_games_*"
	indices, err := s.repo.GetIndices(ctx, pattern)
	if err != nil {
		return err
	}
	
	// Backup each index
	for _, index := range indices {
		err := s.repo.CreateIndexBackup(ctx, index)
		if err != nil {
			log.Printf("Error backing up index %s: %v", index, err)
			// Continue with other indices even if one fails
		}
	}
	
	return nil
}
