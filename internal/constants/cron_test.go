package constants

import (
	"fmt"
	"testing"
)

func TestCronConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "CronDefaultUserID",
			value: CronDefaultUserID,
		},
		{
			name:  "CronJobIDFormat",
			value: CronJobIDFormat,
		},
		{
			name:  "CronJobsFile",
			value: CronJobsFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}
		})
	}
}

func TestCronDefaultUserID(t *testing.T) {
	if CronDefaultUserID != "cli" {
		t.Errorf("CronDefaultUserID = %s, want 'cli'", CronDefaultUserID)
	}
}

func TestCronJobIDFormat(t *testing.T) {
	// Test that the format string is valid and uses printf-style formatting
	if CronJobIDFormat != "job_%d" {
		t.Errorf("CronJobIDFormat = %s, want 'job_%%d'", CronJobIDFormat)
	}

	// Test that the format string can be used with fmt.Sprintf
	testID := fmt.Sprintf(CronJobIDFormat, 42)
	expectedID := "job_42"

	if testID != expectedID {
		t.Errorf("fmt.Sprintf(CronJobIDFormat, 42) = %s, want %s", testID, expectedID)
	}

	// Test with different values
	for i := 0; i < 10; i++ {
		result := fmt.Sprintf(CronJobIDFormat, i)
		expected := fmt.Sprintf("job_%d", i)
		if result != expected {
			t.Errorf("fmt.Sprintf(CronJobIDFormat, %d) = %s, want %s", i, result, expected)
		}
	}
}

func TestCronJobsFile(t *testing.T) {
	if CronJobsFile != "jobs.json" {
		t.Errorf("CronJobsFile = %s, want 'jobs.json'", CronJobsFile)
	}

	// Check file extension
	if len(CronJobsFile) < 5 || CronJobsFile[len(CronJobsFile)-5:] != ".json" {
		t.Errorf("CronJobsFile should have .json extension, got: %s", CronJobsFile)
	}
}

func TestCronConsistency(t *testing.T) {
	// Test that job ID format produces valid identifiers
	// This ensures that job IDs can be used as filenames
	for i := 0; i < 100; i++ {
		jobID := fmt.Sprintf(CronJobIDFormat, i)
		if jobID == "" {
			t.Errorf("Job ID should not be empty for i=%d", i)
		}

		// Check that job ID doesn't contain invalid filename characters
		invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
		for _, char := range invalidChars {
			if len(jobID) > 0 && contains(jobID, char) {
				t.Errorf("Job ID '%s' should not contain invalid character '%s'", jobID, char)
			}
		}
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
