package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCronToolFieldsPreservation tests that Tool, Payload, SessionID fields
// are correctly saved to storage and loaded back
func TestCronToolFieldsPreservation(t *testing.T) {
	// Create temporary directory for storage
	tmpDir := t.TempDir()
	cronDir := filepath.Join(tmpDir, "cron")
	require.NoError(t, os.MkdirAll(cronDir, 0755))

	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	storage := cron.NewStorage(tmpDir, log)

	// Create scheduler and adapter
	scheduler := cron.NewScheduler(log, nil, nil, storage)
	adapter := cron.NewCronSchedulerAdapter(scheduler, storage)
	tool := NewCronTool(adapter, log)

	// Test 1: Create oneshot job with all fields
	executeAt := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	payload := map[string]string{"message": "Test reminder"}
	payloadJSON, _ := json.Marshal(payload)

	args := map[string]interface{}{
		"action":     "add_oneshot",
		"execute_at": executeAt,
		"tool":       "send_message",
		"payload":    string(payloadJSON),
		"session_id": "telegram:123456",
	}

	argsJSON, _ := json.Marshal(args)
	result, err := tool.Execute(string(argsJSON))
	require.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "One-time job added successfully", "Result should contain success message")

	// Load jobs from storage and verify fields
	jobs, err := storage.Load()
	require.NoError(t, err, "Load should not return error")
	require.Len(t, jobs, 1, "Should have exactly one job")

	job := jobs[0]
	assert.NotEmpty(t, job.ID, "Job ID should not be empty")
	assert.Equal(t, "oneshot", job.Type, "Job type should be oneshot")
	assert.Equal(t, "send_message", job.Tool, "Tool should be send_message")
	assert.NotEmpty(t, job.Payload, "Payload should not be empty")
	assert.Equal(t, "telegram:123456", job.SessionID, "SessionID should match")

	// Verify payload content
	if msg, ok := job.Payload["message"].(string); ok {
		assert.Equal(t, "Test reminder", msg, "Message should match")
	} else {
		t.Error("Payload message should be a string")
	}

	// Test 2: List jobs via adapter and verify fields are preserved
	listedJobs := adapter.ListJobs()
	require.Len(t, listedJobs, 1, "Should have one listed job")

	listedJob := listedJobs[0]
	assert.Equal(t, job.ID, listedJob.ID, "Job ID should match")
	assert.Equal(t, job.Tool, listedJob.Tool, "Tool should match")
	assert.Equal(t, job.SessionID, listedJob.SessionID, "SessionID should match")

	if msg, ok := listedJob.Payload["message"].(string); ok {
		assert.Equal(t, "Test reminder", msg, "Payload message should match")
	} else {
		t.Error("Listed payload message should be a string")
	}

	// Test 3: Verify fields persist after restart
	// Simulate restart by creating new storage instance
	newStorage := cron.NewStorage(tmpDir, log)
	reloadedJobs, err := newStorage.Load()
	require.NoError(t, err, "Reload should not return error")
	require.Len(t, reloadedJobs, 1, "Should have one job after reload")

	reloadedJob := reloadedJobs[0]
	assert.Equal(t, "send_message", reloadedJob.Tool, "Tool should persist after reload")
	assert.NotEmpty(t, reloadedJob.Payload, "Payload should persist after reload")
	assert.Equal(t, "telegram:123456", reloadedJob.SessionID, "SessionID should persist after reload")

	if msg, ok := reloadedJob.Payload["message"].(string); ok {
		assert.Equal(t, "Test reminder", msg, "Message should persist after reload")
	} else {
		t.Error("Reloaded payload message should be a string")
	}

	// Test 4: Test recurring job fields preservation
	payload2 := map[string]string{"message": "Recurring test"}
	payloadJSON2, _ := json.Marshal(payload2)

	args2 := map[string]interface{}{
		"action":     "add_recurring",
		"schedule":   "0 0 * * * *",
		"tool":       "agent",
		"payload":    string(payloadJSON2),
		"session_id": "telegram:789012",
	}

	argsJSON2, _ := json.Marshal(args2)
	result2, err := tool.Execute(string(argsJSON2))
	require.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result2, "Recurring job added successfully", "Result should contain success message")

	// Load jobs again and verify both jobs
	jobsAfter2, err := storage.Load()
	require.NoError(t, err, "Load should not return error")
	require.Len(t, jobsAfter2, 2, "Should have two jobs")

	// Find the recurring job
	var recurringJob cron.StorageJob
	for _, j := range jobsAfter2 {
		if j.Type == "recurring" {
			recurringJob = j
			break
		}
	}

	assert.Equal(t, "agent", recurringJob.Tool, "Recurring job tool should be agent")
	assert.NotEmpty(t, recurringJob.Payload, "Recurring job payload should not be empty")
	assert.Equal(t, "telegram:789012", recurringJob.SessionID, "Recurring job SessionID should match")

	if msg, ok := recurringJob.Payload["message"].(string); ok {
		assert.Equal(t, "Recurring test", msg, "Recurring job message should match")
	} else {
		t.Error("Recurring job payload message should be a string")
	}
}
