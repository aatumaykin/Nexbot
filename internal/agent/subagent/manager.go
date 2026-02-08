// Package subagent provides subagent management for spawning and managing
// concurrent agent instances with separate sessions.
// Subagents are isolated instances with their own sessions and memory,
// allowing parallel task execution while maintaining separation of concerns.
package subagent

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/agent/session"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/google/uuid"
)

const (
	// SessionIDPrefix is the prefix for subagent session IDs
	SessionIDPrefix = "subagent-"
)

// Subagent represents a spawned agent instance with isolated session.
type Subagent struct {
	ID      string             // Unique subagent ID (UUID)
	Session string             // Session ID for this subagent
	Loop    *loop.Loop         // Agent loop for processing
	Context context.Context    // Context for lifecycle management
	Cancel  context.CancelFunc // Cancel function for graceful shutdown
	Logger  *logger.Logger     // Logger for this subagent
}

// Manager manages subagent lifecycle, including spawning, stopping, and listing.
// It provides thread-safe operations for concurrent subagent management.
type Manager struct {
	subagents   map[string]*Subagent
	mu          sync.RWMutex
	loopFactory func() (*loop.Loop, error) // Factory for creating new loops
	sessionMgr  *session.Manager           // Session manager for subagent sessions
	logger      *logger.Logger
}

// Config holds configuration for the subagent manager.
type Config struct {
	SessionDir string         // Directory for storing subagent sessions
	Logger     *logger.Logger // Logger for manager operations
	LoopConfig loop.Config    // Configuration for creating new loops
}

// NewManager creates a new subagent manager.
func NewManager(cfg Config) (*Manager, error) {
	// Validate configuration
	if cfg.SessionDir == "" {
		return nil, fmt.Errorf("session directory cannot be empty")
	}
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger cannot be empty")
	}

	// Create subagent session directory
	subagentDir := cfg.SessionDir + "/subagents"
	if err := os.MkdirAll(subagentDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create subagent session directory: %w", err)
	}

	// Create session manager for subagents
	sessionMgr, err := session.NewManager(subagentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	return &Manager{
		subagents:  make(map[string]*Subagent),
		sessionMgr: sessionMgr,
		logger:     cfg.Logger,
		loopFactory: func() (*loop.Loop, error) {
			cfg.LoopConfig.SessionDir = subagentDir
			l, err := loop.NewLoop(cfg.LoopConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create loop: %w", err)
			}
			return l, nil
		},
	}, nil
}

// Spawn creates a new subagent with a new isolated session.
// The subagent starts with its own context and session ID.
// Returns the spawned subagent or an error.
func (m *Manager) Spawn(ctx context.Context, parentSession string, task string) (*Subagent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate unique IDs
	subagentID := generateID()
	sessionID := generateSessionID()

	// Create context for this subagent
	subagentCtx, cancel := context.WithCancel(ctx)

	// Create new loop for this subagent
	subagentLoop, err := m.loopFactory()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create loop for subagent: %w", err)
	}

	// Create subagent
	subagent := &Subagent{
		ID:      subagentID,
		Session: sessionID,
		Loop:    subagentLoop,
		Context: subagentCtx,
		Cancel:  cancel,
		Logger:  m.logger,
	}

	// Store in manager
	m.subagents[subagentID] = subagent

	m.logger.Info("subagent spawned",
		logger.Field{Key: "subagent_id", Value: subagentID},
		logger.Field{Key: "session_id", Value: sessionID},
		logger.Field{Key: "parent_session", Value: parentSession},
		logger.Field{Key: "task", Value: task})

	return subagent, nil
}

// Stop stops a subagent by ID, cancelling its context and removing from registry.
// Returns an error if the subagent is not found.
func (m *Manager) Stop(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sub, exists := m.subagents[id]
	if !exists {
		return fmt.Errorf("subagent not found: %s", id)
	}

	// Cancel subagent context
	sub.Cancel()

	// Remove from registry
	delete(m.subagents, id)

	m.logger.Info("subagent stopped",
		logger.Field{Key: "subagent_id", Value: id},
		logger.Field{Key: "session_id", Value: sub.Session})

	return nil
}

// List returns all active subagents.
// Returns a slice of subagent pointers (read-only snapshot).
func (m *Manager) List() []*Subagent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*Subagent, 0, len(m.subagents))
	for _, sub := range m.subagents {
		list = append(list, sub)
	}
	return list
}

// Get retrieves a subagent by ID.
// Returns the subagent or an error if not found.
func (m *Manager) Get(id string) (*Subagent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sub, exists := m.subagents[id]
	if !exists {
		return nil, fmt.Errorf("subagent not found: %s", id)
	}
	return sub, nil
}

// StopAll stops all active subagents.
// This is useful for graceful shutdown of the manager.
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("stopping all subagents",
		logger.Field{Key: "count", Value: len(m.subagents)})

	for id, sub := range m.subagents {
		sub.Cancel()
		m.logger.Debug("subagent stopped",
			logger.Field{Key: "subagent_id", Value: id},
			logger.Field{Key: "session_id", Value: sub.Session})
	}

	m.subagents = make(map[string]*Subagent)

	m.logger.Info("all subagents stopped")
}

// Count returns the number of active subagents.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.subagents)
}

// ExecuteTask spawns a subagent, executes a task, and cleans up after completion.
// This is a one-shot operation: subagent is created, task is executed, and subagent is removed.
// Returns the response from the subagent or an error.
func (m *Manager) ExecuteTask(ctx context.Context, parentSession string, task string, timeout int) (string, error) {
	// Spawn a new subagent for this task
	subagent, err := m.Spawn(ctx, parentSession, task)
	if err != nil {
		return "", fmt.Errorf("failed to spawn subagent: %w", err)
	}

	// Ensure subagent is stopped and session is cleaned up, even on panic
	defer func() {
		// Stop the subagent (removes from registry)
		if stopErr := m.Stop(subagent.ID); stopErr != nil {
			m.logger.Error("failed to stop subagent during cleanup", stopErr,
				logger.Field{Key: "subagent_id", Value: subagent.ID})
		}

		// Delete the subagent session from storage
		if deleteErr := m.sessionMgr.DeleteSession(subagent.Session); deleteErr != nil {
			m.logger.Error("failed to delete subagent session during cleanup", deleteErr,
				logger.Field{Key: "session_id", Value: subagent.Session},
				logger.Field{Key: "subagent_id", Value: subagent.ID})
		}
	}()

	// Set timeout if provided
	taskCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}

	// Process the task through the subagent
	response, err := subagent.Process(taskCtx, task)
	if err != nil {
		return "", fmt.Errorf("failed to execute task in subagent: %w", err)
	}

	m.logger.Info("subagent task completed",
		logger.Field{Key: "subagent_id", Value: subagent.ID},
		logger.Field{Key: "session_id", Value: subagent.Session},
		logger.Field{Key: "response_length", Value: len(response)})

	return response, nil
}

// Process sends a task to a subagent for processing.
// Returns the response or an error.
func (s *Subagent) Process(ctx context.Context, task string) (string, error) {
	s.Logger.DebugCtx(ctx, "processing task in subagent",
		logger.Field{Key: "subagent_id", Value: s.ID},
		logger.Field{Key: "session_id", Value: s.Session},
		logger.Field{Key: "task_length", Value: len(task)})

	// Use subagent's context with timeout
	if _, ok := s.Context.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(s.Context, 5*time.Minute)
		defer cancel()
	}

	// Process task through subagent's loop
	response, err := s.Loop.Process(ctx, s.Session, task)
	if err != nil {
		s.Logger.ErrorCtx(ctx, "failed to process task in subagent", err,
			logger.Field{Key: "subagent_id", Value: s.ID},
			logger.Field{Key: "session_id", Value: s.Session})
		return "", fmt.Errorf("subagent processing failed: %w", err)
	}

	s.Logger.DebugCtx(ctx, "task processed in subagent",
		logger.Field{Key: "subagent_id", Value: s.ID},
		logger.Field{Key: "response_length", Value: len(response)})

	return response, nil
}

// generateID generates a unique subagent ID using UUID.
func generateID() string {
	return uuid.New().String()
}

// generateSessionID generates a session ID for a subagent.
func generateSessionID() string {
	return fmt.Sprintf("%s%d", SessionIDPrefix, time.Now().UnixNano())
}
