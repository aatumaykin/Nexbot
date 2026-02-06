package llm

import (
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

func TestNewZAIProvider(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := ZAIConfig{
		APIKey:         "test-key",
		Model:          "glm-4.7",
		TimeoutSeconds: 30,
	}

	p := NewZAIProvider(cfg, log)

	if p == nil {
		t.Fatal("NewZAIProvider() returned nil")
	}

	if p.config.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want %q", p.config.APIKey, "test-key")
	}

	if p.config.Model != "glm-4.7" {
		t.Errorf("Model = %q, want %q", p.config.Model, "glm-4.7")
	}

	if p.client.Timeout != 30*time.Second {
		t.Errorf("client.Timeout = %v, want 30s", p.client.Timeout)
	}

	if p.apiURL != ZAIEndpoint {
		t.Errorf("apiURL = %q, want %q", p.apiURL, ZAIEndpoint)
	}
}

func TestNewZAIProvider_Defaults(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := ZAIConfig{
		APIKey: "test-key",
		// Model not set
		// TimeoutSeconds not set
	}

	p := NewZAIProvider(cfg, log)

	if p.config.Model != "glm-4.7" {
		t.Errorf("Default Model = %q, want %q", p.config.Model, "glm-4.7")
	}

	if p.client.Timeout != ZAIRequestTimeout {
		t.Errorf("Default Timeout = %v, want %v", p.client.Timeout, ZAIRequestTimeout)
	}
}

func TestTruncateResponse(t *testing.T) {
	tests := []struct {
		name   string
		body   []byte
		maxLen int
		want   string
	}{
		{
			name:   "short body",
			body:   []byte("short"),
			maxLen: 100,
			want:   "short",
		},
		{
			name:   "exact length",
			body:   []byte("exact"),
			maxLen: 5,
			want:   "exact",
		},
		{
			name:   "long body",
			body:   []byte("this is a long string"),
			maxLen: 10,
			want:   "this is a ...",
		},
		{
			name:   "empty body",
			body:   []byte(""),
			maxLen: 100,
			want:   "",
		},
		{
			name:   "zero max len",
			body:   []byte("test"),
			maxLen: 0,
			want:   "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateResponse(tt.body, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateResponse() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestZAIHTTPError_Error(t *testing.T) {
	err := &zaiHTTPError{
		StatusCode: 404,
		Body:       `{"error": "not found"}`,
	}

	got := err.Error()
	want := "HTTP error: status=404, body={\"error\": \"not found\"}"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestZAIProvider_SupportsToolCalling(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	p := NewZAIProvider(ZAIConfig{APIKey: "test"}, log)

	if !p.SupportsToolCalling() {
		t.Error("ZAIProvider should support tool calling")
	}
}
