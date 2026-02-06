// Package cron provides oneshot job management for cron scheduler.
package cron

import (
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

// oneshotTicker starts a ticker that checks for oneshot jobs every minute.
func (s *Scheduler) oneshotTicker() {
	s.ticker = time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-s.ticker.C:
				s.checkAndExecuteOneshots(time.Now())
			}
		}
	}()
}

// checkAndExecuteOneshots checks for and executes overdue oneshot jobs.
// It loads all jobs from storage and executes those that are overdue.
func (s *Scheduler) checkAndExecuteOneshots(now time.Time) {
	// Load all jobs from storage
	storageJobs, err := s.storage.Load()
	if err != nil {
		s.logger.Error("failed to load jobs for oneshot check", err)
		return
	}

	updated := false
	for _, storageJob := range storageJobs {
		// Only check oneshot jobs
		if storageJob.Type != string(JobTypeOneshot) {
			continue
		}

		// Check if job should execute now
		if storageJob.ExecuteAt == nil {
			s.logger.Warn("oneshot job has no execute_at", logger.Field{Key: "job_id", Value: storageJob.ID})
			continue
		}

		if storageJob.ExecuteAt.After(now) || storageJob.Executed {
			continue
		}

		// Submit job to worker pool BEFORE marking as executed
		job := Job{
			ID:         storageJob.ID,
			Type:       JobType(storageJob.Type),
			Schedule:   storageJob.Schedule,
			ExecuteAt:  storageJob.ExecuteAt,
			Command:    storageJob.Command,
			UserID:     storageJob.UserID,
			Metadata:   storageJob.Metadata,
			Executed:   false, // Not yet executed - executeJob will skip if already executed
			ExecutedAt: nil,
		}
		s.executeJob(job)

		// Mark as executed AFTER submission
		storageJob.Executed = true
		storageJob.ExecutedAt = &now
		updated = true

		s.logger.Info("executed oneshot job",
			logger.Field{Key: "job_id", Value: storageJob.ID},
			logger.Field{Key: "execute_at", Value: storageJob.ExecuteAt})
	}

	// Save updated jobs to storage
	if updated {
		if err := s.storage.Save(storageJobs); err != nil {
			s.logger.Error("failed to save jobs after oneshot execution", err)
		}
	}
}

// executedCleanup starts a ticker that cleans up executed oneshot jobs every 24 hours.
func (s *Scheduler) executedCleanup() {
	s.cleanupTicker = time.NewTicker(24 * time.Hour)
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-s.cleanupTicker.C:
				s.CleanupExecutedOneshots()
			}
		}
	}()
}

// CleanupExecutedOneshots removes all executed oneshot jobs from storage.
// It reloads the jobs map from storage to reflect the changes.
// This method is public and can be called manually.
func (s *Scheduler) CleanupExecutedOneshots() {
	// Load all jobs before cleanup to identify which ones to remove
	allJobs, err := s.storage.Load()
	if err != nil {
		s.logger.Error("failed to load jobs before cleanup", err)
		return
	}

	// Identify executed oneshot jobs to remove
	var oneshotJobsToRemove []string
	for _, job := range allJobs {
		if job.Type == string(JobTypeOneshot) && job.Executed {
			oneshotJobsToRemove = append(oneshotJobsToRemove, job.ID)
		}
	}

	// Remove executed oneshots from storage
	if err := s.storage.RemoveExecutedOneshots(); err != nil {
		s.logger.Error("failed to remove executed oneshots", err)
		return
	}

	// Update in-memory jobs map - remove only executed oneshots
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, jobID := range oneshotJobsToRemove {
		delete(s.jobs, jobID)
	}

	s.logger.Info("cleaned up executed oneshot jobs",
		logger.Field{Key: "remaining_jobs", Value: len(s.jobs)})
}
