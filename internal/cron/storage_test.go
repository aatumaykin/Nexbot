package cron

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorage_Load_EmptyFile(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create storage
	storage := NewStorage(tempDir, log)

	// Load from non-existent file
	jobs, err := storage.Load()

	// Should return empty slice and no error
	assert.NoError(t, err)
	assert.NotNil(t, jobs)
	assert.Empty(t, jobs)
}

func TestStorage_Load_WithJobs(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create storage
	storage := NewStorage(tempDir, log)

	// Append test jobs
	job1 := StorageJob{
		ID:       "job-1",
		Type:     "recurring",
		Schedule: "* * * * *",
		UserID:   "user-1",
	}
	job2 := StorageJob{
		ID:       "job-2",
		Type:     "oneshot",
		Schedule: "* * * * *",
		UserID:   "user-2",
	}

	require.NoError(t, storage.Append(job1))
	require.NoError(t, storage.Append(job2))

	// Load jobs
	jobs, err := storage.Load()

	// Should return 2 jobs
	require.NoError(t, err)
	assert.Len(t, jobs, 2)

	// Verify job content
	assert.Equal(t, "job-1", jobs[0].ID)
	assert.Equal(t, "recurring", jobs[0].Type)
	assert.Equal(t, "* * * * *", jobs[0].Schedule)

	assert.Equal(t, "job-2", jobs[1].ID)
	assert.Equal(t, "oneshot", jobs[1].Type)
}

func TestStorage_Append(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create storage
	storage := NewStorage(tempDir, log)

	// Append job
	job := StorageJob{
		ID:       "test-job",
		Type:     "recurring",
		Schedule: "* * * * *",
		UserID:   "user-1",
		Metadata: map[string]string{"key": "value"},
	}

	err = storage.Append(job)

	// Should succeed
	assert.NoError(t, err)

	// Verify file exists
	jobsPath := filepath.Join(tempDir, CronSubdirectory, JobsFilename)
	_, err = os.Stat(jobsPath)
	assert.NoError(t, err)

	// Verify content
	jobs, err := storage.Load()
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "test-job", jobs[0].ID)
}

func TestStorage_Remove(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create storage
	storage := NewStorage(tempDir, log)

	// Append multiple jobs
	job1 := StorageJob{
		ID:       "job-1",
		Type:     "recurring",
		Schedule: "* * * * *",
		UserID:   "user-1",
	}
	job2 := StorageJob{
		ID:       "job-2",
		Type:     "oneshot",
		Schedule: "* * * * *",
		UserID:   "user-2",
	}
	job3 := StorageJob{
		ID:       "job-3",
		Type:     "oneshot",
		Schedule: "* * * * *",
		UserID:   "user-3",
	}

	require.NoError(t, storage.Append(job1))
	require.NoError(t, storage.Append(job2))
	require.NoError(t, storage.Append(job3))

	// Verify all jobs exist
	jobs, err := storage.Load()
	require.NoError(t, err)
	assert.Len(t, jobs, 3)

	// Remove job-2
	err = storage.Remove("job-2")
	assert.NoError(t, err)

	// Verify job-2 is removed
	jobs, err = storage.Load()
	require.NoError(t, err)
	assert.Len(t, jobs, 2)

	// Verify remaining jobs
	jobIDs := make(map[string]bool)
	for _, job := range jobs {
		jobIDs[job.ID] = true
	}
	assert.True(t, jobIDs["job-1"])
	assert.False(t, jobIDs["job-2"])
	assert.True(t, jobIDs["job-3"])
}

func TestStorage_Save(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create storage
	storage := NewStorage(tempDir, log)

	// Create jobs to save
	executedAt := time.Now().Add(time.Hour)
	jobs := []StorageJob{
		{
			ID:       "job-1",
			Type:     "recurring",
			Schedule: "* * * * *",
			UserID:   "user-1",
			Executed: false,
		},
		{
			ID:         "job-2",
			Type:       "oneshot",
			Schedule:   "* * * * *",
			UserID:     "user-2",
			Executed:   true,
			ExecutedAt: &executedAt,
		},
	}

	// Save jobs
	err = storage.Save(jobs)
	assert.NoError(t, err)

	// Verify file exists
	jobsPath := filepath.Join(tempDir, CronSubdirectory, JobsFilename)
	_, err = os.Stat(jobsPath)
	assert.NoError(t, err)

	// Verify content
	loadedJobs, err := storage.Load()
	require.NoError(t, err)
	assert.Len(t, loadedJobs, 2)

	// Verify job details
	assert.Equal(t, "job-1", loadedJobs[0].ID)
	assert.Equal(t, "recurring", loadedJobs[0].Type)
	assert.False(t, loadedJobs[0].Executed)

	assert.Equal(t, "job-2", loadedJobs[1].ID)
	assert.Equal(t, "oneshot", loadedJobs[1].Type)
	assert.True(t, loadedJobs[1].Executed)
	assert.NotNil(t, loadedJobs[1].ExecutedAt)
}

func TestStorage_RemoveExecutedOneshots(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	// Create storage
	storage := NewStorage(tempDir, log)

	// Create test jobs
	now := time.Now()
	jobs := []StorageJob{
		{
			ID:       "recurrent-1",
			Type:     "recurring",
			Schedule: "* * * * *",
		},
		{
			ID:       "recurrent-2",
			Type:     "recurring",
			Schedule: "*/5 * * * *",
		},
		{
			ID:        "oneshot-new",
			Type:      "oneshot",
			ExecuteAt: &now,
			Executed:  false,
		},
		{
			ID:         "oneshot-done",
			Type:       "oneshot",
			ExecuteAt:  &now,
			Executed:   true,
			ExecutedAt: &now,
		},
		{
			ID:         "oneshot-done-2",
			Type:       "oneshot",
			ExecuteAt:  &now,
			Executed:   true,
			ExecutedAt: &now,
		},
	}

	// Save initial jobs
	require.NoError(t, storage.Save(jobs))

	// Verify initial state
	loadedJobs, err := storage.Load()
	require.NoError(t, err)
	assert.Len(t, loadedJobs, 5)

	// Remove executed oneshots
	err = storage.RemoveExecutedOneshots()
	assert.NoError(t, err)

	// Verify final state
	loadedJobs, err = storage.Load()
	require.NoError(t, err)
	assert.Len(t, loadedJobs, 3)

	// Verify correct jobs remain
	jobIDs := make(map[string]bool)
	for _, job := range loadedJobs {
		jobIDs[job.ID] = true
	}

	// Recurring jobs should remain
	assert.True(t, jobIDs["recurrent-1"])
	assert.True(t, jobIDs["recurrent-2"])

	// Unexecuted oneshot should remain
	assert.True(t, jobIDs["oneshot-new"])

	// Executed oneshots should be removed
	assert.False(t, jobIDs["oneshot-done"])
	assert.False(t, jobIDs["oneshot-done-2"])
}

func TestStorageUpsertJobNormalizeOneshot(t *testing.T) {
	tempDir := t.TempDir()
	log := testLogger()
	storage := NewStorage(tempDir, log)

	// Add oneshot job with schedule (should be normalized)
	now := time.Now()
	job := StorageJob{
		ID:        "job-1",
		Type:      "oneshot",
		Schedule:  "* * * * *", // Should be removed by normalization
		ExecuteAt: &now,
	}

	err := storage.UpsertJob(job)
	require.NoError(t, err)

	// Load and verify schedule was removed
	jobs, err := storage.Load()
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Empty(t, jobs[0].Schedule, "Oneshot job schedule should be normalized to empty")
}
