package main

import (
	"os"

	"github.com/aatumaykin/nexbot/internal/constants"
)

var (
	// Version variables set during build
	Version   string = constants.DefaultVersion
	BuildTime string = constants.DefaultBuildTime
	GitCommit string = constants.DefaultGitCommit
	GoVersion string = constants.DefaultGoVersion
)

func main() {
	// Execute root command
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
