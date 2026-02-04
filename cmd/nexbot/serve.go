package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/channels/telegram"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/tools"
	"github.com/aatumaykin/nexbot/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	serveConfigPath string
	serveLogLevel   string
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start Nexbot agent (main command)",
	Long: `Start Nexbot agent with specified configuration.
This will initialize all components (logger, message bus, channels, agent loop)
and handle graceful shutdown.

The serve command is the main entry point for running Nexbot.`,
	Run: serveHandler,
}

func serveHandler(cmd *cobra.Command, args []string) {
	// Load .env file if exists
	envFile := "./.env"
	if _, err := os.Stat(envFile); err == nil {
		data, err := os.ReadFile(envFile)
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					os.Setenv(key, value)
				}
			}
		}
	}

	// Determine config path
	configPath := serveConfigPath
	if configPath == "" {
		configPath = "./config.toml"
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override log level if flag is set
	if serveLogLevel != "" {
		cfg.Logging.Level = serveLogLevel
	}

	// Validate configuration
	if errors := cfg.Validate(); len(errors) > 0 {
		fmt.Printf("❌ Configuration validation failed:\n")
		for _, e := range errors {
			fmt.Printf("  - %v\n", e)
		}
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
	logger.SetDefault(log)

	// Log startup information
	log.Info("🚀 Starting Nexbot",
		logger.Field{Key: "version", Value: Version},
		logger.Field{Key: "git_commit", Value: GitCommit},
		logger.Field{Key: "config", Value: configPath},
		logger.Field{Key: "workspace", Value: cfg.Workspace.Path},
		logger.Field{Key: "llm_provider", Value: cfg.LLM.Provider},
		logger.Field{Key: "message_bus_capacity", Value: cfg.MessageBus.Capacity},
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize message bus
	log.Info("📡 Initializing message bus",
		logger.Field{Key: "capacity", Value: cfg.MessageBus.Capacity})
	messageBus := bus.New(cfg.MessageBus.Capacity, log)
	if err := messageBus.Start(ctx); err != nil {
		log.Error("Failed to start message bus", err,
			logger.Field{Key: "capacity", Value: cfg.MessageBus.Capacity})
		os.Exit(1)
	}

	// Initialize LLM provider
	var llmProvider llm.Provider
	switch cfg.LLM.Provider {
	case "zai":
		llmProvider = llm.NewZAIProvider(llm.ZAIConfig{
			APIKey: cfg.LLM.ZAI.APIKey,
			Model:  cfg.LLM.ZAI.Model,
		}, log)
		log.Info("✅ Z.ai LLM provider initialized")
	default:
		log.Error("Unsupported LLM provider", nil,
			logger.Field{Key: "provider", Value: cfg.LLM.Provider})
		os.Exit(1)
	}

	// Initialize workspace
	ws := workspace.New(cfg.Workspace)
	if err := ws.EnsureDir(); err != nil {
		log.Error("Failed to create workspace directory", err)
		os.Exit(1)
	}

	if err := ws.EnsureSubpath("sessions"); err != nil {
		log.Error("Failed to create sessions directory", err)
		os.Exit(1)
	}

	// Initialize agent loop
	log.Info("🤖 Initializing agent loop")
	agentLoop, err := loop.NewLoop(loop.Config{
		Workspace:   cfg.Workspace.Path,
		SessionDir:  ws.Subpath("sessions"),
		LLMProvider: llmProvider,
		Logger:      log,
		Model:       cfg.Agent.Model,
		MaxTokens:   cfg.Agent.MaxTokens,
		Temperature: cfg.Agent.Temperature,
	})
	if err != nil {
		log.Error("Failed to initialize agent loop", err)
		os.Exit(1)
	}

	// Register tools
	if cfg.Tools.Shell.Enabled {
		shellTool := tools.NewShellExecTool(cfg, log)
		agentLoop.RegisterTool(shellTool)
		log.Info("✅ Shell tool registered")
	}

	if cfg.Tools.File.Enabled {
		readFileTool := tools.NewReadFileTool(ws)
		writeFileTool := tools.NewWriteFileTool(ws)
		listDirTool := tools.NewListDirTool(ws)
		agentLoop.RegisterTool(readFileTool)
		agentLoop.RegisterTool(writeFileTool)
		agentLoop.RegisterTool(listDirTool)
		log.Info("✅ File tools registered")
	}

	// Initialize Telegram connector if enabled
	var telegramConnector *telegram.Connector
	if cfg.Channels.Telegram.Enabled {
		log.Info("📱 Initializing Telegram connector")
		telegramConnector = telegram.New(cfg.Channels.Telegram, log, messageBus, llmProvider)
		if err := telegramConnector.Start(ctx); err != nil {
			log.Error("Failed to start Telegram connector", err)
			os.Exit(1)
		}
	} else {
		log.Warn("Telegram connector is disabled")
	}

	// Subscribe to inbound messages and process them
	inboundCh := messageBus.SubscribeInbound(ctx)
	go func() {
		for msg := range inboundCh {
			log.InfoCtx(ctx, "Processing inbound message",
				logger.Field{Key: "user_id", Value: msg.UserID},
				logger.Field{Key: "session_id", Value: msg.SessionID})

			// Check for special commands in metadata
			if msg.Metadata != nil {
				if cmd, ok := msg.Metadata["command"].(string); ok && cmd == "new_session" {
					// Handle /new command - clear session
					log.InfoCtx(ctx, "Clearing session due to /new command",
						logger.Field{Key: "session_id", Value: msg.SessionID})

					if err := agentLoop.ClearSession(ctx, msg.SessionID); err != nil {
						log.ErrorCtx(ctx, "Failed to clear session", err,
							logger.Field{Key: "session_id", Value: msg.SessionID})
					} else {
						log.InfoCtx(ctx, "Session cleared successfully",
							logger.Field{Key: "session_id", Value: msg.SessionID})

						// Send confirmation message
						confirmationMsg := bus.NewOutboundMessage(
							msg.ChannelType,
							msg.UserID,
							msg.SessionID,
							"✅ Session cleared. Starting a fresh conversation!",
							nil,
						)
						if err := messageBus.PublishOutbound(*confirmationMsg); err != nil {
							log.ErrorCtx(ctx, "Failed to publish confirmation message", err)
						}
					}
					continue
				}

				if cmd, ok := msg.Metadata["command"].(string); ok && cmd == "status" {
					// Handle /status command - show session and bot status
					log.InfoCtx(ctx, "Getting status for session",
						logger.Field{Key: "session_id", Value: msg.SessionID})

					status, err := agentLoop.GetSessionStatus(ctx, msg.SessionID)
					if err != nil {
						log.ErrorCtx(ctx, "Failed to get session status", err,
							logger.Field{Key: "session_id", Value: msg.SessionID})

						errorMsg := bus.NewOutboundMessage(
							msg.ChannelType,
							msg.UserID,
							msg.SessionID,
							"❌ Failed to get status information. Please try again later.",
							nil,
						)
						if err := messageBus.PublishOutbound(*errorMsg); err != nil {
							log.ErrorCtx(ctx, "Failed to publish error message", err)
						}
					} else {
						// Format status message
						statusMsg := formatStatusMessage(status)

						// Send status message
						responseMsg := bus.NewOutboundMessage(
							msg.ChannelType,
							msg.UserID,
							msg.SessionID,
							statusMsg,
							nil,
						)
						if err := messageBus.PublishOutbound(*responseMsg); err != nil {
							log.ErrorCtx(ctx, "Failed to publish status message", err)
						}
					}
					continue
				}
			}

			// Process regular message
			response, err := agentLoop.Process(ctx, msg.SessionID, msg.Content)
			if err != nil {
				log.ErrorCtx(ctx, "Failed to process message", err,
					logger.Field{Key: "session_id", Value: msg.SessionID})
				response = fmt.Sprintf("Error: %v", err)
			}

			if response != "" {
				outboundMsg := bus.NewOutboundMessage(
					msg.ChannelType,
					msg.UserID,
					msg.SessionID,
					response,
					nil,
				)
				if err := messageBus.PublishOutbound(*outboundMsg); err != nil {
					log.ErrorCtx(ctx, "Failed to publish outbound message", err)
				}
			}
		}
	}()

	log.Info("✅ Nexbot is running")

	// Wait for shutdown signal
	sig := <-sigChan
	log.Info("⏳ Received shutdown signal",
		logger.Field{Key: "signal", Value: sig.String()})

	// Graceful shutdown
	log.Info("🛑 Shutting down Nexbot...")
	cancel()

	// Stop Telegram connector if enabled
	if telegramConnector != nil {
		if err := telegramConnector.Stop(); err != nil {
			log.Error("Failed to stop Telegram connector", err)
		}
	}

	// Stop message bus
	if err := messageBus.Stop(); err != nil {
		log.Error("Failed to stop message bus", err)
		os.Exit(1)
	}

	log.Info("👋 Nexbot stopped gracefully")
	os.Exit(0)
}

func formatStatusMessage(status map[string]any) string {
	var builder strings.Builder

	builder.WriteString("📊 **Session Status**\n\n")

	// Session info
	sessionID, _ := status["session_id"].(string)
	builder.WriteString(fmt.Sprintf("**Session ID:** `%s`\n", sessionID))

	// Message count
	msgCount, _ := status["message_count"].(int)
	builder.WriteString(fmt.Sprintf("**Messages:** %d\n", msgCount))

	// File size
	fileSizeHuman, _ := status["file_size_human"].(string)
	builder.WriteString(fmt.Sprintf("**Session Size:** %s\n", fileSizeHuman))

	builder.WriteString("\n**LLM Configuration:**\n")

	// Model
	model, _ := status["model"].(string)
	builder.WriteString(fmt.Sprintf("**Model:** %s\n", model))

	// Temperature
	temperature, _ := status["temperature"].(float64)
	builder.WriteString(fmt.Sprintf("**Temperature:** %.2f\n", temperature))

	// Max tokens
	maxTokens, _ := status["max_tokens"].(int)
	builder.WriteString(fmt.Sprintf("**Max Tokens:** %d\n", maxTokens))

	// Provider
	provider, _ := status["provider"].(string)
	builder.WriteString(fmt.Sprintf("**Provider:** %s\n", provider))

	return builder.String()
}

func init() {
	serveCmd.Flags().StringVarP(&serveConfigPath, "config", "c", "", "Path to configuration file (default: ./config.toml)")
	serveCmd.Flags().StringVarP(&serveLogLevel, "log-level", "l", "", "Override log level (debug, info, warn, error)")
}
