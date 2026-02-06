package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

// ReadFileTool implements the Tool interface for reading file contents.
// It reads a file from the workspace and returns its content.
type ReadFileTool struct {
	fileToolBase
}

// ReadFileArgs represents the arguments for the read_file tool.
type ReadFileArgs struct {
	Path     string `json:"path"`               // Path to the file (relative to workspace or absolute)
	Offset   int    `json:"offset,omitempty"`   // Line offset (0-based, defaults to 0)
	Limit    int    `json:"limit,omitempty"`    // Maximum number of lines to read (defaults to 2000)
	Encoding string `json:"encoding,omitempty"` // File encoding (defaults to "utf-8", only utf-8 is supported currently)
}

// NewReadFileTool creates a new ReadFileTool instance.
// The workspace parameter is used for resolving relative paths.
// The config parameter provides the file tool configuration (whitelist_dirs, etc.).
func NewReadFileTool(ws *workspace.Workspace, cfg *config.Config) *ReadFileTool {
	return &ReadFileTool{
		fileToolBase: fileToolBase{
			workspace: ws,
			cfg:       cfg,
		},
	}
}

// Name returns the tool name.
func (t *ReadFileTool) Name() string {
	return "read_file"
}

// Description returns a description of what the tool does.
func (t *ReadFileTool) Description() string {
	return "Reads the contents of a file from the workspace. Returns file content with line numbers. Use this tool when you need to examine file contents."
}

// Parameters returns the JSON Schema for the tool's parameters.
func (t *ReadFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The path to the file to read. Can be absolute or relative to the workspace directory.",
			},
			"offset": map[string]interface{}{
				"type":        "integer",
				"description": "The line number to start reading from (0-based). Defaults to 0.",
				"default":     0,
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "The maximum number of lines to read. Defaults to 2000.",
				"default":     2000,
			},
			"encoding": map[string]interface{}{
				"type":        "string",
				"description": "The file encoding. Currently only 'utf-8' is supported.",
				"default":     "utf-8",
			},
		},
		"required": []string{"path"},
	}
}

// Execute reads the file content and returns it.
// args is a JSON-encoded string containing the tool's input parameters.
func (t *ReadFileTool) Execute(args string) (string, error) {
	// Parse arguments
	var fileArgs ReadFileArgs
	if err := parseJSON(args, &fileArgs); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Validate arguments
	if fileArgs.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Set defaults
	if fileArgs.Limit <= 0 {
		fileArgs.Limit = 2000
	}

	if fileArgs.Offset < 0 {
		fileArgs.Offset = 0
	}

	if fileArgs.Encoding == "" {
		fileArgs.Encoding = "utf-8"
	}

	// Validate encoding
	if fileArgs.Encoding != "utf-8" {
		return "", fmt.Errorf("unsupported encoding: %s (only utf-8 is supported)", fileArgs.Encoding)
	}

	// Resolve path
	var fullPath string
	var err error

	if filepath.IsAbs(fileArgs.Path) {
		// Clean the path first to normalize it
		cleanPath := filepath.Clean(fileArgs.Path)
		// Check for directory traversal attempts
		if strings.Contains(cleanPath, "..") {
			return "", fmt.Errorf("path contains directory traversal attempt")
		}
		// Check whitelist_dirs on the clean path
		allowed := false
		for _, allowedDir := range t.cfg.Tools.File.WhitelistDirs {
			// Exact check: path must either be equal to allowedDir or start with allowedDir + separator
			if cleanPath == allowedDir || strings.HasPrefix(cleanPath, allowedDir+string(filepath.Separator)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("absolute path is not in whitelist_dirs")
		}
		fullPath = cleanPath
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

	// Check if file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", fullPath)
		}
		return "", fmt.Errorf("failed to access file: %w", err)
	}

	// Check if it's a regular file
	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file: %s", fullPath)
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Split into lines and apply offset/limit
	lines := splitLines(string(content))

	// Apply offset
	if fileArgs.Offset >= len(lines) {
		return fmt.Sprintf("# File: %s\n# Offset %d is beyond file length (%d lines)\n",
			filepath.Clean(fullPath), fileArgs.Offset, len(lines)), nil
	}

	startLine := fileArgs.Offset
	endLine := startLine + fileArgs.Limit

	if endLine > len(lines) {
		endLine = len(lines)
	}

	selectedLines := lines[startLine:endLine]

	// Format output with line numbers
	result := fmt.Sprintf("# File: %s (lines %d-%d of %d)\n",
		filepath.Clean(fullPath), startLine+1, endLine, len(lines))

	for i, line := range selectedLines {
		lineNum := startLine + i + 1
		result += fmt.Sprintf("%06d| %s\n", lineNum, line)
	}

	return result, nil
}
