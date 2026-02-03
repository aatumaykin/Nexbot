package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aatumaykin/nexbot/internal/llm"
)

// Context defines the structure for context components.
type Context struct {
	Workspace string // Workspace directory path
}

// Builder builds system prompts from various context components.
type Builder struct {
	workspace string
	context   Context
}

// Config holds configuration for the context builder.
type Config struct {
	Workspace string // Workspace directory path
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
		context: Context{
			Workspace: config.Workspace,
		},
	}, nil
}

// Build creates a system prompt by combining context components in priority order:
// IDENTITY → AGENTS → SOUL → USER → TOOLS → memory
func (b *Builder) Build() (string, error) {
	var builder strings.Builder

	// 1. IDENTITY - Core identity and purpose
	identity, err := b.readFile("IDENTITY.md")
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

	// 2. AGENTS - Agent instructions and behavior
	agents, err := b.readFile("AGENTS.md")
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

	// 3. SOUL - Personality and tone
	soul, err := b.readFile("SOUL.md")
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read SOUL.md: %w", err)
	}
	if soul != "" {
		processed, err := b.processTemplates(soul)
		if err != nil {
			return "", fmt.Errorf("failed to process SOUL.md templates: %w", err)
		}
		builder.WriteString(processed)
		builder.WriteString("\n\n---\n\n")
	}

	// 4. USER - User profile and preferences
	user, err := b.readFile("USER.md")
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

	// 5. TOOLS - Available tools and operations
	tools, err := b.readFile("TOOLS.md")
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

	// Add session-specific header
	sessionHeader := fmt.Sprintf("# Session: %s\n", sessionID)

	return sessionHeader + "\n" + systemPrompt, nil
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

	data := map[string]string{
		"CURRENT_TIME":      now.Format("15:04:05"),
		"CURRENT_DATE":      now.Format("2006-01-02"),
		"WORKSPACE_PATH":    b.workspace,
		"USER_NAME":         "", // TODO: Get from config
		"USER_TIMEZONE":     "", // TODO: Get from config
		"USER_EMAIL":        "", // TODO: Get from config
		"USER_TELEGRAM":     "", // TODO: Get from config
		"USER_GITHUB":       "", // TODO: Get from config
		"PROJECT_LIST":      "", // TODO: Get from workspace
		"FREQUENT_COMMANDS": "", // TODO: Get from workspace
		"REMINDER_1":        "", // TODO: Get from workspace
		"REMINDER_2":        "", // TODO: Get from workspace
		"REMINDER_3":        "", // TODO: Get from workspace
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
	case "IDENTITY", "AGENTS", "SOUL", "USER", "TOOLS":
		return b.readFile(name + ".md")
	default:
		return "", fmt.Errorf("unknown component: %s", name)
	}
}
