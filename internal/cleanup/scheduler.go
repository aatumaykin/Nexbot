package cleanup

import (
	"context"
	"fmt"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

// Scheduler manages periodic cleanup runs.
type Scheduler struct {
	runner    *Runner
	config    SchedulerConfig
	logger    *logger.Logger
	workspace string
	ctx       context.Context
	cancel    context.CancelFunc
	ticker    *time.Ticker
}

// SchedulerConfig holds configuration for the cleanup scheduler.
type SchedulerConfig struct {
	Enabled         bool // Enable periodic cleanup
	IntervalMinutes int  // Interval between cleanup runs
}

// NewScheduler creates a new cleanup scheduler.
func NewScheduler(
	runner *Runner,
	config SchedulerConfig,
	workspace string,
	log *logger.Logger,
) *Scheduler {
	return &Scheduler{
		runner:    runner,
		config:    config,
		logger:    log,
		workspace: workspace,
	}
}

// Start begins the periodic cleanup scheduler.
func (s *Scheduler) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.logger.Info("cleanup scheduler disabled")
		return nil
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	interval := time.Duration(s.config.IntervalMinutes) * time.Minute
	s.ticker = time.NewTicker(interval)

	s.logger.Info("cleanup scheduler started",
		logger.Field{Key: "interval_minutes", Value: s.config.IntervalMinutes})

	// Run initial cleanup
	go s.runCleanup(s.ctx)

	// Start periodic cleanup
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.runCleanup(s.ctx)
			case <-s.ctx.Done():
				s.ticker.Stop()
				s.logger.Info("cleanup scheduler stopped")
				return
			}
		}
	}()

	return nil
}

// Stop stops the cleanup scheduler.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// runCleanup executes a single cleanup run.
func (s *Scheduler) runCleanup(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	s.logger.Info("starting cleanup")

	// Get active sessions (empty for now - this would be integrated with agent loop)
	activeSessions := make(map[string]bool)

	stats, err := s.runner.Run(s.workspace, activeSessions, s.logger)
	if err != nil {
		s.logger.Error("cleanup failed", err)
		return
	}

	if stats.SessionsCleaned > 0 || stats.SessionsDeleted > 0 {
		s.logger.Info(fmt.Sprintf("cleanup completed: cleaned %d sessions, deleted %d sessions, expired %d messages, freed %dMB",
			stats.SessionsCleaned, stats.SessionsDeleted, stats.MessagesExpired, stats.MBytesFreed),
			logger.Field{Key: "sessions_cleaned", Value: stats.SessionsCleaned},
			logger.Field{Key: "sessions_deleted", Value: stats.SessionsDeleted},
			logger.Field{Key: "messages_expired", Value: stats.MessagesExpired},
			logger.Field{Key: "mb_freed", Value: stats.MBytesFreed},
			logger.Field{Key: "duration_ms", Value: stats.Duration.Milliseconds()})
	} else {
		s.logger.Debug("cleanup completed: no sessions needed cleanup")
	}
}

// Trigger runs cleanup immediately (manual trigger).
func (s *Scheduler) Trigger(activeSessions map[string]bool) (Stats, error) {
	s.logger.Info("manual cleanup triggered")
	return s.runner.Run(s.workspace, activeSessions, s.logger)
}
