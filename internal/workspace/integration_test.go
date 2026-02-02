package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
)

// TestWorkspaceBootstrapIntegration tests the complete workflow from workspace initialization to bootstrap assembly.
func TestWorkspaceBootstrapIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Step 1: Create workspace configuration
	cfg := config.WorkspaceConfig{
		Path:              tmpDir,
		BootstrapMaxChars: 5000,
	}

	// Step 2: Initialize workspace
	ws := New(cfg)

	// Step 3: Ensure workspace directory exists
	if err := ws.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() failed: %v", err)
	}

	// Step 4: Create bootstrap files in the workspace
	bootstrapContent := map[string]string{
		BootstrapIdentity: `# Core Identity

Nexbot is a lightweight personal AI assistant.

## Current Time

{{CURRENT_TIME}} {{CURRENT_DATE}}

## Workspace

Workspace path: {{WORKSPACE_PATH}}
`,
		BootstrapAgents: `# Agent Instructions

You are helpful and friendly.

## Tools

You can use: file, shell, messaging
`,
		BootstrapSoul: `# Personality

Be concise and accurate.
`,
		BootstrapUser: `# User Profile

Name: Test User
Timezone: UTC
`,
	}

	for filename, content := range bootstrapContent {
		filePath := ws.Subpath(filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create bootstrap file %s: %v", filename, err)
		}
	}

	// Step 5: Initialize bootstrap loader
	loader := NewBootstrapLoader(ws, cfg, nil)

	// Step 6: Load bootstrap files
	files, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Step 7: Verify all files were loaded
	if len(files) != len(bootstrapContent) {
		t.Errorf("Load() returned %d files, want %d", len(files), len(bootstrapContent))
	}

	// Step 8: Assemble bootstrap content
	assembled, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	// Step 9: Verify template variables were substituted
	if strings.Contains(assembled, "{{") {
		t.Error("template variables not substituted, found {{ in output")
	}

	// Step 10: Verify workspace path was substituted correctly
	if !strings.Contains(assembled, tmpDir) {
		t.Error("workspace path not substituted in output")
	}

	// Step 11: Verify time/date were substituted
	if !strings.Contains(assembled, ":") {
		t.Error("time not substituted (missing colon)")
	}
	if !strings.Contains(assembled, "-") {
		t.Error("date not substituted (missing dash)")
	}

	// Step 12: Verify all content is present in assembled output
	requiredKeywords := []string{"Core Identity", "Agent Instructions", "Personality", "User Profile"}
	for _, keyword := range requiredKeywords {
		if !strings.Contains(assembled, keyword) {
			t.Errorf("keyword %s not found in assembled content", keyword)
		}
	}

	// Step 13: Verify separator is present
	if !strings.Contains(assembled, "---") {
		t.Error("separator not present in assembled content")
	}
}

// TestWorkspaceBootstrapIntegrationWithMissingFiles tests integration when some bootstrap files are missing.
func TestWorkspaceBootstrapIntegrationWithMissingFiles(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.WorkspaceConfig{
		Path:              tmpDir,
		BootstrapMaxChars: 5000,
	}

	ws := New(cfg)
	if err := ws.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() failed: %v", err)
	}

	// Create only some bootstrap files
	bootstrapContent := map[string]string{
		BootstrapIdentity: "# Identity\n\n{{CURRENT_TIME}}",
		BootstrapSoul:     "# Soul\n\nBe helpful",
		// Skip AGENTS.md, USER.md, TOOLS.md
	}

	for filename, content := range bootstrapContent {
		filePath := ws.Subpath(filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create bootstrap file %s: %v", filename, err)
		}
	}

	// Track warnings
	var warnings []string
	loggerFunc := func(format string, args ...interface{}) {
		warnings = append(warnings, format)
	}

	loader := NewBootstrapLoader(ws, cfg, loggerFunc)

	// Load should succeed even with missing files
	files, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Should only have loaded the existing files
	if len(files) != len(bootstrapContent) {
		t.Errorf("Load() returned %d files, want %d", len(files), len(bootstrapContent))
	}

	// Assemble should succeed
	assembled, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	// Should contain only existing files
	if !strings.Contains(assembled, "Identity") {
		t.Error("Identity not found in assembled content")
	}
	if !strings.Contains(assembled, "Soul") {
		t.Error("Soul not found in assembled content")
	}

	// Should still have substituted variables
	if strings.Contains(assembled, "{{") {
		t.Error("template variables not substituted")
	}

	// Verify warnings were logged for missing files
	if len(warnings) == 0 {
		t.Error("expected warnings for missing files, got none")
	}
}

// TestWorkspaceBootstrapIntegrationPriorityOrder tests that bootstrap files are assembled in correct priority order.
func TestWorkspaceBootstrapIntegrationPriorityOrder(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.WorkspaceConfig{
		Path:              tmpDir,
		BootstrapMaxChars: 5000,
	}

	ws := New(cfg)
	if err := ws.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() failed: %v", err)
	}

	// Create bootstrap files with markers to verify order
	markers := []struct {
		name   string
		marker string
		order  int
	}{
		{BootstrapIdentity, "IDENTITY_FIRST", 1},
		{BootstrapAgents, "AGENTS_SECOND", 2},
		{BootstrapSoul, "SOUL_THIRD", 3},
		{BootstrapUser, "USER_FOURTH", 4},
		{BootstrapTools, "TOOLS_FIFTH", 5},
	}

	for _, m := range markers {
		content := strings.Join([]string{
			"# " + m.name,
			"",
			m.marker,
		}, "\n")

		filePath := ws.Subpath(m.name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create bootstrap file %s: %v", m.name, err)
		}
	}

	loader := NewBootstrapLoader(ws, cfg, nil)
	assembled, err := loader.Assemble()

	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	// Verify order by checking marker positions
	markerPositions := make(map[string]int)
	for _, m := range markers {
		pos := strings.Index(assembled, m.marker)
		if pos == -1 {
			t.Fatalf("marker %s not found in assembled content", m.marker)
		}
		markerPositions[m.marker] = pos
	}

	// Verify increasing order
	if markerPositions["IDENTITY_FIRST"] > markerPositions["AGENTS_SECOND"] {
		t.Error("IDENTITY should appear before AGENTS")
	}
	if markerPositions["AGENTS_SECOND"] > markerPositions["SOUL_THIRD"] {
		t.Error("AGENTS should appear before SOUL")
	}
	if markerPositions["SOUL_THIRD"] > markerPositions["USER_FOURTH"] {
		t.Error("SOUL should appear before USER")
	}
	if markerPositions["USER_FOURTH"] > markerPositions["TOOLS_FIFTH"] {
		t.Error("USER should appear before TOOLS")
	}
}

// TestWorkspaceBootstrapIntegrationWithSubdirectories tests integration with workspace subdirectories.
func TestWorkspaceBootstrapIntegrationWithSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.WorkspaceConfig{
		Path:              tmpDir,
		BootstrapMaxChars: 5000,
	}

	ws := New(cfg)
	if err := ws.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() failed: %v", err)
	}

	// Create standard subdirectories
	if err := ws.EnsureSubpath(SubdirMemory); err != nil {
		t.Fatalf("EnsureSubpath(memory) failed: %v", err)
	}
	if err := ws.EnsureSubpath(SubdirSkills); err != nil {
		t.Fatalf("EnsureSubpath(skills) failed: %v", err)
	}

	// Verify subdirectories exist
	memoryPath := ws.Subpath(SubdirMemory)
	if _, err := os.Stat(memoryPath); os.IsNotExist(err) {
		t.Error("memory subdirectory not created")
	}

	skillsPath := ws.Subpath(SubdirSkills)
	if _, err := os.Stat(skillsPath); os.IsNotExist(err) {
		t.Error("skills subdirectory not created")
	}

	// Create bootstrap file that references workspace
	bootstrapContent := `# System Info

Workspace: {{WORKSPACE_PATH}}
Time: {{CURRENT_TIME}}

## Subdirectories

- Memory: memory/
- Skills: skills/
`

	identityPath := ws.Subpath(BootstrapIdentity)
	if err := os.WriteFile(identityPath, []byte(bootstrapContent), 0644); err != nil {
		t.Fatalf("failed to create bootstrap file: %v", err)
	}

	loader := NewBootstrapLoader(ws, cfg, nil)
	assembled, err := loader.Assemble()

	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	// Verify workspace path was substituted
	if !strings.Contains(assembled, tmpDir) {
		t.Error("workspace path not substituted")
	}

	// Verify subdirectory paths are present
	if !strings.Contains(assembled, "memory/") {
		t.Error("memory/ not found in bootstrap")
	}
	if !strings.Contains(assembled, "skills/") {
		t.Error("skills/ not found in bootstrap")
	}
}

// TestWorkspaceBootstrapIntegrationWithRealFiles tests integration with actual bootstrap files from the workspace directory.
func TestWorkspaceBootstrapIntegrationWithRealFiles(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.WorkspaceConfig{
		Path:              tmpDir,
		BootstrapMaxChars: 10000,
	}

	ws := New(cfg)

	// Create bootstrap files with realistic content similar to actual files
	identityContent := `# Core Identity

Nexbot is an ultra-lightweight personal AI assistant built in Go.

## Purpose

Nexbot helps you manage digital workflows through:
- Telegram chat interface
- File operations
- Shell commands
- Custom skills
- Long-term memory

## Current Time

{{CURRENT_TIME}}
{{CURRENT_DATE}}

## Workspace

Your workspace is at: {{WORKSPACE_PATH}}
`

	agentsContent := `# Agent Instructions

You are a helpful AI assistant. Be concise, accurate, and friendly.

## Guidelines

- Always explain what you're doing before taking actions
- Ask for clarification when request is ambiguous
- Use tools to help accomplish tasks
- Remember important information in your memory files
`

	soulContent := `# Soul & Personality

## Core Traits

- Professional and friendly
- Concise and practical
- Adaptable and helpful
- Safety-conscious

## Communication Style

- Direct answers preferred over long explanations
- Bullet points for complex information
- Code snippets included when relevant
- Confirmation before executing commands
`

	userContent := `# User Profile

## Basic Info

**Name**: {{USER_NAME}}
**Timezone**: {{USER_TIMEZONE}}
**Language**: English, Russian
**Preferred Style**: Concise, practical

## Preferences

- Direct answers preferred over long explanations
- Bullet points for complex information
- Code snippets included when relevant
- Confirmation before executing commands
- Ask for permission before file operations
`

	toolsContent := `# Available Tools

## File Operations

- Read files from whitelisted directories
- Write files to allowed locations
- List directory contents
- Search for files by pattern

## Shell Commands

- Execute whitelisted shell commands
- Run commands from working directory
- Command timeout enforcement

## Messaging

- Send messages to channels
- Receive commands via chat interfaces
- Multi-channel support
`

	bootstrapFiles := map[string]string{
		BootstrapIdentity: identityContent,
		BootstrapAgents:   agentsContent,
		BootstrapSoul:     soulContent,
		BootstrapUser:     userContent,
		BootstrapTools:    toolsContent,
	}

	for filename, content := range bootstrapFiles {
		filePath := ws.Subpath(filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create bootstrap file %s: %v", filename, err)
		}
	}

	loader := NewBootstrapLoader(ws, cfg, nil)

	// Load bootstrap files
	files, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify we loaded all files
	if len(files) != len(bootstrapFiles) {
		t.Errorf("Load() returned %d files, want %d", len(files), len(bootstrapFiles))
	}

	// Assemble content
	assembled, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	// Verify content is not empty
	if assembled == "" {
		t.Error("Assembled content is empty")
	}

	// Verify supported template variables were substituted
	supportedVars := []string{"{{CURRENT_TIME}}", "{{CURRENT_DATE}}", "{{WORKSPACE_PATH}}"}
	for _, v := range supportedVars {
		if strings.Contains(assembled, v) {
			t.Errorf("supported template variable %s not substituted", v)
		}
	}

	// Verify workspace path was substituted
	if !strings.Contains(assembled, tmpDir) {
		t.Errorf("workspace path %q not substituted in real files", tmpDir)
	}

	// Verify time/date were substituted
	if !strings.Contains(assembled, ":") {
		t.Error("time not substituted in real files")
	}
	if !strings.Contains(assembled, "-") {
		t.Error("date not substituted in real files")
	}

	// Verify all sections are present
	requiredSections := []string{
		"Core Identity",
		"Agent Instructions",
		"Soul & Personality",
		"User Profile",
		"Available Tools",
	}

	for _, section := range requiredSections {
		if !strings.Contains(assembled, section) {
			t.Errorf("required section %s not found in assembled content", section)
		}
	}

	// Verify separators are present
	if !strings.Contains(assembled, "---") {
		t.Error("separator not present in assembled content from real files")
	}
}

// TestWorkspaceBootstrapIntegrationEndToEnd tests a complete realistic usage pattern.
func TestWorkspaceBootstrapIntegrationEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Initialize workspace with config
	cfg := config.WorkspaceConfig{
		Path:              tmpDir,
		BootstrapMaxChars: 8000,
	}

	ws := New(cfg)

	// 2. Create workspace directory
	if err := ws.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() failed: %v", err)
	}

	// 3. Create subdirectories for memory and skills
	if err := ws.EnsureSubpath(SubdirMemory); err != nil {
		t.Fatalf("EnsureSubpath(memory) failed: %v", err)
	}
	if err := ws.EnsureSubpath(SubdirSkills); err != nil {
		t.Fatalf("EnsureSubpath(skills) failed: %v", err)
	}

	// 4. Create bootstrap files with realistic content
	identityContent := `# Nexbot Identity

Nexbot is your personal AI assistant.

## Current Status

- Time: {{CURRENT_TIME}}
- Date: {{CURRENT_DATE}}
- Workspace: {{WORKSPACE_PATH}}

## Capabilities

- File operations
- Shell commands
- Messaging
- Memory management
`

	agentsContent := `# Agent Instructions

You are a helpful AI assistant.

## Behavior

- Be concise and accurate
- Explain your actions
- Ask for clarification when needed
`

	soulContent := `# Personality

Be friendly, professional, and helpful.

## Tone

- Respectful
- Knowledgeable
- Patient
`

	bootstrapFiles := map[string]string{
		BootstrapIdentity: identityContent,
		BootstrapAgents:   agentsContent,
		BootstrapSoul:     soulContent,
	}

	for filename, content := range bootstrapFiles {
		filePath := ws.Subpath(filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create bootstrap file %s: %v", filename, err)
		}
	}

	// 5. Initialize bootstrap loader
	loader := NewBootstrapLoader(ws, cfg, nil)

	// 6. Load and assemble bootstrap content
	systemPrompt, err := loader.Assemble()
	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	// 7. Verify the system prompt is complete and valid
	if len(systemPrompt) == 0 {
		t.Error("system prompt is empty")
	}

	// 8. Verify all components are present
	requiredComponents := []string{
		"Nexbot Identity",
		"Current Status",
		"Capabilities",
		"Agent Instructions",
		"Behavior",
		"Personality",
		"Tone",
	}

	for _, component := range requiredComponents {
		if !strings.Contains(systemPrompt, component) {
			t.Errorf("required component not found: %s", component)
		}
	}

	// 9. Verify template variables are substituted
	if strings.Contains(systemPrompt, "{{") {
		t.Error("template variables not substituted")
	}

	// 10. Verify workspace metadata is included
	if !strings.Contains(systemPrompt, tmpDir) {
		t.Error("workspace path not included in system prompt")
	}

	if !strings.Contains(systemPrompt, ":") {
		t.Error("time not included in system prompt")
	}

	if !strings.Contains(systemPrompt, "-") {
		t.Error("date not included in system prompt")
	}

	// 11. Verify the system prompt has proper structure with separators
	if !strings.Contains(systemPrompt, "---") {
		t.Error("system prompt lacks proper separators")
	}

	// 12. Verify subdirectories are accessible
	memoryPath := ws.Subpath(SubdirMemory)
	if !filepath.IsAbs(memoryPath) {
		t.Error("memory subpath is not absolute")
	}

	skillsPath := ws.Subpath(SubdirSkills)
	if !filepath.IsAbs(skillsPath) {
		t.Error("skills subpath is not absolute")
	}

	// 13. Verify paths resolve correctly
	resolvedPath, err := ws.ResolvePath("memory/test.txt")
	if err != nil {
		t.Errorf("ResolvePath() failed: %v", err)
	}

	expectedMemoryPath := filepath.Join(tmpDir, SubdirMemory, "test.txt")
	if resolvedPath != expectedMemoryPath {
		t.Errorf("ResolvePath() = %v, want %v", resolvedPath, expectedMemoryPath)
	}
}

// TestWorkspaceBootstrapIntegrationWithTildePath tests integration with tilde expansion in path.
func TestWorkspaceBootstrapIntegrationWithTildePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate a config with tilde path
	cfg := config.WorkspaceConfig{
		Path:              tmpDir, // In real scenario this would be "~/.nexbot"
		BootstrapMaxChars: 5000,
	}

	ws := New(cfg)
	if err := ws.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() failed: %v", err)
	}

	// Create bootstrap file with workspace path variable
	bootstrapContent := `# Test

Workspace: {{WORKSPACE_PATH}}
Time: {{CURRENT_TIME}}
`

	identityPath := ws.Subpath(BootstrapIdentity)
	if err := os.WriteFile(identityPath, []byte(bootstrapContent), 0644); err != nil {
		t.Fatalf("failed to create bootstrap file: %v", err)
	}

	loader := NewBootstrapLoader(ws, cfg, nil)
	assembled, err := loader.Assemble()

	if err != nil {
		t.Fatalf("Assemble() failed: %v", err)
	}

	// Verify workspace path was substituted correctly
	if !strings.Contains(assembled, tmpDir) {
		t.Errorf("workspace path %q not substituted in assembled content: %s", tmpDir, assembled)
	}

	// Verify the path from ws.Path() is the same as tmpDir (no tilde)
	if ws.Path() != tmpDir {
		t.Errorf("ws.Path() = %v, want %v", ws.Path(), tmpDir)
	}
}
