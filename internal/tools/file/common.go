package file

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

// fileToolBase contains common fields for file tools.
type fileToolBase struct {
	workspace *workspace.Workspace
	cfg       *config.Config
}

// parseJSON is a helper function to parse JSON arguments.
func parseJSON(jsonStr string, v interface{}) error {
	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

// splitLines splits a string into lines, handling various line endings.
func splitLines(s string) []string {
	// Use Split with both \n and \r\n
	var lines []string
	start := 0

	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			// Check for \r\n
			if i > 0 && s[i-1] == '\r' {
				lines = append(lines, s[start:i-1])
			} else {
				lines = append(lines, s[start:i])
			}
			start = i + 1
		}
	}

	// Add the last line if there's content after the last newline
	if start < len(s) {
		lines = append(lines, s[start:])
	}

	return lines
}

// validatePath validates that a path doesn't contain directory traversal attempts.
func (ftb *fileToolBase) validatePath(path string) error {
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path contains directory traversal attempt")
	}
	return nil
}

// resolvePath resolves a path to an absolute path.
// For relative paths, it uses the workspace. For absolute paths, it checks the whitelist.
func (ftb *fileToolBase) resolvePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		// Absolute path - check whitelist_dirs
		allowed := false
		for _, allowedDir := range ftb.cfg.Tools.File.WhitelistDirs {
			// Exact check: path must either be equal to allowedDir or start with allowedDir + separator
			if path == allowedDir || strings.HasPrefix(path, allowedDir+string(filepath.Separator)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("absolute paths are not allowed")
		}
		return path, nil
	}

	// Relative path - resolve against workspace
	if ftb.workspace == nil {
		return "", fmt.Errorf("workspace is not configured")
	}
	return ftb.workspace.ResolvePath(path)
}

// fileExists checks if a file exists at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isDirectory checks if the path is a directory.
func isDirectory(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}
