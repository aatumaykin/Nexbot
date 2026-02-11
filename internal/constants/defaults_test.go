package constants

import (
	"regexp"
	"strings"
	"testing"
)

func TestDefaultConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "DefaultVersion",
			value: DefaultVersion,
		},
		{
			name:  "DefaultBuildTime",
			value: DefaultBuildTime,
		},
		{
			name:  "DefaultGitCommit",
			value: DefaultGitCommit,
		},
		{
			name:  "DefaultGoVersion",
			value: DefaultGoVersion,
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

func TestDefaultVersion(t *testing.T) {
	// Test that version follows semantic versioning pattern
	// Format: MAJOR.MINOR.PATCH-PRERELEASE (e.g., "0.1.0-dev")
	versionPattern := `^\d+\.\d+\.\d+(-[\w\.-]+)?$`
	matched, err := regexp.MatchString(versionPattern, DefaultVersion)
	if err != nil {
		t.Fatalf("Failed to compile version pattern: %v", err)
	}

	if !matched {
		t.Errorf("DefaultVersion = %s, should follow semantic versioning pattern (e.g., 0.1.0-dev)", DefaultVersion)
	}

	// Test specific value
	if DefaultVersion != "0.1.0-dev" {
		t.Errorf("DefaultVersion = %s, want '0.1.0-dev'", DefaultVersion)
	}
}

func TestDefaultBuildTime(t *testing.T) {
	if DefaultBuildTime != "unknown" {
		t.Errorf("DefaultBuildTime = %s, want 'unknown'", DefaultBuildTime)
	}
}

func TestDefaultGitCommit(t *testing.T) {
	if DefaultGitCommit != "unknown" {
		t.Errorf("DefaultGitCommit = %s, want 'unknown'", DefaultGitCommit)
	}
}

func TestDefaultGoVersion(t *testing.T) {
	if DefaultGoVersion != "unknown" {
		t.Errorf("DefaultGoVersion = %s, want 'unknown'", DefaultGoVersion)
	}
}

func TestDefaultUnknownValues(t *testing.T) {
	// Test that placeholder values are consistent
	unknownValues := []string{DefaultBuildTime, DefaultGitCommit, DefaultGoVersion}
	for i, value := range unknownValues {
		if value != "unknown" {
			t.Errorf("Default constant at index %d = %s, want 'unknown'", i, value)
		}
	}
}

func TestDefaultVersionFormat(t *testing.T) {
	// Test that version has major, minor, and patch parts
	parts := strings.Split(DefaultVersion, ".")
	if len(parts) < 3 {
		t.Errorf("DefaultVersion should have at least 3 parts separated by dots, got %d parts", len(parts))
	}

	// Check that first three parts are numeric
	for i := range 3 {
		if !isNumeric(parts[i]) {
			// Remove any pre-release suffix before checking
			cleanPart := strings.Split(parts[i], "-")[0]
			if !isNumeric(cleanPart) {
				t.Errorf("Version part %d should be numeric, got: %s", i, parts[i])
			}
		}
	}
}

func TestDefaultVersionDevSuffix(t *testing.T) {
	// Test that the version has a dev suffix indicating development build
	if !strings.Contains(DefaultVersion, "-") {
		t.Errorf("DefaultVersion should have a pre-release suffix, got: %s", DefaultVersion)
	}

	if !strings.Contains(DefaultVersion, "dev") {
		t.Errorf("DefaultVersion should indicate development build, got: %s", DefaultVersion)
	}
}

func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}
