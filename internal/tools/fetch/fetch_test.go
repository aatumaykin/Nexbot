package fetch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig creates a test configuration with default values.
func testConfig() *config.Config {
	return &config.Config{
		Tools: config.ToolsConfig{
			Fetch: config.FetchToolConfig{
				Enabled:         true,
				TimeoutSeconds:  10,
				MaxResponseSize: 1024 * 1024,
				UserAgent:       "nexbot/1.0",
			},
		},
	}
}

func TestFetchTool_Name(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	assert.Equal(t, "web_fetch", tool.Name())
}

func TestFetchTool_Description(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	desc := tool.Description()

	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "URL")
	assert.Contains(t, desc, "Fetch")
}

func TestFetchTool_Parameters(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	params := tool.Parameters()

	// Check structure
	assert.Equal(t, "object", params["type"])
	properties, ok := params["properties"].(map[string]interface{})
	require.True(t, ok)

	// Check url property
	urlProp, ok := properties["url"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "string", urlProp["type"])
	assert.Contains(t, urlProp["description"].(string), "http://")
	assert.Contains(t, urlProp["description"].(string), "https://")

	// Check format property
	formatProp, ok := properties["format"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "string", formatProp["type"])
	assert.Contains(t, formatProp["enum"], "text")
	assert.Contains(t, formatProp["enum"], "html")
	assert.Equal(t, "text", formatProp["default"])

	// Check required fields
	required, ok := params["required"].([]interface{})
	require.True(t, ok)
	assert.Len(t, required, 1)
	assert.Equal(t, "url", required[0])
}

func TestFetchTool_Execute_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]string{
		"url": server.URL,
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	// Parse result JSON
	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Equal(t, server.URL, resultJSON["url"])
	assert.Equal(t, float64(200), resultJSON["status"])
	assert.Equal(t, "Hello, World!", resultJSON["content"])
}

func TestFetchTool_Execute_InvalidURL(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	tests := []struct {
		name          string
		url           string
		errorContains string
	}{
		{
			name:          "ftp protocol not allowed",
			url:           "ftp://example.com/file.txt",
			errorContains: "http:// or https://",
		},
		{
			name:          "file protocol not allowed",
			url:           "file:///etc/passwd",
			errorContains: "http:// or https://",
		},
		{
			name:          "no protocol",
			url:           "example.com",
			errorContains: "http:// or https://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := json.Marshal(map[string]string{
				"url": tt.url,
			})

			_, err := tool.Execute(string(args))

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContains)
		})
	}
}

func TestFetchTool_Execute_HTMLStrip(t *testing.T) {
	// Create mock server with HTML content
	htmlContent := `<html>
<head><title>Test Page</title></head>
<body>
<h1>Header</h1>
<p>Paragraph with <strong>bold</strong> text.</p>
<ul>
<li>Item 1</li>
<li>Item 2</li>
</ul>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]string{
		"url":    server.URL,
		"format": "text",
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	// Parse result JSON
	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	content := resultJSON["content"].(string)

	// Check that HTML tags are stripped
	assert.NotContains(t, content, "<html>")
	assert.NotContains(t, content, "<head>")
	assert.NotContains(t, content, "<body>")
	assert.NotContains(t, content, "<h1>")
	assert.NotContains(t, content, "<p>")
	assert.NotContains(t, content, "<script>")
	assert.NotContains(t, content, "<style>")

	// Check that text content is present
	assert.Contains(t, content, "Header")
	assert.Contains(t, content, "Paragraph")
	assert.Contains(t, content, "bold")
	assert.Contains(t, content, "text")
	assert.Contains(t, content, "Item 1")
	assert.Contains(t, content, "Item 2")
}

func TestFetchTool_Execute_RawHTML(t *testing.T) {
	htmlContent := `<html><body><p>Test</p></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]string{
		"url":    server.URL,
		"format": "html",
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	// Parse result JSON
	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	// HTML format returns raw HTML
	assert.Equal(t, htmlContent, resultJSON["content"])
}

func TestFetchTool_Execute_Timeout(t *testing.T) {
	// Create mock server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Response"))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Fetch: config.FetchToolConfig{
				Enabled:         true,
				TimeoutSeconds:  1,
				MaxResponseSize: 1024 * 1024,
				UserAgent:       "nexbot/1.0",
			},
		},
	}
	tool := NewFetchTool(cfg, log)

	args, _ := json.Marshal(map[string]string{
		"url": server.URL,
	})

	_, err := tool.Execute(string(args))

	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "timeout")
}

func TestFetchTool_Execute_SizeLimit(t *testing.T) {
	// Create mock server that returns large content
	largeContent := strings.Repeat("x", 1024*1024) // 1MB

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeContent))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Fetch: config.FetchToolConfig{
				Enabled:         true,
				TimeoutSeconds:  10,
				MaxResponseSize: 100, // Only allow 100 bytes
				UserAgent:       "nexbot/1.0",
			},
		},
	}
	tool := NewFetchTool(cfg, log)

	args, _ := json.Marshal(map[string]string{
		"url": server.URL,
	})

	_, err := tool.Execute(string(args))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
}

func TestFetchTool_Execute_JSONResponse(t *testing.T) {
	jsonResponse := `{"name":"Test","value":123,"nested":{"key":"value"}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jsonResponse))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]string{
		"url":    server.URL,
		"format": "text",
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	// Parse result JSON
	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	// JSON format returns raw JSON string in content
	assert.Contains(t, resultJSON["content"].(string), `"name":"Test"`)
	assert.Contains(t, resultJSON["content"].(string), `"value":123`)
}

func TestFetchTool_Execute_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]string{
		"url": server.URL,
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	// Parse result JSON
	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	// Should still return result with error status
	assert.Equal(t, float64(404), resultJSON["status"])
	assert.Equal(t, "Not Found", resultJSON["content"])
}

func TestFetchTool_Execute_Disabled(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Fetch: config.FetchToolConfig{
				Enabled: false,
			},
		},
	}
	tool := NewFetchTool(cfg, log)

	args, _ := json.Marshal(map[string]string{
		"url": "http://example.com",
	})

	_, err := tool.Execute(string(args))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestFetchTool_Execute_MissingURL(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	tests := []struct {
		name string
		args string
	}{
		{
			name: "empty args",
			args: `{}`,
		},
		{
			name: "only format",
			args: `{"format": "text"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.Execute(tt.args)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "url")
		})
	}
}

func TestFetchTool_Execute_UserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")
		if userAgent != "CustomAgent/1.0" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Fetch: config.FetchToolConfig{
				Enabled:         true,
				TimeoutSeconds:  10,
				MaxResponseSize: 1024 * 1024,
				UserAgent:       "CustomAgent/1.0",
			},
		},
	}
	tool := NewFetchTool(cfg, log)

	args, _ := json.Marshal(map[string]string{
		"url": server.URL,
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	// Parse result JSON
	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Equal(t, "OK", resultJSON["content"])
}

func TestStripHTML(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "simple HTML",
			html:     `<p>Hello World</p>`,
			expected: "Hello World",
		},
		{
			name:     "nested tags",
			html:     `<div><p>Hello <strong>World</strong></p></div>`,
			expected: "Hello World",
		},
		{
			name:     "script tag removal",
			html:     `<p>Content</p><script>alert('xss')</script>`,
			expected: "Content",
		},
		{
			name:     "style tag removal",
			html:     `<p>Content</p><style>body{color:red}</style>`,
			expected: "Content",
		},
		{
			name:     "empty HTML",
			html:     ``,
			expected: ``,
		},
		{
			name:     "text only",
			html:     `Plain text`,
			expected: "Plain text",
		},
		{
			name:     "whitespace normalization",
			html:     `<p>  Multiple   spaces  </p>`,
			expected: "Multiple spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.stripHTML(tt.html)
			assert.Equal(t, tt.expected, result)
		})
	}
}
