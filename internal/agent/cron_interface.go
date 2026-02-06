package agent

import "time"

// Job represents a scheduled job (domain model)
type Job struct {
	ID         string            // Unique job identifier
	Type       string            // Job type: "recurring" or "oneshot"
	Schedule   string            // Cron expression (e.g., "0 * * * *")
	ExecuteAt  *time.Time        // Execution time for oneshot jobs
	Command    string            // Message to send to agent when job executes
	UserID     string            // User ID for the message
	Metadata   map[string]string // Additional job metadata
	Executed   bool              // Whether the job has been executed
	ExecutedAt *time.Time        // When the job was executed
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
