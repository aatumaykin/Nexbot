package loop

import (
	stdcontext "context"
	"fmt"

	"github.com/aatumaykin/nexbot/internal/agent/session"
	"github.com/aatumaykin/nexbot/internal/llm"
)

// SessionOperations handles session-related operations for the loop.
type SessionOperations struct {
	sessionMgr *session.Manager
}

// NewSessionOperations creates a new session operations handler.
func NewSessionOperations(sessionMgr *session.Manager) *SessionOperations {
	return &SessionOperations{
		sessionMgr: sessionMgr,
	}
}

// AddMessageToSession adds a message to the session history.
func (so *SessionOperations) AddMessageToSession(ctx stdcontext.Context, sessionID string, message llm.Message) error {
	sess, _, err := so.sessionMgr.GetOrCreate(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get or create session: %w", err)
	}
	return sess.Append(message)
}

// GetSessionHistory returns the message history for a session.
func (so *SessionOperations) GetSessionHistory(ctx stdcontext.Context, sessionID string) ([]llm.Message, error) {
	sess, _, err := so.sessionMgr.GetOrCreate(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create session: %w", err)
	}
	return sess.Read()
}

// ClearSession clears all messages from a session.
func (so *SessionOperations) ClearSession(ctx stdcontext.Context, sessionID string) error {
	sess, _, err := so.sessionMgr.GetOrCreate(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get or create session: %w", err)
	}
	return sess.Clear()
}

// DeleteSession deletes a session entirely.
func (so *SessionOperations) DeleteSession(ctx stdcontext.Context, sessionID string) error {
	sess, _, err := so.sessionMgr.GetOrCreate(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get or create session: %w", err)
	}
	return sess.Delete()
}

// GetSessionStatus returns status information about a session.
func (so *SessionOperations) GetSessionStatus(ctx stdcontext.Context, sessionID string, loop *Loop) (map[string]any, error) {
	sess, _, err := so.sessionMgr.GetOrCreate(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Get message count
	msgCount, err := sess.MessageCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get message count: %w", err)
	}

	// Get session file size
	fileSize := int64(0)
	if fileInfo, err := getFileInfo(sess.File); err == nil {
		fileSize = fileInfo.Size()
	}

	return map[string]any{
		"session_id":      sessionID,
		"message_count":   msgCount,
		"file_size":       fileSize,
		"file_size_human": formatBytes(fileSize),
		"model":           loop.config.Model,
		"temperature":     loop.config.Temperature,
		"max_tokens":      loop.config.MaxTokens,
	}, nil
}
