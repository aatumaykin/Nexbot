package prompts

import (
	"os"
	"path/filepath"
)

const DefaultPromptsDir = "/usr/local/share/subagent/prompts"

type PromptLoader struct {
	promptsDir string
}

func NewPromptLoader(promptsDir string) *PromptLoader {
	if promptsDir == "" {
		promptsDir = DefaultPromptsDir
	}
	return &PromptLoader{promptsDir: promptsDir}
}

func (l *PromptLoader) LoadIdentity() (string, error) {
	if content := os.Getenv("SUBAGENT_IDENTITY"); content != "" {
		return content, nil
	}

	path := filepath.Join(l.promptsDir, "identity.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return DefaultIdentity, nil
	}
	return string(content), nil
}

func (l *PromptLoader) LoadSecurity() (string, error) {
	if content := os.Getenv("SUBAGENT_SECURITY"); content != "" {
		return content, nil
	}

	path := filepath.Join(l.promptsDir, "security.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return DefaultSecurity, nil
	}
	return string(content), nil
}
