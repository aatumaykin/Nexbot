package version

import (
	"strings"
	"testing"
)

func TestSetInfo(t *testing.T) {
	originalVersion := Version
	originalBuildTime := BuildTime
	originalGitCommit := GitCommit
	originalGoVersion := GoVersion

	defer func() {
		Version = originalVersion
		BuildTime = originalBuildTime
		GitCommit = originalGitCommit
		GoVersion = originalGoVersion
	}()

	SetInfo("1.0.0", "2024-01-01T00:00:00Z", "abc123", "go1.21")

	if Version != "1.0.0" {
		t.Errorf("Version = %s, want 1.0.0", Version)
	}
	if BuildTime != "2024-01-01T00:00:00Z" {
		t.Errorf("BuildTime = %s, want 2024-01-01T00:00:00Z", BuildTime)
	}
	if GitCommit != "abc123" {
		t.Errorf("GitCommit = %s, want abc123", GitCommit)
	}
	if GoVersion != "go1.21" {
		t.Errorf("GoVersion = %s, want go1.21", GoVersion)
	}
}

func TestSetInfoEmptyValues(t *testing.T) {
	originalVersion := Version

	defer func() { Version = originalVersion }()

	Version = "test-version"
	SetInfo("", "", "", "")

	if Version != "test-version" {
		t.Errorf("Version should not change with empty value, got %s", Version)
	}
}

func TestFormatStartupMessage(t *testing.T) {
	originalVersion := Version
	originalBuildTime := BuildTime

	defer func() {
		Version = originalVersion
		BuildTime = originalBuildTime
	}()

	Version = "1.2.3"
	BuildTime = "2024-06-15T10:30:00Z"

	msg := FormatStartupMessage()

	if !strings.Contains(msg, "1.2.3") {
		t.Errorf("Message should contain version, got: %s", msg)
	}
	if !strings.Contains(msg, "2024-06-15T10:30:00Z") {
		t.Errorf("Message should contain build time, got: %s", msg)
	}
	if !strings.Contains(msg, "Nexbot") {
		t.Errorf("Message should contain Nexbot, got: %s", msg)
	}
}
