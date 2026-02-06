package main

import (
	"fmt"
	"os"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/constants"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Validate and manage Nexbot configuration.`,
}

// configValidateCmd represents the config validate command
var configValidateCmd = &cobra.Command{
	Use:   "validate [config-file]",
	Short: "Validate configuration file",
	Long:  `Validate the configuration file and check for errors.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize a minimal logger for this command
		log, err := logger.New(logger.Config{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
			os.Exit(1)
		}

		configPath := constants.DefaultConfigPath
		if len(args) > 0 {
			configPath = args[0]
		}

		log.Info("Validating configuration", logger.Field{Key: "path", Value: configPath})

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			log.Error("Failed to load config", err)
			os.Exit(1)
		}

		// Validate configuration
		errors := cfg.Validate()
		if len(errors) > 0 {
			log.Error("Config validation failed", fmt.Errorf("%d errors", len(errors)))
			for _, e := range errors {
				log.Error("Validation error", e)
			}
			os.Exit(1)
		}

		log.Info("Configuration is valid")
	},
}

func init() {
	configCmd.AddCommand(configValidateCmd)
}
