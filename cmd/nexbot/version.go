package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Display the version, build time, git commit and Go version of Nexbot.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Nexbot - Ultra-Lightweight Personal AI Agent")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Go Version: %s\n", GoVersion)
	},
}
