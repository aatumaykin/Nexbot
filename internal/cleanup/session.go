package cleanup

import (
	"os"
	"path/filepath"
	"time"
)

// SessionInfo holds information about a session file.
type SessionInfo struct {
	ID        string
	Path      string
	Size      int64
	ModTime   time.Time
	LineCount int
}

// GetSessionDir returns the path to the sessions directory.
func (r *Runner) GetSessionDir(workspacePath string) string {
	return filepath.Join(workspacePath, "sessions")
}

// ListSessions lists all session files in the sessions directory.
func (r *Runner) ListSessions(sessionDir string) ([]SessionInfo, error) {
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return nil, err
	}

	var sessions []SessionInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		filePath := filepath.Join(sessionDir, entry.Name())
		sessionID := entry.Name()

		// Remove file extension
		if ext := filepath.Ext(sessionID); ext != "" {
			sessionID = sessionID[:len(sessionID)-len(ext)]
		}

		lineCount, err := r.countLines(filePath)
		if err != nil {
			lineCount = 0
		}

		sessions = append(sessions, SessionInfo{
			ID:        sessionID,
			Path:      filePath,
			Size:      info.Size(),
			ModTime:   info.ModTime(),
			LineCount: lineCount,
		})
	}

	return sessions, nil
}

// countLines counts the number of lines in a file.
func (r *Runner) countLines(filePath string) (int, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}

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

// ShouldCleanup determines if a session should be cleaned up.
func (r *Runner) ShouldCleanup(session SessionInfo, activeSessions map[string]bool) bool {
	// Never cleanup active sessions
	if activeSessions[session.ID] {
		return false
	}

	now := time.Now()

	// Check session TTL
	if r.config.SessionTTLDays > 0 {
		ttl := time.Duration(r.config.SessionTTLDays) * 24 * time.Hour
		if now.Sub(session.ModTime) > ttl {
			return true
		}
	}

	// Check max session size
	if r.config.MaxSessionSizeMB > 0 {
		maxSize := r.config.MaxSessionSizeMB * 1024 * 1024
		// Only apply size limit after keep active period
		if r.config.KeepActiveDays > 0 {
			keepActive := time.Duration(r.config.KeepActiveDays) * 24 * time.Hour
			if now.Sub(session.ModTime) > keepActive && session.Size > maxSize {
				return true
			}
		} else if session.Size > maxSize {
			return true
		}
	}

	return false
}

// DeleteSession removes a session file.
func (r *Runner) DeleteSession(sessionPath string) error {
	if err := os.Remove(sessionPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
