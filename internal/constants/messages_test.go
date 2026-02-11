package constants

import (
	"fmt"
	"strings"
	"testing"
)

func TestCommandMessages(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "MsgSessionCleared",
			value: MsgSessionCleared,
		},
		{
			name:  "MsgStatusError",
			value: MsgStatusError,
		},
		{
			name:  "MsgRestarting",
			value: MsgRestarting,
		},
		{
			name:  "MsgErrorFormat",
			value: MsgErrorFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}

			// Test that format strings can be used
			if strings.Contains(tt.value, "%") {
				testMsg := fmt.Sprintf(tt.value, "test")
				if testMsg == "" {
					t.Errorf("%s should produce valid formatted string", tt.name)
				}
			}
		})
	}
}

func TestStatusMessages(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "MsgStatusHeader",
			value: MsgStatusHeader,
		},
		{
			name:  "MsgStatusSessionID",
			value: MsgStatusSessionID,
		},
		{
			name:  "MsgStatusMessages",
			value: MsgStatusMessages,
		},
		{
			name:  "MsgStatusSessionSize",
			value: MsgStatusSessionSize,
		},
		{
			name:  "MsgStatusLLMConfig",
			value: MsgStatusLLMConfig,
		},
		{
			name:  "MsgStatusModel",
			value: MsgStatusModel,
		},
		{
			name:  "MsgStatusTemp",
			value: MsgStatusTemp,
		},
		{
			name:  "MsgStatusMaxTokens",
			value: MsgStatusMaxTokens,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}

			// Test format strings
			if strings.Contains(tt.value, "%") {
				var testVal any
				if strings.Contains(tt.value, "%s") {
					testVal = "test"
				} else if strings.Contains(tt.value, "%d") {
					testVal = 42
				} else if strings.Contains(tt.value, "%.2f") {
					testVal = 0.7
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

func TestConfigMessages(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "MsgConfigValidating",
			value: MsgConfigValidating,
		},
		{
			name:  "MsgConfigLoadError",
			value: MsgConfigLoadError,
		},
		{
			name:  "MsgConfigValidationError",
			value: MsgConfigValidationError,
		},
		{
			name:  "MsgConfigValid",
			value: MsgConfigValid,
		},
		{
			name:  "MsgConfigValidatePrefix",
			value: MsgConfigValidatePrefix,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}

			// Test format strings
			if strings.Contains(tt.value, "%") {
				testMsg := fmt.Sprintf(tt.value, "test")
				if testMsg == "" {
					t.Errorf("%s should produce valid formatted string", tt.name)
				}
			}
		})
	}
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "MsgErrorLoadingJobs",
			value: MsgErrorLoadingJobs,
		},
		{
			name:  "MsgErrorSavingJobs",
			value: MsgErrorSavingJobs,
		},
		{
			name:  "MsgErrorJobNotFound",
			value: MsgErrorJobNotFound,
		},
		{
			name:  "MsgErrorNoJobsFound",
			value: MsgErrorNoJobsFound,
		},
		{
			name:  "MsgErrorConfigLoad",
			value: MsgErrorConfigLoad,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}

			// Test format strings
			if strings.Contains(tt.value, "%") {
				testMsg := fmt.Sprintf(tt.value, "test error")
				if testMsg == "" {
					t.Errorf("%s should produce valid formatted string", tt.name)
				}
			}
		})
	}
}

func TestJobMessages(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "MsgJobAdded",
			value: MsgJobAdded,
		},
		{
			name:  "MsgJobID",
			value: MsgJobID,
		},
		{
			name:  "MsgJobSchedule",
			value: MsgJobSchedule,
		},
		{
			name:  "MsgJobCommand",
			value: MsgJobCommand,
		},
		{
			name:  "MsgJobRemoveNote",
			value: MsgJobRemoveNote,
		},
		{
			name:  "MsgJobRemoved",
			value: MsgJobRemoved,
		},
		{
			name:  "MsgJobNotFoundHint",
			value: MsgJobNotFoundHint,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}

			// Test format strings
			if strings.Contains(tt.value, "%") {
				testMsg := fmt.Sprintf(tt.value, "test")
				if testMsg == "" {
					t.Errorf("%s should produce valid formatted string", tt.name)
				}
			}
		})
	}
}

func TestJobsListMessages(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "MsgJobsListHeader",
			value: MsgJobsListHeader,
		},
		{
			name:  "MsgJobsListSep",
			value: MsgJobsListSep,
		},
		{
			name:  "MsgJobsMetadata",
			value: MsgJobsMetadata,
		},
		{
			name:  "MsgJobsTotal",
			value: MsgJobsTotal,
		},
		{
			name:  "MsgJobsNotFound",
			value: MsgJobsNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}

			// Test format strings
			if strings.Contains(tt.value, "%") {
				testMsg := fmt.Sprintf(tt.value, 5)
				if testMsg == "" {
					t.Errorf("%s should produce valid formatted string", tt.name)
				}
			}
		})
	}
}

func TestTelegramMessages(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "TelegramMsgAuthError",
			value: TelegramMsgAuthError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}
		})
	}
}

func TestMessagesContainEmojis(t *testing.T) {
	messagesWithEmojis := []struct {
		name      string
		value     string
		wantEmoji bool
	}{
		{"MsgSessionCleared", MsgSessionCleared, true},
		{"MsgStatusError", MsgStatusError, true},
		{"MsgRestarting", MsgRestarting, true},
		{"MsgConfigValid", MsgConfigValid, true},
		{"MsgJobAdded", MsgJobAdded, true},
		{"MsgJobRemoved", MsgJobRemoved, true},
		{"TelegramMsgAuthError", TelegramMsgAuthError, true},
	}

	for _, tt := range messagesWithEmojis {
		t.Run(tt.name, func(t *testing.T) {
			hasEmoji := containsEmoji(tt.value)
			if tt.wantEmoji && !hasEmoji {
				t.Errorf("%s should contain emoji, got: %s", tt.name, tt.value)
			}
		})
	}
}

func TestMessageFormatConsistency(t *testing.T) {
	// Test that all messages with newlines end properly
	messagesWithNewlines := []string{
		MsgStatusSessionID,
		MsgStatusMessages,
		MsgStatusSessionSize,
		MsgStatusModel,
		MsgStatusTemp,
		MsgStatusMaxTokens,
		MsgJobID,
		MsgJobSchedule,
		MsgJobCommand,
	}

	for _, msg := range messagesWithNewlines {
		if !strings.HasSuffix(msg, "\n") {
			t.Errorf("Message should end with newline: %q", msg)
		}
	}
}

func containsEmoji(s string) bool {
	// Basic check for common emoji ranges
	for _, r := range s {
		if (r >= 0x1F600 && r <= 0x1F64F) || // Emoticons
			(r >= 0x1F300 && r <= 0x1F5FF) || // Miscellaneous Symbols and Pictographs
			(r >= 0x1F680 && r <= 0x1F6FF) || // Transport and Map
			(r >= 0x1F700 && r <= 0x1F77F) || // Alchemical Symbols
			(r >= 0x1F900 && r <= 0x1F9FF) || // Supplemental Symbols and Pictographs
			(r >= 0x1FA00 && r <= 0x1FA6F) || // Chess Symbols
			(r >= 0x1FA70 && r <= 0x1FAFF) || // Symbols and Pictographs Extended-A
			(r >= 0x2600 && r <= 0x26FF) || // Misc symbols
			(r >= 0x2700 && r <= 0x27BF) { // Dingbats
			return true
		}
	}
	return false
}
