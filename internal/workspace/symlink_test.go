package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
)

func TestResolveSymlinks(t *testing.T) {
	tempDir := t.TempDir()

	// Test 1: Resolve regular file path
	regularFile := filepath.Join(tempDir, "regular.txt")
	os.WriteFile(regularFile, []byte("content"), 0644)

	ws := New(config.WorkspaceConfig{Path: tempDir})

	resolved, err := ws.ResolveSymlinks(regularFile)
	if err != nil {
		t.Fatalf("failed to resolve regular file: %v", err)
	}
	// Resolved path should be the real path (may resolve system symlinks like /var -> /private/var)
	absRegularFile, err := filepath.Abs(regularFile)
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}
	evalResolved, err := filepath.EvalSymlinks(absRegularFile)
	if err == nil {
		if resolved != evalResolved {
			t.Errorf("expected resolved path %s, got %s", evalResolved, resolved)
		}
	} else {
		if resolved != absRegularFile {
			t.Errorf("expected resolved path %s, got %s", absRegularFile, resolved)
		}
	}

	// Test 2: Resolve symlink
	symlinkPath := filepath.Join(tempDir, "link.txt")
	if err := os.Symlink(regularFile, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	resolved, err = ws.ResolveSymlinks(symlinkPath)
	if err != nil {
		t.Fatalf("failed to resolve symlink: %v", err)
	}
	// Should resolve to the real path of regularFile (accounting for system symlinks)
	evalTarget, err := filepath.EvalSymlinks(absRegularFile)
	if err == nil {
		if resolved != evalTarget {
			t.Errorf("expected resolved path %s, got %s", evalTarget, resolved)
		}
	} else {
		if resolved != absRegularFile {
			t.Errorf("expected resolved path %s, got %s", absRegularFile, resolved)
		}
	}

	// Test 3: Test caching
	resolved2, err := ws.ResolveSymlinks(symlinkPath)
	if err != nil {
		t.Fatalf("failed to resolve symlink (cached): %v", err)
	}
	if resolved2 != resolved {
		t.Errorf("expected cached resolved path %s, got %s", resolved, resolved2)
	}
}

func TestValidatePath_SymlinkAttack(t *testing.T) {
	tempDir := t.TempDir()

	wsDir := filepath.Join(tempDir, "workspace")
	os.MkdirAll(wsDir, 0755)

	ws := New(config.WorkspaceConfig{Path: wsDir})

	// Test 1: Symlink pointing outside workspace
	symlinkPath := filepath.Join(wsDir, "evil_symlink")
	outsideFile := filepath.Join(tempDir, "outside.txt")
	os.WriteFile(outsideFile, []byte("secret"), 0644)

	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	err := ws.ValidatePath(symlinkPath)
	if err == nil {
		t.Error("expected error for symlink pointing outside workspace")
	}

	expectedMsg := "path attempts to escape workspace"
	if err != nil && !containsSubstring(err.Error(), expectedMsg) {
		t.Errorf("expected error containing '%s', got: %v", expectedMsg, err)
	}

	// Test 2: Internal symlink should be allowed
	internalFile := filepath.Join(wsDir, "internal.txt")
	os.WriteFile(internalFile, []byte("internal data"), 0644)

	internalSymlink := filepath.Join(wsDir, "internal_link")
	if err := os.Symlink(internalFile, internalSymlink); err != nil {
		t.Fatalf("failed to create internal symlink: %v", err)
	}

	err = ws.ValidatePath(internalSymlink)
	if err != nil {
		t.Errorf("expected no error for internal symlink, got: %v", err)
	}
}

func TestValidatePath_NestedSymlinkAttack(t *testing.T) {
	tempDir := t.TempDir()

	wsDir := filepath.Join(tempDir, "workspace")
	os.MkdirAll(wsDir, 0755)

	subDir := filepath.Join(wsDir, "subdir")
	os.MkdirAll(subDir, 0755)

	ws := New(config.WorkspaceConfig{Path: wsDir})

	outsideFile := filepath.Join(tempDir, "outside.txt")
	os.WriteFile(outsideFile, []byte("secret"), 0644)

	symlinkPath := filepath.Join(subDir, "nested_evil")
	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	err := ws.ValidatePath(symlinkPath)
	if err == nil {
		t.Error("expected error for nested symlink pointing outside workspace")
	}

	expectedMsg := "path attempts to escape workspace"
	if err != nil && !containsSubstring(err.Error(), expectedMsg) {
		t.Errorf("expected error containing '%s', got: %v", expectedMsg, err)
	}
}

func TestValidatePath_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()

	wsDir := filepath.Join(tempDir, "workspace")
	os.MkdirAll(wsDir, 0755)

	ws := New(config.WorkspaceConfig{Path: wsDir})

	// Non-existent file should not error
	nonExistent := filepath.Join(wsDir, "nonexistent.txt")
	err := ws.ValidatePath(nonExistent)
	if err != nil {
		t.Errorf("expected no error for non-existent file, got: %v", err)
	}
}

func TestValidatePath_NonExistentWithSymlinkParent(t *testing.T) {
	tempDir := t.TempDir()

	wsDir := filepath.Join(tempDir, "workspace")
	os.MkdirAll(wsDir, 0755)

	ws := New(config.WorkspaceConfig{Path: wsDir})

	// Create symlink parent pointing outside
	symlinkDir := filepath.Join(wsDir, "symlink_dir")
	outsideDir := filepath.Join(tempDir, "outside")
	os.MkdirAll(outsideDir, 0755)

	if err := os.Symlink(outsideDir, symlinkDir); err != nil {
		t.Fatalf("failed to create symlink directory: %v", err)
	}

	// Non-existent file under symlink parent should error
	nonExistent := filepath.Join(symlinkDir, "file.txt")
	err := ws.ValidatePath(nonExistent)
	if err == nil {
		t.Error("expected error for file under symlink parent pointing outside workspace")
	}

	// Should mention "symlink" and "escape" in error message
	if err != nil && !containsSubstring(err.Error(), "symlink") && !containsSubstring(err.Error(), "escape") {
		t.Errorf("expected error to mention symlink or escape, got: %v", err)
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
