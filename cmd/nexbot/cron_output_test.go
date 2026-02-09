package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCronListCommandOutput(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Change to temp directory
	_ = os.Chdir(tempDir)

	// Create jobs file with multiple jobs
	jobsPath := filepath.Join(tempDir, "jobs.json")
	jobsContent := `{
   "job_1": {
     "id": "job_1",
     "schedule": "* * * * *",
     "tool": "agent",
     "payload": {"message": "command 1"},
     "user_id": "cli",
     "metadata": {
       "env": "production"
     }
   },
   "job_2": {
     "id": "job_2",
     "schedule": "*/5 * * * *",
     "tool": "agent",
     "payload": {"message": "command 2"},
     "user_id": "cli"
   }
 }`
	err := os.WriteFile(jobsPath, []byte(jobsContent), 0644)
	require.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run list command
	args := []string{"cron", "list"}
	cmd := cronListCmd
	cmd.SetArgs(args[1:])

	runCronList(cmd, args[1:])

	// Close writer and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	output := buf.String()

	// Verify output contains expected information (logger structured format)
	assert.Contains(t, output, "Scheduled jobs:")
	assert.Contains(t, output, "id=job_1")
	assert.Contains(t, output, "tool=agent")
	assert.Contains(t, output, "message=\"command 1\"")
	assert.Contains(t, output, "id=job_2")
	assert.Contains(t, output, "tool=agent")
	assert.Contains(t, output, "message=\"command 2\"")
	assert.Contains(t, output, "Total jobs")
	assert.Contains(t, output, "count=2")
}

func TestCronRemoveCommandOutput(t *testing.T) {
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

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run remove command
	args := []string{"cron", "remove", "job_1"}
	cmd := cronRemoveCmd
	cmd.SetArgs(args[1:])

	runCronRemove(cmd, []string{"job_1"})

	// Close writer and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	output := buf.String()

	// Verify output contains success message (logger structured format)
	assert.Contains(t, output, "Job removed")
	assert.Contains(t, output, "job_id=job_1")

	// Verify job was removed
	jobs, _ := cron.LoadJobs(tempDir)
	_, exists := jobs["job_1"]
	assert.False(t, exists)
}
