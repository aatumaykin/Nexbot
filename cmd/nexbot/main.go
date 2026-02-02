package main

import (
	"os"
)

var (
	// Version variables set during build
	Version   string = "0.1.0-dev"
	BuildTime string = "unknown"
	GitCommit string = "unknown"
	GoVersion string = "unknown"
)

func main() {
	// Execute root command
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
