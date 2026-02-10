package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aatumaykin/nexbot/internal/heartbeat"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

// Context defines the structure for context components.
type Context struct {
	Workspace string // Workspace directory path
}

// Builder builds system prompts from various context components.
type Builder struct {
	workspace string
	timezone  string
}

// Config holds configuration for the context builder.
type Config struct {
	Workspace string // Workspace directory path
	Timezone  string // User timezone (e.g., "Europe/Moscow")
}

// NewBuilder creates a new context builder.
func NewBuilder(config Config) (*Builder, error) {
	if config.Workspace == "" {
		return nil, fmt.Errorf("workspace path cannot be empty")
	}

	// Verify workspace exists
	if _, err := os.Stat(config.Workspace); err != nil {
		return nil, fmt.Errorf("workspace directory not found: %w", err)
	}

	return &Builder{
		workspace: config.Workspace,
		timezone:  config.Timezone,
	}, nil
}

// Build creates a system prompt by combining context components in priority order:
// AGENTS → IDENTITY → USER → TOOLS → HEARTBEAT → memory
func (b *Builder) Build() (string, error) {
	var builder strings.Builder

	// 1. AGENTS - Agent instructions and behavior
	agents, err := b.readFile(workspace.BootstrapAgents)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read AGENTS.md: %w", err)
	}
	if agents != "" {
		processed, err := b.processTemplates(agents)
		if err != nil {
			return "", fmt.Errorf("failed to process AGENTS.md templates: %w", err)
		}
		builder.WriteString(processed)
		builder.WriteString("\n\n---\n\n")
	}

	// 2. IDENTITY - Core identity and purpose
	identity, err := b.readFile(workspace.BootstrapIdentity)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read IDENTITY.md: %w", err)
	}
	if identity != "" {
		processed, err := b.processTemplates(identity)
		if err != nil {
			return "", fmt.Errorf("failed to process IDENTITY.md templates: %w", err)
		}
		builder.WriteString(processed)
		builder.WriteString("\n\n---\n\n")
	}

	// 3. USER - User profile and preferences
	user, err := b.readFile(workspace.BootstrapUser)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read USER.md: %w", err)
	}
	if user != "" {
		processed, err := b.processTemplates(user)
		if err != nil {
			return "", fmt.Errorf("failed to process USER.md templates: %w", err)
		}
		builder.WriteString(processed)
		builder.WriteString("\n\n---\n\n")
	}

	// 4. TOOLS - Available tools and operations
	tools, err := b.readFile(workspace.BootstrapTools)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read TOOLS.md: %w", err)
	}
	if tools != "" {
		processed, err := b.processTemplates(tools)
		if err != nil {
			return "", fmt.Errorf("failed to process TOOLS.md templates: %w", err)
		}
		builder.WriteString(processed)
		builder.WriteString("\n\n---\n\n")
	}

	// 5. HEARTBEAT - Active periodic tasks
	heartbeatContent, err := b.buildHeartbeatContext()
	if err != nil {
		return "", fmt.Errorf("failed to build heartbeat context: %w", err)
	}
	if heartbeatContent != "" {
		builder.WriteString(heartbeatContent)
		builder.WriteString("\n\n---\n\n")
	}

	return builder.String(), nil
}

// BuildWithMemory creates a system prompt with memory context appended.
// This includes all components from Build() plus memory messages.
func (b *Builder) BuildWithMemory(messages []llm.Message) (string, error) {
	systemPrompt, err := b.Build()
	if err != nil {
		return "", err
	}

	if len(messages) == 0 {
		return systemPrompt, nil
	}

	// Append memory section
	var memoryBuilder strings.Builder
	memoryBuilder.WriteString("## Recent Conversation Memory\n\n")

	for _, msg := range messages {
		switch msg.Role {
		case llm.RoleSystem:
			memoryBuilder.WriteString(fmt.Sprintf("**System:** %s\n\n", msg.Content))
		case llm.RoleUser:
			memoryBuilder.WriteString(fmt.Sprintf("**User:** %s\n\n", msg.Content))
		case llm.RoleAssistant:
			memoryBuilder.WriteString(fmt.Sprintf("**Assistant:** %s\n\n", msg.Content))
		case llm.RoleTool:
			memoryBuilder.WriteString(fmt.Sprintf("**Tool [%s]:** %s\n\n", msg.ToolCallID, msg.Content))
		}
	}

	memorySection := memoryBuilder.String()

	return systemPrompt + memorySection, nil
}

// BuildForSession creates a system prompt optimized for a specific session.
// This can be customized per session based on session-specific context.
func (b *Builder) BuildForSession(sessionID string, messages []llm.Message) (string, error) {
	systemPrompt, err := b.BuildWithMemory(messages)
	if err != nil {
		return "", err
	}

	// Parse sessionID to extract channel and chat_id
	var sessionInfo string
	if strings.Contains(sessionID, ":") {
		parts := strings.SplitN(sessionID, ":", 2)
		if len(parts) == 2 {
			channel := parts[0]
			chatID := parts[1]
			sessionInfo = fmt.Sprintf("# Session Information\n\n- **Session ID:** %s\n- **Channel:** %s\n- **Chat ID:** %s\n\n", sessionID, channel, chatID)
		} else {
			sessionInfo = fmt.Sprintf("# Session: %s\n\n", sessionID)
		}
	} else {
		sessionInfo = fmt.Sprintf("# Session: %s\n\n", sessionID)
	}

	// DEBUG: Log what's in the system prompt
	// We'll split it into parts for readability
	systemPromptWithSession := sessionInfo + systemPrompt

	return systemPromptWithSession, nil
}

// ReadMemory reads memory files from the workspace memory directory.
func (b *Builder) ReadMemory() ([]llm.Message, error) {
	memoryDir := filepath.Join(b.workspace, "memory")

	// Check if memory directory exists
	if _, err := os.Stat(memoryDir); os.IsNotExist(err) {
		return []llm.Message{}, nil
	}

	// Read all files in memory directory
	entries, err := os.ReadDir(memoryDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read memory directory: %w", err)
	}

	var messages []llm.Message

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Skip non-markdown files
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(memoryDir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read memory file %s: %w", entry.Name(), err)
		}

		// Create a system message from file content
		messages = append(messages, llm.Message{
			Role:    llm.RoleSystem,
			Content: string(content),
		})
	}

	return messages, nil
}

// processTemplates replaces template variables with actual values.
func (b *Builder) processTemplates(content string) (string, error) {
	now := time.Now()

	timezone := b.timezone
	if timezone == "" {
		timezone = "UTC"
	}

	data := map[string]string{
		"CURRENT_TIME":   now.Format("15:04:05"),
		"CURRENT_DATE":   now.Format("2006-01-02"),
		"WORKSPACE_PATH": b.workspace,
		"TIMEZONE":       timezone,
	}

	result := content
	for key, value := range data {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result, nil
}

// readFile reads a file from the workspace.
func (b *Builder) readFile(filename string) (string, error) {
	filePath := filepath.Join(b.workspace, filename)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// GetWorkspace returns the workspace path.
func (b *Builder) GetWorkspace() string {
	return b.workspace
}

// GetComponent returns a specific context component by name.
func (b *Builder) GetComponent(name string) (string, error) {
	switch name {
	case "IDENTITY":
		return b.readFile(workspace.BootstrapIdentity)
	case "AGENTS":
		return b.readFile(workspace.BootstrapAgents)
	case "USER":
		return b.readFile(workspace.BootstrapUser)
	case "TOOLS":
		return b.readFile(workspace.BootstrapTools)
	default:
		return "", fmt.Errorf("unknown component: %s", name)
	}
}

// buildHeartbeatContext builds heartbeat context from HEARTBEAT.md file.
func (b *Builder) buildHeartbeatContext() (string, error) {
	// Read HEARTBEAT.md file
	heartbeatFile, err := b.readFile(workspace.BootstrapHeartbeat)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read HEARTBEAT.md: %w", err)
	}

	// If file doesn't exist, return empty string
	if err != nil && os.IsNotExist(err) {
		return "", nil
	}

	// Parse heartbeat file
	parser := heartbeat.NewParser()
	tasks, err := parser.Parse(heartbeatFile)
	if err != nil {
		return "", fmt.Errorf("failed to parse HEARTBEAT.md: %w", err)
	}

	// Return formatted context
	return "## Heartbeat Tasks\n\n" + heartbeat.FormatContext(tasks) + "\n", nil
}
