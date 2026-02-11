package cleanup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListSessions(t *testing.T) {
	tempDir := t.TempDir()
	runner := NewRunner(Config{})
	sessionDir := filepath.Join(tempDir, "sessions")

	// Create sessions directory
	if err := os.Mkdir(sessionDir, 0755); err != nil {
		t.Fatalf("failed to create sessions dir: %v", err)
	}

	// Create test session files
	sessions := []string{"session1.jsonl", "session2.jsonl"}
	for _, session := range sessions {
		path := filepath.Join(sessionDir, session)
		if err := os.WriteFile(path, []byte("{}\n"), 0644); err != nil {
			t.Fatalf("failed to create session file: %v", err)
		}
	}

	// Create a subdirectory
	if err := os.Mkdir(filepath.Join(sessionDir, "dir1"), 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// List sessions
	listedSessions, err := runner.ListSessions(sessionDir)
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	// Should only list files, not directories
	if len(listedSessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(listedSessions))
	}
}

func TestShouldCleanup(t *testing.T) {
	tests := []struct {
		name           string
		session        SessionInfo
		activeSessions map[string]bool
		config         Config
		shouldCleanup  bool
	}{
		{
			name: "active session",
			session: SessionInfo{
				ID:      "session1",
				Size:    100 * 1024 * 1024, // 100MB
				ModTime: time.Now().Add(-time.Hour),
			},
			activeSessions: map[string]bool{"session1": true},
			config:         Config{MaxSessionSizeMB: 100, KeepActiveDays: 1},
			shouldCleanup:  false,
		},
		{
			name: "inactive session exceeds size limit",
			session: SessionInfo{
				ID:      "session1",
				Size:    150 * 1024 * 1024, // 150MB
				ModTime: time.Now().Add(-48 * time.Hour),
			},
			activeSessions: map[string]bool{},
			config:         Config{MaxSessionSizeMB: 100, KeepActiveDays: 1},
			shouldCleanup:  true,
		},
		{
			name: "session within TTL",
			session: SessionInfo{
				ID:      "session1",
				Size:    50 * 1024 * 1024,
				ModTime: time.Now().Add(-24 * time.Hour),
			},
			activeSessions: map[string]bool{},
			config:         Config{SessionTTLDays: 90},
			shouldCleanup:  false,
		},
		{
			name: "session expired TTL",
			session: SessionInfo{
				ID:      "session1",
				Size:    50 * 1024 * 1024,
				ModTime: time.Now().Add(-100 * 24 * time.Hour),
			},
			activeSessions: map[string]bool{},
			config:         Config{SessionTTLDays: 90},
			shouldCleanup:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewRunner(tt.config)
			result := runner.ShouldCleanup(tt.session, tt.activeSessions)
			if result != tt.shouldCleanup {
				t.Errorf("expected cleanup=%v, got %v", tt.shouldCleanup, result)
			}
		})
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "empty file",
			content: "",
			want:    0,
		},
		{
			name:    "single line",
			content: "line1",
			want:    1,
		},
		{
			name:    "multiple lines",
			content: "line1\nline2\nline3\n",
			want:    3,
		},
		{
			name:    "no trailing newline",
			content: "line1\nline2",
			want:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewRunner(Config{})
			tempFile := t.TempDir() + "/test.txt"
			if err := os.WriteFile(tempFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			got, err := runner.countLines(tempFile)
			if err != nil {
				t.Fatalf("countLines failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %d lines, got %d", tt.want, got)
			}
		})
	}
}

func TestDeleteSession(t *testing.T) {
	runner := NewRunner(Config{})
	tempFile := t.TempDir() + "/session.jsonl"

	// Create test file
	if err := os.WriteFile(tempFile, []byte("{}\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Delete session
	if err := runner.DeleteSession(tempFile); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Error("file still exists after deletion")
	}
}

func TestCleanupExpiredMessages(t *testing.T) {
	runner := NewRunner(Config{MessageTTLDays: 30})
	tempFile := t.TempDir() + "/session.jsonl"

	// Create test file with 20 lines
	var content strings.Builder
	for i := 1; i <= 20; i++ {
		content.WriteString(`{"role":"user","content":"message ` + string(rune('0'+i)) + `"}` + "\n")
	}
	if err := os.WriteFile(tempFile, []byte(content.String()), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Cleanup expired messages
	expired, kept, err := runner.cleanupExpiredMessages(tempFile, 20)
	if err != nil {
		t.Fatalf("cleanupExpiredMessages failed: %v", err)
	}

	// Should keep at least 10 messages
	if kept < 10 {
		t.Errorf("expected at least 10 kept messages, got %d", kept)
	}

	// Should expire some messages
	if expired < 1 {
		t.Errorf("expected at least 1 expired message, got %d", expired)
	}

	// Verify file content
	data, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	expectedLines := kept
	actualLines := 0
	for _, b := range data {
		if b == '\n' {
			actualLines++
		}
	}

	if actualLines != expectedLines {
		t.Errorf("expected %d lines in file, got %d", expectedLines, actualLines)
	}
}

func TestCleanupSubagentDirs(t *testing.T) {
	runner := NewRunner(Config{})
	sessionDir := t.TempDir()

	// Create empty subagent directory
	subagentDir := filepath.Join(sessionDir, "subagent-123")
	if err := os.Mkdir(subagentDir, 0755); err != nil {
		t.Fatalf("failed to create subagent dir: %v", err)
	}

	// Create a non-empty subagent directory
	nonEmptyDir := filepath.Join(sessionDir, "subagent-456")
	if err := os.Mkdir(nonEmptyDir, 0755); err != nil {
		t.Fatalf("failed to create non-empty subagent dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file in subagent dir: %v", err)
	}

	// Create a regular directory (not prefixed)
	regularDir := filepath.Join(sessionDir, "regular")
	if err := os.Mkdir(regularDir, 0755); err != nil {
		t.Fatalf("failed to create regular dir: %v", err)
	}

	// Cleanup subagent dirs
	cleaned, _ := runner.CleanupSubagentDirs(sessionDir, "subagent-", nil)

	// Should clean only empty subagent dir
	if cleaned != 1 {
		t.Errorf("expected 1 cleaned directory, got %d", cleaned)
	}

	// Verify empty dir is deleted
	if _, err := os.Stat(subagentDir); !os.IsNotExist(err) {
		t.Error("empty subagent directory still exists")
	}

	// Verify non-empty dir still exists
	if _, err := os.Stat(nonEmptyDir); os.IsNotExist(err) {
		t.Error("non-empty subagent directory was deleted")
	}

	// Verify regular dir still exists
	if _, err := os.Stat(regularDir); os.IsNotExist(err) {
		t.Error("regular directory was deleted")
	}
}

func TestRun(t *testing.T) {
	tempDir := t.TempDir()
	sessionDir := filepath.Join(tempDir, "sessions")

	// Create sessions directory
	if err := os.Mkdir(sessionDir, 0755); err != nil {
		t.Fatalf("failed to create sessions dir: %v", err)
	}

	// Create test session files
	for i := 1; i <= 3; i++ {
		sessionFile := filepath.Join(sessionDir, "session"+string(rune('0'+i))+".jsonl")
		var content strings.Builder
		for j := 1; j <= 10; j++ {
			content.WriteString(`{"role":"user","content":"message"}` + "\n")
		}
		if err := os.WriteFile(sessionFile, []byte(content.String()), 0644); err != nil {
			t.Fatalf("failed to create session file: %v", err)
		}
	}

	runner := NewRunner(Config{SessionTTLDays: 90})

	stats, err := runner.Run(tempDir, make(map[string]bool), nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should not cleanup any sessions (within TTL)
	if stats.SessionsDeleted > 0 {
		t.Errorf("expected no sessions to be deleted, got %d", stats.SessionsDeleted)
	}
}

func TestGetSessionDir(t *testing.T) {
	runner := NewRunner(Config{})
	expected := filepath.Join("/workspace", "sessions")
	got := runner.GetSessionDir("/workspace")
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}
