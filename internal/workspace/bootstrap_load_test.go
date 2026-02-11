package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
)

// TestLoad tests loading all bootstrap files
func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	// Create test bootstrap files
	testFiles := map[string]string{
		BootstrapIdentity: "# Identity\n\n{{CURRENT_TIME}}",
		BootstrapAgents:   "# Agents\n\n{{WORKSPACE_PATH}}",
		BootstrapSoul:     "# Soul\n\n{{CURRENT_DATE}}",
	}

	for name, content := range testFiles {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	loader := NewBootstrapLoader(ws, cfg, nil)

	files, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check that all files were loaded
	if len(files) != len(testFiles) {
		t.Errorf("Load() returned %d files, want %d", len(files), len(testFiles))
	}

	// Check that template variables were substituted
	for name, content := range files {
		if strings.Contains(content, "{{") {
			t.Errorf("template variables not substituted in %s: %s", name, content)
		}
	}
}

// TestLoadMissingFiles tests loading when some files are missing
func TestLoadMissingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	// Create only some files
	testFiles := map[string]string{
		BootstrapIdentity: "# Identity",
		BootstrapAgents:   "# Agents",
	}

	for name, content := range testFiles {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	// Track warnings
	var warnings []string
	loggerFunc := func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}

	loader := NewBootstrapLoader(ws, cfg, loggerFunc)

	files, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check that only existing files were loaded
	if len(files) != len(testFiles) {
		t.Errorf("Load() returned %d files, want %d", len(files), len(testFiles))
	}

	// Check that warnings were logged for missing files
	if len(warnings) == 0 {
		t.Error("expected warnings for missing files, got none")
	}
}
