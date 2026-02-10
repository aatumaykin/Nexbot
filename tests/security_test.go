package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/tools/file"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

// TestSymlinkAttackPrevention tests that symlink attacks are prevented
func TestSymlinkAttackPrevention(t *testing.T) {
	tempDir := t.TempDir()
	wsDir := filepath.Join(tempDir, "workspace")
	os.MkdirAll(wsDir, 0755)

	wsCfg := config.WorkspaceConfig{Path: wsDir}
	ws := workspace.New(wsCfg)
	ws.EnsureDir()

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			File: config.FileToolConfig{
				WhitelistDirs: []string{},
			},
		},
	}

	// Create a symlink inside workspace pointing outside
	symlinkPath := filepath.Join(wsDir, "evil_symlink")
	outsideFile := filepath.Join(tempDir, "outside.txt")

	// Create the target file
	os.WriteFile(outsideFile, []byte("secret data"), 0644)

	// Create symlink
	err := os.Symlink(outsideFile, symlinkPath)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Try to read via symlink - should fail
	readTool := file.NewReadFileTool(ws, cfg)
	_, err = readTool.Execute(`{"path": "evil_symlink"}`)
	if err == nil {
		t.Error("Expected error when reading symlink that escapes workspace, got nil")
	}

	expectedMsg := "path attempts to escape workspace"
	if err != nil && !containsString(err.Error(), expectedMsg) {
		t.Errorf("Expected error containing '%s', got: %v", expectedMsg, err)
	}
}

// TestNestedSymlinkAttackPrevention tests nested symlink attacks
func TestNestedSymlinkAttackPrevention(t *testing.T) {
	tempDir := t.TempDir()
	wsDir := filepath.Join(tempDir, "workspace")
	os.MkdirAll(wsDir, 0755)

	wsCfg := config.WorkspaceConfig{Path: wsDir}
	ws := workspace.New(wsCfg)
	ws.EnsureDir()

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			File: config.FileToolConfig{
				WhitelistDirs: []string{},
			},
		},
	}

	// Create nested symlinks
	subDir := filepath.Join(wsDir, "subdir")
	os.MkdirAll(subDir, 0755)

	outsideFile := filepath.Join(tempDir, "outside.txt")
	os.WriteFile(outsideFile, []byte("secret data"), 0644)

	symlinkPath := filepath.Join(subDir, "nested_evil")
	err := os.Symlink(outsideFile, symlinkPath)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Try to read via nested symlink - should fail
	readTool := file.NewReadFileTool(ws, cfg)
	_, err = readTool.Execute(`{"path": "subdir/nested_evil"}`)
	if err == nil {
		t.Error("Expected error when reading nested symlink that escapes workspace, got nil")
	}

	expectedMsg := "path attempts to escape workspace"
	if err != nil && !containsString(err.Error(), expectedMsg) {
		t.Errorf("Expected error containing '%s', got: %v", expectedMsg, err)
	}
}

// TestSymlinkToSystemFile tests symlink to system file attack
func TestSymlinkToSystemFile(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Skipping test: not running as root")
	}

	tempDir := t.TempDir()
	wsDir := filepath.Join(tempDir, "workspace")
	os.MkdirAll(wsDir, 0755)

	wsCfg := config.WorkspaceConfig{Path: wsDir}
	ws := workspace.New(wsCfg)
	ws.EnsureDir()

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			File: config.FileToolConfig{
				WhitelistDirs: []string{},
			},
		},
	}

	// Create symlink to /etc/passwd
	symlinkPath := filepath.Join(wsDir, "passwd_symlink")
	err := os.Symlink("/etc/passwd", symlinkPath)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Try to read /etc/passwd via symlink - should fail
	readTool := file.NewReadFileTool(ws, cfg)
	_, err = readTool.Execute(`{"path": "passwd_symlink"}`)
	if err == nil {
		t.Error("Expected error when reading symlink to /etc/passwd, got nil")
	}

	expectedMsg := "path attempts to escape workspace"
	if err != nil && !containsString(err.Error(), expectedMsg) {
		t.Errorf("Expected error containing '%s', got: %v", expectedMsg, err)
	}
}

// TestSymlinkWritePrevention tests that symlink attacks are prevented on write
func TestSymlinkWritePrevention(t *testing.T) {
	tempDir := t.TempDir()
	wsDir := filepath.Join(tempDir, "workspace")
	os.MkdirAll(wsDir, 0755)

	wsCfg := config.WorkspaceConfig{Path: wsDir}
	ws := workspace.New(wsCfg)
	ws.EnsureDir()

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			File: config.FileToolConfig{
				WhitelistDirs: []string{},
			},
		},
	}

	// Create symlink inside workspace pointing outside
	symlinkPath := filepath.Join(wsDir, "evil_symlink")
	outsideFile := filepath.Join(tempDir, "outside.txt")

	// Create symlink
	err := os.Symlink(outsideFile, symlinkPath)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Try to write via symlink - should fail
	writeTool := file.NewWriteFileTool(ws, cfg)
	_, err = writeTool.Execute(`{"path": "evil_symlink", "content": "malicious", "mode": "overwrite"}`)
	if err == nil {
		t.Error("Expected error when writing to symlink that escapes workspace, got nil")
	}

	expectedMsg := "path attempts to escape workspace"
	if err != nil && !containsString(err.Error(), expectedMsg) {
		t.Errorf("Expected error containing '%s', got: %v", expectedMsg, err)
	}
}

// TestInternalSymlinkAllowed tests that internal symlinks (within workspace) are allowed
func TestInternalSymlinkAllowed(t *testing.T) {
	tempDir := t.TempDir()
	wsDir := filepath.Join(tempDir, "workspace")
	os.MkdirAll(wsDir, 0755)

	wsCfg := config.WorkspaceConfig{Path: wsDir}
	ws := workspace.New(wsCfg)
	ws.EnsureDir()

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			File: config.FileToolConfig{
				WhitelistDirs: []string{},
			},
		},
	}

	// Create a file inside workspace
	targetFile := filepath.Join(wsDir, "target.txt")
	os.WriteFile(targetFile, []byte("internal data"), 0644)

	// Create symlink inside workspace pointing to internal file
	symlinkPath := filepath.Join(wsDir, "internal_symlink")
	err := os.Symlink(targetFile, symlinkPath)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Try to read via internal symlink - should succeed
	readTool := file.NewReadFileTool(ws, cfg)
	result, err := readTool.Execute(`{"path": "internal_symlink"}`)
	if err != nil {
		t.Errorf("Expected success when reading internal symlink, got error: %v", err)
	}

	// Verify content
	expectedContent := "internal data"
	if !containsString(result, expectedContent) {
		t.Errorf("Expected content '%s', got: %s", expectedContent, result)
	}
}

// TestRecursiveSymlinkResolution tests that recursive symlinks are handled correctly
func TestRecursiveSymlinkResolution(t *testing.T) {
	tempDir := t.TempDir()
	wsDir := filepath.Join(tempDir, "workspace")
	os.MkdirAll(wsDir, 0755)

	wsCfg := config.WorkspaceConfig{Path: wsDir}
	ws := workspace.New(wsCfg)
	ws.EnsureDir()

	// Create circular symlinks
	link1 := filepath.Join(wsDir, "link1")
	link2 := filepath.Join(wsDir, "link2")

	os.Symlink(link2, link1)
	os.Symlink(link1, link2)

	// Try to resolve recursive symlinks - should error
	_, err := ws.ResolveSymlinks(link1)
	if err == nil {
		t.Error("Expected error when resolving recursive symlinks, got nil")
	}
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[0:len(substr)] == substr ||
			(len(s) > len(substr) && s[1:len(substr)+1] == substr) ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
