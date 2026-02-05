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
	ID       string            `json:"id"`                 // Unique job identifier
	Schedule string            `json:"schedule"`           // Cron expression (e.g., "0 * * * *")
	Command  string            `json:"command"`            // Message to send to agent when job executes
	UserID   string            `json:"user_id"`            // User ID for the message
	Metadata map[string]string `json:"metadata,omitempty"` // Additional job metadata
}

// Scheduler manages cron job scheduling and execution
type Scheduler struct {
	cron       *cron.Cron
	logger     *logger.Logger
	bus        *bus.MessageBus
	workerPool WorkerPool // Worker pool for async task execution
	ctx        context.Context
	cancel     context.CancelFunc
	started    bool
	mu         sync.RWMutex

	// Job registry for tracking jobs by ID
	jobs        map[string]Job
	jobIDs      map[cron.EntryID]string // cron.EntryID -> Job.ID
	jobEntryIDs map[string]cron.EntryID // Job.ID -> cron.EntryID
}

// NewScheduler creates a new cron scheduler instance
func NewScheduler(logger *logger.Logger, messageBus *bus.MessageBus, workerPool WorkerPool) *Scheduler {
	return &Scheduler{
		cron:        cron.New(cron.WithSeconds()),
		logger:      logger,
		bus:         messageBus,
		workerPool:  workerPool,
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

	// Wait for context cancellation
	go func() {
		<-s.ctx.Done()
		s.cron.Stop()
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

	// Wrap job execution with error handling and logging
	wrappedFunc := func() {
		s.executeJob(job)
	}

	// Add job to cron scheduler (this validates the expression)
	entryID, err := s.cron.AddFunc(job.Schedule, wrappedFunc)
	if err != nil {
		return "", fmt.Errorf("invalid cron expression: %w", err)
	}

	// Store job in registry
	s.jobs[job.ID] = job
	s.jobIDs[entryID] = job.ID
	s.jobEntryIDs[job.ID] = entryID

	s.logger.Info("cron job added",
		logger.Field{Key: "job_id", Value: job.ID},
		logger.Field{Key: "schedule", Value: job.Schedule},
		logger.Field{Key: "entry_id", Value: entryID})

	return job.ID, nil
}

// RemoveJob removes a cron job from the scheduler
func (s *Scheduler) RemoveJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, exists := s.jobEntryIDs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	s.cron.Remove(entryID)
	delete(s.jobs, jobID)
	delete(s.jobIDs, entryID)
	delete(s.jobEntryIDs, jobID)

	s.logger.Info("cron job removed",
		logger.Field{Key: "job_id", Value: jobID},
		logger.Field{Key: "entry_id", Value: entryID})

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

// GenerateJobID генерирует уникальный ID для job (пакетная функция)
func GenerateJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}
