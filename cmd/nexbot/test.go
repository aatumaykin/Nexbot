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

		// Load configuration
		fmt.Printf(constants.TestMsgLoadingConfig, configPath)
		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Printf(constants.MsgConfigLoadError, err)
			os.Exit(1)
		}

		fmt.Println(constants.TestMsgConfigLoaded)

		// Validate LLM configuration
		if cfg.Agent.Provider != "zai" {
			fmt.Printf(constants.TestMsgProviderNotSupported, cfg.Agent.Provider)
			os.Exit(1)
		}

		if cfg.LLM.ZAI.APIKey == "" {
			fmt.Println(constants.TestMsgAPIKeyNotConfigured)
			os.Exit(1)
		}

		// Initialize logger
		log, err := logger.New(logger.Config{
			Level:  cfg.Logging.Level,
			Format: cfg.Logging.Format,
			Output: cfg.Logging.Output,
		})
		if err != nil {
			fmt.Printf(constants.TestMsgFailedToInitLogger, err)
			os.Exit(1)
		}

		// Create LLM provider
		fmt.Printf(constants.TestMsgInitializingProvider)

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

		fmt.Printf(constants.TestMsgProviderInitialized, model)

		// Prepare test request
		testMessage := constants.TestMessage

		fmt.Printf(constants.TestMsgSendingRequest)
		fmt.Printf(constants.TestMsgSendingRequestMessage, testMessage)

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
			fmt.Printf(constants.TestMsgRequestFailed, err)

			// Provide friendly error messages
			fmt.Println(constants.TestMsgPossibleCauses)
			fmt.Println(constants.TestMsgCauseAPIKey)
			fmt.Println(constants.TestMsgCauseNetwork)
			fmt.Println(constants.TestMsgCauseUnavail)
			fmt.Println(constants.TestMsgCauseRateLimit)
			fmt.Println(constants.TestMsgTroubleshooting)
			fmt.Println(constants.TestMsgStepVerifyAPIKey)
			fmt.Println(constants.TestMsgCheckConnection)
			fmt.Println(constants.TestMsgTryAgain)
			fmt.Println(constants.TestMsgCheckStatus)
			os.Exit(1)
		}

		// Display success message
		fmt.Printf(constants.TestMsgRequestSuccessful)

		// Display response details
		fmt.Printf(constants.TestMsgResponseDetails)
		fmt.Printf(constants.TestMsgResponseModel, resp.Model)
		fmt.Printf(constants.TestMsgResponseLatency, latency)
		fmt.Printf(constants.TestMsgFinishReason, resp.FinishReason)

		// Display response content
		fmt.Printf(constants.TestMsgResponseContent)
		fmt.Printf(constants.TestMsgResponseContentText, resp.Content)

		// Display token usage
		fmt.Printf(constants.TestMsgTokenUsage)
		fmt.Printf(constants.TestMsgPromptTokens, resp.Usage.PromptTokens)
		fmt.Printf(constants.TestMsgCompletionTokens, resp.Usage.CompletionTokens)
		fmt.Printf(constants.TestMsgTotalTokens, resp.Usage.TotalTokens)

		// Display tool calls if any
		if len(resp.ToolCalls) > 0 {
			fmt.Printf(constants.TestMsgToolCalls, len(resp.ToolCalls))
			for i, tc := range resp.ToolCalls {
				fmt.Printf(constants.TestMsgToolCallItem, i+1, tc.Name, tc.Arguments)
			}
			fmt.Println()
		}

		// Check finish reason
		switch resp.FinishReason {
		case llm.FinishReasonStop:
			fmt.Println(constants.TestMsgStopNormal)
		case llm.FinishReasonLength:
			fmt.Println(constants.TestMsgStopLength)
		case llm.FinishReasonToolCalls:
			fmt.Println(constants.TestMsgStopToolCalls)
		case llm.FinishReasonError:
			fmt.Println(constants.TestMsgStopError)
		}

		fmt.Println(constants.TestMsgAllPassed)
	},
}

func init() {
	// Add test llm subcommand to test command
	testCmd.AddCommand(testLLMCmd)

	// Add flags
	testLLMCmd.Flags().StringVarP(&testConfigPath, "config", "c", "", "Path to configuration file (default: ./config.toml)")
	testLLMCmd.Flags().StringP("model", "m", "", "Override model to use (e.g., glm-4.7)")
}
