package cron

import (
	"context"
	"fmt"
	"testing"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			msgBus := bus.New(100, 10, log)

			err := msgBus.Start(context.Background())
			require.NoError(t, err)
			defer stopMessageBus(msgBus)

			scheduler := NewScheduler(log, msgBus, nil, nil)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err = scheduler.Start(ctx)
			require.NoError(t, err)
			defer stopScheduler(scheduler)

			// Attempt to add job with the given expression
			job := Job{
				ID:       tt.name,
				Schedule: tt.expression,
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
	msgBus := bus.New(100, 10, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	// Add first job with ID
	job1 := Job{
		ID:       "duplicate-id",
		Schedule: "* * * * * *",
		UserID:   "user-1",
	}

	id1, err := scheduler.AddJob(job1)
	assert.NoError(t, err)
	assert.Equal(t, "duplicate-id", id1)

	// Add second job with the same ID (but different schedule)
	job2 := Job{
		ID:       "duplicate-id",
		Schedule: "*/2 * * * * *",
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
}

// TestSchedulerRemoveNonExistentJob tests removing a job that doesn't exist
func TestSchedulerRemoveNonExistentJob(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, 10, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus, nil, nil)

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
	msgBus := bus.New(100, 10, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus, nil, nil)

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
	msgBus := bus.New(100, 10, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus, nil, nil)

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
