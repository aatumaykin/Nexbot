package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/constants"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/spf13/cobra"
)

var (
	testConfigPath string
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test Nexbot components",
	Long:  `Run tests on various Nexbot components to verify configuration and connectivity.`,
}

// testLLMCmd represents the test llm command
var testLLMCmd = &cobra.Command{
	Use:   "llm",
	Short: "Test LLM provider connectivity",
	Long: `Send a test request to the configured LLM provider (Z.ai)
to verify connectivity and functionality.

This command will:
1. Load the configuration file
2. Initialize the logger
3. Create the LLM provider from configuration
4. Send a test request ("Hello, world!")
5. Display the response, latency, and token usage

Example usage:
  nexbot test llm
  nexbot test llm --config custom-config.toml
  nexbot test llm --model glm-4.7`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get model flag
		modelOverride, _ := cmd.Flags().GetString("model")

		// Determine config path
		configPath := testConfigPath
		if configPath == "" {
			configPath = constants.DefaultConfigPath
		}

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

		// Load configuration
		log.Info("Loading configuration", logger.Field{Key: "path", Value: configPath})
		cfg, err := config.Load(configPath)
		if err != nil {
			log.Error("Failed to load config", err)
			os.Exit(1)
		}

		log.Info("Configuration loaded")

		// Validate LLM configuration
		if cfg.Agent.Provider != "zai" {
			log.Error("Provider not supported", fmt.Errorf("provider: %s", cfg.Agent.Provider))
			os.Exit(1)
		}

		if cfg.LLM.ZAI.APIKey == "" {
			log.Error("API key not configured", nil)
			os.Exit(1)
		}

		// Create LLM provider
		log.Info("Initializing LLM provider")

		// Determine model to use
		model := modelOverride
		if model == "" {
			model = cfg.Agent.Model
			if model == "" {
				model = constants.TestDefaultModel
			}
		}

		provider := llm.NewZAIProvider(llm.ZAIConfig{
			APIKey: cfg.LLM.ZAI.APIKey,
		}, log)

		log.Info("Provider initialized", logger.Field{Key: "model", Value: model})

		// Prepare test request
		testMessage := constants.TestMessage

		log.Info("Sending test request", logger.Field{Key: "message", Value: testMessage})

		// Measure latency
		startTime := time.Now()

		// Send request with context timeout
		ctx, cancel := context.WithTimeout(context.Background(), constants.TestRequestTimeout)
		defer cancel()

		req := llm.ChatRequest{
			Messages: []llm.Message{
				{
					Role:    llm.RoleUser,
					Content: testMessage,
				},
			},
			Model:       model,
			Temperature: constants.TestTemperature,
			MaxTokens:   constants.TestMaxTokens,
		}

		// Apply model override if specified
		if modelOverride != "" {
			req.Model = modelOverride
		}

		resp, err := provider.Chat(ctx, req)

		latency := time.Since(startTime)

		if err != nil {
			log.Error("Request failed", err)
			os.Exit(1)
		}

		// Display success message
		log.Info("Request successful")

		// Display response details
		log.Info("Response details",
			logger.Field{Key: "model", Value: resp.Model},
			logger.Field{Key: "latency", Value: latency},
			logger.Field{Key: "finish_reason", Value: resp.FinishReason})

		// Display response content
		log.Info("Response content", logger.Field{Key: "content", Value: resp.Content})

		// Display token usage
		log.Info("Token usage",
			logger.Field{Key: "prompt_tokens", Value: resp.Usage.PromptTokens},
			logger.Field{Key: "completion_tokens", Value: resp.Usage.CompletionTokens},
			logger.Field{Key: "total_tokens", Value: resp.Usage.TotalTokens})

		// Display tool calls if any
		if len(resp.ToolCalls) > 0 {
			log.Info("Tool calls", logger.Field{Key: "count", Value: len(resp.ToolCalls)})
			for i, tc := range resp.ToolCalls {
				log.Info(fmt.Sprintf("Tool call %d", i+1),
					logger.Field{Key: "name", Value: tc.Name},
					logger.Field{Key: "arguments", Value: tc.Arguments})
			}
		}

		// Check finish reason
		switch resp.FinishReason {
		case llm.FinishReasonStop:
			log.Info("Stop reason: normal")
		case llm.FinishReasonLength:
			log.Info("Stop reason: max tokens reached")
		case llm.FinishReasonToolCalls:
			log.Info("Stop reason: tool calls")
		case llm.FinishReasonError:
			log.Error("Stop reason: error", nil)
		}

		log.Info("All tests passed")
	},
}

func init() {
	// Add test llm subcommand to test command
	testCmd.AddCommand(testLLMCmd)

	// Add flags
	testLLMCmd.Flags().StringVarP(&testConfigPath, "config", "c", "", "Path to configuration file (default: ./config.toml)")
	testLLMCmd.Flags().StringP("model", "m", "", "Override model to use (e.g., glm-4.7)")
}
