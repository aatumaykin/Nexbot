package memory

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aatumaykin/nexbot/internal/llm"
)

// StorageFormat defines the interface for memory storage formats
type StorageFormat interface {
	// FormatMessage converts a message to storage format
	FormatMessage(msg llm.Message) string

	// ParseMessage parses a message from storage format
	ParseMessage(line string) (llm.Message, error)

	// FileExtension returns the file extension for this format
	FileExtension() string
}

// JSONLFormat implements StorageFormat for JSONL
type JSONLFormat struct{}

func (f *JSONLFormat) FormatMessage(msg llm.Message) string {
	data, _ := json.Marshal(msg)
	return string(data) + "\n"
}

func (f *JSONLFormat) ParseMessage(line string) (llm.Message, error) {
	var msg llm.Message
	err := json.Unmarshal([]byte(line), &msg)
	return msg, err
}

func (f *JSONLFormat) FileExtension() string {
	return ".jsonl"
}

// MarkdownFormat implements StorageFormat for Markdown
type MarkdownFormat struct{}

func (f *MarkdownFormat) FormatMessage(msg llm.Message) string {
	var sb strings.Builder
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	switch msg.Role {
	case llm.RoleSystem:
		sb.WriteString(fmt.Sprintf("\n## System [%s]\n\n%s\n", timestamp, msg.Content))
	case llm.RoleUser:
		sb.WriteString(fmt.Sprintf("\n### User [%s]\n\n%s\n", timestamp, msg.Content))
	case llm.RoleAssistant:
		sb.WriteString(fmt.Sprintf("\n### Assistant [%s]\n\n%s\n", timestamp, msg.Content))
	case llm.RoleTool:
		sb.WriteString(fmt.Sprintf("\n#### Tool: %s [%s]\n\n%s\n", msg.ToolCallID, timestamp, msg.Content))
	default:
		sb.WriteString(fmt.Sprintf("\n### %s [%s]\n\n%s\n", msg.Role, timestamp, msg.Content))
	}

	return sb.String()
}

func (f *MarkdownFormat) ParseMessage(line string) (llm.Message, error) {
	// Markdown parsing is more complex and requires full content
	// This is handled in parseMarkdown function
	return llm.Message{}, nil
}

func (f *MarkdownFormat) FileExtension() string {
	return ".markdown"
}

// NewStorageFormat creates a StorageFormat instance from a Format string
func NewStorageFormat(fmt Format) StorageFormat {
	switch fmt {
	case FormatJSONL:
		return &JSONLFormat{}
	case FormatMarkdown:
		return &MarkdownFormat{}
	default:
		return &JSONLFormat{} // Default to JSONL
	}
}
