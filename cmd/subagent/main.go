package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/subagent/prompts"
	"github.com/aatumaykin/nexbot/internal/subagent/sanitizer"
	"github.com/aatumaykin/nexbot/internal/tools"
)

const (
	ProtocolVersion   = "1.0"
	DefaultSkillsPath = "/workspace/skills"
	DefaultTimezone   = "UTC"
	MaxRequestSize    = 1 * 1024 * 1024
)

type SubagentRequest struct {
	Version       string            `json:"version"`
	ID            string            `json:"id"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	Type          string            `json:"type"`
	Task          string            `json:"task"`
	Timeout       int               `json:"timeout"`
	Deadline      int64             `json:"deadline,omitempty"`
	Secrets       map[string]string `json:"secrets,omitempty"`
	LLMAPIKey     string            `json:"llm_api_key,omitempty"`
}

type Subagent struct {
	secretsStore interface {
		SetAll(secrets map[string]string) error
		Clear()
	}
	llmProvider  llm.Provider
	loop         *loop.Loop
	registry     *tools.Registry
	systemPrompt string
	validator    *sanitizer.Validator
	promptLoader *prompts.PromptLoader
	log          *logger.Logger
}

func NewSubagent() *Subagent {
	registry := tools.NewRegistry()
	registerSubagentTools(registry)

	log, _ := logger.New(logger.Config{Level: "info"})

	return &Subagent{
		secretsStore: newSimpleSecretsStore(),
		registry:     registry,
		validator:    sanitizer.NewValidator(sanitizer.SanitizerConfig{}),
		promptLoader: prompts.NewPromptLoader(""),
		log:          log,
	}
}

type simpleSecretsStore struct {
	secrets map[string]string
}

func newSimpleSecretsStore() *simpleSecretsStore {
	return &simpleSecretsStore{secrets: make(map[string]string)}
}

func (s *simpleSecretsStore) SetAll(secrets map[string]string) error {
	s.secrets = secrets
	return nil
}

func (s *simpleSecretsStore) Clear() {
	s.secrets = make(map[string]string)
}

func (s *Subagent) InitLLM(apiKey string) error {
	s.llmProvider = llm.NewZAIProvider(llm.ZAIConfig{
		APIKey: apiKey,
		Model:  "glm-4-flash",
	}, s.log)

	skillsPath := getSkillsPath()
	promptBuilder := prompts.NewSubagentPromptBuilder(DefaultTimezone, s.registry, skillsPath)
	s.systemPrompt = promptBuilder.Build()

	workspace := getWorkspacePath()
	sessionDir := workspace + "/sessions"

	loopInstance, err := loop.NewLoop(loop.Config{
		Workspace:         workspace,
		SessionDir:        sessionDir,
		Timezone:          DefaultTimezone,
		LLMProvider:       s.llmProvider,
		Logger:            s.log,
		Model:             "glm-4-flash",
		MaxToolIterations: 10,
	})
	if err != nil {
		return fmt.Errorf("failed to create loop: %w", err)
	}
	s.loop = loopInstance

	return nil
}

func main() {
	subagent := NewSubagent()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		logInfo("", "shutdown", "received shutdown signal", nil)
		cancel()
	}()

	limitedReader := io.LimitReader(os.Stdin, MaxRequestSize)
	scanner := bufio.NewScanner(limitedReader)
	scanner.Buffer(make([]byte, 64*1024), MaxRequestSize)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			logInfo("", "shutdown", "shutdown complete", nil)
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req SubagentRequest
		if err := json.Unmarshal(line, &req); err != nil {
			sendError("", "", fmt.Errorf("invalid JSON: %w", err))
			continue
		}

		compatible, deprecated := isCompatibleVersion(req.Version)
		if !compatible {
			sendError(req.ID, req.CorrelationID,
				fmt.Errorf("unsupported protocol version: %s", req.Version))
			continue
		}
		if deprecated {
			logInfo(req.ID, "protocol", "deprecated version", map[string]interface{}{
				"version": req.Version,
			})
		}

		if req.Type == "ping" {
			sendPong(req.ID, req.CorrelationID)
			continue
		}

		if req.LLMAPIKey != "" && subagent.llmProvider == nil {
			if err := subagent.InitLLM(req.LLMAPIKey); err != nil {
				sendError(req.ID, req.CorrelationID, fmt.Errorf("failed to initialize LLM: %w", err))
				continue
			}
			logInfo(req.ID, "init", "LLM client initialized", nil)
		}

		if subagent.loop == nil {
			sendError(req.ID, req.CorrelationID,
				fmt.Errorf("LLM not initialized: send llm_api_key"))
			continue
		}

		if req.Secrets != nil {
			if err := subagent.secretsStore.SetAll(req.Secrets); err != nil {
				sendError(req.ID, req.CorrelationID, fmt.Errorf("secrets validation: %w", err))
				continue
			}
		}

		logInfo(req.ID, "task", "processing", map[string]interface{}{
			"correlation_id": req.CorrelationID,
			"secret_count":   len(req.Secrets),
		})

		preparedTask := sanitizer.PrepareTask(req.Task)

		taskCtx := ctx
		var taskCancel context.CancelFunc
		if req.Deadline > 0 {
			deadline := time.Unix(req.Deadline, 0)
			if time.Now().After(deadline) {
				sendError(req.ID, req.CorrelationID, fmt.Errorf("request expired"))
				continue
			}
			taskCtx, taskCancel = context.WithDeadline(ctx, deadline)
		}

		result, err := subagent.loop.Process(taskCtx, req.ID, preparedTask)

		if taskCancel != nil {
			taskCancel()
		}

		if result != "" {
			result = subagent.validator.SanitizeToolOutput(result)
		}

		subagent.secretsStore.Clear()

		if err != nil {
			logError(req.ID, "task", "failed", err.Error())
		} else {
			logInfo(req.ID, "task", "completed", nil)
		}

		sendResponse(req.ID, req.CorrelationID, result, err)
	}
}

func isCompatibleVersion(v string) (compatible bool, deprecated bool) {
	switch v {
	case "", "1.0":
		return true, false
	case "0.9":
		return true, true
	default:
		return false, false
	}
}

func getSkillsPath() string {
	if path := os.Getenv("SKILLS_PATH"); path != "" {
		return path
	}
	return DefaultSkillsPath
}

func getWorkspacePath() string {
	if path := os.Getenv("WORKSPACE_PATH"); path != "" {
		return path
	}
	return "/workspace"
}

func sendResponse(id, correlationID, result string, err error) {
	resp := map[string]interface{}{
		"id":             id,
		"correlation_id": correlationID,
		"version":        ProtocolVersion,
		"status":         "success",
		"result":         result,
	}
	if err != nil {
		resp["status"] = "error"
		resp["error"] = err.Error()
	}
	data, _ := json.Marshal(resp)
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))
}

func sendError(id, correlationID string, err error) {
	resp := map[string]interface{}{
		"id":             id,
		"correlation_id": correlationID,
		"version":        ProtocolVersion,
		"status":         "error",
		"error":          err.Error(),
	}
	data, _ := json.Marshal(resp)
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))
}

func sendPong(id, correlationID string) {
	resp := map[string]interface{}{
		"id":             id,
		"correlation_id": correlationID,
		"version":        ProtocolVersion,
		"status":         "pong",
	}
	data, _ := json.Marshal(resp)
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))
}

type LogEntry struct {
	Time          string                 `json:"time"`
	Level         string                 `json:"level"`
	Message       string                 `json:"message"`
	TaskID        string                 `json:"task_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

func logInfo(taskID, event, msg string, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["event"] = event
	entry := LogEntry{
		Time:     time.Now().Format(time.RFC3339),
		Level:    "info",
		Message:  msg,
		TaskID:   taskID,
		Metadata: metadata,
	}
	data, _ := json.Marshal(entry)
	os.Stderr.Write(data)
	os.Stderr.Write([]byte("\n"))
}

func logError(taskID, event, msg, errMsg string) {
	entry := LogEntry{
		Time:    time.Now().Format(time.RFC3339),
		Level:   "error",
		Message: msg,
		TaskID:  taskID,
		Metadata: map[string]interface{}{
			"event": event,
			"error": errMsg,
		},
	}
	data, _ := json.Marshal(entry)
	os.Stderr.Write(data)
	os.Stderr.Write([]byte("\n"))
}
