package telegram

import (
	"testing"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/mymmrac/telego"
)

func TestBuildInlineKeyboard(t *testing.T) {
	connector := &Connector{}

	tests := []struct {
		name     string
		keyboard *bus.InlineKeyboard
		wantNil  bool
		wantRows int
	}{
		{
			name:     "nil keyboard",
			keyboard: nil,
			wantNil:  true,
		},
		{
			name:     "empty keyboard",
			keyboard: &bus.InlineKeyboard{Rows: [][]bus.InlineButton{}},
			wantNil:  false,
			wantRows: 0,
		},
		{
			name: "single button",
			keyboard: &bus.InlineKeyboard{
				Rows: [][]bus.InlineButton{
					{{Text: "Button 1", Data: "data1"}},
				},
			},
			wantNil:  false,
			wantRows: 1,
		},
		{
			name: "multiple buttons in single row",
			keyboard: &bus.InlineKeyboard{
				Rows: [][]bus.InlineButton{
					{{Text: "Button 1", Data: "data1"}, {Text: "Button 2", Data: "data2"}},
				},
			},
			wantNil:  false,
			wantRows: 1,
		},
		{
			name: "multiple rows",
			keyboard: &bus.InlineKeyboard{
				Rows: [][]bus.InlineButton{
					{{Text: "Button 1", Data: "data1"}},
					{{Text: "Button 2", Data: "data2"}},
					{{Text: "Button 3", Data: "data3"}},
				},
			},
			wantNil:  false,
			wantRows: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := connector.buildInlineKeyboard(tt.keyboard)

			if tt.wantNil && got != nil {
				t.Errorf("buildInlineKeyboard() = %v, want nil", got)
			}

			if !tt.wantNil {
				if got == nil {
					t.Errorf("buildInlineKeyboard() = nil, want non-nil")
					return
				}

				if len(got.InlineKeyboard) != tt.wantRows {
					t.Errorf("buildInlineKeyboard() row count = %d, want %d", len(got.InlineKeyboard), tt.wantRows)
				}
			}
		})
	}
}

func TestBuildInlineKeyboard_ButtonContent(t *testing.T) {
	connector := &Connector{}

	keyboard := &bus.InlineKeyboard{
		Rows: [][]bus.InlineButton{
			{{Text: "Button A", Data: "action_a"}},
			{{Text: "Button B", Data: "action_b"}},
		},
	}

	result := connector.buildInlineKeyboard(keyboard)

	if result == nil {
		t.Fatal("buildInlineKeyboard() returned nil")
	}

	if len(result.InlineKeyboard) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result.InlineKeyboard))
	}

	// Check first row
	if len(result.InlineKeyboard[0]) != 1 {
		t.Errorf("expected 1 button in first row, got %d", len(result.InlineKeyboard[0]))
	}
	if result.InlineKeyboard[0][0].Text != "Button A" {
		t.Errorf("expected button text 'Button A', got '%s'", result.InlineKeyboard[0][0].Text)
	}
	if result.InlineKeyboard[0][0].CallbackData != "action_a" {
		t.Errorf("expected callback data 'action_a', got '%s'", result.InlineKeyboard[0][0].CallbackData)
	}

	// Check second row
	if len(result.InlineKeyboard[1]) != 1 {
		t.Errorf("expected 1 button in second row, got %d", len(result.InlineKeyboard[1]))
	}
	if result.InlineKeyboard[1][0].Text != "Button B" {
		t.Errorf("expected button text 'Button B', got '%s'", result.InlineKeyboard[1][0].Text)
	}
	if result.InlineKeyboard[1][0].CallbackData != "action_b" {
		t.Errorf("expected callback data 'action_b', got '%s'", result.InlineKeyboard[1][0].CallbackData)
	}
}

func TestBuildInlineKeyboard_TypeCompatibility(t *testing.T) {
	connector := &Connector{}

	keyboard := &bus.InlineKeyboard{
		Rows: [][]bus.InlineButton{
			{{Text: "Test", Data: "test_data"}},
		},
	}

	result := connector.buildInlineKeyboard(keyboard)

	// Verify result type is correct
	var _ *telego.InlineKeyboardMarkup = result

	if result == nil {
		t.Fatal("buildInlineKeyboard() returned nil")
	}

	// Verify structure
	if len(result.InlineKeyboard) == 0 {
		t.Error("expected at least one row")
	}
}
