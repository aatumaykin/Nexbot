package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

// WriteFileTool implements the Tool interface for writing file content.
// It writes content to a file in the workspace.
type WriteFileTool struct {
	fileToolBase
}

// WriteFileArgs represents the arguments for the write_file tool.
type WriteFileArgs struct {
	Path    string `json:"path"`           // Path to the file (relative to workspace or absolute)
	Content string `json:"content"`        // Content to write to the file
	Mode    string `json:"mode,omitempty"` // Write mode: "create" (default), "append", "overwrite"
}

// NewWriteFileTool creates a new WriteFileTool instance.
// The workspace parameter is used for resolving relative paths.
// The config parameter provides the file tool configuration (whitelist_dirs, etc.).
func NewWriteFileTool(ws *workspace.Workspace, cfg *config.Config) *WriteFileTool {
	return &WriteFileTool{
		fileToolBase: fileToolBase{
			workspace: ws,
			cfg:       cfg,
		},
	}
}

// Name returns the tool name.
func (t *WriteFileTool) Name() string {
	return "write_file"
}

// Description returns a description of what the tool does.
func (t *WriteFileTool) Description() string {
	return "Write content to a file in workspace. Supports create, append, overwrite modes."
}

// Parameters returns the JSON Schema for the tool's parameters.
func (t *WriteFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the file to write. Can be absolute or relative to the workspace directory. Examples: {\"path\": \"newfile.txt\", \"content\": \"Hello World\", \"mode\": \"create\"}",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file. Examples: {\"path\": \"logs.txt\", \"content\": \"New entry\", \"mode\": \"append\"}",
			},
			"mode": map[string]any{
				"type":        "string",
				"description": "Write mode: 'create' (fails if file exists), 'append' (append to existing file), 'overwrite' (replace file content). Defaults to 'create'. Examples: {\"path\": \"config.json\", \"content\": \"{\\\"key\\\": \\\"value\\\"}\", \"mode\": \"overwrite\"}",
				"enum":        []string{"create", "append", "overwrite"},
				"default":     "create",
			},
		},
		"required": []string{"path", "content"},
	}
}

// Execute writes content to a file.
// args is a JSON-encoded string containing the tool's input parameters.
func (t *WriteFileTool) Execute(args string) (string, error) {
	// Parse arguments
	var fileArgs WriteFileArgs
	if err := parseJSON(args, &fileArgs); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Validate arguments
	if fileArgs.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	if fileArgs.Content == "" {
		return "", fmt.Errorf("content is required")
	}

	// Set defaults
	if fileArgs.Mode == "" {
		fileArgs.Mode = "create"
	}

	// Validate mode
	validModes := map[string]bool{"create": true, "append": true, "overwrite": true}
	if !validModes[fileArgs.Mode] {
		return "", fmt.Errorf("invalid mode '%s', must be one of: create, append, overwrite", fileArgs.Mode)
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

	// Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(fullPath)

	// Validate skill files
	if isSkillPath(cleanPath) {
		workspaceRoot := ""
		if t.workspace != nil {
			workspaceRoot = t.workspace.Path()
		}

		if err := validateSkillPath(cleanPath, workspaceRoot); err != nil {
			return "", err
		}

		// Validate skill content if enabled in config
		if t.cfg.Tools.File.ValidateSkillContent {
			if err := validateSkillContent(fileArgs.Content); err != nil {
				return "", fmt.Errorf("skill content validation failed: %w", err)
			}
		}
	}

	// Check for directory traversal attempts
	if filepath.IsAbs(fileArgs.Path) {
		// Check whitelist_dirs
		allowed := false
		for _, allowedDir := range t.cfg.Tools.File.WhitelistDirs {
			// Exact check: path must either be equal to allowedDir or start with allowedDir + separator
			if fullPath == allowedDir || strings.HasPrefix(fullPath, allowedDir+string(filepath.Separator)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("absolute paths are not allowed")
		}
		// Additional check for directory traversal
		if strings.Contains(cleanPath, "..") {
			return "", fmt.Errorf("path contains directory traversal attempt")
		}
	}

	// Create parent directories if they don't exist
	parentDir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create parent directories: %w", err)
	}

	// Check if file exists
	_, err = os.Stat(cleanPath)
	fileExists := err == nil

	// Handle different modes
	var file *os.File
	defer func() {
		if file != nil {
			file.Close()
		}
	}()

	switch fileArgs.Mode {
	case "create":
		if fileExists {
			return "", fmt.Errorf("file already exists and mode is 'create': %s", cleanPath)
		}
		file, err = os.Create(cleanPath)
		if err != nil {
			return "", fmt.Errorf("failed to create file: %w", err)
		}

	case "append":
		if !fileExists {
			return "", fmt.Errorf("file does not exist and mode is 'append': %s", cleanPath)
		}
		file, err = os.OpenFile(cleanPath, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return "", fmt.Errorf("failed to open file for appending: %w", err)
		}

	case "overwrite":
		file, err = os.Create(cleanPath)
		if err != nil {
			return "", fmt.Errorf("failed to create/overwrite file: %w", err)
		}

	default:
		return "", fmt.Errorf("unknown mode: %s", fileArgs.Mode)
	}

	// Write content
	if _, err := file.WriteString(fileArgs.Content); err != nil {
		return "", fmt.Errorf("failed to write content: %w", err)
	}

	// Ensure content is flushed to disk
	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(fileArgs.Content), cleanPath), nil
}
