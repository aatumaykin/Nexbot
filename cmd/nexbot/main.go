package main

import (
	"os"

	"github.com/aatumaykin/nexbot/internal/version"
)

var (
	Version   string = "0.1.0-dev"
	BuildTime string = "unknown"
	GitCommit string = "unknown"
	GoVersion string = "unknown"
)

func init() {
	version.SetInfo(Version, BuildTime, GitCommit, GoVersion)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
