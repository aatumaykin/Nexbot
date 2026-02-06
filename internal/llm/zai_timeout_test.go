package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

func TestZAIProvider_Chat_Timeout(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := NewZAIProvider(ZAIConfig{
		APIKey:         "test-key",
		TimeoutSeconds: 0, // Very short timeout
	}, log)
	p.apiURL = server.URL

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	}

	_, err = p.Chat(ctx, req)
	if err == nil {
		t.Fatal("Chat() expected timeout error, got nil")
	}
}
