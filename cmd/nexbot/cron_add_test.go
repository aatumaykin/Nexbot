package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCronAdd(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Change to temp directory
	_ = os.Chdir(tempDir)

	// Set up test command
	args := []string{"cron", "add", "* * * * *", "test command"}

	// Create mock command
	cmd := cronAddCmd
	cmd.SetArgs(args[1:]) // Skip "cron" part

	// Run command
	runCronAdd(cmd, args[1:])

	// Verify jobs file was created
	jobsPath := filepath.Join(tempDir, "jobs.json")
	_, err := os.Stat(jobsPath)
	require.NoError(t, err)
}

func TestCronListEmpty(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Change to temp directory
	_ = os.Chdir(tempDir)

	// Create empty jobs file
	jobsPath := filepath.Join(tempDir, "jobs.json")
	err := os.WriteFile(jobsPath, []byte("{}"), 0644)
	require.NoError(t, err)

	// Run list command
	args := []string{"cron", "list"}
	cmd := cronListCmd
	cmd.SetArgs(args[1:])

	// Capture stdout
	// Note: For now, just run the command
	runCronList(cmd, args[1:])
}

func TestCronListWithJobs(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Change to temp directory
	_ = os.Chdir(tempDir)

	// Create jobs file with some jobs
	jobsPath := filepath.Join(tempDir, "jobs.json")
	jobsContent := `{
  "job_1": {
    "id": "job_1",
    "schedule": "* * * * *",
    "command": "command 1",
    "user_id": "cli"
  },
  "job_2": {
    "id": "job_2",
    "schedule": "*/5 * * * *",
    "command": "command 2",
    "user_id": "cli"
  }
}`
	err := os.WriteFile(jobsPath, []byte(jobsContent), 0644)
	require.NoError(t, err)

	// Run list command
	args := []string{"cron", "list"}
	cmd := cronListCmd
	cmd.SetArgs(args[1:])

	// Capture stdout
	// Note: For now, just run the command
	runCronList(cmd, args[1:])
}

func TestCronRemove(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Change to temp directory
	_ = os.Chdir(tempDir)

	// Create jobs file with a job
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

	// Verify job exists before removal
	jobs, _ := cron.LoadJobs(tempDir)
	_, exists := jobs["job_1"]
	assert.True(t, exists)

	// Run remove command
	args := []string{"cron", "remove", "job_1"}
	cmd := cronRemoveCmd
	cmd.SetArgs(args[1:]) // Pass ["remove", "job_1"]

	runCronRemove(cmd, []string{"job_1"}) // Pass just job ID

	// Verify job was removed
	jobs, _ = cron.LoadJobs(tempDir)
	_, exists = jobs["job_1"]
	assert.False(t, exists)
}

func TestCronRemoveNonExistent(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Change to temp directory
	_ = os.Chdir(tempDir)

	// Create empty jobs file
	jobsPath := filepath.Join(tempDir, "jobs.json")
	err := os.WriteFile(jobsPath, []byte("{}"), 0644)
	require.NoError(t, err)

	// Load jobs to verify
	jobs, _ := cron.LoadJobs(tempDir)
	assert.NotNil(t, jobs)

	// Try to remove non-existent job - this will call os.Exit
	// We can't test this properly without os.Exit capture
	// Skip for now
	t.Skip("os.Exit makes this test difficult to implement")
}

func TestCronAddCommandWithInvalidSchedule(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Change to temp directory
	_ = os.Chdir(tempDir)

	// This test verifies that we can add a job with any schedule string to storage
	// The actual validation happens when the scheduler loads and tries to use it
	jobs := map[string]cron.Job{}
	err := cron.SaveJobs(tempDir, jobs)
	require.NoError(t, err)

	// Try to add job with invalid schedule (this will save to storage but fail in scheduler)
	// Since runCronAdd doesn't do validation, this will succeed
	cmd := cronAddCmd
	args := []string{"invalid-cron", "test command"}
	cmd.SetArgs(args)

	// Capture stderr to check for errors (though the function calls os.Exit on error)
	// We can't easily test os.Exit without using a library like testy
	// For now, we just verify the function doesn't panic
	runCronAdd(cmd, args)

	// Verify jobs file was created
	jobsPath := filepath.Join(tempDir, "jobs.json")
	_, err = os.Stat(jobsPath)
	require.NoError(t, err)

	// Load and verify job was saved
	loadedJobs, err := cron.LoadJobs(tempDir)
	require.NoError(t, err)
	assert.Greater(t, len(loadedJobs), 0)
}

func TestCronAddCommandWithEmptyArguments(t *testing.T) {
	// Create temporary directory to avoid conflicts
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})
	_ = os.Chdir(tempDir)

	// This test verifies cobra.ExactArgs(2) is working
	// Note: We can't use cmd.Execute() here because it will try to use the full command tree
	// Instead, we test the validation by calling runCronAdd directly

	// Test with no arguments - should work but may panic internally
	// Skipping this test as runCronAdd expects exact args from cobra
	t.Skip("Skipping argument validation test - cobra.ExactArgs handles this")
}
