package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
)

// TestLoadFromMainSubdirectory tests loading bootstrap files from main/ subdirectory
func TestLoadFromMainSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	// Create main subdirectory
	mainDir := filepath.Join(tmpDir, "main")
	if err := os.Mkdir(mainDir, 0755); err != nil {
		t.Fatalf("failed to create main directory: %v", err)
	}

	testFiles := map[string]string{
		BootstrapIdentity: "# Main Identity\n\nFirst",
		BootstrapAgents:   "# Main Agents\n\nSecond",
		BootstrapUser:     "# Main User\n\nFourth",
		BootstrapTools:    "# Main Tools\n\nFifth",
	}

	for name, content := range testFiles {
		filePath := filepath.Join(mainDir, name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	loader := NewBootstrapLoader(ws, cfg, nil, "main")

	// Load individual file
	identity, err := loader.LoadFile(BootstrapIdentity)
	if err != nil {
		t.Fatalf("LoadFile(BootstrapIdentity) failed: %v", err)
	}

	if !strings.Contains(identity, "Main Identity") {
		t.Errorf("expected 'Main Identity' in content, got: %s", identity)
	}

	// Load all files
	files, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(files) != 4 {
		t.Errorf("expected 4 files, got %d", len(files))
	}

	for name := range testFiles {
		if _, ok := files[name]; !ok {
			t.Errorf("expected file %s not found", name)
		}
	}
}

// TestLoadFromSubagentSubdirectory tests loading bootstrap files from subagent/ subdirectory
func TestLoadFromSubagentSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	// Create subagent subdirectory
	subagentDir := filepath.Join(tmpDir, "subagent")
	if err := os.Mkdir(subagentDir, 0755); err != nil {
		t.Fatalf("failed to create subagent directory: %v", err)
	}

	testFiles := map[string]string{
		BootstrapAgents: "# Subagent Agents\n\nSecond",
	}

	for name, content := range testFiles {
		filePath := filepath.Join(subagentDir, name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	loader := NewBootstrapLoader(ws, cfg, nil, "subagent")

	// Load individual file
	agents, err := loader.LoadFile(BootstrapAgents)
	if err != nil {
		t.Fatalf("LoadFile(BootstrapAgents) failed: %v", err)
	}

	if !strings.Contains(agents, "Subagent Agents") {
		t.Errorf("expected 'Subagent Agents' in content, got: %s", agents)
	}

	// Load all files
	files, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}

	if _, ok := files[BootstrapAgents]; !ok {
		t.Errorf("expected file %s not found", BootstrapAgents)
	}
}

// TestAssembleFromMainSubdirectory tests assembling bootstrap files from main/ subdirectory
func TestAssembleFromMainSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	// Create main subdirectory
	mainDir := filepath.Join(tmpDir, "main")
	if err := os.Mkdir(mainDir, 0755); err != nil {
		t.Fatalf("failed to create main directory: %v", err)
	}

	testFiles := map[string]string{
		BootstrapIdentity: "# Main Identity\n\nFirst",
		BootstrapAgents:   "# Main Agents\n\nSecond",
		BootstrapUser:     "# Main User\n\nFourth",
		BootstrapTools:    "# Main Tools\n\nFifth",
	}

	for name, content := range testFiles {
		filePath := filepath.Join(mainDir, name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	loader := NewBootstrapLoader(ws, cfg, nil, "main")

	assembled, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	if !strings.Contains(assembled, "Main Identity") {
		t.Errorf("expected 'Main Identity' in assembled content, got: %s", assembled)
	}

	if !strings.Contains(assembled, "Main Agents") {
		t.Errorf("expected 'Main Agents' in assembled content, got: %s", assembled)
	}

	if !strings.Contains(assembled, "---") {
		t.Error("separator not present in assembled content")
	}

	for _, keyword := range []string{"First", "Second", "Fourth", "Fifth"} {
		if !strings.Contains(assembled, keyword) {
			t.Errorf("keyword %s not found in assembled content", keyword)
		}
	}
}

// TestAssembleFromSubagentSubdirectory tests assembling bootstrap files from subagent/ subdirectory
func TestAssembleFromSubagentSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	// Create subagent subdirectory
	subagentDir := filepath.Join(tmpDir, "subagent")
	if err := os.Mkdir(subagentDir, 0755); err != nil {
		t.Fatalf("failed to create subagent directory: %v", err)
	}

	testFiles := map[string]string{
		BootstrapAgents: "# Subagent Agents\n\nContent",
	}

	for name, content := range testFiles {
		filePath := filepath.Join(subagentDir, name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	loader := NewBootstrapLoader(ws, cfg, nil, "subagent")

	assembled, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	if !strings.Contains(assembled, "Subagent Agents") {
		t.Errorf("expected 'Subagent Agents' in assembled content, got: %s", assembled)
	}

	if !strings.Contains(assembled, "Content") {
		t.Errorf("expected 'Content' in assembled content, got: %s", assembled)
	}
}

// TestMainSubdirectoryFallbackToEmbedDefaults tests fallback to embedded defaults when main/ files are missing
func TestMainSubdirectoryFallbackToEmbedDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	// Create empty main subdirectory
	mainDir := filepath.Join(tmpDir, "main")
	if err := os.Mkdir(mainDir, 0755); err != nil {
		t.Fatalf("failed to create main directory: %v", err)
	}

	var warnings []string
	loggerFunc := func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}

	loader := NewBootstrapLoader(ws, cfg, loggerFunc, "main")

	assembled, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	if assembled == "" {
		t.Error("expected non-empty assembled content from embedded defaults")
	}

	// Check that we got warnings about using defaults
	if len(warnings) == 0 {
		t.Error("expected warnings about using embedded defaults")
	}

	// Verify we got the embedded default content
	if !strings.Contains(assembled, "Soul") && !strings.Contains(assembled, "helpful") {
		t.Error("expected embedded default IDENTITY content")
	}
}

// TestSubagentSubdirectoryFallbackToEmbedDefaults tests fallback to embedded defaults when subagent/ files are missing
func TestSubagentSubdirectoryFallbackToEmbedDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	// Create empty subagent subdirectory
	subagentDir := filepath.Join(tmpDir, "subagent")
	if err := os.Mkdir(subagentDir, 0755); err != nil {
		t.Fatalf("failed to create subagent directory: %v", err)
	}

	var warnings []string
	loggerFunc := func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}

	loader := NewBootstrapLoader(ws, cfg, loggerFunc, "subagent")

	assembled, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	if assembled == "" {
		t.Error("expected non-empty assembled content from embedded defaults")
	}

	// Check that we got warnings about using defaults
	if len(warnings) == 0 {
		t.Error("expected warnings about using embedded defaults")
	}

	// Verify we got the embedded default content (subagent only has AGENTS.md)
	if !strings.Contains(assembled, "agent") && !strings.Contains(assembled, "Agent") {
		t.Error("expected embedded default AGENTS content")
	}
}

// TestEmptySubdirectoryLoadsFromRoot tests that empty subdirectory loads from workspace root
func TestEmptySubdirectoryLoadsFromRoot(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.WorkspaceConfig{Path: tmpDir}
	ws := New(cfg)

	testFiles := map[string]string{
		BootstrapIdentity: "# Root Identity",
	}

	for name, content := range testFiles {
		filePath := filepath.Join(tmpDir, name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	loader := NewBootstrapLoader(ws, cfg, nil, "")

	identity, err := loader.LoadFile(BootstrapIdentity)
	if err != nil {
		t.Fatalf("LoadFile(BootstrapIdentity) failed: %v", err)
	}

	if !strings.Contains(identity, "Root Identity") {
		t.Errorf("expected 'Root Identity' in content, got: %s", identity)
	}
}
