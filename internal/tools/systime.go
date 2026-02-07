package tools

import (
	"fmt"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

// SystemTimeArgs represents the arguments for the system_time tool.
type SystemTimeArgs struct{}

// SystemTimeTool implements the Tool interface for getting system time.
type SystemTimeTool struct {
	logger *logger.Logger
}

// NewSystemTimeTool creates a new SystemTimeTool instance.
func NewSystemTimeTool(logger *logger.Logger) *SystemTimeTool {
	return &SystemTimeTool{
		logger: logger,
	}
}

// Name returns the tool name.
func (t *SystemTimeTool) Name() string {
	return "system_time"
}

// Description returns a description of what the tool does.
func (t *SystemTimeTool) Description() string {
	return "Возвращает текущее системное время и дату"
}

// Parameters returns the JSON Schema for the tool's parameters.
func (t *SystemTimeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

// ToSchema returns the OpenAI-compatible schema for this tool.
func (t *SystemTimeTool) ToSchema() map[string]interface{} {
	return t.Parameters()
}

// Execute executes the system time tool.
func (t *SystemTimeTool) Execute(args string) (string, error) {
	now := time.Now().Local()

	result := fmt.Sprintf("RFC3339: %s\n", now.Format(time.RFC3339))
	result += fmt.Sprintf("Human readable: %s", now.Format("Monday, 02 January 2006, 15:04:05 MST"))

	return result, nil
}
