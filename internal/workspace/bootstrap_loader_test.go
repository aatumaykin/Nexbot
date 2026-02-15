package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/config"
)

// TestNewBootstrapLoader tests constructor
func TestNewBootstrapLoader(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir, BootstrapMaxChars: 10000}
	ws := New(cfg)

	loader := NewBootstrapLoader(ws, cfg, nil, "")

	if loader.workspace != ws {
		t.Error("workspace not set correctly")
	}

	if loader.maxChars != 10000 {
		t.Errorf("maxChars = %d, want 10000", loader.maxChars)
	}

	if loader.GetMaxChars() != 10000 {
		t.Errorf("GetMaxChars() = %d, want 10000", loader.GetMaxChars())
	}
}

// TestNewBootstrapLoaderDefault tests constructor with default maxChars
func TestNewBootstrapLoaderDefault(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir} // BootstrapMaxChars = 0
	ws := New(cfg)

	loader := NewBootstrapLoader(ws, cfg, nil, "")

	if loader.maxChars != 20000 { // Default should be 20000
		t.Errorf("maxChars = %d, want 20000 (default)", loader.maxChars)
	}
}

// TestLoadFile tests loading individual bootstrap files
func TestLoadFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	// Create test file
	testFile := filepath.Join(tmpDir, "TEST.md")
	testContent := "# Test\n\n{{CURRENT_TIME}}"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	loader := NewBootstrapLoader(ws, cfg, nil, "")

	content, err := loader.LoadFile("TEST.md")
	if err != nil {
		t.Fatalf("LoadFile() failed: %v", err)
	}

	// Check that template variable was substituted
	if strings.Contains(content, "{{CURRENT_TIME}}") {
		t.Error("template variable was not substituted")
	}

	// Check that time was substituted (not empty)
	now := time.Now()
	timeStr := now.Format("15:04:05")
	if !strings.Contains(content, timeStr) && !strings.Contains(content, ":") {
		t.Error("time was not substituted correctly")
	}
}

// TestLoadFileErrors tests error cases for LoadFile
func TestLoadFileErrors(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	loader := NewBootstrapLoader(ws, cfg, nil, "")

	tests := []struct {
		name        string
		filename    string
		setup       func(t *testing.T) func()
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty filename",
			filename:    "",
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "file not found",
			filename:    "NONEXISTENT.md",
			wantErr:     true,
			errContains: "no such file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				cleanup := tt.setup(t)
				defer cleanup()
			}

			_, err := loader.LoadFile(tt.filename)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(strings.ToLower(err.Error()), tt.errContains) {
					t.Errorf("LoadFile() error = %v, want containing %v", err.Error(), tt.errContains)
				}
			}
		})
	}
}

// TestSetMaxChars tests setting maxChars
func TestSetMaxChars(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	loader := NewBootstrapLoader(ws, cfg, nil, "")

	loader.SetMaxChars(5000)

	if loader.maxChars != 5000 {
		t.Errorf("maxChars = %d, want 5000", loader.maxChars)
	}

	if loader.GetMaxChars() != 5000 {
		t.Errorf("GetMaxChars() = %d, want 5000", loader.GetMaxChars())
	}
}
