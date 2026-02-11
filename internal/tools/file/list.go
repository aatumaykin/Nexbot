package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

// ListDirTool implements the Tool interface for listing directory contents.
// It lists files and directories in a workspace directory.
type ListDirTool struct {
	fileToolBase
}

// ListDirArgs represents the arguments for the list_dir tool.
type ListDirArgs struct {
	Path          string `json:"path"`                     // Path to the directory (relative to workspace or absolute)
	Recursive     bool   `json:"recursive,omitempty"`      // Whether to list recursively (default: false)
	IncludeHidden bool   `json:"include_hidden,omitempty"` // Whether to include hidden files/directories (default: false)
}

// NewListDirTool creates a new ListDirTool instance.
// The workspace parameter is used for resolving relative paths.
// The config parameter provides the file tool configuration (whitelist_dirs, etc.).
func NewListDirTool(ws *workspace.Workspace, cfg *config.Config) *ListDirTool {
	return &ListDirTool{
		fileToolBase: fileToolBase{
			workspace: ws,
			cfg:       cfg,
		},
	}
}

// Name returns the tool name.
func (t *ListDirTool) Name() string {
	return "list_dir"
}

// Description returns a description of what the tool does.
func (t *ListDirTool) Description() string {
	return "List directory contents in workspace. Supports recursive listing."
}

// Parameters returns the JSON Schema for the tool's parameters.
func (t *ListDirTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the directory to list. Can be absolute or relative to the workspace directory. Examples: {\"path\": \"src\"}",
			},
			"recursive": map[string]any{
				"type":        "boolean",
				"description": "Whether to list directory contents recursively. Examples: {\"path\": \"src\", \"recursive\": true}",
				"default":     false,
			},
			"include_hidden": map[string]any{
				"type":        "boolean",
				"description": "Whether to include hidden files and directories (those starting with '.'). Examples: {\"path\": \"config\", \"include_hidden\": true}",
				"default":     false,
			},
		},
		"required": []string{"path"},
	}
}

// Execute lists directory contents.
// args is a JSON-encoded string containing the tool's input parameters.
func (t *ListDirTool) Execute(args string) (string, error) {
	// Parse arguments
	var dirArgs ListDirArgs
	if err := parseJSON(args, &dirArgs); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Validate arguments
	if dirArgs.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Resolve path
	var fullPath string
	var err error

	if filepath.IsAbs(dirArgs.Path) {
		// Absolute path - use as-is
		fullPath = dirArgs.Path
	} else {
		// Relative path - resolve against workspace
		if t.workspace == nil {
			return "", fmt.Errorf("workspace is not configured")
		}
		fullPath, err = t.workspace.ResolvePath(dirArgs.Path)
		if err != nil {
			return "", fmt.Errorf("failed to resolve path: %w", err)
		}
	}

	// Clean the path
	cleanPath := filepath.Clean(fullPath)

	// Check if it's a directory
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("directory not found: %s", cleanPath)
		}
		return "", fmt.Errorf("failed to access directory: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", cleanPath)
	}

	// Collect directory entries
	var entries []string
	if dirArgs.Recursive {
		entries, err = t.listRecursive(cleanPath, dirArgs.IncludeHidden)
		if err != nil {
			return "", fmt.Errorf("failed to list directory recursively: %w", err)
		}
	} else {
		entries, err = t.listFlat(cleanPath, dirArgs.IncludeHidden)
		if err != nil {
			return "", fmt.Errorf("failed to list directory: %w", err)
		}
	}

	// Format output
	var result strings.Builder
	result.WriteString(fmt.Sprintf("# Directory: %s\n", filepath.Clean(cleanPath)))
	if dirArgs.Recursive {
		result.WriteString(fmt.Sprintf("# Recursive: %d items\n\n", len(entries)))
	} else {
		result.WriteString(fmt.Sprintf("# %d items\n\n", len(entries)))
	}

	for _, entry := range entries {
		result.WriteString(entry + "\n")
	}

	return result.String(), nil
}

// listFlat lists entries in a single directory (non-recursive).
func (t *ListDirTool) listFlat(dirPath string, includeHidden bool) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, entry := range entries {
		// Skip hidden files unless requested
		if !includeHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Determine entry type
		entryType := "FILE"
		if entry.IsDir() {
			entryType = "DIR "
		}

		result = append(result, fmt.Sprintf("%s %s", entryType, entry.Name()))
	}

	return result, nil
}

// listRecursive lists entries recursively.
func (t *ListDirTool) listRecursive(dirPath string, includeHidden bool) ([]string, error) {
	var result []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		// Skip hidden files unless requested
		if !includeHidden && strings.HasPrefix(relPath, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Determine entry type
		entryType := "FILE"
		if info.IsDir() {
			entryType = "DIR "
		}

		// Add entry to result
		result = append(result, fmt.Sprintf("%s %s", entryType, relPath))

		return nil
	})

	return result, err
}
