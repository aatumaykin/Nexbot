package telegram

import (
	"context"
	"testing"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/mymmrac/telego"
)

func TestMessageSender_PrepareMessage_ParseMode(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 100, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{}
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	tests := []struct {
		name     string
		content  string
		wantMode string
		wantText string
	}{
		{
			name:     "plain text",
			content:  "Just plain text",
			wantMode: "",
			wantText: "Just plain text",
		},
		{
			name:     "markdown bold",
			content:  "**Bold text**",
			wantMode: telego.ModeMarkdown,
			wantText: "**Bold text**",
		},
		{
			name:     "markdown italic",
			content:  "*italic text*",
			wantMode: telego.ModeMarkdown,
			wantText: "*italic text*",
		},
		{
			name:     "markdown link",
			content:  "[link](http://example.com)",
			wantMode: telego.ModeMarkdown,
			wantText: "[link](http://example.com)",
		},
		{
			name:     "code block",
			content:  "```go\ncode here\n```",
			wantMode: telego.ModeHTML,
			wantText: "<pre><code>code here</code></pre>",
		},
		{
			name:     "inline code",
			content:  "text with `inline code`",
			wantMode: telego.ModeHTML,
			wantText: "text with <code>inline code</code>",
		},
		{
			name:     "empty string",
			content:  "",
			wantMode: "",
			wantText: "",
		},
		{
			name:     "mixed markdown and code",
			content:  "**bold** and `code`",
			wantMode: telego.ModeHTML,
			wantText: "<b>bold</b> and <code>code</code>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := conn.prepareMessage(tt.content, 123)
			if err != nil {
				t.Fatalf("prepareMessage() failed: %v", err)
			}

			if params.ParseMode != tt.wantMode {
				t.Errorf("prepareMessage() ParseMode = %v, want %v", params.ParseMode, tt.wantMode)
			}

			if params.Text != tt.wantText {
				t.Errorf("prepareMessage() Text = %v, want %v", params.Text, tt.wantText)
			}
		})
	}
}

func TestMessageSender_PrepareEditMessageParams_ParseMode(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 100, log)

	cfg := config.TelegramConfig{}
	conn := New(cfg, log, msgBus)
	conn.ctx = context.Background()

	tests := []struct {
		name      string
		content   string
		messageID string
		wantMode  string
		wantText  string
	}{
		{
			name:      "plain text",
			content:   "Just plain text",
			messageID: "1",
			wantMode:  "",
			wantText:  "Just plain text",
		},
		{
			name:      "markdown bold",
			content:   "**Bold text**",
			messageID: "1",
			wantMode:  telego.ModeMarkdown,
			wantText:  "**Bold text**",
		},
		{
			name:      "markdown italic",
			content:   "*italic text*",
			messageID: "1",
			wantMode:  telego.ModeMarkdown,
			wantText:  "*italic text*",
		},
		{
			name:      "code block",
			content:   "```go\ncode here\n```",
			messageID: "1",
			wantMode:  telego.ModeHTML,
			wantText:  "<pre><code>code here</code></pre>",
		},
		{
			name:      "inline code",
			content:   "text with `inline code`",
			messageID: "1",
			wantMode:  telego.ModeHTML,
			wantText:  "text with <code>inline code</code>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := conn.prepareEditMessageParams(tt.content, 123, tt.messageID)

			if params.ParseMode != tt.wantMode {
				t.Errorf("prepareEditMessageParams() ParseMode = %v, want %v", params.ParseMode, tt.wantMode)
			}

			if params.Text != tt.wantText {
				t.Errorf("prepareEditMessageParams() Text = %v, want %v", params.Text, tt.wantText)
			}
		})
	}
}
