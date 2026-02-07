// Package cron provides job execution logic for cron scheduler.
package cron

import (
	"fmt"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
)

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
			Command:   job.Command,
			Tool:      job.Tool,
			Payload:   job.Payload,
			SessionID: job.SessionID,
			Metadata:  job.Metadata,
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

	// Determine session ID
	sessionID := job.SessionID
	if sessionID == "" {
		sessionID = generateSessionID(job.ID)
	}

	// Create inbound message
	msg := bus.NewInboundMessage(
		ChannelTypeCron,
		job.UserID, // Use job.UserID for backward compatibility
		sessionID,
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
