// Package workspace provides workspace management functionality for Nexbot.
// It handles workspace directory creation, path resolution, and subdirectory management.
//
// The workspace is the root directory where Nexbot stores its data, including:
//   - memory/: Persistent memory storage
//   - skills/: Custom skill definitions
//
// Example usage:
//
//	cfg := config.WorkspaceConfig{Path: "~/.nexbot"}
//	ws, err := workspace.New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if err := ws.EnsureDir(); err != nil {
//	    log.Fatal(err)
//	}
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aatumaykin/nexbot/internal/config"
)

const (
	// SubdirMemory is the subdirectory name for persistent memory storage
	SubdirMemory = "memory"
	// SubdirSkills is the subdirectory name for custom skill definitions
	SubdirSkills = "skills"
)

// Workspace represents a Nexbot workspace with path management capabilities.
type Workspace struct {
	path         string   // Expanded workspace path
	basePath     string   // Original path from config (may contain ~)
	symlinkCache sync.Map // Cache for resolved symlink paths
}

// New creates a new Workspace from the given configuration.
// The path from config is stored as-is in basePath and expanded in path.
func New(cfg config.WorkspaceConfig) *Workspace {
	expandedPath := expandHome(cfg.Path)
	return &Workspace{
		path:     expandedPath,
		basePath: cfg.Path,
	}
}

// Path returns the expanded workspace path (with ~ expanded to home directory).
func (w *Workspace) Path() string {
	return w.path
}

// BasePath returns the original path from config (may contain ~).
func (w *Workspace) BasePath() string {
	return w.basePath
}

// EnsureDir creates the workspace directory if it doesn't exist.
// Returns an error if the directory cannot be created or if permissions are insufficient.
func (w *Workspace) EnsureDir() error {
	// Check if path is empty
	if w.path == "" {
		return fmt.Errorf("workspace path is empty")
	}

	// Check if directory already exists
	info, err := os.Stat(w.path)
	if err == nil {
		// Path exists, check if it's a directory
		if !info.IsDir() {
			return fmt.Errorf("workspace path exists but is not a directory: %s", w.path)
		}
		// Directory exists and is valid
		return nil
	}

	// Check if error is not "does not exist"
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to access workspace path %s: %w", w.path, err)
	}

	// Create directory with appropriate permissions
	if err := os.MkdirAll(w.path, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory %s: %w", w.path, err)
	}

	return nil
}

// ResolvePath resolves a relative path within the workspace.
// If the given path is absolute, it's returned as-is.
// If the path is relative, it's joined with the workspace path.
// Returns an error if the path would escape the workspace directory.
func (w *Workspace) ResolvePath(relPath string) (string, error) {
	if relPath == "" {
		return "", fmt.Errorf("path is empty")
	}

	// If path is already absolute, return it as-is
	if filepath.IsAbs(relPath) {
		absRelPath, err := filepath.Abs(relPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path: %w", err)
		}
		return absRelPath, nil
	}

	// Clean the relative path to normalize it
	cleanPath := filepath.Clean(relPath)

	// Check for directory traversal attempts before joining
	// If the path starts with "..", it would escape the workspace
	if cleanPath == ".." || (len(cleanPath) > 2 && (cleanPath[:2] == ".." && (cleanPath[2] == filepath.Separator || cleanPath[2] == '/' || cleanPath[2] == '\\'))) {
		return "", fmt.Errorf("path attempts to escape workspace: %s", relPath)
	}

	// Also check if any component of the path is ".."
	pathParts := filepath.SplitList(cleanPath)
	for _, part := range pathParts {
		if part == ".." {
			return "", fmt.Errorf("path attempts to escape workspace: %s", relPath)
		}
	}

	// Join with workspace path
	joinedPath := filepath.Join(w.path, cleanPath)

	// Get absolute paths for comparison
	absWorkspace, err := filepath.Abs(w.path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute workspace path: %w", err)
	}

	absJoined, err := filepath.Abs(joinedPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute joined path: %w", err)
	}

	// Verify the joined path doesn't escape the workspace
	relToWorkspace, err := filepath.Rel(absWorkspace, absJoined)
	if err != nil {
		return "", fmt.Errorf("failed to check path relationship: %w", err)
	}

	// If the relative path from workspace starts with "..", it escapes
	if filepath.IsAbs(relToWorkspace) || (len(relToWorkspace) > 1 && relToWorkspace[0:2] == "..") {
		return "", fmt.Errorf("path attempts to escape workspace: %s", relPath)
	}

	return absJoined, nil
}

// Subpath returns a path for a standard workspace subdirectory.
// Common subdirectories: "memory", "skills"
func (w *Workspace) Subpath(name string) string {
	return filepath.Join(w.path, name)
}

// EnsureSubpath creates a subdirectory within the workspace if it doesn't exist.
// Returns an error if the workspace directory doesn't exist or cannot be created.
func (w *Workspace) EnsureSubpath(name string) error {
	// First ensure workspace exists
	if err := w.EnsureDir(); err != nil {
		return fmt.Errorf("failed to ensure workspace: %w", err)
	}

	if name == "" {
		return fmt.Errorf("subdirectory name is empty")
	}

	subpath := w.Subpath(name)

	// Check if subdirectory already exists
	info, err := os.Stat(subpath)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("subdirectory path exists but is not a directory: %s", subpath)
		}
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to access subdirectory %s: %w", subpath, err)
	}

	// Create subdirectory
	if err := os.MkdirAll(subpath, 0755); err != nil {
		return fmt.Errorf("failed to create subdirectory %s: %w", subpath, err)
	}

	return nil
}

// ResolveSymlinks resolves all symlinks in the path and returns the final real path.
// Uses caching to avoid repeated filesystem calls.
func (w *Workspace) ResolveSymlinks(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}

	// Get absolute path first
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check cache
	if cached, ok := w.symlinkCache.Load(absPath); ok {
		if resolved, ok := cached.(string); ok {
			return resolved, nil
		}
	}

	// Resolve symlinks
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If path doesn't exist, return absolute path
		if os.IsNotExist(err) {
			w.symlinkCache.Store(absPath, absPath)
			return absPath, nil
		}
		return "", fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Cache the result
	w.symlinkCache.Store(absPath, resolved)
	return resolved, nil
}

// ValidatePath validates that a path is safe for file operations within the workspace.
// It checks for:
//   - Path traversal attempts (.. components)
//   - Symlinks pointing outside the workspace
//   - Non-existent files with symlink parents pointing outside
func (w *Workspace) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get absolute workspace path
	absWorkspace, err := filepath.Abs(w.path)
	if err != nil {
		return fmt.Errorf("failed to get absolute workspace path: %w", err)
	}

	// Resolve both workspace and path to their real paths to handle system symlinks
	resolvedWorkspace := absWorkspace
	if rw, err := filepath.EvalSymlinks(absWorkspace); err == nil {
		resolvedWorkspace = rw
	}

	// Check for existing file first
	fileInfo, err := os.Lstat(absPath)
	if err == nil {
		// File exists, check if it's a symlink
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			// It's a symlink, resolve it
			resolved, err := w.ResolveSymlinks(absPath)
			if err != nil {
				return fmt.Errorf("failed to resolve symlinks: %w", err)
			}

			// Verify resolved path is within resolved workspace
			rel, err := filepath.Rel(resolvedWorkspace, resolved)
			if err != nil {
				return fmt.Errorf("failed to check path relationship: %w", err)
			}

			// Check if path starts with ".." (escapes workspace)
			if filepath.IsAbs(rel) || (len(rel) >= 2 && rel[0:2] == "..") {
				return fmt.Errorf("path attempts to escape workspace: %s", path)
			}
		} else {
			// Not a symlink, just check if it's within workspace
			rel, err := filepath.Rel(absWorkspace, absPath)
			if err != nil {
				return fmt.Errorf("failed to check path relationship: %w", err)
			}

			// Check if path starts with ".." (escapes workspace)
			if filepath.IsAbs(rel) || (len(rel) >= 2 && rel[0:2] == "..") {
				return fmt.Errorf("path attempts to escape workspace: %s", path)
			}
		}
	} else if os.IsNotExist(err) {
		// File doesn't exist, check parent directories for symlinks
		parentDir := filepath.Dir(absPath)

		// Check if parent is within workspace
		relParent, err := filepath.Rel(absWorkspace, parentDir)
		if err != nil {
			return fmt.Errorf("failed to check parent path relationship: %w", err)
		}

		// If parent is outside workspace, that's an error
		if filepath.IsAbs(relParent) || (len(relParent) >= 2 && relParent[0:2] == "..") {
			return fmt.Errorf("path attempts to escape workspace: %s", path)
		}

		// Check each parent directory until we reach workspace
		absWorkspaceClean := filepath.Clean(absWorkspace)
		for parentDir != absWorkspaceClean && parentDir != "/" {
			// Check if this directory exists
			parentInfo, err := os.Lstat(parentDir)
			if err == nil {
				// Directory exists, check if it's a symlink
				if parentInfo.Mode()&os.ModeSymlink != 0 {
					// It's a symlink, resolve it
					resolvedParent, err := filepath.EvalSymlinks(parentDir)
					if err != nil {
						return fmt.Errorf("failed to resolve parent symlink: %w", err)
					}

					// Verify resolved parent is within resolved workspace
					relResolved, err := filepath.Rel(resolvedWorkspace, resolvedParent)
					if err != nil {
						return fmt.Errorf("failed to check resolved parent path relationship: %w", err)
					}

					// Check if resolved parent starts with ".." (escapes workspace)
					if filepath.IsAbs(relResolved) || (len(relResolved) >= 2 && relResolved[0:2] == "..") {
						return fmt.Errorf("path attempts to escape workspace via symlink parent: %s", path)
					}
				}
			} else if !os.IsNotExist(err) {
				// Some other error
				return fmt.Errorf("failed to stat parent directory: %w", err)
			}

			// Move up to next parent
			newParent := filepath.Dir(parentDir)
			if newParent == parentDir {
				// Reached root
				break
			}
			parentDir = newParent
		}
	} else {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	return nil
}

// expandHome expands ~ to the user's home directory.
// If the path doesn't start with ~/, it's returned unchanged.
func expandHome(path string) string {
	if len(path) > 0 && path[0] == '~' && (len(path) == 1 || path[1] == '/') {
		home, err := os.UserHomeDir()
		if err != nil {
			// If we can't get home directory, return original path
			return path
		}
		if len(path) == 1 {
			return home
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
