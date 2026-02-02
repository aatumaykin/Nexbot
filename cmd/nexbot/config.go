package main

import (
	"fmt"
	"os"

	"github.com/aatumaykin/nexbot/internal/config"
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
		configPath := "config.toml"
		if len(args) > 0 {
			configPath = args[0]
		}

		fmt.Printf("Validating configuration: %s\n", configPath)

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Printf("❌ Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		// Validate configuration
		errors := cfg.Validate()
		if len(errors) > 0 {
			fmt.Printf("❌ Configuration validation failed with %d error(s):\n", len(errors))
			for i, e := range errors {
				fmt.Printf("  %d. %v\n", i+1, e)
			}
			os.Exit(1)
		}

		fmt.Println("✅ Configuration is valid")
	},
}

func init() {
	configCmd.AddCommand(configValidateCmd)
}
