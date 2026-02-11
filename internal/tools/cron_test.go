package tools

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestEnvironment creates a test environment with scheduler, storage, and logger.
func setupTestEnvironment(t *testing.T) (*cron.Scheduler, *cron.Storage, *logger.Logger, func()) {
	// Create temporary directory for storage
	tempDir := t.TempDir()

	// Create logger
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	// Create message bus
	messageBus := bus.New(100, 10, log)

	// Create storage
	storage := cron.NewStorage(tempDir, log)

	// Create scheduler with nil worker pool (not needed for tests)
	scheduler := cron.NewScheduler(log, messageBus, nil, storage)

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err, "Failed to start scheduler")

	// Cleanup function
	cleanup := func() {
		cancel()
		_ = scheduler.Stop()
		os.RemoveAll(tempDir)
	}

	return scheduler, storage, log, cleanup
}

// setupCronTool creates a CronTool for testing.
func setupCronTool(t *testing.T) *CronTool {
	scheduler, storage, log, _ := setupTestEnvironment(t)
	cronAdapter := cron.NewCronSchedulerAdapter(scheduler, storage)
	return NewCronTool(cronAdapter, log)
}

// TestCronTool_Name tests that the tool returns the correct name.
func TestCronTool_Name(t *testing.T) {
	tool := setupCronTool(t)
	assert.Equal(t, "cron", tool.Name(), "Tool name should be 'cron'")
}

// TestCronTool_Description tests that the tool returns a non-empty description.
func TestCronTool_Description(t *testing.T) {
	tool := setupCronTool(t)
	desc := tool.Description()
	assert.NotEmpty(t, desc, "Description should not be empty")
	assert.Contains(t, desc, "cron", "Description should mention 'cron'")
}

// TestCronTool_Parameters tests that the tool returns valid parameters.
func TestCronTool_Parameters(t *testing.T) {
	tool := setupCronTool(t)
	params := tool.Parameters()

	assert.NotNil(t, params, "Parameters should not be nil")
	assert.Equal(t, "object", params["type"], "Type should be 'object'")

	props, ok := params["properties"].(map[string]interface{})
	assert.True(t, ok, "Properties should be a map")

	// Check action property
	actionProp, ok := props["action"].(map[string]interface{})
	assert.True(t, ok, "Action property should be a map")
	assert.Equal(t, "string", actionProp["type"], "Action type should be 'string'")
	assert.Contains(t, actionProp["enum"], "add_recurring", "Action enum should contain 'add_recurring'")
	assert.Contains(t, actionProp["enum"], "add_oneshot", "Action enum should contain 'add_oneshot'")
	assert.Contains(t, actionProp["enum"], "remove", "Action enum should contain 'remove'")
	assert.Contains(t, actionProp["enum"], "list", "Action enum should contain 'list'")

	// Check required fields - try both types
	required := params["required"]
	switch v := required.(type) {
	case []interface{}:
		assert.Contains(t, v, "action", "Required should contain 'action'")
	case []string:
		assert.Contains(t, v, "action", "Required should contain 'action'")
	default:
		assert.Fail(t, "Required should be a slice")
	}
}

// TestCronToolAddRecurring tests adding a recurring job.
func TestCronToolAddRecurring(t *testing.T) {
	tool := setupCronTool(t)

	args := `{
		"action": "add_recurring",
		"schedule": "0 0 0 * * *",
		"tool": "send_message",
		"payload": "{\"message\": \"test command\"}",
		"session_id": "telegram:123456789"
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "Recurring job added successfully", "Result should contain success message")
	assert.Contains(t, result, "0 0 0 * * *", "Result should contain schedule")
	assert.Contains(t, result, "test command", "Result should contain message")
}

// TestCronToolAddOneshot tests adding a one-time job.
func TestCronToolAddOneshot(t *testing.T) {
	tool := setupCronTool(t)

	executeAt := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	args := `{
		"action": "add_oneshot",
		"execute_at": "` + executeAt + `",
		"tool": "send_message",
		"payload": "{\"message\": \"test command\"}",
		"session_id": "telegram:123456789"
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "One-time job added successfully", "Result should contain success message")
	assert.Contains(t, result, "test command", "Result should contain message")
}

// TestCronToolRemoveJob tests removing a job.
func TestCronToolRemoveJob(t *testing.T) {
	scheduler, storage, log, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cronAdapter := cron.NewCronSchedulerAdapter(scheduler, storage)
	tool := NewCronTool(cronAdapter, log)

	// First, add a job
	addArgs := `{
		"action": "add_recurring",
		"schedule": "0 0 0 * * *",
		"tool": "send_message",
		"payload": "{\"message\": \"test command\"}",
		"session_id": "telegram:123456789"
	}`

	addResult, err := tool.Execute(addArgs)
	require.NoError(t, err, "Failed to add job")

	// Extract job ID from result (format: "   Job ID: <id>")
	// Use string parsing instead of scanf
	lines := strings.Split(addResult, "\n")
	jobID := ""
	for _, line := range lines {
		if strings.Contains(line, "Job ID:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				jobID = strings.TrimSpace(parts[1])
			}
			break
		}
	}
	require.NotEmpty(t, jobID, "Failed to extract job ID from result")

	// Now remove the job
	removeArgs := `{
		"action": "remove",
		"job_id": "` + jobID + `"
	}`

	result, err := tool.Execute(removeArgs)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "Job removed successfully", "Result should contain success message")
	assert.Contains(t, result, jobID, "Result should contain job ID")
}

// TestCronToolListJobs tests listing jobs.
func TestCronToolListJobs(t *testing.T) {
	scheduler, storage, log, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cronAdapter := cron.NewCronSchedulerAdapter(scheduler, storage)
	tool := NewCronTool(cronAdapter, log)

	// Add a job first
	addArgs := `{
		"action": "add_recurring",
		"schedule": "0 0 0 * * *",
		"tool": "send_message",
		"payload": "{\"message\": \"command1\"}",
		"session_id": "telegram:123456789"
	}`

	_, err := tool.Execute(addArgs)
	require.NoError(t, err, "Failed to add job")

	// List jobs
	args := `{
		"action": "list"
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "command1", "Result should contain command1")
}

// TestCronToolListJobs_Empty tests listing jobs when no jobs exist.
func TestCronToolListJobs_Empty(t *testing.T) {
	tool := setupCronTool(t)

	args := `{
		"action": "list"
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "No scheduled jobs found", "Result should indicate no jobs")
}

// TestCronToolInvalidAction tests handling of invalid action.
func TestCronToolInvalidAction(t *testing.T) {
	tool := setupCronTool(t)

	args := `{
		"action": "invalid_action"
	}`

	result, err := tool.Execute(args)
	assert.Error(t, err, "Execute should return error for invalid action")
	assert.Empty(t, result, "Result should be empty on error")
	assert.Contains(t, err.Error(), "invalid action", "Error should mention 'invalid action'")
}

// TestCronToolMissingRequiredParams tests handling of missing required parameters.
func TestCronToolMissingRequiredParams(t *testing.T) {
	tests := []struct {
		name        string
		args        string
		expectedErr string
	}{
		{
			name: "missing schedule for add_recurring",
			args: `{
				"action": "add_recurring",
				"tool": "send_message",
				"payload": "{\"message\": \"test command\"}",
				"user_id": "user123"
			}`,
			expectedErr: "schedule parameter is required",
		},
		{
			name: "missing tool for add_recurring",
			args: `{
				"action": "add_recurring",
				"schedule": "0 * * * *",
				"user_id": "user123"
			}`,
			expectedErr: "tool parameter is required",
		},
		{
			name: "missing execute_at for add_oneshot",
			args: `{
				"action": "add_oneshot",
				"tool": "send_message",
				"payload": "{\"message\": \"test command\"}",
				"user_id": "user123"
			}`,
			expectedErr: "execute_at parameter is required",
		},
		{
			name: "missing job_id for remove",
			args: `{
				"action": "remove"
			}`,
			expectedErr: "job_id parameter is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tool := setupCronTool(t)
			_, err := tool.Execute(tc.args)
			assert.Error(t, err, "Execute should return error")
			assert.Contains(t, err.Error(), tc.expectedErr, "Error should contain expected message")
		})
	}
}

// TestCronToolInvalidJSON tests handling of invalid JSON.
func TestCronToolInvalidJSON(t *testing.T) {
	tool := setupCronTool(t)

	args := `{invalid json`

	_, err := tool.Execute(args)
	assert.Error(t, err, "Execute should return error for invalid JSON")
	assert.Contains(t, err.Error(), "failed to parse cron arguments", "Error should mention parse error")
}

// TestCronToolInvalidExecuteAt tests handling of invalid execute_at format.
func TestCronToolInvalidExecuteAt(t *testing.T) {
	tool := setupCronTool(t)

	args := `{
		"action": "add_oneshot",
		"execute_at": "not-a-valid-date",
		"tool": "send_message",
		"payload": "{\"message\":\"test\"}",
		"session_id": "telegram:123456789"
	}`

	_, err := tool.Execute(args)
	assert.Error(t, err, "Execute should return error for invalid date")
	assert.Contains(t, err.Error(), "invalid execute_at format", "Error should mention date format")
}

// TestCronToolToSchema tests the ToSchema method.
func TestCronToolToSchema(t *testing.T) {
	tool := setupCronTool(t)
	schema := tool.ToSchema()
	assert.NotNil(t, schema, "Schema should not be nil")
	assert.Equal(t, tool.Parameters(), schema, "Schema should match parameters")
}
