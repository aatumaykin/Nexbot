package main

import (
	"fmt"
	"os"

	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Display the version, build time, git commit and Go version of Nexbot.`,
	Run: func(cmd *cobra.Command, args []string) {
		log, err := logger.New(logger.Config{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
			os.Exit(1)
		}

		log.Info("Nexbot - Ultra-Lightweight Personal AI Agent")
		log.Info("Version", logger.Field{Key: "version", Value: Version})
		log.Info("Build Time", logger.Field{Key: "build_time", Value: BuildTime})
		log.Info("Git Commit", logger.Field{Key: "git_commit", Value: GitCommit})
		log.Info("Go Version", logger.Field{Key: "go_version", Value: GoVersion})
	},
}
