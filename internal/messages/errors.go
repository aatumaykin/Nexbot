package messages

import (
	"fmt"
	"strings"

	"github.com/aatumaykin/nexbot/internal/constants"
)

// FormatError formats a general error message with the error prefix.
//
// Parameters:
//   - err: The error to format
//
// Returns:
//   - Formatted error string with "Error: " prefix
func FormatError(err error) string {
	return fmt.Sprintf(constants.MsgErrorFormat, err)
}

// FormatConfigLoadError formats a configuration loading error message.
//
// Parameters:
//   - err: The error that occurred during configuration loading
//
// Returns:
//   - Formatted configuration load error string
func FormatConfigLoadError(err error) string {
	return fmt.Sprintf(constants.MsgConfigLoadError, err)
}

// FormatValidationErrors formats a list of validation errors with numbering.
//
// Parameters:
//   - errs: Slice of validation errors to format
//
// Returns:
//   - Formatted string with all validation errors numbered (1, 2, 3...)
func FormatValidationErrors(errs []error) string {
	if len(errs) == 0 {
		return ""
	}

	builder := &strings.Builder{}

	// Add validation error header
	builder.WriteString(constants.MsgConfigValidationError)

	// Add each validation error with numbering
	for i, err := range errs {
		builder.WriteString(fmt.Sprintf(constants.MsgConfigValidatePrefix, fmt.Sprintf("%d. %v", i+1, err)))
	}

	return builder.String()
}
