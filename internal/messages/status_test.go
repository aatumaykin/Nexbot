package messages

import (
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/constants"
)

func TestFormatStatusMessage(t *testing.T) {
	tests := []struct {
		name           string
		sessionID      string
		msgCount       int
		fileSizeHuman  string
		model          string
		temperature    float64
		maxTokens      int
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:          "standard status",
			sessionID:     "abc123",
			msgCount:      10,
			fileSizeHuman: "1.2 MB",
			model:         "gpt-4",
			temperature:   0.7,
			maxTokens:     2048,
			wantContains: []string{
				"📊 **Session Status**",
				"**Session ID:** `abc123`",
				"**Messages:** 10",
				"**Session Size:** 1.2 MB",
				"**LLM Configuration:**",
				"**Model:** gpt-4",
				"**Temperature:** 0.70",
				"**Max Tokens:** 2048",
			},
		},
		{
			name:          "empty session",
			sessionID:     "",
			msgCount:      0,
			fileSizeHuman: "0 B",
			model:         "",
			temperature:   0.0,
			maxTokens:     0,
			wantContains: []string{
				"📊 **Session Status**",
				"**Session ID:** ``",
				"**Messages:** 0",
				"**Session Size:** 0 B",
				"**LLM Configuration:**",
				"**Model:** ",
				"**Temperature:** 0.00",
				"**Max Tokens:** 0",
			},
		},
		{
			name:          "large session",
			sessionID:     "def456",
			msgCount:      1000,
			fileSizeHuman: "25.5 MB",
			model:         "gpt-4-turbo",
			temperature:   0.5,
			maxTokens:     4096,
			wantContains: []string{
				"📊 **Session Status**",
				"**Session ID:** `def456`",
				"**Messages:** 1000",
				"**Session Size:** 25.5 MB",
				"**LLM Configuration:**",
				"**Model:** gpt-4-turbo",
				"**Temperature:** 0.50",
				"**Max Tokens:** 4096",
			},
		},
		{
			name:          "high temperature",
			sessionID:     "ghi789",
			msgCount:      5,
			fileSizeHuman: "256 KB",
			model:         "claude-3",
			temperature:   0.95,
			maxTokens:     8192,
			wantContains: []string{
				"📊 **Session Status**",
				"**Temperature:** 0.95",
			},
		},
		{
			name:          "low temperature",
			sessionID:     "jkl012",
			msgCount:      3,
			fileSizeHuman: "128 KB",
			model:         "llama-2",
			temperature:   0.01,
			maxTokens:     1024,
			wantContains: []string{
				"📊 **Session Status**",
				"**Temperature:** 0.01",
			},
		},
		{
			name:          "model with special characters",
			sessionID:     "mno345",
			msgCount:      7,
			fileSizeHuman: "512 KB",
			model:         "gpt-4-32k",
			temperature:   0.7,
			maxTokens:     32768,
			wantContains: []string{
				"📊 **Session Status**",
				"**Model:** gpt-4-32k",
			},
		},
		{
			name:          "file size with decimals",
			sessionID:     "pqr678",
			msgCount:      15,
			fileSizeHuman: "1.23 MB",
			model:         "gpt-3.5-turbo",
			temperature:   0.8,
			maxTokens:     4096,
			wantContains: []string{
				"📊 **Session Status**",
				"**Session Size:** 1.23 MB",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatStatusMessage(
				tt.sessionID,
				tt.msgCount,
				tt.fileSizeHuman,
				tt.model,
				tt.temperature,
				tt.maxTokens,
			)

			// Check that all expected strings are present
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("FormatStatusMessage() output should contain %q, got:\n%s", want, got)
				}
			}

			// Check that strings that should not be present are absent
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("FormatStatusMessage() output should NOT contain %q, got:\n%s", notWant, got)
				}
			}
		})
	}
}

func TestFormatStatusMessage_ConstantCheck(t *testing.T) {
	// Test that the function uses the correct constants
	got := FormatStatusMessage(
		"test-session",
		42,
		"5.5 MB",
		"test-model",
		0.75,
		3072,
	)

	// Check that it contains all status constants
	expectedConstants := []string{
		constants.MsgStatusHeader,
		constants.MsgStatusLLMConfig,
	}

	for _, constant := range expectedConstants {
		if !strings.Contains(got, constant) {
			t.Errorf("FormatStatusMessage() should contain constant %q, got:\n%s", constant, got)
		}
	}
}

func TestFormatStatusMessage_Format(t *testing.T) {
	// Test the exact format with specific inputs
	got := FormatStatusMessage(
		"session-123",
		25,
		"3.5 MB",
		"gpt-4",
		0.7,
		2048,
	)

	expected := "📊 **Session Status**\n\n" +
		"**Session ID:** `session-123`\n" +
		"**Messages:** 25\n" +
		"**Session Size:** 3.5 MB\n" +
		"\n**LLM Configuration:**\n" +
		"**Model:** gpt-4\n" +
		"**Temperature:** 0.70\n" +
		"**Max Tokens:** 2048\n"

	if got != expected {
		t.Errorf("FormatStatusMessage() =\n%s\n\nwant\n%s", got, expected)
	}
}

func TestFormatStatusMessage_TemperaturePrecision(t *testing.T) {
	tests := []struct {
		name        string
		temperature float64
		wantFormat  string
	}{
		{
			name:        "two decimal places needed",
			temperature: 0.125,
			wantFormat:  "0.12",
		},
		{
			name:        "one decimal place needed",
			temperature: 0.5,
			wantFormat:  "0.50",
		},
		{
			name:        "zero decimal places",
			temperature: 1.0,
			wantFormat:  "1.00",
		},
		{
			name:        "very small value",
			temperature: 0.001,
			wantFormat:  "0.00",
		},
		{
			name:        "rounding down",
			temperature: 0.749,
			wantFormat:  "0.75",
		},
		{
			name:        "rounding up",
			temperature: 0.751,
			wantFormat:  "0.75",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatStatusMessage(
				"test-session",
				1,
				"1 KB",
				"test-model",
				tt.temperature,
				100,
			)

			expectedPart := "**Temperature:** " + tt.wantFormat
			if !strings.Contains(got, expectedPart) {
				t.Errorf("FormatStatusMessage() should contain %q, got:\n%s", expectedPart, got)
			}
		})
	}
}

func TestFormatStatusMessage_SessionIDFormat(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		wantPart  string
	}{
		{
			name:      "alphanumeric session ID",
			sessionID: "abc123",
			wantPart:  "**Session ID:** `abc123`",
		},
		{
			name:      "session ID with hyphens",
			sessionID: "session-abc-123",
			wantPart:  "**Session ID:** `session-abc-123`",
		},
		{
			name:      "session ID with underscores",
			sessionID: "session_abc_123",
			wantPart:  "**Session ID:** `session_abc_123`",
		},
		{
			name:      "empty session ID",
			sessionID: "",
			wantPart:  "**Session ID:** ``",
		},
		{
			name:      "UUID-like session ID",
			sessionID: "550e8400-e29b-41d4-a716-446655440000",
			wantPart:  "**Session ID:** `550e8400-e29b-41d4-a716-446655440000`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatStatusMessage(
				tt.sessionID,
				1,
				"1 KB",
				"test-model",
				0.7,
				100,
			)

			if !strings.Contains(got, tt.wantPart) {
				t.Errorf("FormatStatusMessage() should contain %q, got:\n%s", tt.wantPart, got)
			}
		})
	}
}

func TestFormatStatusMessage_CompleteOutput(t *testing.T) {
	// Test complete output structure
	sessionID := "test-session-id-12345"
	msgCount := 42
	fileSizeHuman := "2.5 MB"
	model := "gpt-4-turbo-preview"
	temperature := 0.75
	maxTokens := 4096

	got := FormatStatusMessage(sessionID, msgCount, fileSizeHuman, model, temperature, maxTokens)

	// Verify structure: header -> session info -> llm config header -> llm config
	lines := strings.Split(got, "\n")

	// Should have header line
	if !strings.Contains(lines[0], "📊 **Session Status**") {
		t.Error("First line should be status header")
	}

	// Should have empty line after header
	if lines[1] != "" {
		t.Error("Second line should be empty")
	}

	// Should have session info section before LLM config
	llmConfigLine := -1
	sessionIDLine := -1
	for i, line := range lines {
		if strings.Contains(line, "**LLM Configuration:**") {
			llmConfigLine = i
		}
		if strings.Contains(line, "**Session ID:**") {
			sessionIDLine = i
		}
	}

	if sessionIDLine == -1 {
		t.Error("Should have session ID line")
	}
	if llmConfigLine == -1 {
		t.Error("Should have LLM config line")
	}
	if sessionIDLine >= llmConfigLine {
		t.Error("Session info should come before LLM config")
	}
}
