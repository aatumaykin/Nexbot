package workspace

import (
	"os"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
)

// TestWorkspaceBootstrapIntegration tests the complete workflow from workspace initialization to bootstrap assembly.
func TestWorkspaceBootstrapIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir, BootstrapMaxChars: 5000}
	ws := New(cfg)

	if err := ws.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() failed: %v", err)
	}

	bootstrapContent := map[string]string{
		BootstrapIdentity: `# Core Identity\n\nNexbot is a lightweight personal AI assistant.\n\n## Current Time\n\n{{CURRENT_TIME}} {{CURRENT_DATE}}\n\n## Workspace\n\nWorkspace path: {{WORKSPACE_PATH}}`,
		BootstrapAgents:   `# Agent Instructions\n\nYou are helpful and friendly.\n\n## Tools\n\nYou can use: file, shell, messaging`,
		BootstrapSoul:     `# Personality\n\nBe concise and accurate.`,
		BootstrapUser:     `# User Profile\n\nName: Test User\nTimezone: UTC`,
	}

	for filename, content := range bootstrapContent {
		filePath := ws.Subpath(filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create bootstrap file %s: %v", filename, err)
		}
	}

	loader := NewBootstrapLoader(ws, cfg, nil, "")
	files, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(files) != len(bootstrapContent) {
		t.Errorf("Load() returned %d files, want %d", len(files), len(bootstrapContent))
	}

	assembled, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	if strings.Contains(assembled, "{{") {
		t.Error("template variables not substituted, found {{ in output")
	}

	if !strings.Contains(assembled, tmpDir) {
		t.Error("workspace path not substituted in output")
	}

	requiredKeywords := []string{"Core Identity", "Agent Instructions", "Personality", "User Profile"}
	for _, keyword := range requiredKeywords {
		if !strings.Contains(assembled, keyword) {
			t.Errorf("keyword %s not found in assembled content", keyword)
		}
	}
}
