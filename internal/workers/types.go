// Package workers provides an async worker pool for background task execution.
// It supports multiple task types (cron, subagent) and provides result channels
// for asynchronous execution monitoring.
package workers

import (
	"context"
	"time"
)

// Task represents a unit of work to be executed by a worker.
type Task struct {
	ID      string                 // Unique task identifier
	Type    string                 // Task type: "cron" or "subagent"
	Payload interface{}            // Task payload (command, agent config, etc.)
	Context context.Context        // Task-specific context for cancellation/timeout
	Metrics map[string]interface{} // Optional metrics to track
}

// Result represents the outcome of a task execution.
type Result struct {
	TaskID   string                 // ID of the executed task
	Error    error                  // Error if execution failed
	Output   string                 // Task output
	Duration time.Duration          // Execution duration
	Metrics  map[string]interface{} // Task execution metrics
}

// PoolMetrics tracks execution metrics for the worker pool.
type PoolMetrics struct {
	TasksSubmitted uint64
	TasksCompleted uint64
	TasksFailed    uint64
	TotalDuration  time.Duration
}

// CronTask is a type alias for compatibility with cron package
type CronTask struct {
	ID      string
	Type    string
	Payload interface{}
	Context context.Context
}

// TaskExecutor defines the interface for task-specific execution logic
type TaskExecutor func(context.Context, Task) (string, error)

// Constants for worker pool configuration
const (
	DefaultTaskTimeout = 30 * time.Second
	DefaultPoolSize    = 5
	DefaultQueueSize   = 100
)
