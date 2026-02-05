package main

import (
	"fmt"
	"os"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/constants"
	"github.com/aatumaykin/nexbot/internal/messages"
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
		configPath := constants.DefaultConfigPath
		if len(args) > 0 {
			configPath = args[0]
		}

		fmt.Printf(constants.MsgConfigValidating, configPath)

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Println(messages.FormatConfigLoadError(err))
			os.Exit(1)
		}

		// Validate configuration
		errors := cfg.Validate()
		if len(errors) > 0 {
			fmt.Println(messages.FormatValidationErrors(errors))
			os.Exit(1)
		}

		fmt.Println(constants.MsgConfigValid)
	},
}

func init() {
	configCmd.AddCommand(configValidateCmd)
}
