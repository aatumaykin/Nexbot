// Package cron provides utility functions for managing cron jobs.
package cron

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aatumaykin/nexbot/internal/constants"
)

// LoadJobs loads cron jobs from the workspace.
// Returns an empty map if the jobs file doesn't exist.
// Parameters:
//   - workspacePath: Path to the workspace directory
//
// Returns:
//   - map[string]Job: Map of job ID to Job
//   - error: Error if loading fails for reasons other than file not existing
func LoadJobs(workspacePath string) (map[string]Job, error) {
	jobsPath := filepath.Join(workspacePath, constants.CronJobsFile)

	data, err := os.ReadFile(jobsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]Job), nil
		}
		return nil, fmt.Errorf("failed to read jobs file: %w", err)
	}

	var jobs map[string]Job
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, fmt.Errorf("failed to parse jobs file: %w", err)
	}

	if jobs == nil {
		jobs = make(map[string]Job)
	}

	return jobs, nil
}

// SaveJobs saves cron jobs to the workspace.
// Parameters:
//   - workspacePath: Path to the workspace directory
//   - jobs: Map of job ID to Job to save
//
// Returns:
//   - error: Error if saving fails
func SaveJobs(workspacePath string, jobs map[string]Job) error {
	jobsPath := filepath.Join(workspacePath, constants.CronJobsFile)

	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jobs: %w", err)
	}

	if err := os.WriteFile(jobsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write jobs file: %w", err)
	}

	return nil
}

// GenerateJobID generates a unique job ID using the process ID.
// Returns a string in the format "job_<pid>".
func GenerateJobID() string {
	return fmt.Sprintf(constants.CronJobIDFormat, os.Getpid())
}
