// Package app provides graceful shutdown and restart functionality for the application.
// It ensures all components are stopped in the correct order and allows for
// internal restarts without process termination.
package app

import (
	"context"
	"fmt"
	"time"

	"github.com/aatumaykin/nexbot/internal/ipc"
)

// Shutdown performs graceful shutdown of all components.
// It stops the application in the following order:
//  1. Cancels the application context
//  2. Stops the Telegram connector (if running)
//  3. Stops the cron scheduler (if running)
//  4. Stops the message bus
//
// The method is thread-safe and can be called from multiple goroutines.
func (a *App) Shutdown() error {
	// Get mutex for thread safety
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.shutdownInternal()
}

// Restart performs an internal application restart without terminating the process.
// It performs the following steps:
//  1. Logs the restart attempt
//  2. Calls Shutdown() to stop all components
//  3. Creates a new context
//  4. Reinitializes all components via Initialize()
//  5. Restarts message processing via StartMessageProcessing()
//
// This method is thread-safe and can be called from any goroutine.
// Only one restart can be in progress at a time.
func (a *App) Restart() error {
	// Serialize all Restart() calls to prevent race conditions
	a.restartMutex.Lock()
	defer a.restartMutex.Unlock()

	a.logger.Info("Restarting application")

	// Shutdown existing components
	if err := a.shutdownInternal(); err != nil {
		return fmt.Errorf("failed to shutdown: %w", err)
	}

	// Create new context
	a.ctx, a.cancel = context.WithCancel(context.Background())

	// Reinitialize all components
	if err := a.Initialize(a.ctx); err != nil {
		return fmt.Errorf("failed to reinitialize: %w", err)
	}

	// Restart message processing
	if err := a.StartMessageProcessing(a.ctx); err != nil {
		return fmt.Errorf("failed to restart message processing: %w", err)
	}

	a.logger.Info("Application restarted successfully")
	return nil
}

// shutdownInternal performs shutdown without holding the mutex.
// This is used by Restart() which already holds the mutex.
func (a *App) shutdownInternal() error {
	// If not started, nothing to do
	if !a.started {
		return nil
	}

	// Cancel context to stop all background operations
	a.cancel()

	// Cleanup IPC
	if a.ipcHandler != nil {
		if err := a.ipcHandler.Stop(); err != nil {
			a.logger.Error("failed to stop IPC handler", err)
		}
	}

	// Remove PID file and socket
	if err := ipc.Cleanup(a.config.Workspace.Path); err != nil {
		a.logger.Error("failed to cleanup IPC files", err)
	}

	// Stop telegram connector if not nil
	if a.telegram != nil {
		if err := a.telegram.Stop(); err != nil {
			a.logger.Error("Failed to stop telegram connector", err)
		}
	}

	// Stop cron scheduler if not nil
	if a.cronScheduler != nil {
		if err := a.cronScheduler.Stop(); err != nil {
			a.logger.Error("Failed to stop cron scheduler", err)
		}
	}

	// Stop worker pool if not nil
	if a.workerPool != nil {
		a.workerPool.Stop()
	}

	// Stop subagent manager if not nil
	if a.subagentManager != nil {
		a.logger.Info("🛑 Stopping subagent manager")
		a.subagentManager.StopAll()
	}

	// Stop Docker pool if not nil
	if a.dockerPool != nil {
		a.logger.Info("🛑 Stopping Docker pool")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := a.dockerPool.Stop(shutdownCtx); err != nil {
			a.logger.Error("Failed to stop Docker pool", err)
		}
	}

	// Clear secrets store
	if a.secretsStore != nil {
		a.secretsStore.Clear()
	}

	// Stop message bus
	var busErr error
	if a.messageBus != nil {
		busErr = a.messageBus.Stop()
		if busErr != nil {
			a.logger.Error("Failed to stop message bus", busErr)
		}
	}

	// Mark application as stopped
	a.started = false

	// Log completion
	a.logger.Info("Application shutdown complete")

	// Return message bus error if occurred
	return busErr
}
