package messages

import (
	"fmt"
	"strings"

	"github.com/aatumaykin/nexbot/internal/constants"
)

// FormatStatusMessage formats the session status message with session info and LLM configuration.
//
// Parameters:
//   - sessionID: Unique identifier for the current session
//   - msgCount: Total number of messages in the session
//   - fileSizeHuman: Human-readable file size string (e.g., "1.2 MB")
//   - model: LLM model name being used
//   - temperature: Temperature parameter for LLM generation
//   - maxTokens: Maximum tokens allowed for generation
//
// Returns:
//   - Formatted status message string ready for display
func FormatStatusMessage(
	sessionID string,
	msgCount int,
	fileSizeHuman string,
	model string,
	temperature float64,
	maxTokens int,
) string {
	// Start building the status message
	builder := &strings.Builder{}

	// Add status header
	builder.WriteString(constants.MsgStatusHeader)

	// Add session information
	builder.WriteString(fmt.Sprintf(constants.MsgStatusSessionID, sessionID))
	builder.WriteString(fmt.Sprintf(constants.MsgStatusMessages, msgCount))
	builder.WriteString(fmt.Sprintf(constants.MsgStatusSessionSize, fileSizeHuman))

	// Add LLM configuration header
	builder.WriteString(constants.MsgStatusLLMConfig)

	// Add LLM configuration details
	builder.WriteString(fmt.Sprintf(constants.MsgStatusModel, model))
	builder.WriteString(fmt.Sprintf(constants.MsgStatusTemp, temperature))
	builder.WriteString(fmt.Sprintf(constants.MsgStatusMaxTokens, maxTokens))

	return builder.String()
}
