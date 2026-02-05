package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aatumaykin/nexbot/internal/llm"
)

// Session represents a chat session with messages stored in JSONL format.
type Session struct {
	ID     string     // Unique session identifier
	File   string     // Path to JSONL file
	mu     sync.Mutex // Protects file operations
	loaded bool       // Track if session was just created
}

// Entry represents a single entry in the JSONL session file.
type Entry struct {
	Message   llm.Message `json:"message"`
	Timestamp string      `json:"timestamp,omitempty"`
	Metadata  interface{} `json:"metadata,omitempty"`
}

// Manager manages sessions stored as JSONL files.
type Manager struct {
	baseDir string // Base directory for session files
	mu      sync.RWMutex
}

// NewManager creates a new session manager with the specified base directory.
func NewManager(baseDir string) (*Manager, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("base directory cannot be empty")
	}

	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &Manager{
		baseDir: baseDir,
	}, nil
}

// Exists проверяет существует ли сессия
// Returns true if session file exists, false otherwise
func (m *Manager) Exists(sessionID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessionFile := filepath.Join(m.baseDir, sessionID+".jsonl")
	_, err := os.Stat(sessionFile)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetOrCreate retrieves an existing session or creates a new one.
// Returns the session and a boolean indicating whether it was newly created.
func (m *Manager) GetOrCreate(sessionID string) (*Session, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessionFile := filepath.Join(m.baseDir, sessionID+".jsonl")

	// Check if session file exists
	_, err := os.Stat(sessionFile)
	if os.IsNotExist(err) {
		// Create new session
		session := &Session{
			ID:     sessionID,
			File:   sessionFile,
			loaded: false,
		}

		// Create empty file
		if err := os.WriteFile(sessionFile, []byte{}, 0644); err != nil {
			return nil, false, fmt.Errorf("failed to create session file: %w", err)
		}

		return session, true, nil
	}

	if err != nil {
		return nil, false, fmt.Errorf("failed to check session file: %w", err)
	}

	// Return existing session
	return &Session{
		ID:     sessionID,
		File:   sessionFile,
		loaded: true,
	}, false, nil
}

// Append adds a message to the session.
// The message is appended as a JSON line to the session file.
func (s *Session) Append(msg llm.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := Entry{
		Message:   msg,
		Timestamp: "", // TODO: Add timestamp if needed
		Metadata:  nil,
	}

	// Marshal entry to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Open file in append mode
	file, err := os.OpenFile(s.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	// Append JSON line with newline
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// Read reads all messages from the session.
// Returns messages in chronological order (as they were appended).
func (s *Session) Read() ([]llm.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read file content
	data, err := os.ReadFile(s.File)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var messages []llm.Message

	// Parse JSONL line by line
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			// Skip malformed lines
			continue
		}

		messages = append(messages, entry.Message)
	}

	return messages, nil
}

// splitLines splits byte data into lines, handling both \n and \r\n.
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

// Delete removes the session file.
func (s *Session) Delete() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Remove(s.File); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	return nil
}

// Exists checks if the session file exists.
func (s *Session) Exists() bool {
	_, err := os.Stat(s.File)
	return !os.IsNotExist(err)
}

// MessageCount returns the number of messages in the session.
func (s *Session) MessageCount() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.File)
	if err != nil {
		return 0, fmt.Errorf("failed to read session file: %w", err)
	}

	// Count non-empty lines
	count := 0
	for _, b := range data {
		if b == '\n' {
			count++
		}
	}

	// Account for file that might not end with newline
	if len(data) > 0 && data[len(data)-1] != '\n' {
		count++
	}

	return count, nil
}

// Clear removes all messages from the session.
func (s *Session) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.WriteFile(s.File, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to clear session file: %w", err)
	}

	return nil
}
