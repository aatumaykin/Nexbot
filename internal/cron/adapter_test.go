package cron

import (
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchedulerAdapter_ConvertJobToAgentJob verifies conversion from cron.Job to agent.Job
func TestSchedulerAdapter_ConvertJobToAgentJob(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    Job
		expected agent.Job
	}{
		{
			name: "recurring job",
			input: Job{
				ID:         "job_123",
				Type:       JobTypeRecurring,
				Schedule:   "0 * * * *",
				UserID:     "user_1",
				Metadata:   map[string]string{"key": "value"},
				Executed:   false,
				ExecutedAt: nil,
			},
			expected: agent.Job{
				ID:         "job_123",
				Type:       "recurring",
				Schedule:   "0 * * * *",
				ExecuteAt:  nil,
				UserID:     "user_1",
				Metadata:   map[string]string{"key": "value"},
				Executed:   false,
				ExecutedAt: nil,
			},
		},
		{
			name: "oneshot job",
			input: Job{
				ID:         "job_456",
				Type:       JobTypeOneshot,
				Schedule:   "0 0 0 1 1 *",
				ExecuteAt:  &now,
				UserID:     "user_2",
				Metadata:   nil,
				Executed:   true,
				ExecutedAt: &now,
			},
			expected: agent.Job{
				ID:         "job_456",
				Type:       "oneshot",
				Schedule:   "0 0 0 1 1 *",
				ExecuteAt:  &now,
				UserID:     "user_2",
				Metadata:   nil,
				Executed:   true,
				ExecutedAt: &now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert cron job to agent job
			agentJob := convertToAgentJob(tt.input)

			// Verify conversion
			assert.Equal(t, tt.expected.ID, agentJob.ID)
			assert.Equal(t, tt.expected.Type, agentJob.Type)
			assert.Equal(t, tt.expected.Schedule, agentJob.Schedule)
			assert.Equal(t, tt.expected.UserID, agentJob.UserID)
			assert.Equal(t, tt.expected.Executed, agentJob.Executed)

			if tt.expected.ExecuteAt != nil {
				require.NotNil(t, agentJob.ExecuteAt)
				assert.True(t, tt.expected.ExecuteAt.Equal(*agentJob.ExecuteAt))
			} else {
				assert.Nil(t, agentJob.ExecuteAt)
			}

			if tt.expected.ExecutedAt != nil {
				require.NotNil(t, agentJob.ExecutedAt)
				assert.True(t, tt.expected.ExecutedAt.Equal(*agentJob.ExecutedAt))
			} else {
				assert.Nil(t, agentJob.ExecutedAt)
			}

			assert.Equal(t, tt.expected.Metadata, agentJob.Metadata)
		})
	}
}

// TestSchedulerAdapter_ConvertStorageJobToAgentJob verifies conversion from cron.StorageJob to agent.Job
func TestSchedulerAdapter_ConvertStorageJobToAgentJob(t *testing.T) {
	tests := []struct {
		name     string
		input    StorageJob
		expected agent.Job
	}{
		{
			name: "storage job with metadata",
			input: StorageJob{
				ID:         "storage_job_1",
				Type:       "recurring",
				Schedule:   "0 * * * *",
				UserID:     "user_1",
				Metadata:   map[string]string{"created_by": "cron_tool"},
				Executed:   false,
				ExecutedAt: nil,
			},
			expected: agent.Job{
				ID:         "storage_job_1",
				Type:       "recurring",
				Schedule:   "0 * * * *",
				ExecuteAt:  nil,
				UserID:     "user_1",
				Metadata:   map[string]string{"created_by": "cron_tool"},
				Executed:   false,
				ExecutedAt: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert storage job to agent job
			agentJob := convertStorageToAgentJob(tt.input)

			// Verify conversion
			assert.Equal(t, tt.expected.ID, agentJob.ID)
			assert.Equal(t, tt.expected.Type, agentJob.Type)
			assert.Equal(t, tt.expected.Schedule, agentJob.Schedule)
			assert.Equal(t, tt.expected.UserID, agentJob.UserID)
			assert.Equal(t, tt.expected.Executed, agentJob.Executed)
			assert.Equal(t, tt.expected.Metadata, agentJob.Metadata)
		})
	}
}

// TestSchedulerAdapter_ConvertAgentJobToCronJob verifies conversion from agent.Job to cron.Job
func TestSchedulerAdapter_ConvertAgentJobToCronJob(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    agent.Job
		expected Job
	}{
		{
			name: "agent recurring job",
			input: agent.Job{
				ID:         "agent_job_1",
				Type:       "recurring",
				Schedule:   "0 * * * *",
				ExecuteAt:  nil,
				UserID:     "user_1",
				Metadata:   map[string]string{"key": "value"},
				Executed:   false,
				ExecutedAt: nil,
			},
			expected: Job{
				ID:         "agent_job_1",
				Type:       JobTypeRecurring,
				Schedule:   "0 * * * *",
				ExecuteAt:  nil,
				UserID:     "user_1",
				Metadata:   map[string]string{"key": "value"},
				Executed:   false,
				ExecutedAt: nil,
			},
		},
		{
			name: "agent oneshot job",
			input: agent.Job{
				ID:         "agent_job_2",
				Type:       "oneshot",
				Schedule:   "0 0 0 1 1 *",
				ExecuteAt:  &now,
				UserID:     "user_2",
				Metadata:   nil,
				Executed:   true,
				ExecutedAt: &now,
			},
			expected: Job{
				ID:         "agent_job_2",
				Type:       JobTypeOneshot,
				Schedule:   "0 0 0 1 1 *",
				ExecuteAt:  &now,
				UserID:     "user_2",
				Metadata:   nil,
				Executed:   true,
				ExecutedAt: &now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert agent job to cron job
			cronJob := convertAgentToCronJob(tt.input)

			// Verify conversion
			assert.Equal(t, tt.expected.ID, cronJob.ID)
			assert.Equal(t, tt.expected.Type, cronJob.Type)
			assert.Equal(t, tt.expected.Schedule, cronJob.Schedule)
			assert.Equal(t, tt.expected.UserID, cronJob.UserID)
			assert.Equal(t, tt.expected.Executed, cronJob.Executed)

			if tt.expected.ExecuteAt != nil {
				require.NotNil(t, cronJob.ExecuteAt)
				assert.True(t, tt.expected.ExecuteAt.Equal(*cronJob.ExecuteAt))
			} else {
				assert.Nil(t, cronJob.ExecuteAt)
			}

			if tt.expected.ExecutedAt != nil {
				require.NotNil(t, cronJob.ExecutedAt)
				assert.True(t, tt.expected.ExecutedAt.Equal(*cronJob.ExecutedAt))
			} else {
				assert.Nil(t, cronJob.ExecutedAt)
			}

			assert.Equal(t, tt.expected.Metadata, cronJob.Metadata)
		})
	}
}

// TestSchedulerAdapter_ConvertAgentJobToStorageJob verifies conversion from agent.Job to cron.StorageJob
func TestSchedulerAdapter_ConvertAgentJobToStorageJob(t *testing.T) {
	_ = time.Now() // Avoid unused variable warning

	tests := []struct {
		name     string
		input    agent.Job
		expected StorageJob
	}{
		{
			name: "agent job to storage",
			input: agent.Job{
				ID:         "agent_job_1",
				Type:       "recurring",
				Schedule:   "0 * * * *",
				ExecuteAt:  nil,
				UserID:     "user_1",
				Metadata:   map[string]string{"key": "value"},
				Executed:   false,
				ExecutedAt: nil,
			},
			expected: StorageJob{
				ID:         "agent_job_1",
				Type:       "recurring",
				Schedule:   "0 * * * *",
				ExecuteAt:  nil,
				UserID:     "user_1",
				Metadata:   map[string]string{"key": "value"},
				Executed:   false,
				ExecutedAt: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert agent job to storage job
			storageJob := convertAgentToStorageJob(tt.input)

			// Verify conversion
			assert.Equal(t, tt.expected.ID, storageJob.ID)
			assert.Equal(t, tt.expected.Type, storageJob.Type)
			assert.Equal(t, tt.expected.Schedule, storageJob.Schedule)
			assert.Equal(t, tt.expected.UserID, storageJob.UserID)
			assert.Equal(t, tt.expected.Executed, storageJob.Executed)
			assert.Equal(t, tt.expected.Metadata, storageJob.Metadata)
		})
	}
}

// Helper functions for conversions (these are used internally by the adapter)

func convertToAgentJob(job Job) agent.Job {
	return agent.Job{
		ID:         job.ID,
		Type:       string(job.Type),
		Schedule:   job.Schedule,
		ExecuteAt:  job.ExecuteAt,
		UserID:     job.UserID,
		Metadata:   job.Metadata,
		Executed:   job.Executed,
		ExecutedAt: job.ExecutedAt,
	}
}

func convertStorageToAgentJob(job StorageJob) agent.Job {
	return agent.Job{
		ID:         job.ID,
		Type:       job.Type,
		Schedule:   job.Schedule,
		ExecuteAt:  job.ExecuteAt,
		UserID:     job.UserID,
		Metadata:   job.Metadata,
		Executed:   job.Executed,
		ExecutedAt: job.ExecutedAt,
	}
}

func convertAgentToCronJob(job agent.Job) Job {
	return Job{
		ID:         job.ID,
		Type:       JobType(job.Type),
		Schedule:   job.Schedule,
		ExecuteAt:  job.ExecuteAt,
		UserID:     job.UserID,
		Metadata:   job.Metadata,
		Executed:   job.Executed,
		ExecutedAt: job.ExecutedAt,
	}
}

func convertAgentToStorageJob(job agent.Job) StorageJob {
	return StorageJob{
		ID:         job.ID,
		Type:       job.Type,
		Schedule:   job.Schedule,
		ExecuteAt:  job.ExecuteAt,
		UserID:     job.UserID,
		Metadata:   job.Metadata,
		Executed:   job.Executed,
		ExecutedAt: job.ExecutedAt,
	}
}
