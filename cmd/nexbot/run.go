package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/spf13/cobra"
)

var (
	runConfigPath string
	runWorkspace  string
	runDebug      bool
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start Nexbot agent",
	Long: `Start Nexbot agent with specified configuration.
This will initialize all components (logger, message bus, channels, agent loop)
and handle graceful shutdown.`,
	Run: runHandler,
}

func runHandler(cmd *cobra.Command, args []string) {
	// Determine config path
	configPath := runConfigPath
	if configPath == "" {
		configPath = "config.toml"
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Validate configuration
	if errors := cfg.Validate(); len(errors) > 0 {
		fmt.Printf("❌ Configuration validation failed:\n")
		for _, e := range errors {
			fmt.Printf("  - %v\n", e)
		}
		os.Exit(1)
	}

	// Enable debug mode if flag is set
	if runDebug {
		cfg.Logging.Level = "debug"
	}

	// Override workspace if specified
	if runWorkspace != "" {
		cfg.Workspace.Path = runWorkspace
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
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start components (placeholder for now - will be implemented in later tasks)
	log.Info("📡 Initializing message bus")
	// TODO: Initialize message bus

	log.Info("🤖 Initializing agent loop")
	// TODO: Initialize agent loop

	if cfg.Channels.Telegram.Enabled {
		log.Info("📱 Initializing Telegram connector")
		// TODO: Initialize Telegram connector
	} else {
		log.Warn("Telegram connector is disabled")
	}

	log.Info("✅ Nexbot is running")

	// Wait for shutdown signal
	_ = ctx // TODO: Use ctx for component initialization
	sig := <-sigChan
	log.Info("⏳ Received shutdown signal",
		logger.Field{Key: "signal", Value: sig.String()})

	// Graceful shutdown
	log.Info("🛑 Shutting down Nexbot...")
	cancel() // Cancel context

	// TODO: Stop all components gracefully
	// TODO: Stop message bus
	// TODO: Stop agent loop
	// TODO: Stop channels

	log.Info("👋 Nexbot stopped gracefully")
}

func init() {
	runCmd.Flags().StringVarP(&runConfigPath, "config", "c", "", "Path to configuration file (default: ./config.toml)")
	runCmd.Flags().StringVarP(&runWorkspace, "workspace", "w", "", "Path to workspace directory (overrides config)")
	runCmd.Flags().BoolVarP(&runDebug, "debug", "d", false, "Enable debug logging")
}
