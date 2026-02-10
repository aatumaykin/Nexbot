package cleanup

import "time"

// Stats holds statistics about cleanup operations.
type Stats struct {
	SessionsCleaned int           // Number of sessions cleaned up
	SessionsDeleted int           // Number of sessions deleted
	MessagesExpired int           // Number of messages expired (by TTL)
	MBytesFreed     int64         // Megabytes freed (rounded)
	Duration        time.Duration // Time taken for cleanup
}

// Config holds configuration for cleanup operations.
type Config struct {
	MessageTTLDays   int64 // TTL for messages in days (0 = no TTL)
	SessionTTLDays   int64 // TTL for sessions in days (0 = no TTL)
	MaxSessionSizeMB int64 // Maximum session size in MB (0 = no limit)
	KeepActiveDays   int64 // Keep sessions active for N days after last modification
}

// Runner manages periodic cleanup operations.
type Runner struct {
	config  Config
	stats   Stats
	running bool
	lastRun time.Time
}

// NewRunner creates a new cleanup runner.
func NewRunner(config Config) *Runner {
	return &Runner{
		config: config,
	}
}
