package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

// DeleteFileTool implements the Tool interface for deleting files and directories.
// It deletes a file or directory from the workspace.
type DeleteFileTool struct {
	fileToolBase
}

// DeleteFileArgs represents the arguments for the delete_file tool.
type DeleteFileArgs struct {
	Path      string `json:"path"`                // Path to the file or directory (relative to workspace or absolute)
	Recursive bool   `json:"recursive,omitempty"` // Whether to delete directories recursively (default: false)
}

// NewDeleteFileTool creates a new DeleteFileTool instance.
// The workspace parameter is used for resolving relative paths.
// The config parameter provides the file tool configuration (whitelist_dirs, etc.).
func NewDeleteFileTool(ws *workspace.Workspace, cfg *config.Config) *DeleteFileTool {
	return &DeleteFileTool{
		fileToolBase: fileToolBase{
			workspace: ws,
			cfg:       cfg,
		},
	}
}

// Name returns the tool name.
func (t *DeleteFileTool) Name() string {
	return "delete_file"
}

// Description returns a description of what the tool does.
func (t *DeleteFileTool) Description() string {
	return "Delete file or directory from workspace. Supports recursive deletion."
}

// Parameters returns the JSON Schema for the tool's parameters.
func (t *DeleteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The path to the file or directory to delete. Can be absolute or relative to the workspace directory. Examples: {\"path\": \"temp.txt\"}",
			},
			"recursive": map[string]interface{}{
				"type":        "boolean",
				"description": "For directories, whether to delete recursively. Required for non-empty directories. Examples: {\"path\": \"logs\", \"recursive\": true}",
				"default":     false,
			},
		},
		"required": []string{"path"},
	}
}

// Execute deletes a file or directory.
// args is a JSON-encoded string containing the tool's input parameters.
func (t *DeleteFileTool) Execute(args string) (string, error) {
	// Parse arguments
	var fileArgs DeleteFileArgs
	if err := parseJSON(args, &fileArgs); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Validate arguments
	if fileArgs.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Resolve path
	var fullPath string
	var err error

	if filepath.IsAbs(fileArgs.Path) {
		// Absolute path - use as-is
		fullPath = fileArgs.Path
	} else {
		// Relative path - resolve against workspace
		if t.workspace == nil {
			return "", fmt.Errorf("workspace is not configured")
		}
		fullPath, err = t.workspace.ResolvePath(fileArgs.Path)
		if err != nil {
			return "", fmt.Errorf("failed to resolve path: %w", err)
		}
	}

	// Clean the path
	cleanPath := filepath.Clean(fullPath)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path contains directory traversal attempt")
	}

	// Check if file/directory exists
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file or directory not found: %s", cleanPath)
		}
		return "", fmt.Errorf("failed to access path: %w", err)
	}

	// Perform deletion
	if info.IsDir() {
		// Directory
		if !fileArgs.Recursive {
			// Check if directory is empty
			entries, err := os.ReadDir(cleanPath)
			if err != nil {
				return "", fmt.Errorf("failed to check directory: %w", err)
			}
			if len(entries) > 0 {
				return "", fmt.Errorf("directory is not empty, use recursive=true to delete: %s", cleanPath)
			}
			// Remove empty directory
			if err := os.Remove(cleanPath); err != nil {
				return "", fmt.Errorf("failed to delete directory: %w", err)
			}
		} else {
			// Remove directory recursively
			if err := os.RemoveAll(cleanPath); err != nil {
				return "", fmt.Errorf("failed to delete directory recursively: %w", err)
			}
		}
	} else {
		// Regular file
		if err := os.Remove(cleanPath); err != nil {
			return "", fmt.Errorf("failed to delete file: %w", err)
		}
	}

	return fmt.Sprintf("Successfully deleted %s", cleanPath), nil
}
