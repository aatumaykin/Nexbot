package main

import (
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/tools"
	"github.com/aatumaykin/nexbot/internal/tools/fetch"
)

func registerSubagentTools(registry *tools.Registry) {
	log, _ := logger.New(logger.Config{Level: "info"})

	fetchCfg := &config.Config{
		Tools: config.ToolsConfig{
			Fetch: config.FetchToolConfig{
				Enabled:         true,
				TimeoutSeconds:  30,
				MaxResponseSize: 5 * 1024 * 1024,
				UserAgent:       "Nexbot-Subagent/1.0",
			},
		},
	}
	registry.Register(fetch.NewFetchTool(fetchCfg, log))
}
