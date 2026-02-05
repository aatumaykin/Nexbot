package constants

import (
	"fmt"
	"testing"
	"time"
)

func TestTestConstants(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{
			name:  "TestRequestTimeout",
			value: TestRequestTimeout,
		},
		{
			name:  "TestTemperature",
			value: TestTemperature,
		},
		{
			name:  "TestMaxTokens",
			value: TestMaxTokens,
		},
		{
			name:  "TestDefaultModel",
			value: TestDefaultModel,
		},
		{
			name:  "TestMessage",
			value: TestMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch v := tt.value.(type) {
			case string:
				if v == "" {
					t.Errorf("%s should not be empty", tt.name)
				}
			case time.Duration:
				if v <= 0 {
					t.Errorf("%s should be positive, got: %v", tt.name, v)
				}
			case float64:
				if v <= 0 {
					t.Errorf("%s should be positive, got: %v", tt.name, v)
				}
			case int:
				if v <= 0 {
					t.Errorf("%s should be positive, got: %v", tt.name, v)
				}
			}
		})
	}
}

func TestTestRequestTimeout(t *testing.T) {
	expected := 30 * time.Second
	if TestRequestTimeout != expected {
		t.Errorf("TestRequestTimeout = %v, want %v", TestRequestTimeout, expected)
	}

	// Test that timeout is reasonable (between 1 second and 5 minutes)
	minTimeout := 1 * time.Second
	maxTimeout := 5 * time.Minute

	if TestRequestTimeout < minTimeout {
		t.Errorf("TestRequestTimeout should be at least %v, got: %v", minTimeout, TestRequestTimeout)
	}

	if TestRequestTimeout > maxTimeout {
		t.Errorf("TestRequestTimeout should not exceed %v, got: %v", maxTimeout, TestRequestTimeout)
	}
}

func TestTestTemperature(t *testing.T) {
	if TestTemperature != 0.7 {
		t.Errorf("TestTemperature = %v, want 0.7", TestTemperature)
	}

	// Test that temperature is in valid range [0.0, 2.0]
	if TestTemperature < 0.0 || TestTemperature > 2.0 {
		t.Errorf("TestTemperature should be between 0.0 and 2.0, got: %v", TestTemperature)
	}
}

func TestTestMaxTokens(t *testing.T) {
	if TestMaxTokens != 200 {
		t.Errorf("TestMaxTokens = %v, want 200", TestMaxTokens)
	}

	// Test that max tokens is reasonable (between 1 and 100000)
	if TestMaxTokens < 1 || TestMaxTokens > 100000 {
		t.Errorf("TestMaxTokens should be between 1 and 100000, got: %v", TestMaxTokens)
	}
}

func TestTestDefaultModel(t *testing.T) {
	if TestDefaultModel != "glm-4.7" {
		t.Errorf("TestDefaultModel = %s, want 'glm-4.7'", TestDefaultModel)
	}

	// Test that model name is not empty
	if TestDefaultModel == "" {
		t.Error("TestDefaultModel should not be empty")
	}

	// Test that model name contains alphanumeric characters, hyphens, and dots
	for _, r := range TestDefaultModel {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.') {
			t.Errorf("TestDefaultModel should only contain lowercase letters, numbers, hyphens, and dots, got: %s", TestDefaultModel)
			break
		}
	}
}

func TestTestMessage(t *testing.T) {
	if TestMessage != "Hello, world! Please respond with a friendly greeting." {
		t.Errorf("TestMessage = %s, want 'Hello, world! Please respond with a friendly greeting.'", TestMessage)
	}

	// Test that message is not empty
	if TestMessage == "" {
		t.Error("TestMessage should not be empty")
	}

	// Test that message is reasonably long for testing
	if len(TestMessage) < 10 {
		t.Errorf("TestMessage should be at least 10 characters, got: %d", len(TestMessage))
	}
}

func TestTestConfigMessages(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"TestMsgLoadingConfig", TestMsgLoadingConfig},
		{"TestMsgConfigLoaded", TestMsgConfigLoaded},
		{"TestMsgProviderNotSupported", TestMsgProviderNotSupported},
		{"TestMsgAPIKeyNotConfigured", TestMsgAPIKeyNotConfigured},
		{"TestMsgFailedToInitLogger", TestMsgFailedToInitLogger},
		{"TestMsgInitializingProvider", TestMsgInitializingProvider},
		{"TestMsgProviderInitialized", TestMsgProviderInitialized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}

			// Test format strings
			if containsString(tt.value, "%") {
				testMsg := fmt.Sprintf(tt.value, "test")
				if testMsg == "" {
					t.Errorf("%s should produce valid formatted string", tt.name)
				}
			}
		})
	}
}

func TestTestRequestMessages(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"TestMsgSendingRequest", TestMsgSendingRequest},
		{"TestMsgSendingRequestMessage", TestMsgSendingRequestMessage},
		{"TestMsgRequestFailed", TestMsgRequestFailed},
		{"TestMsgPossibleCauses", TestMsgPossibleCauses},
		{"TestMsgCauseAPIKey", TestMsgCauseAPIKey},
		{"TestMsgCauseNetwork", TestMsgCauseNetwork},
		{"TestMsgCauseUnavail", TestMsgCauseUnavail},
		{"TestMsgCauseRateLimit", TestMsgCauseRateLimit},
		{"TestMsgTroubleshooting", TestMsgTroubleshooting},
		{"TestMsgStepVerifyAPIKey", TestMsgStepVerifyAPIKey},
		{"TestMsgCheckConnection", TestMsgCheckConnection},
		{"TestMsgTryAgain", TestMsgTryAgain},
		{"TestMsgCheckStatus", TestMsgCheckStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}

			// Test format strings
			if containsString(tt.value, "%") {
				testMsg := fmt.Sprintf(tt.value, "test")
				if testMsg == "" {
					t.Errorf("%s should produce valid formatted string", tt.name)
				}
			}
		})
	}
}

func TestTestResponseMessages(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"TestMsgRequestSuccessful", TestMsgRequestSuccessful},
		{"TestMsgResponseDetails", TestMsgResponseDetails},
		{"TestMsgResponseModel", TestMsgResponseModel},
		{"TestMsgResponseLatency", TestMsgResponseLatency},
		{"TestMsgFinishReason", TestMsgFinishReason},
		{"TestMsgResponseContent", TestMsgResponseContent},
		{"TestMsgResponseContentText", TestMsgResponseContentText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}

			// Test format strings
			if containsString(tt.value, "%") {
				var testVal interface{}
				if containsString(tt.value, "%s") {
					testVal = "test"
				} else if containsString(tt.value, "%v") {
					testVal = 42
				}

				if testVal != nil {
					testMsg := fmt.Sprintf(tt.value, testVal)
					if testMsg == "" {
						t.Errorf("%s should produce valid formatted string", tt.name)
					}
				}
			}
		})
	}
}

func TestTestTokenMessages(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"TestMsgTokenUsage", TestMsgTokenUsage},
		{"TestMsgPromptTokens", TestMsgPromptTokens},
		{"TestMsgCompletionTokens", TestMsgCompletionTokens},
		{"TestMsgTotalTokens", TestMsgTotalTokens},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}

			// Test format strings
			if containsString(tt.value, "%") {
				testMsg := fmt.Sprintf(tt.value, 100)
				if testMsg == "" {
					t.Errorf("%s should produce valid formatted string", tt.name)
				}
			}
		})
	}
}

func TestTestToolMessages(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"TestMsgToolCalls", TestMsgToolCalls},
		{"TestMsgToolCallItem", TestMsgToolCallItem},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}

			// Test format strings
			if containsString(tt.value, "%") {
				testMsg := fmt.Sprintf(tt.value, 1, "test_func", "args")
				if testMsg == "" {
					t.Errorf("%s should produce valid formatted string", tt.name)
				}
			}
		})
	}
}

func TestTestStopMessages(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"TestMsgStopNormal", TestMsgStopNormal},
		{"TestMsgStopLength", TestMsgStopLength},
		{"TestMsgStopToolCalls", TestMsgStopToolCalls},
		{"TestMsgStopError", TestMsgStopError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}
		})
	}
}

func TestTestMessagesWithEmojis(t *testing.T) {
	messagesWithEmojis := []struct {
		name      string
		value     string
		wantEmoji bool
	}{
		{"TestMsgConfigLoaded", TestMsgConfigLoaded, true},
		{"TestMsgProviderNotSupported", TestMsgProviderNotSupported, true},
		{"TestMsgAPIKeyNotConfigured", TestMsgAPIKeyNotConfigured, true},
		{"TestMsgFailedToInitLogger", TestMsgFailedToInitLogger, true},
		{"TestMsgProviderInitialized", TestMsgProviderInitialized, true},
		{"TestMsgSendingRequest", TestMsgSendingRequest, true},
		{"TestMsgRequestFailed", TestMsgRequestFailed, true},
		{"TestMsgRequestSuccessful", TestMsgRequestSuccessful, true},
		{"TestMsgStopNormal", TestMsgStopNormal, true},
		{"TestMsgAllPassed", TestMsgAllPassed, true},
	}

	for _, tt := range messagesWithEmojis {
		t.Run(tt.name, func(t *testing.T) {
			hasEmoji := containsTestEmoji(tt.value)
			if tt.wantEmoji && !hasEmoji {
				t.Errorf("%s should contain emoji, got: %s", tt.name, tt.value)
			}
		})
	}
}

func TestTestAllPassed(t *testing.T) {
	expected := "\n✨ All checks passed! Your LLM provider is working correctly."
	if TestMsgAllPassed != expected {
		t.Errorf("TestMsgAllPassed = %s, want %s", TestMsgAllPassed, expected)
	}
}

func TestTestMessageFormatting(t *testing.T) {
	// Test that messages with %s format produce valid output
	formatted := fmt.Sprintf(TestMsgResponseModel, "glm-4.7")
	if formatted == "" {
		t.Error("Formatted message should not be empty")
	}

	// Test that messages with %v format produce valid output
	formatted = fmt.Sprintf(TestMsgResponseLatency, 250*time.Millisecond)
	if formatted == "" {
		t.Error("Formatted message should not be empty")
	}

	// Test that messages with %d format produce valid output
	formatted = fmt.Sprintf(TestMsgTotalTokens, 1500)
	if formatted == "" {
		t.Error("Formatted message should not be empty")
	}

	// Test that messages with %.2f format produce valid output
	formatted = fmt.Sprintf(TestMsgResponseLatency, 0.75)
	if formatted == "" {
		t.Error("Formatted message should not be empty")
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper function to check if string contains emoji
func containsTestEmoji(s string) bool {
	for _, r := range s {
		if (r >= 0x1F600 && r <= 0x1F64F) || // Emoticons
			(r >= 0x1F300 && r <= 0x1F5FF) || // Miscellaneous Symbols and Pictographs
			(r >= 0x1F680 && r <= 0x1F6FF) || // Transport and Map
			(r >= 0x1F900 && r <= 0x1F9FF) || // Supplemental Symbols and Pictographs
			(r >= 0x2600 && r <= 0x26FF) || // Misc symbols
			(r >= 0x2700 && r <= 0x27BF) { // Dingbats
			return true
		}
	}
	return false
}
