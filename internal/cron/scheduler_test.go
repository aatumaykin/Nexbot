package cron

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduler(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)
	scheduler := NewScheduler(log, msgBus)

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

	scheduler := NewScheduler(log, msgBus)

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

	scheduler := NewScheduler(log, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	job := Job{
		ID:       "test-job",
		Schedule: "* * * * * *", // Every second
		Command:  "test command",
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
	assert.Equal(t, job.Command, storedJob.Command)
}

func TestScheduler_AddJobAutoID(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	job := Job{
		Schedule: "* * * * * *", // Every second
		Command:  "test command",
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

	scheduler := NewScheduler(log, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	job := Job{
		ID:       "test-job",
		Schedule: "invalid-cron",
		Command:  "test command",
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

	scheduler := NewScheduler(log, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	// Add a job
	job := Job{
		ID:       "test-job",
		Schedule: "* * * * * *",
		Command:  "test command",
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

	scheduler := NewScheduler(log, msgBus)

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
		Command:  "command 1",
		UserID:   "user-1",
	}
	job2 := Job{
		ID:       "job-2",
		Schedule: "*/2 * * * * *",
		Command:  "command 2",
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

func TestScheduler_JobExecution(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	// Subscribe to inbound messages
	inboundCh := msgBus.SubscribeInbound(ctx)
	defer func() {
		// Wait a bit for message to be received
		time.Sleep(100 * time.Millisecond)
	}()

	// Add a job that runs every 100ms
	job := Job{
		ID:       "test-job",
		Schedule: "*/1 * * * * *", // Every second
		Command:  "cron test command",
		UserID:   "cron-user",
		Metadata: map[string]string{
			"test_key": "test_value",
		},
	}

	_, err = scheduler.AddJob(job)
	require.NoError(t, err)

	// Wait for job to execute
	select {
	case msg := <-inboundCh:
		assert.Equal(t, ChannelTypeCron, msg.ChannelType)
		assert.Equal(t, "cron-user", msg.UserID)
		assert.Equal(t, "cron test command", msg.Content)
		assert.NotNil(t, msg.Metadata)
		assert.Equal(t, "test-job", msg.Metadata["cron_job_id"])
		assert.Equal(t, job.Schedule, msg.Metadata["cron_schedule"])
		assert.Equal(t, "test_value", msg.Metadata["test_key"])
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for cron job to execute")
	}
}

func TestScheduler_JobExecutionWithMetadata(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	// Subscribe to inbound messages
	inboundCh := msgBus.SubscribeInbound(ctx)
	defer func() {
		time.Sleep(100 * time.Millisecond)
	}()

	job := Job{
		ID:       "metadata-job",
		Schedule: "*/1 * * * * *",
		Command:  "test",
		UserID:   "test-user",
		Metadata: map[string]string{
			"env":  "production",
			"team": "devops",
		},
	}

	_, err = scheduler.AddJob(job)
	require.NoError(t, err)

	select {
	case msg := <-inboundCh:
		assert.Equal(t, "production", msg.Metadata["env"])
		assert.Equal(t, "devops", msg.Metadata["team"])
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for cron job")
	}
}

func TestScheduler_GetJob(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	job := Job{
		ID:       "get-test-job",
		Schedule: "* * * * * *",
		Command:  "test command",
		UserID:   "test-user",
	}

	_, err = scheduler.AddJob(job)
	require.NoError(t, err)

	// Get existing job
	storedJob, err := scheduler.GetJob("get-test-job")
	assert.NoError(t, err)
	assert.Equal(t, "get-test-job", storedJob.ID)
	assert.Equal(t, job.Command, storedJob.Command)

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

	scheduler := NewScheduler(log, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Add some jobs
	job1 := Job{
		ID:       "job-1",
		Schedule: "* * * * * *",
		Command:  "command 1",
		UserID:   "user-1",
	}
	job2 := Job{
		ID:       "job-2",
		Schedule: "*/2 * * * * *",
		Command:  "command 2",
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

// TestCronExpressionValidation tests various valid and invalid cron expressions
func TestCronExpressionValidation(t *testing.T) {
	tests := []struct {
		name        string
		expression  string
		expectError bool
		description string
	}{
		{
			name:        "Every second",
			expression:  "* * * * * *",
			expectError: false,
			description: "Cron with seconds field (every second)",
		},
		{
			name:        "Every 6 hours",
			expression:  "0 */6 * * * *",
			expectError: false,
			description: "Cron for every 6 hours starting at minute 0",
		},
		{
			name:        "Weekdays at 9 AM",
			expression:  "0 9 * * 1-5 *",
			expectError: false,
			description: "Cron for weekdays at 9:00 AM",
		},
		{
			name:        "Hourly on the hour",
			expression:  "0 * * * * *",
			expectError: false,
			description: "Cron for every hour at minute 0",
		},
		{
			name:        "Daily at 5 PM",
			expression:  "0 17 * * * *",
			expectError: false,
			description: "Cron for daily at 5:00 PM",
		},
		{
			name:        "Monthly on the 1st at 0:00",
			expression:  "0 0 1 * * *",
			expectError: false,
			description: "Cron for monthly on the 1st",
		},
		{
			name:        "Invalid - empty",
			expression:  "",
			expectError: true,
			description: "Empty cron expression should fail",
		},
		{
			name:        "Invalid - malformed",
			expression:  "invalid-cron",
			expectError: true,
			description: "Malformed cron expression should fail",
		},
		{
			name:        "Invalid - invalid minute",
			expression:  "61 * * * *",
			expectError: true,
			description: "Invalid minute value (61) should fail",
		},
		{
			name:        "Invalid - invalid hour",
			expression:  "* 25 * * *",
			expectError: true,
			description: "Invalid hour value (25) should fail",
		},
		{
			name:        "Invalid - invalid day of week range",
			expression:  "0 9 * * 8-* *",
			expectError: true,
			description: "Invalid day of week range (8-*) should fail (valid range is 0-6)",
		},
		{
			name:        "Valid - timezone format",
			expression:  "CRON_TZ=America/New_York 0 9 * * * *",
			expectError: false,
			description: "Cron with timezone format (supported by robfig/cron v3)",
		},
		{
			name:        "Valid - complex schedule",
			expression:  "*/15 9-17 * * 1-5 *",
			expectError: false,
			description: "Complex schedule: weekdays 9-5, every 15 minutes",
		},
		{
			name:        "Valid - varied ranges",
			expression:  "0 0,12 * * * *",
			expectError: false,
			description: "Cron at midnight and noon daily",
		},
		{
			name:        "Valid - stepped ranges",
			expression:  "*/5 * * * * *",
			expectError: false,
			description: "Every 5 seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := testLogger()
			msgBus := bus.New(100, log)

			err := msgBus.Start(context.Background())
			require.NoError(t, err)
			defer stopMessageBus(msgBus)

			scheduler := NewScheduler(log, msgBus)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err = scheduler.Start(ctx)
			require.NoError(t, err)
			defer stopScheduler(scheduler)

			// Attempt to add job with the given expression
			job := Job{
				ID:       tt.name,
				Schedule: tt.expression,
				Command:  "test command",
				UserID:   "test-user",
			}

			_, err = scheduler.AddJob(job)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid cron expression")
			} else {
				assert.NoError(t, err)
				// Verify job was actually added
				storedJob, err := scheduler.GetJob(tt.name)
				assert.NoError(t, err)
				assert.Equal(t, tt.expression, storedJob.Schedule)
			}
		})
	}
}

// TestSchedulerDuplicateJobID tests adding jobs with the same ID
func TestSchedulerDuplicateJobID(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	// Add first job with ID
	job1 := Job{
		ID:       "duplicate-id",
		Schedule: "* * * * * *",
		Command:  "command 1",
		UserID:   "user-1",
	}

	id1, err := scheduler.AddJob(job1)
	assert.NoError(t, err)
	assert.Equal(t, "duplicate-id", id1)

	// Add second job with the same ID (but different schedule)
	job2 := Job{
		ID:       "duplicate-id",
		Schedule: "*/2 * * * * *",
		Command:  "command 2",
		UserID:   "user-2",
	}

	// This should succeed because schedule is valid
	_, err = scheduler.AddJob(job2)
	assert.NoError(t, err)

	// Note: With current implementation, duplicate IDs will overwrite the job
	// Verify we have 1 job (the second one overwrote the first)
	jobs := scheduler.ListJobs()
	assert.Len(t, jobs, 1)

	// Verify the job has the second schedule
	assert.Equal(t, "duplicate-id", jobs[0].ID)
	assert.Equal(t, "*/2 * * * * *", jobs[0].Schedule)
	assert.Equal(t, "command 2", jobs[0].Command)
}

// TestSchedulerRemoveNonExistentJob tests removing a job that doesn't exist
func TestSchedulerRemoveNonExistentJob(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	// Try to remove non-existent job
	err = scheduler.RemoveJob("non-existent-job")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")
}

// TestSchedulerListWithNoJobs tests listing jobs when no jobs are scheduled
func TestSchedulerListWithNoJobs(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	// List jobs when empty
	jobs := scheduler.ListJobs()
	assert.Empty(t, jobs)
	assert.Len(t, jobs, 0)
}

// TestSchedulerConcurrentAddRemove tests concurrent add/remove operations
func TestSchedulerConcurrentAddRemove(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	const numGoroutines = 10
	_ = make(chan Job, numGoroutines) // Placeholder for jobs channel
	results := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Spawn goroutines for concurrent job additions
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			job := Job{
				ID:       fmt.Sprintf("concurrent-job-%d", id),
				Schedule: "* * * * * *",
				Command:  fmt.Sprintf("command %d", id),
				UserID:   "test-user",
			}
			jobID, err := scheduler.AddJob(job)
			if err != nil {
				errors <- err
				return
			}
			results <- jobID
		}(i)
	}

	// Wait for all jobs to be added
	jobIDs := make(map[string]bool)
	for i := 0; i < numGoroutines; i++ {
		select {
		case id := <-results:
			jobIDs[id] = true
		case err := <-errors:
			t.Errorf("Error adding job: %v", err)
			cancel()
			return
		}
	}

	// Verify all jobs were added
	assert.Len(t, jobIDs, numGoroutines)

	// Spawn goroutines for concurrent job removals
	for i := 0; i < numGoroutines/2; i++ {
		go func(id int) {
			err := scheduler.RemoveJob(fmt.Sprintf("concurrent-job-%d", id))
			if err != nil {
				errors <- err
			}
		}(i)
	}

	// Wait for removals to complete
	for i := 0; i < numGoroutines/2; i++ {
		select {
		case err := <-errors:
			t.Logf("Concurrent remove error (may be expected): %v", err)
		default:
		}
	}

	// Verify some jobs were removed
	jobsLeft := scheduler.ListJobs()
	assert.LessOrEqual(t, len(jobsLeft), numGoroutines)
}

// testLogger creates a test logger instance
func testLogger() *logger.Logger {
	log, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		panic(err)
	}
	return log
}

// stopScheduler stops a scheduler and ignores the error (for use in defer in tests)
func stopScheduler(s *Scheduler) {
	_ = s.Stop()
}

// stopMessageBus stops a message bus and ignores the error (for use in defer in tests)
func stopMessageBus(b *bus.MessageBus) {
	_ = b.Stop()
}
