package file

import (
	"encoding/json"
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
