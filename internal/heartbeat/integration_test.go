package heartbeat

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_HeartbeatWithScheduler(t *testing.T) {
	// Create temporary workspace
	tmpDir := t.TempDir()

	// Create logger
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create message bus
	messageBus := bus.New(100, log)

	// Create scheduler
	scheduler := cron.NewScheduler(log, messageBus, nil, nil)

	// Create HEARTBEAT.md file with invalid tasks
	content := `# Heartbeat Tasks

## Periodic Reviews

### Daily Standup
- Schedule: "0 0 9 * * *"
- Task: "Review daily progress, check for blocked tasks, update priorities"

### Weekly Summary
- Schedule: "0 0 17 * * 5"
- Task: "Generate weekly summary of completed tasks and planned work"

### Health Check
- Schedule: "0 0 */6 * * *"
- Task: "Check system health, monitor logs, alert on errors"
`
	heartbeatPath := filepath.Join(tmpDir, "HEARTBEAT.md")
	err = os.WriteFile(heartbeatPath, []byte(content), 0644)
	require.NoError(t, err)

	// Load heartbeat tasks
	loader := NewLoader(tmpDir, log)
	tasks, err := loader.Load()
	require.NoError(t, err)
	assert.Len(t, tasks, 3)

	// Register tasks with scheduler
	for _, task := range tasks {
		job := cron.Job{
			ID:       fmt.Sprintf("heartbeat_%s", task.Name),
			Schedule: task.Schedule,
			Command:  task.Task,
			UserID:   "system",
			Metadata: map[string]string{
				"type":        "heartbeat",
				"task_name":   task.Name,
				"source_file": "HEARTBEAT.md",
			},
		}

		jobID, err := scheduler.AddJob(job)
		require.NoError(t, err)
		assert.NotEmpty(t, jobID)
	}

	// List jobs from scheduler
	jobs := scheduler.ListJobs()
	assert.Len(t, jobs, 3)

	// Verify jobs are registered correctly
	for i, job := range jobs {
		assert.Contains(t, job.ID, "heartbeat_")
		assert.NotEmpty(t, job.Schedule)
		assert.NotEmpty(t, job.Command)
		assert.Equal(t, "system", job.UserID)
		assert.Equal(t, "heartbeat", job.Metadata["type"])
		assert.Equal(t, tasks[i].Name, job.Metadata["task_name"])
	}

	// Verify context format
	context := loader.GetContext()
	assert.Contains(t, context, "Active heartbeat tasks: 3")
	assert.Contains(t, context, "Daily Standup")
	assert.Contains(t, context, "Weekly Summary")
	assert.Contains(t, context, "Health Check")
}

func TestIntegration_HeartbeatWithInvalidTasks(t *testing.T) {
	// Create temporary workspace
	tmpDir := t.TempDir()

	// Create logger
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create message bus
	messageBus := bus.New(100, log)

	// Create scheduler
	scheduler := cron.NewScheduler(log, messageBus, nil, nil)

	// Create HEARTBEAT.md file with invalid tasks
	content := `# Heartbeat Tasks

## Periodic Reviews

### Invalid Task
- Schedule: "invalid-cron"
- Task: "This task should be skipped"

### Valid Task
- Schedule: "0 0 9 * * *"
- Task: "This task should be registered"
`
	heartbeatPath := filepath.Join(tmpDir, "HEARTBEAT.md")
	err = os.WriteFile(heartbeatPath, []byte(content), 0644)
	require.NoError(t, err)

	// Load heartbeat tasks
	loader := NewLoader(tmpDir, log)
	tasks, err := loader.Load()
	require.NoError(t, err)
	assert.Len(t, tasks, 1) // Only valid task should be loaded

	// Register valid task with scheduler
	for _, task := range tasks {
		job := cron.Job{
			ID:       fmt.Sprintf("heartbeat_%s", task.Name),
			Schedule: task.Schedule,
			Command:  task.Task,
			UserID:   "system",
			Metadata: map[string]string{
				"type":        "heartbeat",
				"task_name":   task.Name,
				"source_file": "HEARTBEAT.md",
			},
		}

		jobID, err := scheduler.AddJob(job)
		require.NoError(t, err)
		assert.NotEmpty(t, jobID)
	}

	// List jobs from scheduler
	jobs := scheduler.ListJobs()
	assert.Len(t, jobs, 1)

	// Verify only valid task was registered
	assert.Equal(t, "Valid Task", jobs[0].Metadata["task_name"])
}

func TestIntegration_HeartbeatContextInBuilder(t *testing.T) {
	// Create temporary workspace
	tmpDir := t.TempDir()

	// Create logger
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create HEARTBEAT.md file
	content := `# Heartbeat Tasks

## Periodic Reviews

### Daily Standup
- Schedule: "0 0 9 * * *"
- Task: "Review daily progress, check for blocked tasks, update priorities"

### Weekly Summary
- Schedule: "0 0 17 * * 5"
- Task: "Generate weekly summary of completed tasks and planned work"
`
	heartbeatPath := filepath.Join(tmpDir, "HEARTBEAT.md")
	err = os.WriteFile(heartbeatPath, []byte(content), 0644)
	require.NoError(t, err)

	// Load heartbeat tasks
	loader := NewLoader(tmpDir, log)
	_, err = loader.Load()
	require.NoError(t, err)

	// Get context
	context := loader.GetContext()
	assert.NotEmpty(t, context)
	assert.Contains(t, context, "Active heartbeat tasks: 2")
	assert.Contains(t, context, "Daily Standup")
	assert.Contains(t, context, "Weekly Summary")
}

func TestIntegration_HeartbeatEmptyWorkspace(t *testing.T) {
	// Create temporary workspace
	tmpDir := t.TempDir()

	// Create logger
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Load heartbeat tasks (file doesn't exist)
	loader := NewLoader(tmpDir, log)
	tasks, err := loader.Load()
	require.NoError(t, err)
	assert.Nil(t, tasks)

	// Get context
	context := loader.GetContext()
	assert.Equal(t, "No active heartbeat tasks", context)

	// Try to register tasks with scheduler (should be no tasks)
	messageBus := bus.New(100, log)
	scheduler := cron.NewScheduler(log, messageBus, nil, nil)

	// List jobs from scheduler (should be empty)
	jobs := scheduler.ListJobs()
	assert.Len(t, jobs, 0)
}
