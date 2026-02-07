package tools

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSystemTimeTool creates a SystemTimeTool for testing.
func setupSystemTimeTool(t *testing.T) *SystemTimeTool {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	return NewSystemTimeTool(log)
}

// TestSystemTimeToolName tests that tool returns correct name.
func TestSystemTimeToolName(t *testing.T) {
	tool := setupSystemTimeTool(t)
	assert.Equal(t, "system_time", tool.Name(), "Tool name should be 'system_time'")
}

// TestSystemTimeToolDescription tests that tool returns a non-empty description.
func TestSystemTimeToolDescription(t *testing.T) {
	tool := setupSystemTimeTool(t)
	desc := tool.Description()
	assert.NotEmpty(t, desc, "Description should not be empty")
	assert.Contains(t, desc, "время", "Description should mention 'время'")
	assert.Contains(t, desc, "дату", "Description should mention 'дату'")
}

// TestSystemTimeToolParameters tests that tool returns valid parameters.
func TestSystemTimeToolParameters(t *testing.T) {
	tool := setupSystemTimeTool(t)
	params := tool.Parameters()

	assert.NotNil(t, params, "Parameters should not be nil")
	assert.Equal(t, "object", params["type"], "Type should be 'object'")

	props, ok := params["properties"].(map[string]interface{})
	assert.True(t, ok, "Properties should be a map")
	assert.Empty(t, props, "Properties should be empty")

	// Check required fields - try both types
	required := params["required"]
	switch v := required.(type) {
	case []interface{}:
		assert.Empty(t, v, "Required should be empty")
	case []string:
		assert.Empty(t, v, "Required should be empty")
	default:
		assert.Fail(t, "Required should be a slice")
	}
}

// TestSystemTimeToolExecute tests that execution returns time in correct format.
func TestSystemTimeToolExecute(t *testing.T) {
	tool := setupSystemTimeTool(t)

	result, err := tool.Execute("")
	assert.NoError(t, err, "Execute should not return error")
	assert.NotEmpty(t, result, "Result should not be empty")

	// Check that result contains RFC3339 format
	lines := strings.Split(result, "\n")
	assert.Len(t, lines, 2, "Result should have 2 lines")

	// First line should be RFC3339 format
	assert.Contains(t, lines[0], "RFC3339:", "First line should contain 'RFC3339:'")
	rfc3339Line := strings.TrimSpace(strings.TrimPrefix(lines[0], "RFC3339:"))

	// RFC3339 pattern: 2006-01-02T15:04:05-07:00
	rfc3339Pattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$`)
	assert.True(t, rfc3339Pattern.MatchString(rfc3339Line), "RFC3339 format should match pattern")

	// Verify the time is parseable
	_, err = time.Parse(time.RFC3339, rfc3339Line)
	assert.NoError(t, err, "RFC3339 time should be parseable")

	// Second line should be human readable format
	assert.Contains(t, lines[1], "Human readable:", "Second line should contain 'Human readable:'")
	humanReadableLine := strings.TrimSpace(strings.TrimPrefix(lines[1], "Human readable:"))
	assert.NotEmpty(t, humanReadableLine, "Human readable time should not be empty")
}

// TestSystemTimeToolToSchema tests that ToSchema returns correct schema.
func TestSystemTimeToolToSchema(t *testing.T) {
	tool := setupSystemTimeTool(t)
	schema := tool.ToSchema()
	assert.NotNil(t, schema, "Schema should not be nil")
	assert.Equal(t, tool.Parameters(), schema, "Schema should match parameters")
}
