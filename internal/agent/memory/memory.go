package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aatumaykin/nexbot/internal/llm"
)

// Format represents the storage format for memory files.
type Format string

const (
	FormatJSONL    Format = "jsonl"    // JSONL format
	FormatMarkdown Format = "markdown" // Markdown format
)

// Store manages message history storage in various formats.
type Store struct {
	baseDir string // Base directory for memory files
	format  Format // Storage format (jsonl or markdown)
	mu      sync.RWMutex
}

// Config holds configuration for the memory store.
type Config struct {
	BaseDir string // Base directory for memory files
	Format  Format // Storage format
}

// NewStore creates a new memory store with the specified configuration.
func NewStore(config Config) (*Store, error) {
	if config.BaseDir == "" {
		return nil, fmt.Errorf("base directory cannot be empty")
	}

	if config.Format == "" {
		config.Format = FormatJSONL
	}

	// Ensure base directory exists
	if err := os.MkdirAll(config.BaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &Store{
		baseDir: config.BaseDir,
		format:  config.Format,
	}, nil
}

// Write adds a message to the memory store for a given session.
func (s *Store) Write(sessionID string, msg llm.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.format {
	case FormatJSONL:
		return s.writeJSONL(sessionID, msg)
	case FormatMarkdown:
		return s.writeMarkdown(sessionID, msg)
	default:
		return fmt.Errorf("unsupported format: %s", s.format)
	}
}

// Read retrieves all messages for a given session.
func (s *Store) Read(sessionID string) ([]llm.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch s.format {
	case FormatJSONL:
		return s.readJSONL(sessionID)
	case FormatMarkdown:
		return s.readMarkdown(sessionID)
	default:
		return nil, fmt.Errorf("unsupported format: %s", s.format)
	}
}

// Append adds multiple messages to the memory store for a given session.
func (s *Store) Append(sessionID string, messages []llm.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.format {
	case FormatJSONL:
		return s.appendJSONL(sessionID, messages)
	case FormatMarkdown:
		return s.appendMarkdown(sessionID, messages)
	default:
		return fmt.Errorf("unsupported format: %s", s.format)
	}
}

// GetLastN retrieves the last N messages for a given session.
func (s *Store) GetLastN(sessionID string, n int) ([]llm.Message, error) {
	allMessages, err := s.Read(sessionID)
	if err != nil {
		return nil, err
	}

	if len(allMessages) <= n {
		return allMessages, nil
	}

	return allMessages[len(allMessages)-n:], nil
}

// Clear removes all messages for a given session.
func (s *Store) Clear(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.format {
	case FormatJSONL:
		return s.clearJSONL(sessionID)
	case FormatMarkdown:
		return s.clearMarkdown(sessionID)
	default:
		return fmt.Errorf("unsupported format: %s", s.format)
	}
}

// Exists checks if memory exists for a given session.
func (s *Store) Exists(sessionID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch s.format {
	case FormatJSONL:
		return s.existsJSONL(sessionID)
	case FormatMarkdown:
		return s.existsMarkdown(sessionID)
	default:
		return false
	}
}

// GetSessions returns all session IDs that have stored memory.
func (s *Store) GetSessions() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read memory directory: %w", err)
	}

	var sessions []string
	suffix := "." + string(s.format)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, suffix) {
			sessionID := strings.TrimSuffix(name, suffix)
			sessions = append(sessions, sessionID)
		}
	}

	return sessions, nil
}

// JSONL implementation

func (s *Store) writeJSONL(sessionID string, msg llm.Message) error {
	filePath := s.getFilePath(sessionID)

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Open file in append mode
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

func (s *Store) readJSONL(sessionID string) ([]llm.Message, error) {
	filePath := s.getFilePath(sessionID)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []llm.Message{}, nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var messages []llm.Message
	lines := splitLines(data)

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var msg llm.Message
		if err := json.Unmarshal(line, &msg); err != nil {
			// Skip malformed lines
			continue
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (s *Store) appendJSONL(sessionID string, messages []llm.Message) error {
	filePath := s.getFilePath(sessionID)

	// Open file in append mode
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}

		if _, err := file.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}
	}

	return nil
}

func (s *Store) clearJSONL(sessionID string) error {
	filePath := s.getFilePath(sessionID)

	if err := os.WriteFile(filePath, []byte{}, 0644); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear file: %w", err)
	}

	return nil
}

func (s *Store) existsJSONL(sessionID string) bool {
	filePath := s.getFilePath(sessionID)
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// Markdown implementation

func (s *Store) writeMarkdown(sessionID string, msg llm.Message) error {
	filePath := s.getFilePath(sessionID)

	// Open file in append mode
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	line := s.formatMessageAsMarkdown(msg)
	if _, err := file.WriteString(line); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

func (s *Store) readMarkdown(sessionID string) ([]llm.Message, error) {
	filePath := s.getFilePath(sessionID)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []llm.Message{}, nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return s.parseMarkdown(string(data)), nil
}

func (s *Store) appendMarkdown(sessionID string, messages []llm.Message) error {
	filePath := s.getFilePath(sessionID)

	// Open file in append mode
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	for _, msg := range messages {
		line := s.formatMessageAsMarkdown(msg)
		if _, err := file.WriteString(line); err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}
	}

	return nil
}

func (s *Store) clearMarkdown(sessionID string) error {
	filePath := s.getFilePath(sessionID)

	if err := os.WriteFile(filePath, []byte{}, 0644); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear file: %w", err)
	}

	return nil
}

func (s *Store) existsMarkdown(sessionID string) bool {
	filePath := s.getFilePath(sessionID)
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func (s *Store) formatMessageAsMarkdown(msg llm.Message) string {
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

func (s *Store) parseMarkdown(content string) []llm.Message {
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
				currentRole = llm.RoleTool
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

// Helper functions

func (s *Store) getFilePath(sessionID string) string {
	return filepath.Join(s.baseDir, sessionID+"."+string(s.format))
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0

	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		} else if b == '\r' && i+1 < len(data) && data[i+1] == '\n' {
			lines = append(lines, data[start:i])
			start = i + 2
		}
	}

	// Add last line if not empty
	if start < len(data) {
		lines = append(lines, data[start:])
	}

	return lines
}
