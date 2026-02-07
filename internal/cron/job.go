// Package cron provides types and helper functions for cron jobs.
package cron

import (
	"context"
	"fmt"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
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
	Command   string            // Deprecated: Used only for backward compatibility when tool=""
	Tool      string            // Internal tool: "" (legacy) | "send_message" | "agent"
	Payload   map[string]any    // Tool parameters (JSON). For send_message/agent: {"message": "text"}
	SessionID string            // Session ID for context (format: "channel:chat_id")
	Metadata  map[string]string // Job metadata
}

// Job represents a scheduled cron job
type Job struct {
	ID         string            `json:"id"`                    // Unique job identifier
	Type       JobType           `json:"type"`                  // Job type: recurring or oneshot
	Schedule   string            `json:"schedule"`              // Cron expression (e.g., "0 * * * *")
	ExecuteAt  *time.Time        `json:"execute_at,omitempty"`  // Execution time for oneshot jobs
	Command    string            `json:"command"`               // Message to send to agent when job executes
	UserID     string            `json:"user_id"`               // User ID for the message
	Tool       string            `json:"tool"`                  // Внутренний инструмент: "" | "send_message" | "agent"
	Payload    map[string]any    `json:"payload"`               // Параметры для инструмента (JSON)
	SessionID  string            `json:"session_id"`            // Контекст сессии (опциональный)
	Metadata   map[string]string `json:"metadata,omitempty"`    // Additional job metadata
	Executed   bool              `json:"executed,omitempty"`    // Whether the job has been executed
	ExecutedAt *time.Time        `json:"executed_at,omitempty"` // When the job was executed
}

// generateSessionID generates a session ID for a cron job
func generateSessionID(jobID string) string {
	return fmt.Sprintf("cron_%s", jobID)
}
