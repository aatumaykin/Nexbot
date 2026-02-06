package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
)

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

// TestPriorityOrder tests that files are loaded in correct priority order
func TestPriorityOrder(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

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

	markers := []string{"FIRST", "SECOND", "THIRD", "FOURTH", "FIFTH"}
	positions := make(map[string]int)
	for _, marker := range markers {
		pos := strings.Index(assembled, marker)
		if pos == -1 {
			t.Fatalf("marker %s not found in assembled content", marker)
		}
		positions[marker] = pos
	}

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

// TestIntegrationBootstrapLoader tests integration with Workspace
func TestIntegrationBootstrapLoader(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir, BootstrapMaxChars: 5000}
	ws := New(cfg)

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

	files, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(files) != len(testFiles) {
		t.Errorf("Load() returned %d files, want %d", len(files), len(testFiles))
	}

	assembled, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	if strings.Contains(assembled, "{{") {
		t.Error("template variables not substituted")
	}

	for _, keyword := range []string{"Identity", "Agents", "Soul"} {
		if !strings.Contains(assembled, keyword) {
			t.Errorf("keyword %s not found in assembled content", keyword)
		}
	}

	if !strings.Contains(assembled, "---") {
		t.Error("separator not present in assembled content")
	}
}
