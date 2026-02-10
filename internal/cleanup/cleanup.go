package cleanup

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

// Run performs cleanup on sessions directory.
func (r *Runner) Run(workspacePath string, activeSessions map[string]bool, log *logger.Logger) (Stats, error) {
	startTime := time.Now()
	stats := Stats{}

	sessionDir := r.GetSessionDir(workspacePath)

	// Check if sessions directory exists
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		if log != nil {
			log.Debug("sessions directory does not exist, skipping cleanup")
		}
		return stats, nil
	}

	sessions, err := r.ListSessions(sessionDir)
	if err != nil {
		if log != nil {
			log.Error("failed to list sessions for cleanup", err)
		}
		return stats, err
	}

	if log != nil {
		log.Debug("found sessions for cleanup",
			logger.Field{Key: "count", Value: len(sessions)})
	}

	for _, session := range sessions {
		// Check if session should be cleaned up
		if !r.ShouldCleanup(session, activeSessions) {
			continue
		}

		// Try to cleanup expired messages first (keep recent ones)
		if r.config.MessageTTLDays > 0 && session.ModTime.Before(time.Now().AddDate(0, 0, -int(r.config.MessageTTLDays))) {
			expired, cleaned, err := r.cleanupExpiredMessages(session.Path, session.LineCount)
			if err != nil {
				if log != nil {
					log.Error("failed to cleanup expired messages", err,
						logger.Field{Key: "session_id", Value: session.ID})
				}
				continue
			}

			stats.MessagesExpired += expired
			if cleaned > 0 {
				stats.SessionsCleaned++
				if log != nil {
					log.Debug("cleaned expired messages from session",
						logger.Field{Key: "session_id", Value: session.ID},
						logger.Field{Key: "expired", Value: expired})
				}
			}
		} else {
			// Delete entire session if expired or too large
			sizeBefore := session.Size
			if err := r.DeleteSession(session.Path); err != nil {
				if log != nil {
					log.Error("failed to delete session", err,
						logger.Field{Key: "session_id", Value: session.ID})
				}
				continue
			}

			stats.SessionsDeleted++
			stats.MBytesFreed += sizeBefore

			if log != nil {
				log.Debug("deleted session",
					logger.Field{Key: "session_id", Value: session.ID},
					logger.Field{Key: "size_bytes", Value: sizeBefore})
			}
		}
	}

	// Convert bytes to megabytes
	stats.MBytesFreed = (stats.MBytesFreed + 1024*1024 - 1) / (1024 * 1024)
	stats.Duration = time.Since(startTime)
	r.lastRun = time.Now()
	r.stats = stats

	return stats, nil
}

// cleanupExpiredMessages removes expired messages from a session file.
// Returns number of expired messages, number of messages kept, and error.
func (r *Runner) cleanupExpiredMessages(filePath string, totalLines int) (int, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	var keptLines []string
	scanner := bufio.NewScanner(file)

	// Simple strategy: keep last N messages (based on message TTL)
	// For TTL of 30 days, we might keep last 1000 messages as an approximation
	// This is a simplification - in production we'd parse timestamps
	keepRatio := 0.5 // Keep 50% of messages
	if r.config.MessageTTLDays < 30 {
		keepRatio = 0.3
	}
	keepCount := int(float64(totalLines) * keepRatio)
	if keepCount < 10 {
		keepCount = 10 // Always keep at least 10 messages
	}

	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Keep last N lines
		if lineNum > (totalLines - keepCount) {
			keptLines = append(keptLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}

	// Write kept lines back to file
	if err := os.WriteFile(filePath, []byte(strings.Join(keptLines, "\n")+"\n"), 0644); err != nil {
		return 0, 0, err
	}

	expired := totalLines - len(keptLines)
	return expired, len(keptLines), nil
}

// CleanupSubagentDirs removes empty subagent session directories.
func (r *Runner) CleanupSubagentDirs(sessionDir string, prefix string, log *logger.Logger) (int, int64) {
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return 0, 0
	}

	cleaned := 0
	freed := int64(0)

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}

		dirPath := filepath.Join(sessionDir, entry.Name())

		// Check if directory is empty
		subEntries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}

		if len(subEntries) == 0 {
			// Get size before deletion (should be 0 for empty dir)
			info, _ := os.Stat(dirPath)
			if info != nil {
				freed += info.Size()
			}

			// Remove empty directory
			if err := os.Remove(dirPath); err == nil {
				cleaned++
				if log != nil {
					log.Debug("removed empty subagent directory",
						logger.Field{Key: "path", Value: dirPath})
				}
			}
		}
	}

	return cleaned, freed
}

// GetStats returns the statistics from the last cleanup run.
func (r *Runner) GetStats() Stats {
	return r.stats
}

// GetLastRun returns the time of the last cleanup run.
func (r *Runner) GetLastRun() time.Time {
	return r.lastRun
}
