package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/config"
)

// TestNewBootstrapLoader tests the constructor
func TestNewBootstrapLoader(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir, BootstrapMaxChars: 10000}
	ws := New(cfg)

	loader := NewBootstrapLoader(ws, cfg, nil)

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

	loader := NewBootstrapLoader(ws, cfg, nil)

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

	loader := NewBootstrapLoader(ws, cfg, nil)

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

	loader := NewBootstrapLoader(ws, cfg, nil)

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
	loggerFunc := func(format string, args ...interface{}) {
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

// TestAssemble tests assembling all bootstrap files
func TestAssemble(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	// Create test bootstrap files in priority order
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

	// Check that files are assembled in priority order
	if !strings.Contains(assembled, "First\n\n---\n\n# Agents\n\nSecond") {
		t.Errorf("files not assembled in priority order. Assembled: %q", assembled)
	}

	// Check that separator is present
	if !strings.Contains(assembled, "---") {
		t.Error("separator not present in assembled content")
	}

	// Check that all files are present
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

	// Create large test file
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

	// Check that content was truncated
	if len(assembled) != 100 {
		t.Errorf("Assemble() returned %d characters, want 100", len(assembled))
	}

	// Check that warning was logged
	if len(warnings) == 0 {
		t.Error("expected truncation warning, got none")
	}

	// Check that warning contains truncation info
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

	// Create test file
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

	// Check that content was not truncated
	if assembled != testContent {
		t.Errorf("Assemble() = %v, want %v", assembled, testContent)
	}
}

// TestAssembleEmptyFiles tests assembling when some files are empty
func TestAssembleEmptyFiles(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	// Create test files with empty content
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

	// Check that non-empty files are present
	if !strings.Contains(assembled, "# Identity") {
		t.Error("Identity not found in assembled content")
	}
	if !strings.Contains(assembled, "# Soul") {
		t.Error("Soul not found in assembled content")
	}

	// Check that separators are still present even with empty files
	// Empty files should just add extra separators
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

	// Create test file with malformed markdown
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

	// Malformed markdown should still be assembled (loader doesn't validate markdown)
	if !strings.Contains(assembled, "# No newline") {
		t.Error("malformed content not in assembled output")
	}
	if !strings.Contains(assembled, "# Another header") {
		t.Error("malformed content not in assembled output")
	}
}

// TestSubstituteVariables tests template variable substitution
func TestSubstituteVariables(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	loader := NewBootstrapLoader(ws, cfg, nil)

	tests := []struct {
		name  string
		input string
		check func(string) bool
	}{
		{
			name:  "CURRENT_TIME substitution",
			input: "Time: {{CURRENT_TIME}}",
			check: func(s string) bool {
				return !strings.Contains(s, "{{CURRENT_TIME}}") && strings.Contains(s, ":")
			},
		},
		{
			name:  "CURRENT_DATE substitution",
			input: "Date: {{CURRENT_DATE}}",
			check: func(s string) bool {
				return !strings.Contains(s, "{{CURRENT_DATE}}") && strings.Contains(s, "-")
			},
		},
		{
			name:  "WORKSPACE_PATH substitution",
			input: "Path: {{WORKSPACE_PATH}}",
			check: func(s string) bool {
				return !strings.Contains(s, "{{WORKSPACE_PATH}}") && strings.Contains(s, tmpDir)
			},
		},
		{
			name:  "multiple substitutions",
			input: "{{CURRENT_TIME}} {{CURRENT_DATE}} {{WORKSPACE_PATH}}",
			check: func(s string) bool {
				return !strings.Contains(s, "{{") && strings.Contains(s, ":") && strings.Contains(s, "-") && strings.Contains(s, tmpDir)
			},
		},
		{
			name:  "no substitutions",
			input: "Plain text with no variables",
			check: func(s string) bool {
				return s == "Plain text with no variables"
			},
		},
		{
			name:  "incomplete template (missing closing braces)",
			input: "Text with {{CURRENT_TIME",
			check: func(s string) bool {
				// Incomplete template should not be substituted
				return strings.Contains(s, "{{CURRENT_TIME")
			},
		},
		{
			name:  "unknown template variable",
			input: "Text with {{UNKNOWN_VAR}}",
			check: func(s string) bool {
				// Unknown variable should not be substituted
				return strings.Contains(s, "{{UNKNOWN_VAR}}")
			},
		},
		{
			name:  "partial substitution (some variables known, some unknown)",
			input: "{{CURRENT_TIME}} and {{UNKNOWN}}",
			check: func(s string) bool {
				// Known variable should be substituted, unknown should remain
				return !strings.Contains(s, "{{CURRENT_TIME}}") && strings.Contains(s, "{{UNKNOWN}}") && strings.Contains(s, ":")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.substituteVariables(tt.input)
			if !tt.check(result) {
				t.Errorf("substituteVariables() = %v, check failed", result)
			}
		})
	}
}

// TestSetMaxChars tests setting maxChars
func TestSetMaxChars(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	loader := NewBootstrapLoader(ws, cfg, nil)

	loader.SetMaxChars(5000)

	if loader.maxChars != 5000 {
		t.Errorf("maxChars = %d, want 5000", loader.maxChars)
	}

	if loader.GetMaxChars() != 5000 {
		t.Errorf("GetMaxChars() = %d, want 5000", loader.GetMaxChars())
	}
}

// TestIntegrationBootstrapLoader tests integration with Workspace
func TestIntegrationBootstrapLoader(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir, BootstrapMaxChars: 5000}
	ws := New(cfg)

	// Create bootstrap files
	testFiles := map[string]string{
		BootstrapIdentity: "# Identity\n\nTime: {{CURRENT_TIME}}",
		BootstrapAgents:   "# Agents\n\nPath: {{WORKSPACE_PATH}}",
		BootstrapSoul:     "# Soul\n\nDate: {{CURRENT_DATE}}",
	}

	for name, content := range testFiles {
		filePath := filepath.Join(tmpDir, name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	loader := NewBootstrapLoader(ws, cfg, nil)

	// Test Load
	files, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(files) != len(testFiles) {
		t.Errorf("Load() returned %d files, want %d", len(files), len(testFiles))
	}

	// Test Assemble
	assembled, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	// Check that variables were substituted
	if strings.Contains(assembled, "{{") {
		t.Error("template variables not substituted")
	}

	// Check that all files are present
	for _, keyword := range []string{"Identity", "Agents", "Soul"} {
		if !strings.Contains(assembled, keyword) {
			t.Errorf("keyword %s not found in assembled content", keyword)
		}
	}

	// Check that separator is present
	if !strings.Contains(assembled, "---") {
		t.Error("separator not present in assembled content")
	}
}

// TestPriorityOrder tests that files are loaded in correct priority order
func TestPriorityOrder(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	// Create all bootstrap files with markers
	files := []struct {
		name   string
		marker string
	}{
		{BootstrapIdentity, "FIRST"},
		{BootstrapAgents, "SECOND"},
		{BootstrapSoul, "THIRD"},
		{BootstrapUser, "FOURTH"},
		{BootstrapTools, "FIFTH"},
	}

	for _, f := range files {
		content := fmt.Sprintf("# %s\n\n%s", f.name, f.marker)
		filePath := filepath.Join(tmpDir, f.name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", f.name, err)
		}
	}

	loader := NewBootstrapLoader(ws, cfg, nil)
	assembled, err := loader.Assemble()

	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	// Find positions of markers
	markers := []string{"FIRST", "SECOND", "THIRD", "FOURTH", "FIFTH"}
	positions := make(map[string]int)
	for _, marker := range markers {
		pos := strings.Index(assembled, marker)
		if pos == -1 {
			t.Fatalf("marker %s not found in assembled content", marker)
		}
		positions[marker] = pos
	}

	// Check that positions are in correct order (increasing)
	if positions["FIRST"] > positions["SECOND"] {
		t.Error("FIRST should appear before SECOND")
	}
	if positions["SECOND"] > positions["THIRD"] {
		t.Error("SECOND should appear before THIRD")
	}
	if positions["THIRD"] > positions["FOURTH"] {
		t.Error("THIRD should appear before FOURTH")
	}
	if positions["FOURTH"] > positions["FIFTH"] {
		t.Error("FOURTH should appear before FIFTH")
	}
}
