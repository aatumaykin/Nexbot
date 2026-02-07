package cron

import (
	"context"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduler_JobExecution(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	workerPool := &mockWorkerPool{}
	scheduler := NewScheduler(log, msgBus, workerPool, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	// Add a job that runs every 30 seconds
	job := Job{
		ID:       "test-job",
		Schedule: "*/30 * * * * *", // Every 30 seconds
		Command:  "cron test command",
		UserID:   "cron-user",
		Metadata: map[string]string{
			"test_key": "test_value",
		},
	}

	_, err = scheduler.AddJob(job)
	require.NoError(t, err)

	// Wait for job to execute (schedule is every 30 seconds, wait 35s to ensure it executes once)
	time.Sleep(35 * time.Second)

	// Verify job was submitted to worker pool
	assert.Len(t, workerPool.submittedTasks, 1, "Job should be submitted to worker pool")

	// Verify job details
	task := workerPool.submittedTasks[0]
	assert.Equal(t, "cron", task.Type)

	// Verify command is in payload (deprecated field)
	payload, ok := task.Payload.(CronTaskPayload)
	require.True(t, ok)
	assert.Equal(t, "cron test command", payload.Command)
	assert.Equal(t, "cron-user", payload.Metadata["user_id"])
	assert.Equal(t, "test-job", payload.Metadata["cron_job_id"])
	assert.Equal(t, job.Schedule, payload.Metadata["cron_schedule"])
	assert.Equal(t, "test_value", payload.Metadata["test_key"])
}

func TestScheduler_JobExecutionWithMetadata(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	workerPool := &mockWorkerPool{}
	scheduler := NewScheduler(log, msgBus, workerPool, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	job := Job{
		ID:       "metadata-job",
		Schedule: "*/30 * * * * *", // Every 30 seconds
		Command:  "test",
		UserID:   "test-user",
		Metadata: map[string]string{
			"env":  "production",
			"team": "devops",
		},
	}

	_, err = scheduler.AddJob(job)
	require.NoError(t, err)

	// Wait for job to execute
	time.Sleep(35 * time.Second)

	// Verify job was submitted to worker pool
	assert.Len(t, workerPool.submittedTasks, 1, "Job should be submitted to worker pool")

	// Verify metadata is preserved
	task := workerPool.submittedTasks[0]
	payload, ok := task.Payload.(CronTaskPayload)
	require.True(t, ok)
	assert.Equal(t, "production", payload.Metadata["env"])
	assert.Equal(t, "devops", payload.Metadata["team"])
}
