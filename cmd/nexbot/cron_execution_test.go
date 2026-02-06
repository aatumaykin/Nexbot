package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCronExecutionWithMock(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Change to temp directory
	_ = os.Chdir(tempDir)

	// Create a logger and message bus
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

	// Create scheduler
	scheduler := cron.NewScheduler(log, msgBus, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer func() {
		if err := scheduler.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	// Subscribe to inbound messages to capture job execution
	inboundCh := msgBus.SubscribeInbound(ctx)

	// Add a job that will execute quickly (every 100ms)
	job := cron.Job{
		ID:       "exec-test-job",
		Schedule: "*/1 * * * * *", // Every second
		Command:  "test execution command",
		UserID:   "cli",
		Metadata: map[string]string{
			"test_key": "test_value",
		},
	}

	// Save job to persistent storage
	jobs := map[string]cron.Job{job.ID: job}
	err = cron.SaveJobs(tempDir, jobs)
	require.NoError(t, err)

	// Add job to scheduler
	jobID, err := scheduler.AddJob(job)
	require.NoError(t, err)
	assert.Equal(t, "exec-test-job", jobID)

	// Wait for job to execute
	select {
	case msg := <-inboundCh:
		assert.Equal(t, cron.ChannelTypeCron, msg.ChannelType)
		assert.Equal(t, "cli", msg.UserID)
		assert.Equal(t, "test execution command", msg.Content)
		assert.NotNil(t, msg.Metadata)
		assert.Equal(t, "exec-test-job", msg.Metadata["cron_job_id"])
		assert.Equal(t, job.Schedule, msg.Metadata["cron_schedule"])
		assert.Equal(t, "test_value", msg.Metadata["test_key"])
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for cron job to execute")
	}

	// Verify job is still in scheduler
	storedJobs := scheduler.ListJobs()
	assert.Len(t, storedJobs, 1)
}
