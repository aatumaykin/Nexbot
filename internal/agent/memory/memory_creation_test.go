package memory

import (
	"os"
	"testing"
)

func TestNewStore(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid JSONL store",
			config: Config{
				BaseDir: "/tmp/test-memory-jsonl",
				Format:  FormatJSONL,
			},
			wantErr: false,
		},
		{
			name: "valid Markdown store",
			config: Config{
				BaseDir: "/tmp/test-memory-markdown",
				Format:  FormatMarkdown,
			},
			wantErr: false,
		},
		{
			name: "empty base directory",
			config: Config{
				BaseDir: "",
				Format:  FormatJSONL,
			},
			wantErr: true,
		},
		{
			name: "default format when not specified",
			config: Config{
				BaseDir: "/tmp/test-memory-default",
				Format:  "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				defer os.RemoveAll(tt.config.BaseDir)
			}

			store, err := NewStore(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if store == nil {
					t.Error("NewStore() returned nil store")
				}

				// Verify default format is set
				if tt.config.Format == "" {
					if _, ok := store.format.(*JSONLFormat); !ok {
						t.Errorf("Expected default format JSONL, got %T", store.format)
					}
				}

				// Verify directory was created
				if _, err := os.Stat(tt.config.BaseDir); os.IsNotExist(err) {
					t.Error("Base directory should be created")
				}
			}
		})
	}
}
