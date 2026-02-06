package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	baseDir string        // Base directory for memory files
	format  StorageFormat // Storage format interface
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
		format:  NewStorageFormat(config.Format),
	}, nil
}

// Write adds a message to the memory store for a given session.
func (s *Store) Write(sessionID string, msg llm.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := s.getFilePath(sessionID)

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	line := s.format.FormatMessage(msg)
	if _, err := file.WriteString(line); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	return nil
}

// Read retrieves all messages for a given session.
func (s *Store) Read(sessionID string) ([]llm.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := s.getFilePath(sessionID)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []llm.Message{}, nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Special handling for Markdown format (requires full content parsing)
	if _, isMarkdown := s.format.(*MarkdownFormat); isMarkdown {
		return s.parseMarkdown(string(data)), nil
	}

	// Default line-by-line parsing for JSONL and other formats
	var messages []llm.Message
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		msg, err := s.format.ParseMessage(line)
		if err != nil {
			continue // Skip malformed lines
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// Append adds multiple messages to the memory store for a given session.
func (s *Store) Append(sessionID string, messages []llm.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := s.getFilePath(sessionID)

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	for _, msg := range messages {
		line := s.format.FormatMessage(msg)
		if _, err := file.WriteString(line); err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}
	}

	return nil
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

	filePath := s.getFilePath(sessionID)

	if err := os.WriteFile(filePath, []byte{}, 0644); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear file: %w", err)
	}

	return nil
}

// Exists checks if memory exists for a given session.
func (s *Store) Exists(sessionID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := s.getFilePath(sessionID)
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
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
	suffix := s.format.FileExtension()

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

// Markdown parsing helpers

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
	return filepath.Join(s.baseDir, sessionID+s.format.FileExtension())
}
