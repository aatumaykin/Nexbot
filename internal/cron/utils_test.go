// Package cron provides unit tests for utility functions.
package cron

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadJobs tests loading jobs from an existing file.
func TestLoadJobs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test jobs file
	jobsData := map[string]Job{
		"job1": {
			ID:       "job1",
			Command:  "test command",
			Schedule: "0 * * * *",
			Type:     JobTypeRecurring,
		},
		"job2": {
			ID:       "job2",
			Command:  "another command",
			Schedule: "0 0 * * *",
			Type:     JobTypeOneshot,
		},
	}

	data, _ := json.MarshalIndent(jobsData, "", "  ")
	jobsFile := filepath.Join(tmpDir, "jobs.json")
	if err := os.WriteFile(jobsFile, data, 0644); err != nil {
		t.Fatalf("Failed to write jobs file: %v", err)
	}

	// Load jobs
	jobs, err := LoadJobs(tmpDir)
	if err != nil {
		t.Fatalf("LoadJobs failed: %v", err)
	}

	// Check that jobs are loaded
	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
	}

	// Check job1
	if job, ok := jobs["job1"]; ok {
		if job.Command != "test command" {
			t.Errorf("job1 command = %q, want 'test command'", job.Command)
		}
		if job.Type != JobTypeRecurring {
			t.Errorf("job1 type = %q, want 'recurring'", job.Type)
		}
	} else {
		t.Error("job1 not found in loaded jobs")
	}
}

// TestLoadJobsNotExists tests loading jobs from a non-existent file.
func TestLoadJobsNotExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Try to load from a directory where jobs.json doesn't exist
	jobs, err := LoadJobs(tmpDir)
	if err != nil {
		t.Errorf("LoadJobs should return empty map, not error: %v", err)
	}

	// Check that an empty map is returned
	if jobs == nil {
		t.Error("Expected empty map, got nil")
	}

	if len(jobs) != 0 {
		t.Errorf("Expected 0 jobs, got %d", len(jobs))
	}
}

// TestLoadJobsInvalidJSON tests loading jobs from a file with invalid JSON.
func TestLoadJobsInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with invalid JSON
	jobsFile := filepath.Join(tmpDir, "jobs.json")
	if err := os.WriteFile(jobsFile, []byte("{invalid json}"), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	// Try to load
	jobs, err := LoadJobs(tmpDir)
	if err == nil {
		t.Error("LoadJobs should return error for invalid JSON")
	}

	// Check that nil is returned on error (as per implementation)
	if jobs != nil {
		t.Error("Expected nil on error, got a map")
	}
}

// TestLoadJobsNilJSON tests loading jobs from a file with null JSON.
func TestLoadJobsNilJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with null JSON
	jobsFile := filepath.Join(tmpDir, "jobs.json")
	if err := os.WriteFile(jobsFile, []byte("null"), 0644); err != nil {
		t.Fatalf("Failed to write null JSON file: %v", err)
	}

	// Load jobs
	jobs, err := LoadJobs(tmpDir)
	if err != nil {
		t.Errorf("LoadJobs should handle null JSON, got error: %v", err)
	}

	// Check that an empty map is returned
	if jobs == nil {
		t.Error("Expected empty map for null JSON, got nil")
	}

	if len(jobs) != 0 {
		t.Errorf("Expected 0 jobs for null JSON, got %d", len(jobs))
	}
}

// TestSaveJobs tests saving jobs to a file.
func TestSaveJobs(t *testing.T) {
	tmpDir := t.TempDir()

	// Prepare data to save
	jobs := map[string]Job{
		"job1": {
			ID:       "job1",
			Command:  "test command",
			Schedule: "0 * * * *",
			Type:     JobTypeRecurring,
		},
	}

	// Save jobs
	err := SaveJobs(tmpDir, jobs)
	if err != nil {
		t.Fatalf("SaveJobs failed: %v", err)
	}

	// Check that the file exists
	jobsFile := filepath.Join(tmpDir, "jobs.json")
	if _, err := os.Stat(jobsFile); os.IsNotExist(err) {
		t.Error("Jobs file was not created")
	}

	// Read and verify contents
	data, err := os.ReadFile(jobsFile)
	if err != nil {
		t.Fatalf("Failed to read jobs file: %v", err)
	}

	var loadedJobs map[string]Job
	err = json.Unmarshal(data, &loadedJobs)
	if err != nil {
		t.Errorf("Failed to unmarshal saved jobs: %v", err)
	}

	if len(loadedJobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(loadedJobs))
	}

	if loadedJobs["job1"].Command != "test command" {
		t.Errorf("Expected command 'test command', got '%s'", loadedJobs["job1"].Command)
	}
}

// TestSaveJobsEmpty tests saving an empty job map.
func TestSaveJobsEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// Save empty jobs
	jobs := make(map[string]Job)
	err := SaveJobs(tmpDir, jobs)
	if err != nil {
		t.Fatalf("SaveJobs failed for empty map: %v", err)
	}

	// Check that the file exists
	jobsFile := filepath.Join(tmpDir, "jobs.json")
	if _, err := os.Stat(jobsFile); os.IsNotExist(err) {
		t.Error("Jobs file was not created for empty map")
	}

	// Read and verify contents
	data, err := os.ReadFile(jobsFile)
	if err != nil {
		t.Fatalf("Failed to read jobs file: %v", err)
	}

	var loadedJobs map[string]Job
	err = json.Unmarshal(data, &loadedJobs)
	if err != nil {
		t.Errorf("Failed to unmarshal saved jobs: %v", err)
	}

	if len(loadedJobs) != 0 {
		t.Errorf("Expected 0 jobs, got %d", len(loadedJobs))
	}
}

// TestGenerateJobID tests generating a unique job ID.
func TestGenerateJobID(t *testing.T) {
	// Generate a job ID
	jobID := GenerateJobID()

	// Check that the ID is not empty
	if jobID == "" {
		t.Error("Generated ID is empty")
	}

	// Check that the ID follows the expected format (job_<uuid>)
	if len(jobID) < 40 {
		t.Errorf("Generated ID is too short: %s", jobID)
	}

	// The ID should start with "job_"
	if jobID[:4] != "job_" {
		t.Errorf("Generated ID doesn't start with 'job_': %s", jobID)
	}

	// The rest should be a UUID (36 chars with hyphens)
	uuidPart := jobID[4:]
	if len(uuidPart) != 36 {
		t.Errorf("UUID part is not 36 characters: %s (len=%d)", uuidPart, len(uuidPart))
	}
}

// TestGenerateJobIDMultiple tests that GenerateJobID generates unique IDs.
func TestGenerateJobIDMultiple(t *testing.T) {
	// Generate multiple IDs
	id1 := GenerateJobID()
	id2 := GenerateJobID()

	// IDs should be different since we use UUID
	if id1 == id2 {
		t.Errorf("Generated IDs should be different: %s == %s", id1, id2)
	}
}

// TestLoadSaveRoundTrip tests saving and then loading jobs.
func TestLoadSaveRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	// Prepare original jobs
	originalJobs := map[string]Job{
		"job1": {
			ID:       "job1",
			Command:  "test command",
			Schedule: "0 * * * *",
			Type:     JobTypeRecurring,
		},
		"job2": {
			ID:       "job2",
			Command:  "another command",
			Schedule: "0 0 * * *",
			Type:     JobTypeOneshot,
		},
	}

	// Save jobs
	if err := SaveJobs(tmpDir, originalJobs); err != nil {
		t.Fatalf("SaveJobs failed: %v", err)
	}

	// Load jobs
	loadedJobs, err := LoadJobs(tmpDir)
	if err != nil {
		t.Fatalf("LoadJobs failed: %v", err)
	}

	// Check that all jobs are loaded
	if len(loadedJobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(loadedJobs))
	}

	// Check that job1 is preserved
	if job, ok := loadedJobs["job1"]; ok {
		if job.ID != originalJobs["job1"].ID {
			t.Errorf("job1 ID = %q, want %q", job.ID, originalJobs["job1"].ID)
		}
		if job.Command != originalJobs["job1"].Command {
			t.Errorf("job1 Command = %q, want %q", job.Command, originalJobs["job1"].Command)
		}
	} else {
		t.Error("job1 not found in loaded jobs")
	}

	// Check that job2 is preserved
	if job, ok := loadedJobs["job2"]; ok {
		if job.ID != originalJobs["job2"].ID {
			t.Errorf("job2 ID = %q, want %q", job.ID, originalJobs["job2"].ID)
		}
		if job.Command != originalJobs["job2"].Command {
			t.Errorf("job2 Command = %q, want %q", job.Command, originalJobs["job2"].Command)
		}
	} else {
		t.Error("job2 not found in loaded jobs")
	}
}
