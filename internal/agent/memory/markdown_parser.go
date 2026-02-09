package memory

import (
	"strings"

	"github.com/aatumaykin/nexbot/internal/llm"
)

// MarkdownParser parses markdown-formatted memory content into LLM messages.
type MarkdownParser struct{}

// NewMarkdownParser creates a new markdown parser instance.
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{}
}

// Parse converts markdown content into a slice of LLM messages.
func (p *MarkdownParser) Parse(content string) []llm.Message {
	var messages []llm.Message

	lines := strings.Split(content, "\n")
	var currentRole llm.Role
	var currentContent strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect headers to identify role
		if strings.HasPrefix(trimmed, "### User [") {
			if currentRole != "" && currentContent.Len() > 0 {
				messages = append(messages, llm.Message{
					Role:    currentRole,
					Content: strings.TrimSpace(currentContent.String()),
				})
			}
			currentRole = llm.RoleUser
			currentContent.Reset()
			continue
		} else if strings.HasPrefix(trimmed, "### Assistant [") {
			if currentRole != "" && currentContent.Len() > 0 {
				messages = append(messages, llm.Message{
					Role:    currentRole,
					Content: strings.TrimSpace(currentContent.String()),
				})
			}
			currentRole = llm.RoleAssistant
			currentContent.Reset()
			continue
		} else if strings.HasPrefix(trimmed, "## System [") {
			if currentRole != "" && currentContent.Len() > 0 {
				messages = append(messages, llm.Message{
					Role:    currentRole,
					Content: strings.TrimSpace(currentContent.String()),
				})
			}
			currentRole = llm.RoleSystem
			currentContent.Reset()
			continue
		} else if strings.HasPrefix(trimmed, "#### Tool:") {
			if currentRole != "" && currentContent.Len() > 0 {
				messages = append(messages, llm.Message{
					Role:    currentRole,
					Content: strings.TrimSpace(currentContent.String()),
				})
			}
			// Extract tool call ID
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				toolCallID := strings.TrimSuffix(parts[2], "]")
				// Create tool message and add it immediately
				messages = append(messages, llm.Message{
					Role:       llm.RoleTool,
					ToolCallID: toolCallID,
					Content:    "",
				})
			}
			currentContent.Reset()
			// Reset currentRole to avoid adding duplicate
			currentRole = ""
			continue
		}

		// Skip header lines
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		currentContent.WriteString(line)
		currentContent.WriteString("\n")
	}

	// Add last message if exists
	if currentRole != "" && currentContent.Len() > 0 {
		messages = append(messages, llm.Message{
			Role:    currentRole,
			Content: strings.TrimSpace(currentContent.String()),
		})
	}

	return messages
}
