package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
)

// TestAssemble tests assembling all bootstrap files
func TestAssemble(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	testFiles := map[string]string{
		BootstrapIdentity: "# Identity\n\nFirst",
		BootstrapAgents:   "# Agents\n\nSecond",
		BootstrapSoul:     "# Soul\n\nThird",
		BootstrapUser:     "# User\n\nFourth",
		BootstrapTools:    "# Tools\n\nFifth",
	}

	for name, content := range testFiles {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	loader := NewBootstrapLoader(ws, cfg, nil)

	assembled, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	if !strings.Contains(assembled, "First\n\n---\n\n# Agents\n\nSecond") {
		t.Errorf("files not assembled in priority order. Assembled: %q", assembled)
	}

	if !strings.Contains(assembled, "---") {
		t.Error("separator not present in assembled content")
	}

	for _, keyword := range []string{"First", "Second", "Third", "Fourth", "Fifth"} {
		if !strings.Contains(assembled, keyword) {
			t.Errorf("keyword %s not found in assembled content", keyword)
		}
	}
}

// TestAssembleTruncation tests that content is truncated when exceeding maxChars
func TestAssembleTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir, BootstrapMaxChars: 100}
	ws := New(cfg)

	largeContent := strings.Repeat("This is a very long line. ", 100)
	testFile := filepath.Join(tmpDir, BootstrapIdentity)
	if err := os.WriteFile(testFile, []byte(largeContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var warnings []string
	loggerFunc := func(format string, args ...interface{}) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}

	loader := NewBootstrapLoader(ws, cfg, loggerFunc)

	assembled, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	if len(assembled) != 100 {
		t.Errorf("Assemble() returned %d characters, want 100", len(assembled))
	}

	if len(warnings) == 0 {
		t.Error("expected truncation warning, got none")
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "truncated") {
			found = true
			break
		}
	}
	if !found {
		t.Error("truncation warning not found in logs")
	}
}

// TestAssembleNoLimit tests that content is not truncated when maxChars is 0
func TestAssembleNoLimit(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir, BootstrapMaxChars: 0}
	ws := New(cfg)

	testContent := "# Test\n\nContent"
	testFile := filepath.Join(tmpDir, BootstrapIdentity)
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	loader := NewBootstrapLoader(ws, cfg, nil)

	assembled, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	if assembled != testContent {
		t.Errorf("Assemble() = %v, want %v", assembled, testContent)
	}
}

// TestAssembleEmptyFiles tests assembling when some files are empty
func TestAssembleEmptyFiles(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	testFiles := map[string]string{
		BootstrapIdentity: "# Identity",
		BootstrapAgents:   "", // Empty file
		BootstrapSoul:     "# Soul",
		BootstrapUser:     "", // Empty file
	}

	for name, content := range testFiles {
		filePath := filepath.Join(tmpDir, name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	loader := NewBootstrapLoader(ws, cfg, nil)

	assembled, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	if !strings.Contains(assembled, "# Identity") {
		t.Error("Identity not found in assembled content")
	}
	if !strings.Contains(assembled, "# Soul") {
		t.Error("Soul not found in assembled content")
	}

	sepCount := strings.Count(assembled, "---")
	if sepCount < 1 {
		t.Error("expected at least 1 separator, got 0")
	}
}

// TestAssembleMalformedMarkdown tests handling of malformed markdown
func TestAssembleMalformedMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	malformedContent := "# No newline# Another header\n\nContent{{}}{}"
	testFile := filepath.Join(tmpDir, BootstrapIdentity)
	if err := os.WriteFile(testFile, []byte(malformedContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	loader := NewBootstrapLoader(ws, cfg, nil)

	assembled, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	if !strings.Contains(assembled, "# No newline") {
		t.Error("malformed content not in assembled output")
	}
	if !strings.Contains(assembled, "# Another header") {
		t.Error("malformed content not in assembled output")
	}
}
