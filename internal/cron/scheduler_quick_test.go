package cron

import (
	"context"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchedulerOneshotAlreadyExecutedQuick(t *testing.T) {
	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)
	messageBus := bus.New(100, 10, log)
	workerPool := &mockWorkerPool{}
	storage := NewStorage(tempDir, log)
	scheduler := NewScheduler(log, messageBus, workerPool, storage)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = scheduler.Start(ctx)
	require.NoError(t, err)
	now := time.Now()
	past := now.Add(-1 * time.Minute)

	// Test 1: Add job with Executed=true
	job := Job{
		ID:        "oneshot-already-executed",
		Type:      JobTypeOneshot,
		UserID:    "user-1",
		ExecuteAt: &past,
		Executed:  true,
	}
	jobID, err := scheduler.AddJob(job)
	require.NoError(t, err)
	assert.NotEmpty(t, jobID)

	// Force check by calling checkAndExecuteOneshots directly
	scheduler.checkAndExecuteOneshots(time.Now())

	// Should not have been submitted
	assert.Empty(t, workerPool.submittedTasks, "Oneshot job with Executed=true should not be submitted")

	// Test 2: Add job with Executed=false
	job2 := Job{
		ID:        "oneshot-not-executed",
		Type:      JobTypeOneshot,
		UserID:    "user-1",
		ExecuteAt: &past,
		Executed:  false,
	}
	jobID2, err := scheduler.AddJob(job2)
	require.NoError(t, err)
	assert.NotEmpty(t, jobID2)

	// Force check by calling checkAndExecuteOneshots directly
	scheduler.checkAndExecuteOneshots(time.Now())

	// Should have been submitted
	assert.Len(t, workerPool.submittedTasks, 1, "Oneshot job with Executed=false should be submitted")

	err = scheduler.Stop()
	assert.NoError(t, err)
}
