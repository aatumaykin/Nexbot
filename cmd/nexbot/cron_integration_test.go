package main

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCronCommandsIntegration(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Change to temp directory
	_ = os.Chdir(tempDir)

	// 1. Add first job directly to jobs.json
	jobs := map[string]cron.Job{
		"integration-job-1": {
			ID:       "integration-job-1",
			Schedule: "* * * * *",
			Command:  "test command 1",
			UserID:   "cli",
		},
	}

	err := cron.SaveJobs(tempDir, jobs)
	require.NoError(t, err)

	// Verify job was saved
	jobs, err = cron.LoadJobs(tempDir)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)

	// 2. Add second job
	jobs["integration-job-2"] = cron.Job{
		ID:       "integration-job-2",
		Schedule: "*/5 * * * *",
		Command:  "test command 2",
		UserID:   "cli",
	}

	err = cron.SaveJobs(tempDir, jobs)
	require.NoError(t, err)

	// Verify both jobs exist
	jobs, err = cron.LoadJobs(tempDir)
	require.NoError(t, err)
	assert.Len(t, jobs, 2)

	// 3. Remove first job
	err = cron.SaveJobs(tempDir, map[string]cron.Job{
		"integration-job-2": {
			ID:       "integration-job-2",
			Schedule: "*/5 * * * *",
			Command:  "test command 2",
			UserID:   "cli",
		},
	})
	require.NoError(t, err)

	// 4. Verify only one job remains
	jobs, err = cron.LoadJobs(tempDir)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "integration-job-2", jobs["integration-job-2"].ID)
}

func TestMultipleConcurrentCLIOperations(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Change to temp directory
	_ = os.Chdir(tempDir)

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	var mu sync.Mutex

	// Spawn goroutines to add jobs concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Generate unique job ID with goroutine-safe approach
			jobID := fmt.Sprintf("concurrent-job-%d-%d", os.Getpid(), id)
			job := cron.Job{
				ID:       jobID,
				Schedule: "* * * * *",
				Command:  fmt.Sprintf("concurrent command %d", id),
				UserID:   "cli",
			}

			mu.Lock()
			defer mu.Unlock()

			// Load existing jobs
			jobs, err := cron.LoadJobs(tempDir)
			if err != nil && !os.IsNotExist(err) {
				return
			}

			if jobs == nil {
				jobs = make(map[string]cron.Job)
			}

			// Add new job
			jobs[jobID] = job

			// Save jobs
			_ = cron.SaveJobs(tempDir, jobs)
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all jobs were saved
	jobs, err := cron.LoadJobs(tempDir)
	require.NoError(t, err)
	assert.Len(t, jobs, numGoroutines)

	// Verify each job has unique ID
	ids := make(map[string]bool)
	for _, job := range jobs {
		assert.False(t, ids[job.ID], "Duplicate job ID found: %s", job.ID)
		ids[job.ID] = true
	}
}
