// Package cron provides job execution logic for cron scheduler.
package cron

import (
	"fmt"
	"time"

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

	// Validate worker pool is available
	if s.workerPool == nil {
		s.logger.Error("cron job execution failed: worker pool is not configured",
			fmt.Errorf("worker pool not available"),
			logger.Field{Key: "job_id", Value: job.ID},
			logger.Field{Key: "tool", Value: job.Tool})
		return
	}

	// Prepare task payload - always use Payload
	taskPayload := CronTaskPayload{
		Tool:      job.Tool,
		Payload:   job.Payload,
		SessionID: job.SessionID,
		Metadata:  job.Metadata,
	}

	// Validate required fields based on tool type
	if job.Tool != "" {
		if job.Payload == nil {
			s.logger.Error("cron job execution failed: payload is required when tool is specified",
				fmt.Errorf("payload is required"),
				logger.Field{Key: "job_id", Value: job.ID},
				logger.Field{Key: "tool", Value: job.Tool})
			return
		}

		// For send_message and agent, validate session_id
		if job.Tool == "send_message" || job.Tool == "agent" {
			if job.SessionID == "" {
				s.logger.Error("cron job execution failed: session_id is required for send_message/agent tools",
					fmt.Errorf("session_id is required"),
					logger.Field{Key: "job_id", Value: job.ID},
					logger.Field{Key: "tool", Value: job.Tool})
				return
			}
		}
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
		logger.Field{Key: "tool", Value: job.Tool})
}
