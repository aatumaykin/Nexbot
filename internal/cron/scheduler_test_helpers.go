package cron

import (
	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// testLogger creates a test logger instance
func testLogger() *logger.Logger {
	log, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		panic(err)
	}
	return log
}

// stopScheduler stops a scheduler and ignores the error (for use in defer in tests)
func stopScheduler(s *Scheduler) {
	_ = s.Stop()
}

// stopMessageBus stops a message bus and ignores the error (for use in defer in tests)
func stopMessageBus(b *bus.MessageBus) {
	_ = b.Stop()
}
