package heartbeat

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_Load_FileExists(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create HEARTBEAT.md file
	content := `# Heartbeat Tasks

## Periodic Reviews

### Daily Standup
- Schedule: "0 0 9 * * *"
- Task: "Review daily progress"
`
	heartbeatPath := filepath.Join(tmpDir, "HEARTBEAT.md")
	err = os.WriteFile(heartbeatPath, []byte(content), 0644)
	require.NoError(t, err)

	// Create loader
	loader := NewLoader(tmpDir, log)

	// Load tasks
	tasks, err := loader.Load()

	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "Daily Standup", tasks[0].Name)
	assert.True(t, loader.isLoaded)
}

func TestLoader_Load_FileNotExists(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create loader
	loader := NewLoader(tmpDir, log)

	// Load tasks (file doesn't exist)
	tasks, err := loader.Load()

	require.NoError(t, err)
	assert.Nil(t, tasks)
	assert.True(t, loader.isLoaded)
}

func TestLoader_Load_InvalidTasks(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create HEARTBEAT.md file with invalid tasks
	content := `# Heartbeat Tasks

## Periodic Reviews

### Invalid Task 1
- Schedule: "invalid-cron"
- Task: "Invalid cron expression"

### Invalid Task 2
- Schedule: "0 0 9 * * *"
- Task: ""
`
	heartbeatPath := filepath.Join(tmpDir, "HEARTBEAT.md")
	err = os.WriteFile(heartbeatPath, []byte(content), 0644)
	require.NoError(t, err)

	// Create loader
	loader := NewLoader(tmpDir, log)

	// Load tasks (invalid tasks should be skipped)
	tasks, err := loader.Load()

	require.NoError(t, err)
	assert.Len(t, tasks, 0)
}

func TestLoader_Load_ValidAndInvalidTasks(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create HEARTBEAT.md file with mix of valid and invalid tasks
	content := `# Heartbeat Tasks

## Periodic Reviews

### Invalid Task
- Schedule: "invalid-cron"
- Task: "Invalid cron expression"

### Daily Standup
- Schedule: "0 0 9 * * *"
- Task: "Review daily progress"
`
	heartbeatPath := filepath.Join(tmpDir, "HEARTBEAT.md")
	err = os.WriteFile(heartbeatPath, []byte(content), 0644)
	require.NoError(t, err)

	// Create loader
	loader := NewLoader(tmpDir, log)

	// Load tasks
	tasks, err := loader.Load()

	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "Daily Standup", tasks[0].Name)
}

func TestLoader_GetTasks_AfterLoad(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create HEARTBEAT.md file
	content := `# Heartbeat Tasks

## Periodic Reviews

### Daily Standup
- Schedule: "0 0 9 * * *"
- Task: "Review daily progress"
`
	heartbeatPath := filepath.Join(tmpDir, "HEARTBEAT.md")
	err = os.WriteFile(heartbeatPath, []byte(content), 0644)
	require.NoError(t, err)

	// Create loader
	loader := NewLoader(tmpDir, log)

	// Load tasks
	_, err = loader.Load()
	require.NoError(t, err)

	// Get tasks
	tasks := loader.GetTasks()

	assert.Len(t, tasks, 1)
	assert.Equal(t, "Daily Standup", tasks[0].Name)
}

func TestLoader_GetTasks_BeforeLoad(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create loader
	loader := NewLoader(tmpDir, log)

	// Get tasks before loading
	tasks := loader.GetTasks()

	assert.Nil(t, tasks)
}

func TestLoader_GetContext_AfterLoad(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create HEARTBEAT.md file
	content := `# Heartbeat Tasks

## Periodic Reviews

### Daily Standup
- Schedule: "0 0 9 * * *"
- Task: "Review daily progress"

### Weekly Summary
- Schedule: "0 0 17 * * 5"
- Task: "Generate weekly summary"
`
	heartbeatPath := filepath.Join(tmpDir, "HEARTBEAT.md")
	err = os.WriteFile(heartbeatPath, []byte(content), 0644)
	require.NoError(t, err)

	// Create loader
	loader := NewLoader(tmpDir, log)

	// Load tasks
	_, err = loader.Load()
	require.NoError(t, err)

	// Get context
	context := loader.GetContext()

	assert.Equal(t, "Active heartbeat tasks: 2 (Daily Standup, Weekly Summary)", context)
}

func TestLoader_GetContext_NoTasks(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create loader
	loader := NewLoader(tmpDir, log)

	// Load tasks (file doesn't exist)
	_, err = loader.Load()
	require.NoError(t, err)

	// Get context
	context := loader.GetContext()

	assert.Equal(t, "No active heartbeat tasks", context)
}

func TestLoader_SetTasks(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create loader
	loader := NewLoader(tmpDir, log)

	// Set tasks
	tasks := []HeartbeatTask{
		{Name: "Test Task", Schedule: "0 0 9 * * *", Task: "Test"},
	}
	loader.SetTasks(tasks)

	// Get tasks
	result := loader.GetTasks()

	assert.Len(t, result, 1)
	assert.Equal(t, "Test Task", result[0].Name)
	assert.True(t, loader.isLoaded)
}
