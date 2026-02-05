package main

import (
	"fmt"
	"os"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
)

func main() {
	// Load config
	cfg, err := config.Load("./config.toml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	logger.SetDefault(log)

	// Create message bus (but don't start it - the bot should already be running)
	messageBus := bus.New(cfg.MessageBus.Capacity, log)
	log.Info("Message bus created", logger.Field{Key: "capacity", Value: cfg.MessageBus.Capacity})

	// Note: We can't directly publish to the running bot's message bus from this process
	// because it's a separate process with its own in-memory message bus.

	// Instead, we need to test via Telegram directly or use a different approach.
	log.Info("Note: Cannot test restart via this script as message bus is in-process only.")
	log.Info("Please send /restart command via Telegram app to test.")

	time.Sleep(1 * time.Second)
}
