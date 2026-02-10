package file

import (
	"encoding/json"
	"fmt"
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

// validateSkillPath validates that a skill file path follows the required format.
// Skills must be in skills/ directory and named SKILL.md.
func validateSkillPath(path string, workspaceRoot string) error {
	relPath, err := filepath.Rel(workspaceRoot, path)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if len(parts) < 2 || parts[0] != "skills" {
		return fmt.Errorf("skill files must be created in skills/ directory")
	}

	fileName := parts[len(parts)-1]
	if fileName != "SKILL.md" {
		return fmt.Errorf("skill files must be named SKILL.md, got: %s", fileName)
	}

	return nil
}

// isSkillPath checks if a path should be validated as a skill file.
// Returns true if the filename is SKILL.md.
func isSkillPath(path string) bool {
	return filepath.Base(path) == "SKILL.md"
}

// validateSkillContent validates that skill content has valid YAML frontmatter.
// This is optional and can be toggled via configuration.
func validateSkillContent(content string) error {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 {
		return fmt.Errorf("skill content must have YAML frontmatter delimited by '---'")
	}

	trimmedFirst := strings.TrimSpace(lines[0])
	if trimmedFirst != "---" {
		return fmt.Errorf("skill content must start with YAML frontmatter delimited by '---'")
	}

	foundEnd := false
	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "---" {
			foundEnd = true
			break
		}
	}

	if !foundEnd {
		return fmt.Errorf("YAML frontmatter must be closed with '---'")
	}

	return nil
}
