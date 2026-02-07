package messages

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/aatumaykin/nexbot/internal/constants"
)

// FormatError formats a general error message with error prefix.
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

// CleanContent removes LLM reasoning tags from content.
//
// This function removes full  tags from response content
// to prevent LLM "thinking" content from being sent to users.
//
// Parameters:
//   - content: The content to clean
//
// Returns:
//   - Content with  tags removed, with proper cleanup of whitespace
func CleanContent(content string) string {
	thinkOpen := string([]byte{0x3c, 0x74, 0x68, 0x69, 0x6e, 0x6b, 0x3e})
	thinkClose := string([]byte{0x3c, 0x2f, 0x74, 0x68, 0x69, 0x6e, 0x6b, 0x3e})
	thinkTagRegex := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(thinkOpen) + `.*?` + regexp.QuoteMeta(thinkClose))

	// Remove complete  tags
	cleaned := thinkTagRegex.ReplaceAllString(content, "")

	// Trim whitespace before checking for incomplete tags
	trimmed := strings.TrimSpace(cleaned)

	// Remove incomplete opening tag  at the end
	// Check if there's an unclosed  tag at the end
	lastOpenIndex := strings.LastIndex(cleaned, thinkOpen)
	if lastOpenIndex != -1 {
		// Check if there's a closing tag after this opening tag
		lastCloseIndex := strings.LastIndex(cleaned, thinkClose)
		if lastCloseIndex < lastOpenIndex {
			// No closing tag after this opening tag - it's incomplete
			// Remove everything from the opening tag to the end
			cleaned = strings.TrimSpace(cleaned[:lastOpenIndex])
		}
	}

	// Remove incomplete closing tag  at the beginning
	trimmed = strings.TrimSpace(cleaned)
	if strings.HasPrefix(trimmed, thinkClose) {
		cleaned = strings.TrimSpace(cleaned[len(thinkClose):])
	} else if strings.HasPrefix(trimmed, "\n\n"+thinkClose) || strings.HasPrefix(trimmed, "\n"+thinkClose) || strings.HasPrefix(trimmed, " "+thinkClose) {
		cleaned = regexp.MustCompile(`^\s*`+regexp.QuoteMeta(thinkClose)+`\s*\n*`).ReplaceAllString(cleaned, "")
	}

	// Trim leading/trailing whitespace after tag removal
	cleaned = strings.TrimSpace(cleaned)

	// Remove excessive newlines (more than 2 consecutive)
	cleaned = regexp.MustCompile(`\n{3,}`).ReplaceAllString(cleaned, "\n\n")

	return cleaned
}
