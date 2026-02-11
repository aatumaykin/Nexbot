// Package subagent provides session management for subagents.
// Subagent sessions are isolated from main agent sessions and stored
// in a separate directory to maintain separation of concerns.
package subagent

import (
	"fmt"
	"os"
	"path/filepath"
)

// Storage handles storage operations for subagent sessions.
// It provides isolated storage per subagent using JSONL format.
type Storage struct {
	baseDir string
}

// NewStorage creates a new subagent session storage.
func NewStorage(baseDir string) (*Storage, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("base directory cannot be empty")
	}

	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &Storage{
		baseDir: baseDir,
	}, nil
}

// Save saves a session entry to storage.
// The entry is appended to the subagent's JSONL file.
func (s *Storage) Save(subagentID string, entry any) error {
	// Create subagent session directory if not exists
	subagentPath := filepath.Join(s.baseDir, subagentID)
	if err := os.MkdirAll(subagentPath, 0755); err != nil {
		return fmt.Errorf("failed to create subagent directory: %w", err)
	}

	// Create session file path
	sessionFile := filepath.Join(subagentPath, "session.jsonl")

	// In a full implementation, this would serialize entry and append to JSONL file
	// For now, just ensure the file exists
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		if err := os.WriteFile(sessionFile, []byte(""), 0644); err != nil {
			return fmt.Errorf("failed to create session file: %w", err)
		}
	}

	return nil
}

// Load loads session entries for a subagent.
// Returns entries from the subagent's JSONL file.
func (s *Storage) Load(subagentID string) ([]any, error) {
	// Create subagent session path
	subagentPath := filepath.Join(s.baseDir, subagentID)
	sessionFile := filepath.Join(subagentPath, "session.jsonl")

	// Check if file exists
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		// File doesn't exist yet, return empty slice
		return []any{}, nil
	}

	// In a full implementation, this would read and parse JSONL file
	// For now, return empty slice
	return []any{}, nil
}

// Delete deletes all session data for a subagent.
// This removes the subagent's session directory.
func (s *Storage) Delete(subagentID string) error {
	subagentPath := filepath.Join(s.baseDir, subagentID)

	// Check if directory exists
	if _, err := os.Stat(subagentPath); os.IsNotExist(err) {
		// Directory doesn't exist, nothing to delete
		return nil
	}

	// Remove subagent directory
	if err := os.RemoveAll(subagentPath); err != nil {
		return fmt.Errorf("failed to delete subagent session: %w", err)
	}

	return nil
}

// List lists all subagent session directories.
// Returns a slice of subagent IDs.
func (s *Storage) List() ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	subagentIDs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			subagentIDs = append(subagentIDs, entry.Name())
		}
	}

	return subagentIDs, nil
}
