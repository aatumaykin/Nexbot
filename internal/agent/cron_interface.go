package agent

import "time"

// Job represents a scheduled job (domain model)
type Job struct {
	ID         string            `json:"id"`                    // Unique job identifier
	Type       string            `json:"type"`                  // Job type: "recurring" or "oneshot"
	Schedule   string            `json:"schedule"`              // Cron expression (e.g., "0 * * * *")
	ExecuteAt  *time.Time        `json:"execute_at,omitempty"`  // Execution time for oneshot jobs
	UserID     string            `json:"user_id,omitempty"`     // User ID for the message
	Tool       string            `json:"tool"`                  // Internal tool: "" | "send_message" | "agent"
	Payload    map[string]any    `json:"payload"`               // Tool parameters (JSON)
	SessionID  string            `json:"session_id"`            // Session ID for context
	Metadata   map[string]string `json:"metadata,omitempty"`    // Additional job metadata
	Executed   bool              `json:"executed,omitempty"`    // Whether job has been executed
	ExecutedAt *time.Time        `json:"executed_at,omitempty"` // When job was executed
}

// CronManager is a domain-level interface for job scheduling
type CronManager interface {
	// AddJob adds a new job to the schedule
	AddJob(job Job) (string, error)

	// RemoveJob removes a job by ID
	RemoveJob(jobID string) error

	// ListJobs returns all scheduled jobs
	ListJobs() []Job

	// RemoveFromStorage removes a job from persistent storage
	RemoveFromStorage(jobID string) error

	// AppendJob adds a job to storage (for persistence)
	AppendJob(job Job) error
}
