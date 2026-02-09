package memory

import (
	"testing"

	"github.com/aatumaykin/nexbot/internal/llm"
)

func TestMarkdownParser_Parse(t *testing.T) {
	parser := NewMarkdownParser()

	tests := []struct {
		name    string
		content string
		want    []llm.Message
		wantErr bool
	}{
		{
			name:    "empty content",
			content: "",
			want:    []llm.Message{},
		},
		{
			name: "single user message",
			content: `
### User [2026-01-15 10:00:00]

Hello world
`,
			want: []llm.Message{
				{
					Role:    llm.RoleUser,
					Content: "Hello world",
				},
			},
		},
		{
			name: "single assistant message",
			content: `
### Assistant [2026-01-15 10:00:00]

Hi there!
`,
			want: []llm.Message{
				{
					Role:    llm.RoleAssistant,
					Content: "Hi there!",
				},
			},
		},
		{
			name: "single system message",
			content: `
## System [2026-01-15 10:00:00]

You are a helpful assistant
`,
			want: []llm.Message{
				{
					Role:    llm.RoleSystem,
					Content: "You are a helpful assistant",
				},
			},
		},
		{
			name: "multiple messages",
			content: `
### User [2026-01-15 10:00:00]

Hello

### Assistant [2026-01-15 10:01:00]

Hi! How can I help?

### User [2026-01-15 10:02:00]

What is 2+2?
`,
			want: []llm.Message{
				{
					Role:    llm.RoleUser,
					Content: "Hello",
				},
				{
					Role:    llm.RoleAssistant,
					Content: "Hi! How can I help?",
				},
				{
					Role:    llm.RoleUser,
					Content: "What is 2+2?",
				},
			},
		},
		{
			name: "tool message",
			content: `
#### Tool: abc123 [2026-01-15 10:00:00]

tool result here
`,
			want: []llm.Message{
				{
					Role:       llm.RoleTool,
					ToolCallID: "abc123",
					Content:    "",
				},
			},
		},
		{
			name: "multiline content",
			content: `
### User [2026-01-15 10:00:00]

Line 1
Line 2
Line 3
`,
			want: []llm.Message{
				{
					Role:    llm.RoleUser,
					Content: "Line 1\nLine 2\nLine 3",
				},
			},
		},
		{
			name: "mixed roles",
			content: `
## System [2026-01-15 09:00:00]

System prompt

### User [2026-01-15 10:00:00]

User message

### Assistant [2026-01-15 10:01:00]

Assistant response

#### Tool: tool1 [2026-01-15 10:02:00]

Tool result

### Assistant [2026-01-15 10:03:00]

Final response
`,
			want: []llm.Message{
				{
					Role:    llm.RoleSystem,
					Content: "System prompt",
				},
				{
					Role:    llm.RoleUser,
					Content: "User message",
				},
				{
					Role:    llm.RoleAssistant,
					Content: "Assistant response",
				},
				{
					Role:       llm.RoleTool,
					ToolCallID: "tool1",
					Content:    "",
				},
				{
					Role:    llm.RoleAssistant,
					Content: "Final response",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.Parse(tt.content)

			if len(got) != len(tt.want) {
				t.Errorf("Parse() returned %d messages, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i].Role != tt.want[i].Role {
					t.Errorf("Parse() message %d: role = %v, want %v", i, got[i].Role, tt.want[i].Role)
				}
				if got[i].Content != tt.want[i].Content {
					t.Errorf("Parse() message %d: content = %q, want %q", i, got[i].Content, tt.want[i].Content)
				}
				if got[i].ToolCallID != tt.want[i].ToolCallID {
					t.Errorf("Parse() message %d: toolCallID = %q, want %q", i, got[i].ToolCallID, tt.want[i].ToolCallID)
				}
			}
		})
	}
}

func TestNewMarkdownParser(t *testing.T) {
	parser := NewMarkdownParser()
	if parser == nil {
		t.Error("NewMarkdownParser() returned nil")
	}
}
