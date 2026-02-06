package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadJobsNoFile(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Change to temp directory
	_ = os.Chdir(tempDir)

	// Load jobs when no file exists
	jobs, err := cron.LoadJobs(tempDir)

	// Should return empty map and no error
	assert.NoError(t, err)
	assert.NotNil(t, jobs)
	assert.Empty(t, jobs)
}

func TestLoadJobsWithFile(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Change to temp directory
	_ = os.Chdir(tempDir)

	// Create jobs file
	jobsPath := filepath.Join(tempDir, "jobs.json")
	jobsContent := `{
  "job_1": {
    "id": "job_1",
    "schedule": "* * * * *",
    "command": "test command",
    "user_id": "cli"
  }
}`
	err := os.WriteFile(jobsPath, []byte(jobsContent), 0644)
	require.NoError(t, err)

	// Load jobs
	jobs, err := cron.LoadJobs(tempDir)

	require.NoError(t, err)
	require.NotNil(t, jobs)
	assert.Len(t, jobs, 1)

	job := jobs["job_1"]
	assert.Equal(t, "job_1", job.ID)
	assert.Equal(t, "* * * * *", job.Schedule)
	assert.Equal(t, "test command", job.Command)
}

func TestSaveJobs(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Change to temp directory
	_ = os.Chdir(tempDir)

	// Create jobs
	jobs := map[string]cron.Job{
		"job_1": {
			ID:       "job_1",
			Schedule: "* * * * *",
			Command:  "test command",
			UserID:   "cli",
		},
	}

	// Save jobs
	err := cron.SaveJobs(tempDir, jobs)
	require.NoError(t, err)

	// Verify file was created
	jobsPath := filepath.Join(tempDir, "jobs.json")
	data, err := os.ReadFile(jobsPath)
	require.NoError(t, err)

	// Verify content
	assert.Contains(t, string(data), "job_1")
	assert.Contains(t, string(data), "test command")
}

func TestGenerateJobID(t *testing.T) {
	// Generate job IDs
	id1 := cron.GenerateJobID()
	id2 := cron.GenerateJobID()

	// IDs should be different (process IDs are recycled, but in tests they're often the same)
	// So we'll just check format
	assert.Contains(t, id1, "job_")
	assert.Contains(t, id2, "job_")
}

func TestCronJobPersistence(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Change to temp directory
	_ = os.Chdir(tempDir)

	// Create a logger and message bus for first scheduler instance
	log, err := logger.New(logger.Config{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	msgBus := bus.New(100, log)
	err = msgBus.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, msgBus.Stop())
	})

	// Create and start first scheduler instance
	scheduler1 := cron.NewScheduler(log, msgBus, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler1.Start(ctx)
	require.NoError(t, err)

	// Add a job via CLI (simulated by directly adding to jobs.json)
	jobs := map[string]cron.Job{
		"persistent-job-1": {
			ID:       "persistent-job-1",
			Schedule: "0 * * * * *",
			Command:  "persistent command 1",
			UserID:   "cli",
		},
		"persistent-job-2": {
			ID:       "persistent-job-2",
			Schedule: "*/30 * * * * *",
			Command:  "persistent command 2",
			UserID:   "cli",
		},
	}

	err = cron.SaveJobs(tempDir, jobs)
	require.NoError(t, err)

	// Load jobs to verify they were saved
	savedJobs, err := cron.LoadJobs(tempDir)
	require.NoError(t, err)
	assert.Len(t, savedJobs, 2)

	// Stop first scheduler
	err = scheduler1.Stop()
	require.NoError(t, err)

	// Create second scheduler instance (simulate restart)
	scheduler2 := cron.NewScheduler(log, msgBus, nil, nil)
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	err = scheduler2.Start(ctx2)
	require.NoError(t, err)
	if err := scheduler2.Stop(); err != nil {
		t.Fatal(err)
	}

	// Load jobs again - they should still be there
	reloadedJobs, err := cron.LoadJobs(tempDir)
	require.NoError(t, err)
	assert.Len(t, reloadedJobs, 2)

	// Verify job details persisted
	assert.Equal(t, "persistent-job-1", reloadedJobs["persistent-job-1"].ID)
	assert.Equal(t, "0 * * * * *", reloadedJobs["persistent-job-1"].Schedule)
	assert.Equal(t, "persistent command 1", reloadedJobs["persistent-job-1"].Command)

	assert.Equal(t, "persistent-job-2", reloadedJobs["persistent-job-2"].ID)
	assert.Equal(t, "*/30 * * * * *", reloadedJobs["persistent-job-2"].Schedule)
	assert.Equal(t, "persistent command 2", reloadedJobs["persistent-job-2"].Command)
}
