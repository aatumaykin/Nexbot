// Package workspace provides bootstrap files loading and assembly functionality for Nexbot.
//
// Bootstrap files are markdown files that define the system prompt and behavior:
//   - IDENTITY.md: Core identity and purpose
//   - AGENTS.md: Agent instructions and behavior guidelines
//   - SOUL.md: Personality and tone
//   - USER.md: User preferences and personalization
//   - TOOLS.md: Available tools and their usage
//
// The loader supports template variable substitution:
//   - {{CURRENT_TIME}}: Current time in ISO format
//   - {{CURRENT_DATE}}: Current date in ISO format
//   - {{WORKSPACE_PATH}}: Workspace directory path
//
// Example usage:
//
//	cfg := config.WorkspaceConfig{Path: "~/.nexbot", BootstrapMaxChars: 20000}
//	ws := workspace.New(cfg)
//	loader := workspace.NewBootstrapLoader(ws, cfg)
//
//	// Load and assemble all bootstrap files
//	content, err := loader.Assemble()
//	if err != nil {
//	    log.Fatal(err)
//	}
package workspace

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aatumaykin/nexbot/internal/config"
)

// BootstrapLoader loads and assembles bootstrap files for the system prompt.
type BootstrapLoader struct {
	workspace  *Workspace
	maxChars   int
	loggerFunc func(string, ...interface{}) // For logging warnings about missing files
}

// BootstrapFile represents a bootstrap file with its name and priority.
type BootstrapFile struct {
	Name     string
	Priority int
}

// Bootstrap files in priority order (higher priority = loaded first)
const (
	BootstrapIdentity = "IDENTITY.md"
	BootstrapAgents   = "AGENTS.md"
	BootstrapSoul     = "SOUL.md"
	BootstrapUser     = "USER.md"
	BootstrapTools    = "TOOLS.md"
)

// Default bootstrap files in priority order
var defaultBootstrapFiles = []BootstrapFile{
	{Name: BootstrapIdentity, Priority: 1},
	{Name: BootstrapAgents, Priority: 2},
	{Name: BootstrapSoul, Priority: 3},
	{Name: BootstrapUser, Priority: 4},
	{Name: BootstrapTools, Priority: 5},
}

// NewBootstrapLoader creates a new BootstrapLoader.
// If maxChars is 0, no limit is enforced.
// loggerFunc is an optional function for logging warnings (can be nil).
func NewBootstrapLoader(ws *Workspace, cfg config.WorkspaceConfig, loggerFunc func(string, ...interface{})) *BootstrapLoader {
	maxChars := cfg.BootstrapMaxChars
	if maxChars == 0 {
		maxChars = 20000 // Default from config
	}

	return &BootstrapLoader{
		workspace:  ws,
		maxChars:   maxChars,
		loggerFunc: loggerFunc,
	}
}

// LoadFile loads a single bootstrap file by name.
// Returns the file content with template variables substituted.
// Returns error if the file cannot be read (not found, permission denied, etc).
func (bl *BootstrapLoader) LoadFile(filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("filename is empty")
	}

	filePath := bl.workspace.Subpath(filename)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read bootstrap file %s: %w", filename, err)
	}

	// Substitute template variables
	substituted := bl.substituteVariables(string(content))

	return substituted, nil
}

// Load loads all bootstrap files in priority order.
// Returns a map of filename to content.
// Missing files are skipped (logged if loggerFunc is set).
func (bl *BootstrapLoader) Load() (map[string]string, error) {
	result := make(map[string]string)

	for _, bf := range defaultBootstrapFiles {
		content, err := bl.LoadFile(bf.Name)
		if err != nil {
			// Log warning but don't fail - missing files are handled gracefully
			if bl.loggerFunc != nil {
				bl.loggerFunc("failed to load bootstrap file %s: %v", bf.Name, err)
			}
			continue
		}
		result[bf.Name] = content
	}

	return result, nil
}

// Assemble loads all bootstrap files and assembles them into a single string.
// Files are concatenated in priority order with separator lines.
// Missing files are skipped.
// Content is truncated if it exceeds maxChars.
func (bl *BootstrapLoader) Assemble() (string, error) {
	// Load all bootstrap files
	files, err := bl.Load()
	if err != nil {
		return "", err
	}

	// Assemble in priority order
	var parts []string
	for _, bf := range defaultBootstrapFiles {
		if content, ok := files[bf.Name]; ok {
			parts = append(parts, content)
		}
	}

	assembled := strings.Join(parts, "\n\n---\n\n")

	// Truncate if exceeds maxChars
	if bl.maxChars > 0 && len(assembled) > bl.maxChars {
		assembled = assembled[:bl.maxChars]
		if bl.loggerFunc != nil {
			bl.loggerFunc("bootstrap content truncated to %d characters (was %d)", bl.maxChars, len(assembled))
		}
	}

	return assembled, nil
}

// substituteVariables replaces template variables in the content.
// Supported variables:
//   - {{CURRENT_TIME}}: Current time in ISO format
//   - {{CURRENT_DATE}}: Current date in ISO format
//   - {{WORKSPACE_PATH}}: Workspace directory path
func (bl *BootstrapLoader) substituteVariables(content string) string {
	now := time.Now()

	// Replace CURRENT_TIME
	content = strings.ReplaceAll(content, "{{CURRENT_TIME}}", now.Format("15:04:05"))

	// Replace CURRENT_DATE
	content = strings.ReplaceAll(content, "{{CURRENT_DATE}}", now.Format("2006-01-02"))

	// Replace WORKSPACE_PATH
	content = strings.ReplaceAll(content, "{{WORKSPACE_PATH}}", bl.workspace.Path())

	return content
}

// SetMaxChars sets the maximum number of characters for assembled content.
// If set to 0, no limit is enforced.
func (bl *BootstrapLoader) SetMaxChars(maxChars int) {
	bl.maxChars = maxChars
}

// GetMaxChars returns the current maximum character limit.
func (bl *BootstrapLoader) GetMaxChars() int {
	return bl.maxChars
}
