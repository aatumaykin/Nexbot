package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/logger"
)

func TestZAIProvider_Chat_NetworkError(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	p := NewZAIProvider(ZAIConfig{APIKey: "test-key"}, log)
	p.apiURL = "http://invalid-host-that-does-not-exist.local:9999"

	ctx := context.Background()
	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	}

	_, err = p.Chat(ctx, req)
	if err == nil {
		t.Fatal("Chat() expected network error, got nil")
	}
}

func TestZAIProvider_Chat_HTTPError(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(`{"error": "invalid api key"}`)); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}))
	defer server.Close()

	p := NewZAIProvider(ZAIConfig{APIKey: "invalid-key"}, log)
	p.apiURL = server.URL

	ctx := context.Background()
	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	}

	_, err = p.Chat(ctx, req)
	if err == nil {
		t.Fatal("Chat() expected HTTP error, got nil")
	}

	if httpErr, ok := errors.AsType[*zaiHTTPError](err); ok {
		if httpErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("HTTP error status = %d, want 401", httpErr.StatusCode)
		}
	}
}

func TestZAIProvider_Chat_APIError(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := zaiResponse{
			Error: &zaiAPIError{
				Message: "Rate limit exceeded",
				Type:    "rate_limit_error",
				Code:    "rate_limit",
			},
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Encode failed: %v", err)
		}
	}))
	defer server.Close()

	p := NewZAIProvider(ZAIConfig{APIKey: "test-key"}, log)
	p.apiURL = server.URL

	ctx := context.Background()
	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	}

	_, err = p.Chat(ctx, req)
	if err == nil {
		t.Fatal("Chat() expected API error, got nil")
	}

	if !strings.Contains(err.Error(), "Rate limit exceeded") {
		t.Errorf("Error = %v, should contain 'Rate limit exceeded'", err)
	}
}

func TestZAIProvider_Chat_InvalidJSON(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`invalid json`)); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}))
	defer server.Close()

	p := NewZAIProvider(ZAIConfig{APIKey: "test-key"}, log)
	p.apiURL = server.URL

	ctx := context.Background()
	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	}

	_, err = p.Chat(ctx, req)
	if err == nil {
		t.Fatal("Chat() expected JSON error, got nil")
	}
}
