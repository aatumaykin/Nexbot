package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aatumaykin/nexbot/internal/app"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/constants"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/messages"
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
	// Load .env
	if err := config.LoadEnvOptional(constants.DefaultEnvPath); err != nil {
		fmt.Printf("Warning: failed to load .env file: %v\n", err)
	}

	// Load config
	configPath := serveConfigPath
	if configPath == "" {
		configPath = constants.DefaultConfigPath
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Println(messages.FormatConfigLoadError(err))
		os.Exit(1)
	}

	// Override log level
	if serveLogLevel != "" {
		cfg.Logging.Level = serveLogLevel
	}

	// Validate config
	if errors := cfg.Validate(); len(errors) > 0 {
		fmt.Println(messages.FormatValidationErrors(errors))
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.New(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: cfg.Logging.Output,
	})
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	logger.SetDefault(log)

	// Create and run app
	application := app.New(cfg, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run app in goroutine
	appErr := make(chan error, 1)
	go func() {
		appErr <- application.Run(ctx)
	}()

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		log.Info("Received shutdown signal", logger.Field{Key: "signal", Value: sig.String()})
		cancel()
	case err := <-appErr:
		if err != nil {
			log.Error("Application error", err)
			os.Exit(1)
		}
	}

	log.Info("Application stopped")
}

func init() {
	serveCmd.Flags().StringVarP(&serveConfigPath, "config", "c", "", "Path to configuration file (default: ./config.toml)")
	serveCmd.Flags().StringVarP(&serveLogLevel, "log-level", "l", "", "Override log level (debug, info, warn, error)")
}
