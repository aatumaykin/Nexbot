package main

import (
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "nexbot",
	Short: "Nexbot - Ultra-Lightweight Personal AI Agent",
	Long: `Nexbot is a self-hosted AI agent with message bus architecture,
extensible channels and skills. It's designed to be simple and lightweight.`,
	Version: Version,
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(testCmd)
}
