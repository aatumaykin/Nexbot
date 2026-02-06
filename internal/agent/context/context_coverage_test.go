package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

// TestBuilderGetWorkspace tests the GetWorkspace method
func TestBuilderGetWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	builder, err := NewBuilder(Config{Workspace: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	workspacePath := builder.GetWorkspace()
	if workspacePath != tmpDir {
		t.Errorf("GetWorkspace() = %v, want %v", workspacePath, tmpDir)
	}
}

// TestBuilderGetComponent tests the GetComponent method
func TestBuilderGetComponent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test bootstrap files
	identityContent := "# Identity\nThis is a test identity."
	if err := os.WriteFile(filepath.Join(tmpDir, workspace.BootstrapIdentity), []byte(identityContent), 0644); err != nil {
		t.Fatalf("Failed to create IDENTITY.md: %v", err)
	}

	agentsContent := "# Agents\nTest agents content."
	if err := os.WriteFile(filepath.Join(tmpDir, workspace.BootstrapAgents), []byte(agentsContent), 0644); err != nil {
		t.Fatalf("Failed to create AGENTS.md: %v", err)
	}

	userContent := "# User\nTest user content."
	if err := os.WriteFile(filepath.Join(tmpDir, workspace.BootstrapUser), []byte(userContent), 0644); err != nil {
		t.Fatalf("Failed to create USER.md: %v", err)
	}

	toolsContent := "# Tools\nTest tools content."
	if err := os.WriteFile(filepath.Join(tmpDir, workspace.BootstrapTools), []byte(toolsContent), 0644); err != nil {
		t.Fatalf("Failed to create TOOLS.md: %v", err)
	}

	builder, err := NewBuilder(Config{Workspace: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	tests := []struct {
		name            string
		component       string
		expectedContent string
		expectErr       bool
	}{
		{
			name:            "IDENTITY component",
			component:       "IDENTITY",
			expectedContent: identityContent,
			expectErr:       false,
		},
		{
			name:            "AGENTS component",
			component:       "AGENTS",
			expectedContent: agentsContent,
			expectErr:       false,
		},
		{
			name:            "USER component",
			component:       "USER",
			expectedContent: userContent,
			expectErr:       false,
		},
		{
			name:            "TOOLS component",
			component:       "TOOLS",
			expectedContent: toolsContent,
			expectErr:       false,
		},
		{
			name:            "unknown component",
			component:       "UNKNOWN",
			expectedContent: "",
			expectErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := builder.GetComponent(tt.component)

			if tt.expectErr && err == nil {
				t.Error("GetComponent() expected error but got none")
				return
			}
			if !tt.expectErr && err != nil {
				t.Errorf("GetComponent() unexpected error: %v", err)
				return
			}
			if !tt.expectErr && content != tt.expectedContent {
				t.Errorf("GetComponent() = %v, want %v", content, tt.expectedContent)
			}
		})
	}
}

// TestBuilderGetComponentMissingFile tests GetComponent with missing file
func TestBuilderGetComponentMissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	builder, err := NewBuilder(Config{Workspace: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Try to get IDENTITY component which doesn't exist
	content, err := builder.GetComponent("IDENTITY")

	if err == nil {
		t.Error("GetComponent() should return error for missing file")
	}
	if content != "" {
		t.Errorf("GetComponent() should return empty string for missing file, got %v", content)
	}
}

// TestBuilderBuildWithHeartbeat tests Build with heartbeat context
func TestBuilderBuildWithHeartbeat(t *testing.T) {
	tmpDir := t.TempDir()

	// Create HEARTBEAT.md file with active tasks
	heartbeatContent := `# Heartbeat Tasks

## Daily Review

### Daily Standup
- Schedule: "0 9 * * *"
- Task: "Review daily progress, check for blocked tasks, update priorities"

### Weekly Summary
- Schedule: "0 17 * * 5"
- Task: "Generate weekly summary report"
`
	if err := os.WriteFile(filepath.Join(tmpDir, workspace.BootstrapHeartbeat), []byte(heartbeatContent), 0644); err != nil {
		t.Fatalf("Failed to create HEARTBEAT.md: %v", err)
	}

	builder, err := NewBuilder(Config{Workspace: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// Verify heartbeat content is included
	if !strings.Contains(result, "Daily Standup") || !strings.Contains(result, "Weekly Summary") {
		t.Error("Build() should include heartbeat context")
	}
}

// TestBuilderBuildWithMemoryEmpty tests BuildWithMemory with empty memory
func TestBuilderBuildWithMemoryEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a bootstrap file so Build returns something
	identityContent := "# Identity\nTest identity."
	if err := os.WriteFile(filepath.Join(tmpDir, workspace.BootstrapIdentity), []byte(identityContent), 0644); err != nil {
		t.Fatalf("Failed to create IDENTITY.md: %v", err)
	}

	builder, err := NewBuilder(Config{Workspace: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Build with empty memory
	messages := []llm.Message{}
	result, err := builder.BuildWithMemory(messages)
	if err != nil {
		t.Fatalf("BuildWithMemory() error: %v", err)
	}

	// Verify result is not empty
	if result == "" {
		t.Error("BuildWithMemory() should return non-empty result")
	}

	// Verify memory section is NOT included (empty memory)
	if strings.Contains(result, "Recent Conversation Memory") {
		t.Error("BuildWithMemory() should not include memory section when empty")
	}
}

// TestBuilderBuildWithMemoryMultipleMessages tests BuildWithMemory with multiple messages
func TestBuilderBuildWithMemoryMultipleMessages(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a bootstrap file so Build returns something
	identityContent := "# Identity\nTest identity."
	if err := os.WriteFile(filepath.Join(tmpDir, workspace.BootstrapIdentity), []byte(identityContent), 0644); err != nil {
		t.Fatalf("Failed to create IDENTITY.md: %v", err)
	}

	builder, err := NewBuilder(Config{Workspace: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Build with multiple memory messages
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: "System message"},
		{Role: llm.RoleUser, Content: "User message 1"},
		{Role: llm.RoleAssistant, Content: "Assistant response 1"},
		{Role: llm.RoleUser, Content: "User message 2"},
	}

	result, err := builder.BuildWithMemory(messages)
	if err != nil {
		t.Fatalf("BuildWithMemory() error: %v", err)
	}

	// Verify memory section is included
	if !strings.Contains(result, "Recent Conversation Memory") {
		t.Error("BuildWithMemory() should include memory section")
	}

	// Verify all messages are included
	if !strings.Contains(result, "User message 1") {
		t.Error("BuildWithMemory() should include user message 1")
	}
	if !strings.Contains(result, "Assistant response 1") {
		t.Error("BuildWithMemory() should include assistant response 1")
	}
	if !strings.Contains(result, "User message 2") {
		t.Error("BuildWithMemory() should include user message 2")
	}
}

// TestBuilderBuildForSession tests BuildForSession method
func TestBuilderBuildForSession(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a bootstrap file so Build returns something
	identityContent := "# Identity\nTest identity."
	if err := os.WriteFile(filepath.Join(tmpDir, workspace.BootstrapIdentity), []byte(identityContent), 0644); err != nil {
		t.Fatalf("Failed to create IDENTITY.md: %v", err)
	}

	builder, err := NewBuilder(Config{Workspace: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	sessionID := "test-session-123"
	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}

	result, err := builder.BuildForSession(sessionID, messages)
	if err != nil {
		t.Fatalf("BuildForSession() error: %v", err)
	}

	// Verify session header is included
	if !strings.Contains(result, "# Session: test-session-123") {
		t.Error("BuildForSession() should include session header")
	}
}

// TestBuilderProcessTemplates tests processTemplates with template variables
func TestBuilderProcessTemplates(t *testing.T) {
	tmpDir := t.TempDir()

	// Create IDENTITY.md with template variables
	identityContent := `# Identity
Workspace: {{WORKSPACE_PATH}}
Time: {{CURRENT_TIME}}
Date: {{CURRENT_DATE}}
`
	if err := os.WriteFile(filepath.Join(tmpDir, workspace.BootstrapIdentity), []byte(identityContent), 0644); err != nil {
		t.Fatalf("Failed to create IDENTITY.md: %v", err)
	}

	builder, err := NewBuilder(Config{Workspace: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// Verify template variables are replaced
	if strings.Contains(result, "{{WORKSPACE_PATH}}") {
		t.Error("Template variable WORKSPACE_PATH should be replaced")
	}
	if strings.Contains(result, "{{CURRENT_TIME}}") {
		t.Error("Template variable CURRENT_TIME should be replaced")
	}
	if strings.Contains(result, "{{CURRENT_DATE}}") {
		t.Error("Template variable CURRENT_DATE should be replaced")
	}

	// Verify workspace path is included
	if !strings.Contains(result, tmpDir) {
		t.Error("Result should contain workspace path")
	}
}
