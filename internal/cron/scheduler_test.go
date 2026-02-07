package cron

import (
	"context"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduler(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)
	scheduler := NewScheduler(log, msgBus, nil, nil)

	assert.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.cron)
	assert.NotNil(t, scheduler.logger)
	assert.NotNil(t, scheduler.bus)
	assert.NotNil(t, scheduler.jobs)
	assert.NotNil(t, scheduler.jobIDs)
	assert.NotNil(t, scheduler.jobEntryIDs)
}

func TestScheduler_StartStop(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)

	scheduler := NewScheduler(log, msgBus, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start scheduler
	err = scheduler.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, scheduler.IsStarted())

	// Start again should fail
	err = scheduler.Start(ctx)
	assert.Error(t, err)

	// Stop scheduler
	err = scheduler.Stop()
	assert.NoError(t, err)
	assert.False(t, scheduler.IsStarted())

	// Stop again should fail
	err = scheduler.Stop()
	assert.Error(t, err)

	err = msgBus.Stop()
	require.NoError(t, err)
}

func TestScheduler_AddJob(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	job := Job{
		ID:       "test-job",
		Schedule: "* * * * * *", // Every second
		UserID:   "test-user",
	}

	jobID, err := scheduler.AddJob(job)
	assert.NoError(t, err)
	assert.Equal(t, "test-job", jobID)

	// Verify job is stored
	storedJob, err := scheduler.GetJob("test-job")
	assert.NoError(t, err)
	assert.Equal(t, "test-job", storedJob.ID)
	assert.Equal(t, job.Schedule, storedJob.Schedule)
}

func TestScheduler_AddJobAutoID(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	job := Job{
		Schedule: "* * * * * *", // Every second
		UserID:   "test-user",
	}

	jobID, err := scheduler.AddJob(job)
	assert.NoError(t, err)
	assert.NotEmpty(t, jobID)
	assert.Contains(t, jobID, "job_")

	// Verify job is stored
	storedJob, err := scheduler.GetJob(jobID)
	assert.NoError(t, err)
	assert.Equal(t, jobID, storedJob.ID)
}

func TestScheduler_AddJobInvalidSchedule(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	job := Job{
		ID:       "test-job",
		Schedule: "invalid-cron",
		UserID:   "test-user",
	}

	_, err = scheduler.AddJob(job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron expression")
}

func TestScheduler_RemoveJob(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	// Add a job
	job := Job{
		ID:       "test-job",
		Schedule: "* * * * * *",
		UserID:   "test-user",
	}

	_, err = scheduler.AddJob(job)
	require.NoError(t, err)

	// Remove job
	err = scheduler.RemoveJob("test-job")
	assert.NoError(t, err)

	// Verify job is removed
	_, err = scheduler.GetJob("test-job")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")

	// Remove non-existent job
	err = scheduler.RemoveJob("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")
}

func TestScheduler_ListJobs(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	// Initially empty
	jobs := scheduler.ListJobs()
	assert.Empty(t, jobs)

	// Add multiple jobs
	job1 := Job{
		ID:       "job-1",
		Schedule: "* * * * * *",
		UserID:   "user-1",
	}
	job2 := Job{
		ID:       "job-2",
		Schedule: "*/2 * * * * *",
		UserID:   "user-2",
	}

	_, err = scheduler.AddJob(job1)
	require.NoError(t, err)
	_, err = scheduler.AddJob(job2)
	require.NoError(t, err)

	// List jobs
	jobs = scheduler.ListJobs()
	assert.Len(t, jobs, 2)

	jobIDs := make(map[string]bool)
	for _, job := range jobs {
		jobIDs[job.ID] = true
	}
	assert.True(t, jobIDs["job-1"])
	assert.True(t, jobIDs["job-2"])
}

func TestScheduler_GetJob(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	job := Job{
		ID:       "get-test-job",
		Schedule: "* * * * * *",
		UserID:   "test-user",
	}

	_, err = scheduler.AddJob(job)
	require.NoError(t, err)

	// Get existing job
	storedJob, err := scheduler.GetJob("get-test-job")
	assert.NoError(t, err)
	assert.Equal(t, "get-test-job", storedJob.ID)

	// Get non-existent job
	_, err = scheduler.GetJob("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")
}

func TestScheduler_GracefulShutdown(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Add some jobs
	job1 := Job{
		ID:       "job-1",
		Schedule: "* * * * * *",
		UserID:   "user-1",
	}
	job2 := Job{
		ID:       "job-2",
		Schedule: "*/2 * * * * *",
		UserID:   "user-2",
	}

	_, err = scheduler.AddJob(job1)
	require.NoError(t, err)
	_, err = scheduler.AddJob(job2)
	require.NoError(t, err)

	// Trigger graceful shutdown using Stop()
	err = scheduler.Stop()
	assert.NoError(t, err)

	// Wait a bit for shutdown to complete
	time.Sleep(100 * time.Millisecond)

	// Scheduler should be stopped
	assert.False(t, scheduler.IsStarted())
}

func TestScheduler_AddJobInvalidOneshotWithSchedule(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)
	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	now := time.Now()
	past := now.Add(-1 * time.Minute)
	job := Job{
		ID:        "test-oneshot",
		Type:      JobTypeOneshot,
		Schedule:  "* * * * * *", // Should NOT be allowed for oneshot
		ExecuteAt: &past,
		UserID:    "test-user",
	}

	_, err = scheduler.AddJob(job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "oneshot jobs cannot have schedule field")
}

func TestScheduler_AddJobNormalizeToolCommand(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)
	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	tempDir := t.TempDir()
	storage := NewStorage(tempDir, log)
	scheduler := NewScheduler(log, msgBus, nil, storage)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	now := time.Now()
	past := now.Add(-1 * time.Minute)
	job := Job{
		ID:        "test-tool",
		Type:      JobTypeOneshot,
		Tool:      "send_message",
		Payload:   map[string]any{"message": "hello"},
		SessionID: "telegram:123",
		ExecuteAt: &past,
	}

	jobID, err := scheduler.AddJob(job)
	require.NoError(t, err)

	_, err = scheduler.GetJob(jobID)
	require.NoError(t, err)
}
