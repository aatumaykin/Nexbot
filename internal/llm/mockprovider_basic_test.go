package llm

import (
	"context"
	"testing"
)

func TestNewMockProvider(t *testing.T) {
	tests := []struct {
		name  string
		cfg   MockConfig
		want  MockMode
		repos []string
	}{
		{
			name: "echo mode",
			cfg:  MockConfig{Mode: MockModeEcho},
			want: MockModeEcho,
		},
		{
			name: "fixed mode",
			cfg: MockConfig{
				Mode:      MockModeFixed,
				Responses: []string{"test response"},
			},
			want:  MockModeFixed,
			repos: []string{"test response"},
		},
		{
			name: "fixtures mode",
			cfg: MockConfig{
				Mode:      MockModeFixtures,
				Responses: []string{"resp1", "resp2"},
			},
			want:  MockModeFixtures,
			repos: []string{"resp1", "resp2"},
		},
		{
			name: "error mode",
			cfg:  MockConfig{Mode: MockModeError},
			want: MockModeError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewMockProvider(tt.cfg)
			if p.mode != tt.want {
				t.Errorf("NewMockProvider() mode = %v, want %v", p.mode, tt.want)
			}
			if tt.repos != nil {
				if len(p.responses) != len(tt.repos) {
					t.Errorf("NewMockProvider() responses len = %d, want %d", len(p.responses), len(tt.repos))
				}
			}
		})
	}
}

func TestNewEchoProvider(t *testing.T) {
	p := NewEchoProvider()
	if p.mode != MockModeEcho {
		t.Errorf("NewEchoProvider() mode = %v, want %v", p.mode, MockModeEcho)
	}
}

func TestNewFixedProvider(t *testing.T) {
	response := "fixed response"
	p := NewFixedProvider(response)

	if p.mode != MockModeFixed {
		t.Errorf("NewFixedProvider() mode = %v, want %v", p.mode, MockModeFixed)
	}
	if len(p.responses) != 1 || p.responses[0] != response {
		t.Errorf("NewFixedProvider() responses = %v, want [%s]", p.responses, response)
	}
}

func TestNewFixturesProvider(t *testing.T) {
	responses := []string{"resp1", "resp2", "resp3"}
	p := NewFixturesProvider(responses)

	if p.mode != MockModeFixtures {
		t.Errorf("NewFixturesProvider() mode = %v, want %v", p.mode, MockModeFixtures)
	}
	if len(p.responses) != len(responses) {
		t.Errorf("NewFixturesProvider() responses len = %d, want %d", len(p.responses), len(responses))
	}
}

func TestNewErrorProvider(t *testing.T) {
	p := NewErrorProvider()
	if p.mode != MockModeError {
		t.Errorf("NewErrorProvider() mode = %v, want %v", p.mode, MockModeError)
	}
}

func TestMockProvider_SupportsToolCalling(t *testing.T) {
	p := NewEchoProvider()
	if p.SupportsToolCalling() != false {
		t.Errorf("SupportsToolCalling() = %v, want false", p.SupportsToolCalling())
	}
}

func TestMockProvider_GetCallCount(t *testing.T) {
	p := NewEchoProvider()
	ctx := context.Background()

	if p.GetCallCount() != 0 {
		t.Errorf("GetCallCount() = %d, want 0", p.GetCallCount())
	}

	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Test"}},
	}
	_, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if p.GetCallCount() != 1 {
		t.Errorf("GetCallCount() after 1 call = %d, want 1", p.GetCallCount())
	}

	_, err = p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if p.GetCallCount() != 2 {
		t.Errorf("GetCallCount() after 2 calls = %d, want 2", p.GetCallCount())
	}
}

func TestMockProvider_ResetCallCount(t *testing.T) {
	p := NewEchoProvider()
	ctx := context.Background()

	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Test"}},
	}
	_, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	_, err = p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	p.ResetCallCount()

	if p.GetCallCount() != 0 {
		t.Errorf("After ResetCallCount(), GetCallCount() = %d, want 0", p.GetCallCount())
	}
}

func TestMockProvider_SetErrorAfter(t *testing.T) {
	p := NewMockProvider(MockConfig{
		Mode:       MockModeEcho,
		ErrorAfter: 0,
	})
	ctx := context.Background()

	p.SetErrorAfter(1)

	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Test"}},
	}
	_, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("First call with ErrorAfter=1 should succeed, got error: %v", err)
	}

	_, err = p.Chat(ctx, req)
	if err == nil {
		t.Fatal("Second call with ErrorAfter=1 should fail, got nil")
	}
}

func TestMockProvider_GetResponses(t *testing.T) {
	responses := []string{"resp1", "resp2"}
	p := NewFixturesProvider(responses)

	got := p.GetResponses()
	if len(got) != len(responses) {
		t.Errorf("GetResponses() len = %d, want %d", len(got), len(responses))
	}

	for i, r := range got {
		if r != responses[i] {
			t.Errorf("GetResponses()[%d] = %q, want %q", i, r, responses[i])
		}
	}
}

func TestMockProvider_SetResponses(t *testing.T) {
	p := NewEchoProvider()

	responses := []string{"new1", "new2"}
	p.SetResponses(responses)

	got := p.GetResponses()
	if len(got) != len(responses) {
		t.Errorf("After SetResponses(), len = %d, want %d", len(got), len(responses))
	}

	// Response index should be reset
	if p.responseIndex != 0 {
		t.Errorf("After SetResponses(), responseIndex = %d, want 0", p.responseIndex)
	}

	// Verify responses are actually set
	for i, r := range got {
		if r != responses[i] {
			t.Errorf("After SetResponses(), response[%d] = %q, want %q", i, r, responses[i])
		}
	}
}
