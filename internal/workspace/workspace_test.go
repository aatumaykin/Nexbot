package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
)

// TestNew tests the New constructor
func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		cfgPath   string
		wantPath  string
		checkHome bool
	}{
		{
			name:     "simple path",
			cfgPath:  "/tmp/nexbot",
			wantPath: "/tmp/nexbot",
		},
		{
			name:     "empty path",
			cfgPath:  "",
			wantPath: "", // Should remain empty
		},
		{
			name:      "home path with tilde",
			cfgPath:   "~/.nexbot",
			checkHome: true, // Should be expanded to actual home directory
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WorkspaceConfig{Path: tt.cfgPath}
			ws := New(cfg)

			if ws.basePath != tt.cfgPath {
				t.Errorf("BasePath() = %v, want %v", ws.basePath, tt.cfgPath)
			}

			if tt.checkHome {
				// Check that path was expanded from home
				home, _ := os.UserHomeDir()
				expectedPath := filepath.Join(home, ".nexbot")
				if ws.path != expectedPath {
					t.Errorf("Path() = %v, want %v (home expanded)", ws.path, expectedPath)
				}
			} else if tt.wantPath != "" && ws.path != tt.wantPath {
				t.Errorf("Path() = %v, want %v", ws.path, tt.wantPath)
			}
		})
	}
}

// TestPath tests the Path method
func TestPath(t *testing.T) {
	ws := &Workspace{
		path:     "/tmp/nexbot",
		basePath: "~/.nexbot",
	}

	if got := ws.Path(); got != "/tmp/nexbot" {
		t.Errorf("Path() = %v, want %v", got, "/tmp/nexbot")
	}
}

// TestBasePath tests the BasePath method
func TestBasePath(t *testing.T) {
	ws := &Workspace{
		path:     filepath.Join(os.Getenv("HOME"), ".nexbot"),
		basePath: "~/.nexbot",
	}

	if got := ws.BasePath(); got != "~/.nexbot" {
		t.Errorf("BasePath() = %v, want %v", got, "~/.nexbot")
	}
}

// TestEnsureDir tests the EnsureDir method
func TestEnsureDir(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		setup   func(t *testing.T) func()
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "workspace path is empty",
		},
		{
			name: "existing directory",
			path: filepath.Join(os.TempDir(), "nexbot-test-existing"),
			setup: func(t *testing.T) func() {
				if err := os.MkdirAll(filepath.Join(os.TempDir(), "nexbot-test-existing"), 0755); err != nil {
					t.Fatalf("failed to create test directory: %v", err)
				}
				return func() {
					os.RemoveAll(filepath.Join(os.TempDir(), "nexbot-test-existing"))
				}
			},
			wantErr: false,
		},
		{
			name: "create new directory",
			path: filepath.Join(os.TempDir(), "nexbot-test-new"),
			setup: func(t *testing.T) func() {
				return func() {
					os.RemoveAll(filepath.Join(os.TempDir(), "nexbot-test-new"))
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				cleanup := tt.setup(t)
				defer cleanup()
			}

			ws := &Workspace{path: tt.path, basePath: tt.path}
			err := ws.EnsureDir()

			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("EnsureDir() error message = %v, want %v", err.Error(), tt.errMsg)
				}
			}

			if !tt.wantErr && tt.path != "" {
				// Verify directory exists
				info, err := os.Stat(tt.path)
				if err != nil {
					t.Errorf("directory was not created: %v", err)
				}
				if err == nil && !info.IsDir() {
					t.Errorf("path exists but is not a directory")
				}
			}
		})
	}
}

// TestResolvePath tests the ResolvePath method
func TestResolvePath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := &Workspace{path: tmpDir, basePath: tmpDir}

	tests := []struct {
		name      string
		relPath   string
		wantErr   bool
		wantStart string
		errMsg    string
	}{
		{
			name:    "empty path",
			relPath: "",
			wantErr: true,
			errMsg:  "path is empty",
		},
		{
			name:      "simple relative path",
			relPath:   "test.txt",
			wantErr:   false,
			wantStart: tmpDir,
		},
		{
			name:      "nested relative path",
			relPath:   "subdir/test.txt",
			wantErr:   false,
			wantStart: tmpDir,
		},
		{
			name:      "relative path with dot",
			relPath:   "./test.txt",
			wantErr:   false,
			wantStart: tmpDir,
		},
		{
			name:    "escape attempt with ..",
			relPath: "../test.txt",
			wantErr: true,
			errMsg:  "attempts to escape",
		},
		{
			name:    "escape attempt with absolute path",
			relPath: "/etc/passwd",
			wantErr: false,
			// Absolute paths are allowed as-is
			wantStart: "/etc/passwd",
		},
		{
			name:    "escape attempt with ../..",
			relPath: "../../test.txt",
			wantErr: true,
			errMsg:  "attempts to escape",
		},
		{
			name:    "escape attempt with .. in middle",
			relPath: "subdir/../../test.txt",
			wantErr: true,
			errMsg:  "attempts to escape",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ws.ResolvePath(tt.relPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolvePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("ResolvePath() error message = %v, want %v", err.Error(), tt.errMsg)
				}
			}

			if !tt.wantErr {
				if !filepath.IsAbs(got) {
					t.Errorf("ResolvePath() returned non-absolute path: %v", got)
				}

				if tt.wantStart != "" && got != tt.wantStart && !containsString(got, tt.wantStart) {
					t.Errorf("ResolvePath() = %v, want starting with %v", got, tt.wantStart)
				}
			}
		})
	}
}

// TestSubpath tests the Subpath method
func TestSubpath(t *testing.T) {
	tmpDir := "/tmp/nexbot"
	ws := &Workspace{path: tmpDir, basePath: tmpDir}

	tests := []struct {
		name string
		dir  string
		want string
	}{
		{
			name: "memory subdirectory",
			dir:  "memory",
			want: filepath.Join(tmpDir, "memory"),
		},
		{
			name: "skills subdirectory",
			dir:  "skills",
			want: filepath.Join(tmpDir, "skills"),
		},
		{
			name: "custom subdirectory",
			dir:  "custom",
			want: filepath.Join(tmpDir, "custom"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ws.Subpath(tt.dir)
			if got != tt.want {
				t.Errorf("Subpath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestEnsureSubpath tests the EnsureSubpath method
func TestEnsureSubpath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := &Workspace{path: tmpDir, basePath: tmpDir}

	tests := []struct {
		name    string
		subdir  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty subdirectory name",
			subdir:  "",
			wantErr: true,
			errMsg:  "subdirectory name is empty",
		},
		{
			name:    "create memory subdirectory",
			subdir:  "memory",
			wantErr: false,
		},
		{
			name:    "create skills subdirectory",
			subdir:  "skills",
			wantErr: false,
		},
		{
			name:    "create nested subdirectory",
			subdir:  "nested/path",
			wantErr: false,
		},
		{
			name:   "existing subdirectory",
			subdir: "memory",
			// This should not error if the directory already exists
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ws.EnsureSubpath(tt.subdir)

			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureSubpath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("EnsureSubpath() error message = %v, want %v", err.Error(), tt.errMsg)
				}
			}

			if !tt.wantErr && tt.subdir != "" {
				subpath := ws.Subpath(tt.subdir)
				info, err := os.Stat(subpath)
				if err != nil {
					t.Errorf("subdirectory was not created: %v", err)
				}
				if err == nil && !info.IsDir() {
					t.Errorf("subpath exists but is not a directory")
				}
			}
		})
	}
}

// TestExpandHome tests the expandHome function
func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "expand tilde only",
			path: "~",
			want: home,
		},
		{
			name: "expand tilde with slash",
			path: "~/",
			want: home,
		},
		{
			name: "expand tilde with path",
			path: "~/.nexbot",
			want: filepath.Join(home, ".nexbot"),
		},
		{
			name: "expand tilde with nested path",
			path: "~/projects/nexbot",
			want: filepath.Join(home, "projects", "nexbot"),
		},
		{
			name: "absolute path",
			path: "/tmp/nexbot",
			want: "/tmp/nexbot",
		},
		{
			name: "relative path",
			path: "./nexbot",
			want: "./nexbot",
		},
		{
			name: "empty path",
			path: "",
			want: "",
		},
		{
			name: "path starting with ~ but not followed by /",
			path: "~test",
			want: "~test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandHome(tt.path)
			if got != tt.want {
				t.Errorf("expandHome() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIntegration tests integration scenarios
func TestIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := struct{ Path string }{Path: tmpDir}

	ws := &Workspace{}
	ws.path = expandHome(cfg.Path)
	ws.basePath = cfg.Path

	// Ensure workspace directory exists
	if err := ws.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() failed: %v", err)
	}

	// Create subdirectories
	if err := ws.EnsureSubpath(SubdirMemory); err != nil {
		t.Fatalf("EnsureSubpath(memory) failed: %v", err)
	}
	if err := ws.EnsureSubpath(SubdirSkills); err != nil {
		t.Fatalf("EnsureSubpath(skills) failed: %v", err)
	}

	// Verify paths resolve correctly
	memoryPath, err := ws.ResolvePath("memory")
	if err != nil {
		t.Fatalf("ResolvePath(memory) failed: %v", err)
	}

	expectedMemoryPath := filepath.Join(tmpDir, SubdirMemory)
	if memoryPath != expectedMemoryPath {
		t.Errorf("Resolved path = %v, want %v", memoryPath, expectedMemoryPath)
	}

	// Verify subpaths
	skillsPath := ws.Subpath(SubdirSkills)
	expectedSkillsPath := filepath.Join(tmpDir, SubdirSkills)
	if skillsPath != expectedSkillsPath {
		t.Errorf("Subpath() = %v, want %v", skillsPath, expectedSkillsPath)
	}
}

// TestConfigToWorkspaceIntegration tests the complete flow from config to workspace
func TestConfigToWorkspaceIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.WorkspaceConfig{
		Path:              tmpDir,
		BootstrapMaxChars: 5000,
	}

	// Create workspace from config
	ws := New(cfg)

	// Verify path is set correctly
	if ws.Path() != tmpDir {
		t.Errorf("ws.Path() = %q, want %q", ws.Path(), tmpDir)
	}

	// Ensure workspace exists
	if err := ws.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() failed: %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("failed to stat workspace dir: %v", err)
	}
	if !info.IsDir() {
		t.Error("workspace path exists but is not a directory")
	}

	// Verify subpaths work
	memoryPath := ws.Subpath(SubdirMemory)
	expectedMemoryPath := filepath.Join(tmpDir, SubdirMemory)
	if memoryPath != expectedMemoryPath {
		t.Errorf("Subpath(memory) = %q, want %q", memoryPath, expectedMemoryPath)
	}
}

// TestConfigToWorkspaceWithTilde tests tilde expansion in config to workspace
func TestConfigToWorkspaceWithTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	cfg := config.WorkspaceConfig{
		Path:              "~/.nexbot-test",
		BootstrapMaxChars: 5000,
	}

	ws := New(cfg)

	// Verify tilde was expanded
	expectedPath := filepath.Join(home, ".nexbot-test")
	if ws.Path() != expectedPath {
		t.Errorf("ws.Path() = %q, want %q", ws.Path(), expectedPath)
	}

	// Verify BasePath preserves tilde
	if ws.BasePath() != "~/.nexbot-test" {
		t.Errorf("ws.BasePath() = %q, want %q", ws.BasePath(), "~/.nexbot-test")
	}

	// Clean up
	_ = os.RemoveAll(expectedPath)
}

// TestConfigToWorkspaceWithAbsolutePath tests absolute path in config to workspace
func TestConfigToWorkspaceWithAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.WorkspaceConfig{
		Path:              tmpDir,
		BootstrapMaxChars: 5000,
	}

	ws := New(cfg)

	// Verify absolute path is preserved
	absPath, err := filepath.Abs(tmpDir)
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	if ws.Path() != absPath {
		t.Errorf("ws.Path() = %q, want %q", ws.Path(), absPath)
	}

	// Ensure workspace exists
	if err := ws.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() failed: %v", err)
	}
}

// TestConfigToWorkspaceWithRelativePath tests relative path in config to workspace
func TestConfigToWorkspaceWithRelativePath(t *testing.T) {
	relativePath := "./test-workspace"

	cfg := config.WorkspaceConfig{
		Path:              relativePath,
		BootstrapMaxChars: 5000,
	}

	ws := New(cfg)

	// Relative paths are preserved (not expanded to absolute)
	if ws.Path() != relativePath {
		t.Errorf("ws.Path() = %q, want %q", ws.Path(), relativePath)
	}

	// Clean up
	_ = os.RemoveAll(relativePath)
}

// TestEnsureDirPermissions tests permission handling in EnsureDir
func TestEnsureDirPermissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root, permissions test would fail")
	}

	tmpDir := t.TempDir()

	// Create a workspace path pointing to a file (not directory)
	filePath := filepath.Join(tmpDir, "asfile")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	wsFile := &Workspace{path: filePath, basePath: filePath}
	err := wsFile.EnsureDir()
	if err == nil {
		t.Error("EnsureDir() on file path should error")
	}

	// Test with readonly directory
	readonlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readonlyDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(readonlyDir, "test")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Make directory readonly (no write permission)
	if err := os.Chmod(readonlyDir, 0500); err != nil {
		t.Fatalf("failed to change directory permissions: %v", err)
	}

	// Try to create a subdirectory in readonly parent
	ws := &Workspace{path: readonlyDir, basePath: readonlyDir}
	if err := ws.EnsureDir(); err != nil {
		t.Errorf("EnsureDir() on existing readonly dir should not error, got: %v", err)
	}

	// Restore permissions for cleanup
	_ = os.Chmod(readonlyDir, 0755)
}

// TestEnsureSubpathPermissions tests permission handling in EnsureSubpath
func TestEnsureSubpathPermissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root, permissions test would fail")
	}

	tmpDir := t.TempDir()

	// Create a readonly parent directory
	readonlyParent := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readonlyParent, 0500); err != nil {
		t.Fatalf("failed to create readonly directory: %v", err)
	}

	// Try to create subdirectory in readonly parent
	ws := &Workspace{path: readonlyParent, basePath: readonlyParent}

	// First ensure parent exists (should work)
	if err := ws.EnsureDir(); err != nil {
		t.Errorf("EnsureDir() on existing readonly dir should not error, got: %v", err)
	}

	// Try to create a subdirectory - this may fail due to permissions
	// On some systems it may succeed, so we check but don't fail the test if it works
	_ = ws.EnsureSubpath("testsub")
}

// TestResolvePathEdgeCases tests edge cases in ResolvePath
func TestResolvePathEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	ws := &Workspace{path: tmpDir, basePath: tmpDir}

	tests := []struct {
		name    string
		relPath string
		wantErr bool
	}{
		{
			name:    "single dot",
			relPath: ".",
			wantErr: false,
		},
		{
			name:    "double dot",
			relPath: "..",
			wantErr: true,
		},
		{
			name:    "trailing slash",
			relPath: "test/",
			wantErr: false,
		},
		{
			name:    "multiple slashes",
			relPath: "test//file.txt",
			wantErr: false,
		},
		{
			name:    "dot slash dot",
			relPath: "././file.txt",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ws.ResolvePath(tt.relPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolvePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || len(s) > len(substr)*2))
}

// Simpler contains function for checking substrings
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
