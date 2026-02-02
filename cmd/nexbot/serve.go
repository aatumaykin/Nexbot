package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/spf13/cobra"
)

var (
	serveConfigPath string
	serveLogLevel   string
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start Nexbot agent (main command)",
	Long: `Start Nexbot agent with specified configuration.
This will initialize all components (logger, message bus, channels, agent loop)
and handle graceful shutdown.

The serve command is the main entry point for running Nexbot.`,
	Run: serveHandler,
}

func serveHandler(cmd *cobra.Command, args []string) {
	// Determine config path
	configPath := serveConfigPath
	if configPath == "" {
		configPath = "./config.toml"
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override log level if flag is set
	if serveLogLevel != "" {
		cfg.Logging.Level = serveLogLevel
	}

	// Validate configuration
	if errors := cfg.Validate(); len(errors) > 0 {
		fmt.Printf("❌ Configuration validation failed:\n")
		for _, e := range errors {
			fmt.Printf("  - %v\n", e)
		}
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.New(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: cfg.Logging.Output,
	})
	if err != nil {
		fmt.Printf("❌ Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	logger.SetDefault(log)

	// Log startup information
	log.Info("🚀 Starting Nexbot",
		logger.Field{Key: "version", Value: Version},
		logger.Field{Key: "git_commit", Value: GitCommit},
		logger.Field{Key: "config", Value: configPath},
		logger.Field{Key: "workspace", Value: cfg.Workspace.Path},
		logger.Field{Key: "llm_provider", Value: cfg.LLM.Provider},
		logger.Field{Key: "message_bus_capacity", Value: cfg.MessageBus.Capacity},
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize message bus
	log.Info("📡 Initializing message bus",
		logger.Field{Key: "capacity", Value: cfg.MessageBus.Capacity})
	messageBus := bus.New(cfg.MessageBus.Capacity, log)
	if err := messageBus.Start(ctx); err != nil {
		log.Error("Failed to start message bus", err,
			logger.Field{Key: "capacity", Value: cfg.MessageBus.Capacity})
		os.Exit(1)
	}

	// Subscribe to message bus for components (placeholder)
	// TODO: Subscribe inbound messages for agent processing
	// TODO: Subscribe outbound messages for Telegram connector

	// Initialize agent loop (placeholder)
	log.Info("🤖 Initializing agent loop")
	// TODO: Initialize agent loop

	// Initialize Telegram connector if enabled (placeholder)
	if cfg.Channels.Telegram.Enabled {
		log.Info("📱 Initializing Telegram connector")
		// TODO: Initialize Telegram connector
	} else {
		log.Warn("Telegram connector is disabled")
	}

	log.Info("✅ Nexbot is running")

	// Wait for shutdown signal
	sig := <-sigChan
	log.Info("⏳ Received shutdown signal",
		logger.Field{Key: "signal", Value: sig.String()})

	// Graceful shutdown
	log.Info("🛑 Shutting down Nexbot...")
	cancel()

	// Stop message bus
	if err := messageBus.Stop(); err != nil {
		log.Error("Failed to stop message bus", err)
		os.Exit(1)
	}

	// TODO: Stop agent loop
	// TODO: Stop Telegram connector

	log.Info("👋 Nexbot stopped gracefully")
	os.Exit(0)
}

func init() {
	serveCmd.Flags().StringVarP(&serveConfigPath, "config", "c", "", "Path to configuration file (default: ./config.toml)")
	serveCmd.Flags().StringVarP(&serveLogLevel, "log-level", "l", "", "Override log level (debug, info, warn, error)")
}
