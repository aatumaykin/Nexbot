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

// mockWorkerPool is a mock implementation of WorkerPool for testing
type mockWorkerPool struct {
	submittedTasks []Task
}

func (m *mockWorkerPool) Submit(task Task) {
	m.submittedTasks = append(m.submittedTasks, task)
}

func TestSchedulerOneshotExecution(t *testing.T) {
	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)
	messageBus := bus.New(100, log)
	workerPool := &mockWorkerPool{}
	storage := NewStorage(tempDir, log)
	scheduler := NewScheduler(log, messageBus, workerPool, storage)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = scheduler.Start(ctx)
	require.NoError(t, err)
	now := time.Now()
	past := now.Add(-1 * time.Minute)
	job := Job{
		ID:        "oneshot-1",
		Type:      JobTypeOneshot,
		Command:   "test command",
		UserID:    "user-1",
		ExecuteAt: &past,
		Executed:  false,
	}
	jobID, err := scheduler.AddJob(job)
	require.NoError(t, err)
	assert.NotEmpty(t, jobID)

	// Force check by calling checkAndExecuteOneshots directly
	scheduler.checkAndExecuteOneshots(time.Now())

	assert.Len(t, workerPool.submittedTasks, 1)
	err = scheduler.Stop()
	assert.NoError(t, err)
}

func TestSchedulerOneshotAlreadyExecuted(t *testing.T) {
	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)
	messageBus := bus.New(100, log)
	workerPool := &mockWorkerPool{}
	storage := NewStorage(tempDir, log)
	scheduler := NewScheduler(log, messageBus, workerPool, storage)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = scheduler.Start(ctx)
	require.NoError(t, err)
	now := time.Now()
	past := now.Add(-1 * time.Minute)
	job := Job{
		ID:        "oneshot-2",
		Type:      JobTypeOneshot,
		Command:   "test command",
		UserID:    "user-1",
		ExecuteAt: &past,
		Executed:  true,
	}
	jobID, err := scheduler.AddJob(job)
	require.NoError(t, err)
	assert.NotEmpty(t, jobID)
	time.Sleep(2 * time.Second)
	assert.Empty(t, workerPool.submittedTasks)
	err = scheduler.Stop()
	assert.NoError(t, err)
}

func TestSchedulerCleanupExecuted(t *testing.T) {
	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)
	messageBus := bus.New(100, log)
	workerPool := &mockWorkerPool{}
	storage := NewStorage(tempDir, log)
	scheduler := NewScheduler(log, messageBus, workerPool, storage)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = scheduler.Start(ctx)
	require.NoError(t, err)
	now := time.Now()
	past := now.Add(-1 * time.Minute)
	future := now.Add(1 * time.Hour)
	job1 := Job{
		ID:        "oneshot-new",
		Type:      JobTypeOneshot,
		Command:   "keep this",
		UserID:    "user-1",
		ExecuteAt: &future,
		Executed:  false,
	}
	_, _ = scheduler.AddJob(job1)
	job2 := Job{
		ID:         "oneshot-executed",
		Type:       JobTypeOneshot,
		Command:    "remove this",
		UserID:     "user-1",
		ExecuteAt:  &past,
		Executed:   true,
		ExecutedAt: &past,
	}
	_, _ = scheduler.AddJob(job2)
	jobs := scheduler.ListJobs()
	assert.Len(t, jobs, 2)
	scheduler.CleanupExecutedOneshots()
	jobs = scheduler.ListJobs()
	assert.Len(t, jobs, 1)
	assert.Equal(t, "oneshot-new", jobs[0].ID)
	assert.Equal(t, "keep this", jobs[0].Command)
	remainingJobs, err := storage.Load()
	require.NoError(t, err)
	assert.Len(t, remainingJobs, 1)
	assert.Equal(t, "oneshot-new", remainingJobs[0].ID)
	err = scheduler.Stop()
	assert.NoError(t, err)
}

func TestSchedulerStorageIntegration(t *testing.T) {
	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)
	messageBus := bus.New(100, log)
	storage := NewStorage(tempDir, log)
	scheduler := NewScheduler(log, messageBus, nil, storage)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = scheduler.Start(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, scheduler.Stop())
	})
	now := time.Now()
	job1 := Job{
		ID:       "recurring-1",
		Type:     JobTypeRecurring,
		Schedule: "* * * * * *",
		Command:  "recurring command",
		UserID:   "user-1",
	}
	jobID1, err := scheduler.AddJob(job1)
	require.NoError(t, err)
	assert.NotEmpty(t, jobID1)
	future := now.Add(1 * time.Hour)
	job2 := Job{
		ID:        "oneshot-1",
		Type:      JobTypeOneshot,
		Command:   "oneshot command",
		UserID:    "user-1",
		ExecuteAt: &future,
		Executed:  false,
	}
	jobID2, err := scheduler.AddJob(job2)
	require.NoError(t, err)
	assert.NotEmpty(t, jobID2)
	jobs := scheduler.ListJobs()
	assert.Len(t, jobs, 2)
	storageJobs, err := storage.Load()
	require.NoError(t, err)
	assert.Len(t, storageJobs, 2)
	foundRecurring := false
	foundOneshot := false
	for _, job := range storageJobs {
		if job.ID == "recurring-1" && job.Type == string(JobTypeRecurring) {
			foundRecurring = true
		}
		if job.ID == "oneshot-1" && job.Type == string(JobTypeOneshot) {
			foundOneshot = true
		}
	}
	assert.True(t, foundRecurring, "recurring job type preserved")
	assert.True(t, foundOneshot, "oneshot job type preserved")
	var oneshotJob *StorageJob
	for _, job := range storageJobs {
		if job.ID == "oneshot-1" {
			oneshotJob = &job
			break
		}
	}
	require.NotNil(t, oneshotJob)
	assert.NotNil(t, oneshotJob.ExecuteAt, "ExecuteAt should be set")
	assert.Equal(t, future.Format(time.RFC3339), oneshotJob.ExecuteAt.Format(time.RFC3339))
	var recurringJob *StorageJob
	for _, job := range storageJobs {
		if job.ID == "recurring-1" {
			recurringJob = &job
			break
		}
	}
	require.NotNil(t, recurringJob)
	_ = recurringJob.ExecuteAt
}

func TestSchedulerOneshotNotExecutedTwice(t *testing.T) {
	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)
	messageBus := bus.New(100, log)
	workerPool := &mockWorkerPool{}
	storage := NewStorage(tempDir, log)
	scheduler := NewScheduler(log, messageBus, workerPool, storage)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = scheduler.Start(ctx)
	require.NoError(t, err)
	now := time.Now()
	past := now.Add(-1 * time.Minute)

	// Add oneshot job with tool
	job := Job{
		ID:        "oneshot-tool-test",
		Type:      JobTypeOneshot,
		Tool:      "send_message",
		Payload:   map[string]any{"message": "test"},
		SessionID: "telegram:12345",
		UserID:    "user-1",
		ExecuteAt: &past,
		Executed:  false,
	}

	jobID, err := scheduler.AddJob(job)
	require.NoError(t, err)
	assert.NotEmpty(t, jobID)

	// Wait for first execution (ticker runs every minute, but we check immediately)
	time.Sleep(100 * time.Millisecond)

	// First check - should execute
	scheduler.checkAndExecuteOneshots(time.Now())
	assert.Len(t, workerPool.submittedTasks, 1, "Oneshot job should execute once")

	// Second check - should NOT execute again
	scheduler.checkAndExecuteOneshots(time.Now())
	assert.Len(t, workerPool.submittedTasks, 1, "Oneshot job should not execute twice")

	// Verify job is marked as executed in storage (reload from storage)
	storageJobs, err := storage.Load()
	require.NoError(t, err)
	require.Len(t, storageJobs, 1)
	storedJob := storageJobs[0]
	assert.True(t, storedJob.Executed, "Job should be marked as executed")

	// Verify schedule and command are normalized
	assert.Empty(t, storedJob.Schedule, "Oneshot job should not have schedule")
	assert.Empty(t, storedJob.Command, "Job with tool should not have command")

	err = scheduler.Stop()
	assert.NoError(t, err)
}
