package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aatumaykin/nexbot/internal/workspace"
)

// ReadFileTool implements the Tool interface for reading file contents.
// It reads a file from the workspace and returns its content.
type ReadFileTool struct {
	workspace *workspace.Workspace
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
func NewReadFileTool(ws *workspace.Workspace) *ReadFileTool {
	return &ReadFileTool{
		workspace: ws,
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

// WriteFileTool implements the Tool interface for writing file content.
// It writes content to a file in the workspace.
type WriteFileTool struct {
	workspace *workspace.Workspace
}

// WriteFileArgs represents the arguments for the write_file tool.
type WriteFileArgs struct {
	Path    string `json:"path"`           // Path to the file (relative to workspace or absolute)
	Content string `json:"content"`        // Content to write to the file
	Mode    string `json:"mode,omitempty"` // Write mode: "create" (default), "append", "overwrite"
}

// NewWriteFileTool creates a new WriteFileTool instance.
// The workspace parameter is used for resolving relative paths.
func NewWriteFileTool(ws *workspace.Workspace) *WriteFileTool {
	return &WriteFileTool{
		workspace: ws,
	}
}

// Name returns the tool name.
func (t *WriteFileTool) Name() string {
	return "write_file"
}

// Description returns a description of what the tool does.
func (t *WriteFileTool) Description() string {
	return "Writes content to a file in the workspace. Supports creating new files, appending to existing files, or overwriting files. Creates parent directories if they don't exist."
}

// Parameters returns the JSON Schema for the tool's parameters.
func (t *WriteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The path to the file to write. Can be absolute or relative to the workspace directory.",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The content to write to the file.",
			},
			"mode": map[string]interface{}{
				"type":        "string",
				"description": "Write mode: 'create' (fails if file exists), 'append' (append to existing file), 'overwrite' (replace file content). Defaults to 'create'.",
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

	// Check for directory traversal attempts
	if filepath.IsAbs(fileArgs.Path) {
		// For absolute paths, verify it doesn't escape allowed directories
		// (This is a simplified check - in production you'd want more robust validation)
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

// ListDirTool implements the Tool interface for listing directory contents.
// It lists files and directories in a workspace directory.
type ListDirTool struct {
	workspace *workspace.Workspace
}

// ListDirArgs represents the arguments for the list_dir tool.
type ListDirArgs struct {
	Path          string `json:"path"`                     // Path to the directory (relative to workspace or absolute)
	Recursive     bool   `json:"recursive,omitempty"`      // Whether to list recursively (default: false)
	IncludeHidden bool   `json:"include_hidden,omitempty"` // Whether to include hidden files/directories (default: false)
}

// NewListDirTool creates a new ListDirTool instance.
// The workspace parameter is used for resolving relative paths.
func NewListDirTool(ws *workspace.Workspace) *ListDirTool {
	return &ListDirTool{
		workspace: ws,
	}
}

// Name returns the tool name.
func (t *ListDirTool) Name() string {
	return "list_dir"
}

// Description returns a description of what the tool does.
func (t *ListDirTool) Description() string {
	return "Lists the contents of a directory in the workspace. Can list recursively and optionally include hidden files."
}

// Parameters returns the JSON Schema for the tool's parameters.
func (t *ListDirTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The path to the directory to list. Can be absolute or relative to the workspace directory.",
			},
			"recursive": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to list directory contents recursively.",
				"default":     false,
			},
			"include_hidden": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to include hidden files and directories (those starting with '.').",
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
	result := fmt.Sprintf("# Directory: %s\n", filepath.Clean(cleanPath))
	if dirArgs.Recursive {
		result += fmt.Sprintf("# Recursive: %d items\n\n", len(entries))
	} else {
		result += fmt.Sprintf("# %d items\n\n", len(entries))
	}

	for _, entry := range entries {
		result += entry + "\n"
	}

	return result, nil
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
