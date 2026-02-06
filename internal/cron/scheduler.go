// Package cron provides a cron scheduler for scheduled task execution.
// It uses robfig/cron/v3 library to enable periodic job scheduling.
// Jobs can be added, removed, and listed. Each job executes by sending
// a message to the message bus inbound queue.
package cron

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/robfig/cron/v3"
)

// Scheduler manages cron job scheduling and execution
type Scheduler struct {
	cron          *cron.Cron
	logger        *logger.Logger
	bus           *bus.MessageBus
	workerPool    WorkerPool   // Worker pool for async task execution
	storage       *Storage     // Persistent storage for jobs
	ticker        *time.Ticker // Ticker for oneshot job checking
	cleanupTicker *time.Ticker // Ticker for executed cleanup
	parser        cron.Parser  // Parser for validating cron expressions
	ctx           context.Context
	cancel        context.CancelFunc
	started       bool
	mu            sync.RWMutex

	// Job registry for tracking jobs by ID
	jobs        map[string]Job
	jobIDs      map[cron.EntryID]string // cron.EntryID -> Job.ID
	jobEntryIDs map[string]cron.EntryID // Job.ID -> cron.EntryID
}

// NewScheduler creates a new cron scheduler instance
func NewScheduler(logger *logger.Logger, messageBus *bus.MessageBus, workerPool WorkerPool, storage *Storage) *Scheduler {
	return &Scheduler{
		cron:        cron.New(cron.WithSeconds()),
		logger:      logger,
		bus:         messageBus,
		workerPool:  workerPool,
		storage:     storage,
		parser:      cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
		jobs:        make(map[string]Job),
		jobIDs:      make(map[cron.EntryID]string),
		jobEntryIDs: make(map[string]cron.EntryID),
	}
}

// Start starts the cron scheduler
// It will block until the context is cancelled
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("scheduler already started")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.started = true

	s.cron.Start()
	s.logger.Info("cron scheduler started")

	// Start oneshot ticker
	s.oneshotTicker()

	// Start executed cleanup ticker
	s.executedCleanup()

	// Wait for context cancellation
	go func() {
		<-s.ctx.Done()
		s.cron.Stop()
		if s.ticker != nil {
			s.ticker.Stop()
		}
		if s.cleanupTicker != nil {
			s.cleanupTicker.Stop()
		}
		s.logger.Info("cron scheduler stopped")
	}()

	return nil
}

// Stop stops the cron scheduler gracefully
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return fmt.Errorf("scheduler not started")
	}

	s.cancel()
	s.started = false
	return nil
}

// AddJob adds a new cron job to the scheduler
// Returns the cron entry ID for the job
func (s *Scheduler) AddJob(job Job) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate job ID if not provided
	if job.ID == "" {
		job.ID = GenerateJobID()
	}

	var entryID cron.EntryID
	var err error

	// Only add to cron scheduler for recurring jobs
	// Empty type defaults to recurring for backward compatibility
	if job.Type == JobTypeRecurring || job.Type == "" {
		// Wrap job execution with error handling and logging
		wrappedFunc := func() {
			s.executeJob(job)
		}

		// Validate cron expression first
		if err := validateCronExpression(job.Schedule, s.parser); err != nil {
			return "", err
		}

		// Add job to cron scheduler
		entryID, err = s.cron.AddFunc(job.Schedule, wrappedFunc)
		if err != nil {
			return "", fmt.Errorf("invalid cron expression: %w", err)
		}
	} else if job.Schedule != "" {
		// For non-recurring jobs, validate cron expression without adding to scheduler
		if err := validateCronExpression(job.Schedule, s.parser); err != nil {
			return "", err
		}
	}

	// Store job in registry
	s.jobs[job.ID] = job
	if job.Type == JobTypeRecurring || job.Type == "" {
		s.jobIDs[entryID] = job.ID
		s.jobEntryIDs[job.ID] = entryID
	}

	// Persist job to storage
	if s.storage != nil {
		storageJob := StorageJob{
			ID:         job.ID,
			Type:       string(job.Type),
			Schedule:   job.Schedule,
			ExecuteAt:  job.ExecuteAt,
			Command:    job.Command,
			UserID:     job.UserID,
			Metadata:   job.Metadata,
			Executed:   job.Executed,
			ExecutedAt: job.ExecutedAt,
		}
		if err := s.storage.UpsertJob(storageJob); err != nil {
			s.logger.Error("failed to persist job to storage", err,
				logger.Field{Key: "job_id", Value: job.ID})
			// Continue even if storage fails - job is already in memory
		}
	}

	// For oneshot jobs, execute immediately if time has already passed
	// This is important for testing and ensures jobs don't get missed
	if job.Type == JobTypeOneshot && !job.Executed && job.ExecuteAt != nil {
		now := time.Now()
		if validateOneshotJobExecution(job.ExecuteAt, now) {
			// Execute job first (with Executed=false)
			s.executeJob(job)
			s.logger.Info("executed oneshot job immediately on add",
				logger.Field{Key: "job_id", Value: job.ID})

			// Then mark as executed
			storageJob := StorageJob{
				ID:         job.ID,
				Type:       string(job.Type),
				Schedule:   job.Schedule,
				ExecuteAt:  job.ExecuteAt,
				Command:    job.Command,
				UserID:     job.UserID,
				Metadata:   job.Metadata,
				Executed:   true,
				ExecutedAt: &now,
			}
			if s.storage != nil {
				if err := s.storage.UpsertJob(storageJob); err != nil {
					s.logger.Error("failed to update oneshot job as executed", err,
						logger.Field{Key: "job_id", Value: job.ID})
				}
			}
			// Update in-memory job
			job.Executed = true
			job.ExecutedAt = &now
			s.jobs[job.ID] = job
		}
	}

	// Log job addition
	if job.Type == JobTypeRecurring {
		s.logger.Info("cron job added",
			logger.Field{Key: "job_id", Value: job.ID},
			logger.Field{Key: "schedule", Value: job.Schedule},
			logger.Field{Key: "entry_id", Value: entryID})
	} else {
		s.logger.Info("cron job added",
			logger.Field{Key: "job_id", Value: job.ID},
			logger.Field{Key: "job_type", Value: job.Type},
			logger.Field{Key: "execute_at", Value: job.ExecuteAt})
	}

	return job.ID, nil
}

// RemoveJob removes a cron job from the scheduler
func (s *Scheduler) RemoveJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Only remove from cron scheduler for recurring jobs
	// Empty type defaults to recurring for backward compatibility
	if job.Type == JobTypeRecurring || job.Type == "" {
		if entryID, ok := s.jobEntryIDs[jobID]; ok {
			s.cron.Remove(entryID)
			delete(s.jobIDs, entryID)
			delete(s.jobEntryIDs, jobID)
		}
	}
	delete(s.jobs, jobID)

	// Remove from storage
	if s.storage != nil {
		if err := s.storage.Remove(jobID); err != nil {
			s.logger.Error("failed to remove job from storage", err,
				logger.Field{Key: "job_id", Value: jobID})
			// Continue even if storage fails - job is already removed from memory
		}
	}

	// Log removal with entry ID for recurring jobs
	if job.Type == JobTypeRecurring {
		if entryID, ok := s.jobEntryIDs[jobID]; ok {
			s.logger.Info("cron job removed",
				logger.Field{Key: "job_id", Value: jobID},
				logger.Field{Key: "entry_id", Value: entryID})
		}
	} else {
		s.logger.Info("cron job removed",
			logger.Field{Key: "job_id", Value: jobID},
			logger.Field{Key: "job_type", Value: job.Type})
	}

	return nil
}

// ListJobs returns all scheduled jobs
func (s *Scheduler) ListJobs() []Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// GetJob retrieves a specific job by ID
func (s *Scheduler) GetJob(jobID string) (Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return Job{}, fmt.Errorf("job not found: %s", jobID)
	}
	return job, nil
}

// IsStarted returns true if the scheduler is started
func (s *Scheduler) IsStarted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started
}
