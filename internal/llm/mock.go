package llm

import (
	"context"
	"fmt"
)

// MockProvider is a mock implementation of the Provider interface for testing
// and graceful degradation scenarios.
type MockProvider struct {
	responses     []string // Pre-defined responses (rotates through them)
	responseIndex int      // Current index in responses
	mode          MockMode // Mode of operation (echo, fixed, fixtures)
	delay         int      // Simulated delay in milliseconds (for testing latency)
	errorAfter    int      // Number of successful calls before returning errors
	callCount     int      // Number of Chat() calls made
}

// MockMode defines the operation mode of the mock provider.
type MockMode int

const (
	// MockModeEcho returns the user's message (echo mode)
	MockModeEcho MockMode = iota

	// MockModeFixed returns a fixed response
	MockModeFixed

	// MockModeFixtures returns pre-defined responses in rotation
	MockModeFixtures

	// MockModeError always returns an error
	MockModeError
)

// MockConfig holds configuration for the mock provider.
type MockConfig struct {
	Mode       MockMode // Operation mode
	Responses  []string // Pre-defined responses (for Fixed/Fixtures modes)
	Delay      int      // Simulated delay in milliseconds
	ErrorAfter int      // Number of successful calls before returning errors
}

// NewMockProvider creates a new mock LLM provider.
func NewMockProvider(cfg MockConfig) *MockProvider {
	return &MockProvider{
		mode:          cfg.Mode,
		responses:     cfg.Responses,
		responseIndex: 0,
		delay:         cfg.Delay,
		errorAfter:    cfg.ErrorAfter,
		callCount:     0,
	}
}

// NewEchoProvider creates a mock provider that echoes user messages.
func NewEchoProvider() *MockProvider {
	return NewMockProvider(MockConfig{
		Mode: MockModeEcho,
	})
}

// NewFixedProvider creates a mock provider that always returns a fixed response.
func NewFixedProvider(response string) *MockProvider {
	return NewMockProvider(MockConfig{
		Mode:      MockModeFixed,
		Responses: []string{response},
	})
}

// NewFixturesProvider creates a mock provider that cycles through pre-defined responses.
func NewFixturesProvider(responses []string) *MockProvider {
	return NewMockProvider(MockConfig{
		Mode:      MockModeFixtures,
		Responses: responses,
	})
}

// NewErrorProvider creates a mock provider that always returns errors.
func NewErrorProvider() *MockProvider {
	return NewMockProvider(MockConfig{
		Mode: MockModeError,
	})
}

// Chat implements the Provider interface.
func (m *MockProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	m.callCount++

	// Check if we should return an error
	if m.errorAfter > 0 && m.callCount > m.errorAfter {
		return nil, fmt.Errorf("mock provider error after %d calls", m.errorAfter)
	}

	// Handle error mode
	if m.mode == MockModeError {
		return nil, fmt.Errorf("mock provider error")
	}

	// Get the user message (last message if available)
	var userMessage string
	if len(req.Messages) > 0 {
		lastMsg := req.Messages[len(req.Messages)-1]
		if lastMsg.Role == RoleUser {
			userMessage = lastMsg.Content
		}
	}

	// Determine response based on mode
	var response string
	switch m.mode {
	case MockModeEcho:
		if userMessage != "" {
			response = fmt.Sprintf("Echo: %s", userMessage)
		} else {
			response = "Echo: (no user message)"
		}
	case MockModeFixed:
		if len(m.responses) > 0 {
			response = m.responses[0]
		} else {
			response = "Fixed response: no responses configured"
		}
	case MockModeFixtures:
		if len(m.responses) > 0 {
			response = m.responses[m.responseIndex]
			m.responseIndex = (m.responseIndex + 1) % len(m.responses)
		} else {
			response = "Fixtures: no responses configured"
		}
	default:
		response = "Unknown mock mode"
	}

	// Build response
	return &ChatResponse{
		Content:      response,
		Model:        req.Model,
		FinishReason: "stop",
		Usage: Usage{
			PromptTokens:     len(userMessage),
			CompletionTokens: len(response),
			TotalTokens:      len(userMessage) + len(response),
		},
	}, nil
}

// SupportsToolCalling implements the Provider interface.
// Mock provider does not support tool calling.
func (m *MockProvider) SupportsToolCalling() bool {
	return false
}

// GetDefaultModel implements the Provider interface.
func (m *MockProvider) GetDefaultModel() string {
	return "mock-model"
}

// GetCallCount returns the number of Chat() calls made to this provider.
// Useful for testing.
func (m *MockProvider) GetCallCount() int {
	return m.callCount
}

// ResetCallCount resets the call counter.
// Useful for testing.
func (m *MockProvider) ResetCallCount() {
	m.callCount = 0
}

// SetErrorAfter configures the provider to return errors after N calls.
func (m *MockProvider) SetErrorAfter(n int) {
	m.errorAfter = n
}

// GetResponses returns the current list of responses.
func (m *MockProvider) GetResponses() []string {
	return m.responses
}

// SetResponses sets the list of responses.
func (m *MockProvider) SetResponses(responses []string) {
	m.responses = responses
	m.responseIndex = 0
}
