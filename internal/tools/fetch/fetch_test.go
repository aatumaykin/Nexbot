package fetch

import (
	"encoding/base64"
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
		_, _ = w.Write([]byte("Hello, World!"))
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
		_, _ = w.Write([]byte(htmlContent))
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
		_, _ = w.Write([]byte(htmlContent))
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
		_, _ = w.Write([]byte("Response"))
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
		_, _ = w.Write([]byte(largeContent))
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
		_, _ = w.Write([]byte(jsonResponse))
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
		_, _ = w.Write([]byte("Not Found"))
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
		_, _ = w.Write([]byte("OK"))
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

func TestFetchTool_Execute_FollowRedirects_True(t *testing.T) {
	redirectCount := 0
	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Final Content"))
	}))
	defer finalServer.Close()

	initialServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		if redirectCount == 1 {
			w.Header().Set("Location", finalServer.URL)
			w.WriteHeader(http.StatusFound)
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Should not reach here"))
		}
	}))
	defer initialServer.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]interface{}{
		"url":             initialServer.URL,
		"followRedirects": true,
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Equal(t, float64(200), resultJSON["status"])
	assert.Equal(t, "Final Content", resultJSON["content"])
	assert.Equal(t, 1, redirectCount)
}

func TestFetchTool_Execute_FollowRedirects_False(t *testing.T) {
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Should not reach here"))
	}))
	defer redirectTarget.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", redirectTarget.URL)
		w.WriteHeader(http.StatusFound)
	}))
	defer redirectServer.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]interface{}{
		"url":             redirectServer.URL,
		"followRedirects": false,
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Equal(t, float64(302), resultJSON["status"])
}

func TestFetchTool_Execute_FollowRedirects_Default(t *testing.T) {
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Final Content"))
	}))
	defer redirectTarget.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", redirectTarget.URL)
		w.WriteHeader(http.StatusFound)
	}))
	defer redirectServer.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]string{
		"url": redirectServer.URL,
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Equal(t, float64(200), resultJSON["status"])
	assert.Equal(t, "Final Content", resultJSON["content"])
}

func TestFetchTool_Execute_TimeoutOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Response"))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Fetch: config.FetchToolConfig{
				Enabled:         true,
				TimeoutSeconds:  10,
				MaxResponseSize: 1024 * 1024,
				UserAgent:       "nexbot/1.0",
			},
		},
	}
	tool := NewFetchTool(cfg, log)

	args, _ := json.Marshal(map[string]interface{}{
		"url":     server.URL,
		"timeout": 2,
	})

	_, err := tool.Execute(string(args))

	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "timeout")
}

func TestFetchTool_Execute_TimeoutOverride_Valid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Response"))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Fetch: config.FetchToolConfig{
				Enabled:         true,
				TimeoutSeconds:  10,
				MaxResponseSize: 1024 * 1024,
				UserAgent:       "nexbot/1.0",
			},
		},
	}
	tool := NewFetchTool(cfg, log)

	args, _ := json.Marshal(map[string]interface{}{
		"url":     server.URL,
		"timeout": 5,
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Equal(t, float64(200), resultJSON["status"])
	assert.Equal(t, "Response", resultJSON["content"])
}

func TestFetchTool_Execute_Timeout_TooLow(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	tests := []struct {
		name          string
		timeout       int
		errorContains string
	}{
		{
			name:          "zero",
			timeout:       0,
			errorContains: "at least 1 second",
		},
		{
			name:          "negative",
			timeout:       -1,
			errorContains: "at least 1 second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := json.Marshal(map[string]interface{}{
				"url":     "http://example.com",
				"timeout": tt.timeout,
			})

			_, err := tool.Execute(string(args))

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContains)
		})
	}
}

func TestFetchTool_Execute_Timeout_TooHigh(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	tests := []struct {
		name          string
		timeout       int
		errorContains string
	}{
		{
			name:          "121 seconds",
			timeout:       121,
			errorContains: "exceed 120 seconds",
		},
		{
			name:          "300 seconds",
			timeout:       300,
			errorContains: "exceed 120 seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := json.Marshal(map[string]interface{}{
				"url":     "http://example.com",
				"timeout": tt.timeout,
			})

			_, err := tool.Execute(string(args))

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContains)
		})
	}
}

func TestFetchTool_Execute_Timeout_Default(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Response"))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Fetch: config.FetchToolConfig{
				Enabled:         true,
				TimeoutSeconds:  10,
				MaxResponseSize: 1024 * 1024,
				UserAgent:       "nexbot/1.0",
			},
		},
	}
	tool := NewFetchTool(cfg, log)

	args, _ := json.Marshal(map[string]string{
		"url": server.URL,
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Equal(t, float64(200), resultJSON["status"])
}

func TestFetchTool_Execute_JSONFormat(t *testing.T) {
	jsonResponse := `{"name":"Test","value":123,"nested":{"key":"value"}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jsonResponse))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]string{
		"url":    server.URL,
		"format": "json",
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Contains(t, resultJSON, "json")
	jsonData := resultJSON["json"].(map[string]interface{})
	assert.Equal(t, "Test", jsonData["name"])
	assert.Equal(t, float64(123), jsonData["value"])

	nested := jsonData["nested"].(map[string]interface{})
	assert.Equal(t, "value", nested["key"])
}

func TestFetchTool_Execute_JSONFormat_InvalidJSON(t *testing.T) {
	invalidJSON := `{"invalid": json}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(invalidJSON))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]string{
		"url":    server.URL,
		"format": "json",
	})

	_, err := tool.Execute(string(args))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON response")
}

func TestFetchTool_Execute_StatusText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]string{
		"url": server.URL,
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Contains(t, resultJSON, "statusText")
	assert.Equal(t, "404 Not Found", resultJSON["statusText"])
}

func TestFetchTool_Execute_Headers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Cache-Control", "max-age=3600")
		w.Header().Set("Server", "TestServer/1.0")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]string{
		"url": server.URL,
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Contains(t, resultJSON, "headers")
	headers := resultJSON["headers"].(map[string]interface{})
	assert.Equal(t, "text/html", headers["Content-Type"])
	assert.Equal(t, "max-age=3600", headers["Cache-Control"])
	assert.Equal(t, "TestServer/1.0", headers["Server"])
}

func TestFetchTool_Execute_BasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
		if authHeader != expectedAuth {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Authenticated"))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]interface{}{
		"url": server.URL,
		"basicAuth": map[string]string{
			"username": "testuser",
			"password": "testpass",
		},
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Equal(t, float64(200), resultJSON["status"])
	assert.Equal(t, "Authenticated", resultJSON["content"])
}

func TestFetchTool_Execute_BasicAuth_SecretPassword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:secretvalue"))
		if authHeader != expectedAuth {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Authenticated with secret"))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	tool.SetSecretResolver(func(sessionID, value string) string {
		if value == "$MYPASS" {
			return "secretvalue"
		}
		return value
	})
	tool.SetSessionID("test-session")

	args, _ := json.Marshal(map[string]interface{}{
		"url": server.URL,
		"basicAuth": map[string]string{
			"username": "testuser",
			"password": "$MYPASS",
		},
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Equal(t, float64(200), resultJSON["status"])
	assert.Equal(t, "Authenticated with secret", resultJSON["content"])
}

func TestFetchTool_Execute_BasicAuth_Invalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Basic invalidcredentials" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]interface{}{
		"url": server.URL,
		"basicAuth": map[string]string{
			"username": "testuser",
			"password": "wrongpass",
		},
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Equal(t, float64(401), resultJSON["status"])
}

func TestFetchTool_Execute_Cookies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookieHeader := r.Header.Get("Cookie")
		if cookieHeader == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !strings.Contains(cookieHeader, "sessionid=abc123") || !strings.Contains(cookieHeader, "user_pref=dark") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Cookies received"))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]interface{}{
		"url": server.URL,
		"cookies": map[string]string{
			"sessionid": "abc123",
			"user_pref": "dark",
		},
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Equal(t, float64(200), resultJSON["status"])
	assert.Equal(t, "Cookies received", resultJSON["content"])
}

func TestFetchTool_Execute_Cookies_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookieHeader := r.Header.Get("Cookie")
		if cookieHeader != "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("No cookies"))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]interface{}{
		"url":     server.URL,
		"cookies": map[string]string{},
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Equal(t, float64(200), resultJSON["status"])
	assert.Equal(t, "No cookies", resultJSON["content"])
}

func TestFetchTool_Execute_Cookies_Single(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookieHeader := r.Header.Get("Cookie")
		if cookieHeader != "singlecookie=value1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Single cookie"))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]interface{}{
		"url": server.URL,
		"cookies": map[string]string{
			"singlecookie": "value1",
		},
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Equal(t, float64(200), resultJSON["status"])
	assert.Equal(t, "Single cookie", resultJSON["content"])
}

func TestFetchTool_Execute_BasicAuthAndCookies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
		if authHeader != expectedAuth {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		cookieHeader := r.Header.Get("Cookie")
		if !strings.Contains(cookieHeader, "token=xyz") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Both auth and cookies"))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]interface{}{
		"url": server.URL,
		"basicAuth": map[string]string{
			"username": "user",
			"password": "pass",
		},
		"cookies": map[string]string{
			"token": "xyz",
		},
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	assert.Equal(t, float64(200), resultJSON["status"])
	assert.Equal(t, "Both auth and cookies", resultJSON["content"])
}

func TestFetchTool_Execute_MarkdownFormat(t *testing.T) {
	htmlContent := `<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
<h1>Main Title</h1>
<h2>Subtitle</h2>
<p>Paragraph with <strong>bold</strong> and <em>italic</em> text.</p>
<p>Code: <code>console.log("hello")</code></p>
<ul>
<li>Item 1</li>
<li>Item 2</li>
</ul>
<ol>
<li>First</li>
<li>Second</li>
</ol>
<p><a href="https://example.com">Link text</a></p>
<p><img src="image.jpg" alt="Image"></p>
<pre><code>
code block
</code></pre>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	args, _ := json.Marshal(map[string]string{
		"url":    server.URL,
		"format": "markdown",
	})

	result, err := tool.Execute(string(args))

	require.NoError(t, err)

	var resultJSON map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultJSON)
	require.NoError(t, err)

	content := resultJSON["content"].(string)

	assert.Contains(t, content, "# Main Title")
	assert.Contains(t, content, "## Subtitle")
	assert.Contains(t, content, "**bold**")
	assert.Contains(t, content, "*italic*")
	assert.Contains(t, content, "`console.log(\"hello\")`")
	assert.Contains(t, content, "- Item 1")
	assert.Contains(t, content, "- First")
	assert.Contains(t, content, "[Link text](https://example.com)")
	assert.Contains(t, content, "![Image](image.jpg)")
	assert.Contains(t, content, "```")
	assert.NotContains(t, content, "<h1>")
	assert.NotContains(t, content, "<script>")
	assert.NotContains(t, content, "<style>")
}

func TestHtmlToMarkdown(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	tests := []struct {
		name        string
		html        string
		contains    []string
		notContains []string
	}{
		{
			name:        "headers",
			html:        "<h1>Title</h1><h2>Subtitle</h2><h3>Section</h3>",
			contains:    []string{"# Title", "## Subtitle", "### Section"},
			notContains: []string{"<h1>", "<h2>", "<h3>"},
		},
		{
			name:        "bold and italic",
			html:        "<p><strong>bold</strong> and <b>also bold</b>. <em>italic</em> and <i>also italic</i>.</p>",
			contains:    []string{"**bold**", "**also bold**", "*italic*", "*also italic*"},
			notContains: []string{"<strong>", "<b>", "<em>", "<i>"},
		},
		{
			name:        "code and pre",
			html:        "<p>Inline <code>code</code></p><pre>code block</pre>",
			contains:    []string{"`code`", "```", "code block"},
			notContains: []string{"<code>", "<pre>"},
		},
		{
			name:        "links",
			html:        `<a href="https://example.com">Link text</a>`,
			contains:    []string{"[Link text](https://example.com)"},
			notContains: []string{"<a href="},
		},
		{
			name:        "images with alt",
			html:        `<img src="image.jpg" alt="Description">`,
			contains:    []string{"![Description](image.jpg)"},
			notContains: []string{"<img"},
		},
		{
			name:        "images without alt",
			html:        `<img src="image.jpg">`,
			contains:    []string{"![](image.jpg)"},
			notContains: []string{"<img"},
		},
		{
			name:        "lists",
			html:        "<ul><li>Item 1</li><li>Item 2</li></ul>",
			contains:    []string{"- Item 1", "- Item 2"},
			notContains: []string{"<ul>", "<li>"},
		},
		{
			name:        "ordered lists",
			html:        "<ol><li>First</li><li>Second</li></ol>",
			contains:    []string{"- First", "- Second"},
			notContains: []string{"<ol>", "<li>"},
		},
		{
			name:        "paragraphs",
			html:        "<p>First paragraph</p><p>Second paragraph</p>",
			contains:    []string{"First paragraph", "Second paragraph"},
			notContains: []string{"<p>"},
		},
		{
			name:        "script and style tags",
			html:        `<p>Content</p><script>alert('xss')</script><style>body{color:red}</style>`,
			contains:    []string{"Content"},
			notContains: []string{"<script>", "<style>", "alert", "color"},
		},
		{
			name:        "nested formatting",
			html:        `<p><strong>Bold with <em>italic inside</em></strong></p>`,
			contains:    []string{"**Bold with *italic inside***"},
			notContains: []string{"<strong>", "<em>"},
		},
		{
			name:        "line breaks",
			html:        `<p>Line 1<br>Line 2</p>`,
			contains:    []string{"Line 1", "Line 2"},
			notContains: []string{"<br>"},
		},
		{
			name:        "empty HTML",
			html:        "",
			contains:    []string{""},
			notContains: nil,
		},
		{
			name:        "plain text",
			html:        "Just plain text",
			contains:    []string{"Just plain text"},
			notContains: nil,
		},
		{
			name:        "mixed content",
			html:        `<h1>Title</h1><p>Text with <strong>bold</strong> and <a href="http://test.com">link</a>.</p>`,
			contains:    []string{"# Title", "**bold**", "[link](http://test.com)"},
			notContains: []string{"<h1>", "<strong>", "<a href"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.htmlToMarkdown(tt.html)
			for _, s := range tt.contains {
				assert.Contains(t, result, s)
			}
			for _, s := range tt.notContains {
				assert.NotContains(t, result, s)
			}
		})
	}
}

func TestFetchTool_Parameters_Markdown(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewFetchTool(testConfig(), log)

	params := tool.Parameters()
	properties, ok := params["properties"].(map[string]interface{})
	require.True(t, ok)

	formatProp, ok := properties["format"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, formatProp["enum"], "markdown")
	assert.Contains(t, formatProp["description"], "Markdown")
}
