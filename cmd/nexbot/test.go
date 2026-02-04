package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aatumaykin/nexbot/internal/config"
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
			configPath = "config.toml"
		}

		// Load configuration
		fmt.Printf("📄 Loading configuration: %s\n", configPath)
		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Printf("❌ Failed to load configuration: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ Configuration loaded")

		// Validate LLM configuration
		if cfg.Agent.Provider != "zai" {
			fmt.Printf("❌ LLM provider '%s' is not yet supported (only 'zai' is supported)\n", cfg.Agent.Provider)
			os.Exit(1)
		}

		if cfg.LLM.ZAI.APIKey == "" {
			fmt.Println("❌ Z.ai API key is not configured in [llm.zai.api_key]")
			os.Exit(1)
		}

		// Initialize logger
		log, err := logger.New(logger.Config{
			Level:  cfg.Logging.Level,
			Format: cfg.Logging.Format,
			Output: cfg.Logging.Output,
		})
		if err != nil {
			fmt.Printf("❌ Failed to initialize logger: %v\n", err)
			os.Exit(1)
		}

		// Create LLM provider
		fmt.Printf("🔌 Initializing Z.ai provider...\n")

		// Determine model to use
		model := modelOverride
		if model == "" {
			model = cfg.Agent.Model
			if model == "" {
				model = "glm-4.7"
			}
		}

		provider := llm.NewZAIProvider(llm.ZAIConfig{
			APIKey: cfg.LLM.ZAI.APIKey,
		}, log)

		fmt.Printf("✅ Z.ai provider initialized (model: %s)\n\n", model)

		// Prepare test request
		testMessage := "Hello, world! Please respond with a friendly greeting."

		fmt.Printf("📨 Sending test request...\n")
		fmt.Printf("   Message: %q\n\n", testMessage)

		// Measure latency
		startTime := time.Now()

		// Send request with context timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req := llm.ChatRequest{
			Messages: []llm.Message{
				{
					Role:    llm.RoleUser,
					Content: testMessage,
				},
			},
			Model:       model,
			Temperature: 0.7,
			MaxTokens:   200,
		}

		// Apply model override if specified
		if modelOverride != "" {
			req.Model = modelOverride
		}

		resp, err := provider.Chat(ctx, req)

		latency := time.Since(startTime)

		if err != nil {
			fmt.Printf("\n❌ Request failed: %v\n\n", err)

			// Provide friendly error messages
			fmt.Println("Possible causes:")
			fmt.Println("  • Invalid or expired API key (check ZAI_API_KEY)")
			fmt.Println("  • Network connectivity issues")
			fmt.Println("  • Z.ai API is temporarily unavailable")
			fmt.Println("  • Rate limit exceeded (too many requests)")
			fmt.Println("\nTroubleshooting steps:")
			fmt.Println("  1. Verify your API key in config.toml")
			fmt.Println("  2. Check your internet connection")
			fmt.Println("  3. Try again in a few minutes")
			fmt.Println("  4. Check Z.ai status page")
			os.Exit(1)
		}

		// Display success message
		fmt.Printf("✅ Request successful!\n\n")

		// Display response details
		fmt.Printf("📥 Response Details:\n")
		fmt.Printf("   Model:        %s\n", resp.Model)
		fmt.Printf("   Latency:      %v\n", latency)
		fmt.Printf("   Finish Reason: %s\n\n", resp.FinishReason)

		// Display response content
		fmt.Printf("📝 Response Content:\n")
		fmt.Printf("   %q\n\n", resp.Content)

		// Display token usage
		fmt.Printf("📊 Token Usage:\n")
		fmt.Printf("   Prompt Tokens:     %6d\n", resp.Usage.PromptTokens)
		fmt.Printf("   Completion Tokens: %6d\n", resp.Usage.CompletionTokens)
		fmt.Printf("   Total Tokens:      %6d\n\n", resp.Usage.TotalTokens)

		// Display tool calls if any
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("🔧 Tool Calls: %d\n", len(resp.ToolCalls))
			for i, tc := range resp.ToolCalls {
				fmt.Printf("   %d. %s(%s)\n", i+1, tc.Name, tc.Arguments)
			}
			fmt.Println()
		}

		// Check finish reason
		switch resp.FinishReason {
		case llm.FinishReasonStop:
			fmt.Println("✨ Model completed generation normally")
		case llm.FinishReasonLength:
			fmt.Println("⚠️  Model stopped due to max_tokens limit")
		case llm.FinishReasonToolCalls:
			fmt.Println("🔧 Model requested tool/function calls")
		case llm.FinishReasonError:
			fmt.Println("❌ Model stopped due to an error")
		}

		fmt.Println("\n✨ All checks passed! Your LLM provider is working correctly.")
	},
}

func init() {
	// Add test llm subcommand to test command
	testCmd.AddCommand(testLLMCmd)

	// Add flags
	testLLMCmd.Flags().StringVarP(&testConfigPath, "config", "c", "", "Path to configuration file (default: ./config.toml)")
	testLLMCmd.Flags().StringP("model", "m", "", "Override model to use (e.g., glm-4.7)")
}
