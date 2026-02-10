// Package app provides the main application structure for Nexbot.
// It coordinates all components including the agent loop, message bus,
// channels (Telegram), cron scheduler, and command handlers.
package app

import (
	"context"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/agent/subagent"
	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/channels/telegram"
	"github.com/aatumaykin/nexbot/internal/cleanup"
	"github.com/aatumaykin/nexbot/internal/commands"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/aatumaykin/nexbot/internal/heartbeat"
	"github.com/aatumaykin/nexbot/internal/ipc"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/workers"
	"sync"
)

// App represents the main application structure.
// It holds references to all major components and manages their lifecycle.
type App struct {
	// Configuration and core services
	config *config.Config
	logger *logger.Logger

	// Communication infrastructure
	messageBus *bus.MessageBus

	// Core agent components
	agentLoop      *loop.Loop
	commandHandler *commands.Handler

	// Channels
	telegram *telegram.Connector

	// Scheduled tasks
	cronScheduler *cron.Scheduler

	// Background task execution
	workerPool *workers.WorkerPool

	// Subagent manager
	subagentManager *subagent.Manager

	// Heartbeat checker
	heartbeatChecker *heartbeat.Checker

	// Cleanup scheduler
	cleanupScheduler *cleanup.Scheduler

	// IPC handler
	ipcHandler *ipc.Handler

	// Context management
	ctx    context.Context
	cancel context.CancelFunc

	// Thread-safety
	mu           sync.RWMutex
	started      bool
	restartMutex sync.Mutex // Mutex to serialize Restart() calls
}

// New creates a new App instance with the provided configuration and logger.
// Only initializes config and logger fields; other components are initialized
// in the Initialize() method.
func New(cfg *config.Config, log *logger.Logger) *App {
	return &App{
		config: cfg,
		logger: log,
	}
}

// Run starts the application and blocks until the context is cancelled.
// It performs the following steps:
//  1. Initializes all components via Initialize()
//  2. Starts message processing via StartMessageProcessing()
//  3. Logs that the application is running
//  4. Waits for the context to be cancelled
//  5. Performs graceful shutdown via Shutdown()
func (a *App) Run(ctx context.Context) error {
	// Initialize all components
	if err := a.Initialize(ctx); err != nil {
		return err
	}

	// Start message processing
	if err := a.StartMessageProcessing(ctx); err != nil {
		return err
	}

	// Log that application is running
	a.logger.Info("Application is running")

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	return a.Shutdown()
}

// GetIPC returns the IPC handler instance.
func (a *App) GetIPC() *ipc.Handler {
	return a.ipcHandler
}
