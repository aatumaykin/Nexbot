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

const (
	// ChannelTypeCron is the channel type for cron-scheduled messages
	ChannelTypeCron bus.ChannelType = "cron"
)

// JobType represents the type of a cron job
type JobType string

const (
	// JobTypeRecurring is a repeating job that runs on a schedule
	JobTypeRecurring JobType = "recurring"
	// JobTypeOneshot is a one-time job that runs once at the specified time
	JobTypeOneshot JobType = "oneshot"
)

// Task represents a cron task to be submitted to worker pool
type Task struct {
	ID      string          // Unique task identifier
	Type    string          // Task type: "cron"
	Payload interface{}     // Task payload (command, user_id, metadata, etc.)
	Context context.Context // Task-specific context for cancellation/timeout
}

// WorkerPool is an interface for worker pool operations
type WorkerPool interface {
	Submit(task Task)
}

// CronTaskPayload represents the payload for a cron task
type CronTaskPayload struct {
	Command  string            // Command to execute
	UserID   string            // User ID
	Metadata map[string]string // Job metadata
}

// Job represents a scheduled cron job
type Job struct {
	ID         string            `json:"id"`                    // Unique job identifier
	Type       JobType           `json:"type"`                  // Job type: recurring or oneshot
	Schedule   string            `json:"schedule"`              // Cron expression (e.g., "0 * * * *")
	ExecuteAt  *time.Time        `json:"execute_at,omitempty"`  // Execution time for oneshot jobs
	Command    string            `json:"command"`               // Message to send to agent when job executes
	UserID     string            `json:"user_id"`               // User ID for the message
	Metadata   map[string]string `json:"metadata,omitempty"`    // Additional job metadata
	Executed   bool              `json:"executed,omitempty"`    // Whether the job has been executed
	ExecutedAt *time.Time        `json:"executed_at,omitempty"` // When the job was executed
}

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
		job.ID = generateJobID()
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

		// Add job to cron scheduler (this validates the expression)
		entryID, err = s.cron.AddFunc(job.Schedule, wrappedFunc)
		if err != nil {
			return "", fmt.Errorf("invalid cron expression: %w", err)
		}
	} else if job.Schedule != "" {
		// For non-recurring jobs, validate cron expression without adding to scheduler
		_, err = s.parser.Parse(job.Schedule)
		if err != nil {
			return "", fmt.Errorf("invalid cron expression: %w", err)
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
		if job.ExecuteAt.Before(now) || job.ExecuteAt.Equal(now) {
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

// executeJob executes a cron job by submitting it to the worker pool
func (s *Scheduler) executeJob(job Job) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("cron job panic recovered", fmt.Errorf("panic: %v", r),
				logger.Field{Key: "job_id", Value: job.ID})
		}
	}()

	// Skip execution if oneshot job was already executed
	if job.Type == JobTypeOneshot && job.Executed {
		return
	}

	// Submit to worker pool if available
	if s.workerPool != nil {
		// Prepare task payload
		taskPayload := CronTaskPayload{
			Command:  job.Command,
			UserID:   job.UserID,
			Metadata: job.Metadata,
		}

		// Create task ID
		taskID := fmt.Sprintf("cron_%s_%d", job.ID, time.Now().UnixNano())

		// Create and submit task
		task := Task{
			ID:      taskID,
			Type:    "cron",
			Payload: taskPayload,
			Context: s.ctx,
		}

		s.workerPool.Submit(task)

		s.logger.Info("cron job submitted to worker pool",
			logger.Field{Key: "job_id", Value: job.ID},
			logger.Field{Key: "task_id", Value: taskID},
			logger.Field{Key: "command", Value: job.Command})
	} else {
		// Fallback to message bus if no worker pool
		s.fallbackToMessageBus(job)
	}
}

// fallbackToMessageBus sends the job to the message bus as before
func (s *Scheduler) fallbackToMessageBus(job Job) {
	// Prepare metadata for the message
	metadata := make(map[string]any)
	metadata["cron_job_id"] = job.ID
	metadata["cron_schedule"] = job.Schedule
	for k, v := range job.Metadata {
		metadata[k] = v
	}

	// Create inbound message
	msg := bus.NewInboundMessage(
		ChannelTypeCron,
		job.UserID,
		generateSessionID(job.ID),
		job.Command,
		metadata,
	)

	// Publish to message bus
	if err := s.bus.PublishInbound(*msg); err != nil {
		s.logger.Error("failed to publish cron job message", err,
			logger.Field{Key: "job_id", Value: job.ID})
		return
	}

	s.logger.Info("cron job executed via message bus",
		logger.Field{Key: "job_id", Value: job.ID},
		logger.Field{Key: "command", Value: job.Command})
}

// IsStarted returns true if the scheduler is started
func (s *Scheduler) IsStarted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started
}

// generateJobID generates a unique job ID
func generateJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}

// generateSessionID generates a session ID for a cron job
func generateSessionID(jobID string) string {
	return fmt.Sprintf("cron_%s", jobID)
}

// GenerateJobID генерирует уникальный ID для job (экспортируемый метод)
func (s *Scheduler) GenerateJobID() string {
	return generateJobID()
}

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
