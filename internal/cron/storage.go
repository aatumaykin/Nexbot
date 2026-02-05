// Package cron provides persistent storage for cron jobs using JSONL format.
// Storage handles saving and loading cron jobs to/from disk.
package cron

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

const (
	// CronSubdirectory is the subdirectory name for cron jobs within workspace
	CronSubdirectory = "cron"

	// JobsFilename is the filename for storing cron jobs in JSONL format
	JobsFilename = "jobs.jsonl"
)

// StorageJob represents a cron job persisted in storage
type StorageJob struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Schedule   string            `json:"schedule,omitempty"`
	ExecuteAt  *time.Time        `json:"execute_at,omitempty"`
	Command    string            `json:"command"`
	UserID     string            `json:"user_id,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Executed   bool              `json:"executed,omitempty"`
	ExecutedAt *time.Time        `json:"executed_at,omitempty"`
}

// Storage provides persistent storage for cron jobs.
// It uses JSONL (JSON Lines) format to store jobs one per line.
type Storage struct {
	filePath string         // Full path to the storage file
	logger   *logger.Logger // Logger instance for storage operations
}

// NewStorage creates a new Storage instance for cron jobs.
// The filePath is constructed by joining workspacePath with the "cron" subdirectory and the jobs filename.
//
// Parameters:
//   - workspacePath: Path to the workspace directory
//   - logger: Logger instance for storage operations
//
// Returns:
//   - *Storage: A new storage instance ready for use
func NewStorage(workspacePath string, logger *logger.Logger) *Storage {
	filePath := filepath.Join(workspacePath, CronSubdirectory, JobsFilename)
	return &Storage{
		filePath: filePath,
		logger:   logger,
	}
}

// Load reads cron jobs from the JSONL storage file.
// Returns empty slice if file doesn't exist.
func (s *Storage) Load() ([]StorageJob, error) {
	// Check if file exists
	_, err := os.Stat(s.filePath)
	if os.IsNotExist(err) {
		return []StorageJob{}, nil
	}
	if err != nil {
		s.logger.Error("failed to stat storage file", err,
			logger.Field{Key: "file", Value: s.filePath})
		return nil, err
	}

	// Open file
	file, err := os.Open(s.filePath)
	if err != nil {
		s.logger.Error("failed to open storage file", err,
			logger.Field{Key: "file", Value: s.filePath})
		return nil, err
	}
	defer file.Close()

	var jobs []StorageJob
	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Read file line by line
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		var job StorageJob
		if err := json.Unmarshal([]byte(line), &job); err != nil {
			s.logger.Error("failed to unmarshal job line", err,
				logger.Field{Key: "file", Value: s.filePath},
				logger.Field{Key: "line", Value: lineNum})
			continue
		}

		jobs = append(jobs, job)
	}

	if err := scanner.Err(); err != nil {
		s.logger.Error("error scanning storage file", err,
			logger.Field{Key: "file", Value: s.filePath})
		return nil, err
	}

	return jobs, nil
}

// Append adds a new cron job to the storage file.
// The job is appended to the end of the file with a newline.
// Parameters:
//   - job: The StorageJob to append
//
// Returns:
//   - error: Error if the operation fails
func (s *Storage) Append(job StorageJob) error {
	// Open file with append mode
	file, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		s.logger.Error("failed to open storage file for append", err,
			logger.Field{Key: "file", Value: s.filePath})
		return err
	}
	defer file.Close()

	// Marshal job to JSON
	data, err := json.Marshal(job)
	if err != nil {
		s.logger.Error("failed to marshal job", err,
			logger.Field{Key: "job_id", Value: job.ID})
		return err
	}

	// Write job with newline
	if _, err := file.Write(append(data, '\n')); err != nil {
		s.logger.Error("failed to write job to storage", err,
			logger.Field{Key: "file", Value: s.filePath},
			logger.Field{Key: "job_id", Value: job.ID})
		return err
	}

	s.logger.Debug("job appended to storage",
		logger.Field{Key: "job_id", Value: job.ID},
		logger.Field{Key: "file", Value: s.filePath})

	return nil
}

// Remove removes a cron job from the storage by its ID.
// This operation loads all jobs, filters out the specified job, and saves the rest.
// Parameters:
//   - jobID: The ID of the job to remove
//
// Returns:
//   - error: Error if the operation fails
func (s *Storage) Remove(jobID string) error {
	// Load all jobs
	jobs, err := s.Load()
	if err != nil {
		return err
	}

	// Filter out the job to remove
	var filteredJobs []StorageJob
	removed := false
	for _, job := range jobs {
		if job.ID == jobID {
			removed = true
			continue
		}
		filteredJobs = append(filteredJobs, job)
	}

	if !removed {
		s.logger.Warn("job not found for removal",
			logger.Field{Key: "job_id", Value: jobID})
	}

	// Save filtered jobs
	if err := s.Save(filteredJobs); err != nil {
		return err
	}

	s.logger.Debug("job removed from storage",
		logger.Field{Key: "job_id", Value: jobID},
		logger.Field{Key: "file", Value: s.filePath})

	return nil
}

// Save writes all cron jobs to the storage file using atomic write.
// A temporary file is created first, then renamed to the actual file.
// Parameters:
//   - jobs: Slice of StorageJob to save
//
// Returns:
//   - error: Error if the operation fails
func (s *Storage) Save(jobs []StorageJob) error {
	// Create temporary file path
	tmpPath := s.filePath + ".tmp"

	// Open temporary file
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		s.logger.Error("failed to create temporary storage file", err,
			logger.Field{Key: "file", Value: tmpPath})
		return err
	}
	defer file.Close()

	// Write each job as a JSON line
	for _, job := range jobs {
		data, err := json.Marshal(job)
		if err != nil {
			s.logger.Error("failed to marshal job", err,
				logger.Field{Key: "job_id", Value: job.ID})
			return err
		}

		// Write job with newline
		if _, err := file.Write(append(data, '\n')); err != nil {
			s.logger.Error("failed to write job to temporary file", err,
				logger.Field{Key: "file", Value: tmpPath},
				logger.Field{Key: "job_id", Value: job.ID})
			return err
		}
	}

	// Ensure all data is written to disk
	if err := file.Sync(); err != nil {
		s.logger.Error("failed to sync temporary file", err,
			logger.Field{Key: "file", Value: tmpPath})
		return err
	}

	// Atomically rename temporary file to actual file
	if err := os.Rename(tmpPath, s.filePath); err != nil {
		s.logger.Error("failed to rename temporary file", err,
			logger.Field{Key: "from", Value: tmpPath},
			logger.Field{Key: "to", Value: s.filePath})
		return err
	}

	s.logger.Debug("jobs saved to storage",
		logger.Field{Key: "count", Value: len(jobs)},
		logger.Field{Key: "file", Value: s.filePath})

	return nil
}
