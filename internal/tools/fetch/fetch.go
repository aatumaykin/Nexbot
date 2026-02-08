package fetch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
)

type FetchTool struct {
	cfg    *config.Config
	logger *logger.Logger
}

type FetchArgs struct {
	URL    string `json:"url"`
	Format string `json:"format"`
}

func NewFetchTool(cfg *config.Config, log *logger.Logger) *FetchTool {
	return &FetchTool{
		cfg:    cfg,
		logger: log,
	}
}

func (t *FetchTool) Name() string {
	return "web_fetch"
}

func (t *FetchTool) Description() string {
	return "Fetch content from a URL. Returns formatted text with metadata."
}

func (t *FetchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "The URL to fetch. Must start with http:// or https://",
			},
			"format": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"text", "html"},
				"default":     "text",
				"description": "Output format: 'text' (strips HTML tags) or 'html' (raw HTML)",
			},
		},
		"required": []interface{}{"url"},
	}
}

func (t *FetchTool) Execute(args string) (string, error) {
	var fetchArgs FetchArgs
	if err := json.Unmarshal([]byte(args), &fetchArgs); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if fetchArgs.URL == "" {
		return "", fmt.Errorf("url is required")
	}
	if !strings.HasPrefix(fetchArgs.URL, "http://") && !strings.HasPrefix(fetchArgs.URL, "https://") {
		return "", fmt.Errorf("url must start with http:// or https://")
	}
	if fetchArgs.Format == "" {
		fetchArgs.Format = "text"
	}

	if !t.cfg.Tools.Fetch.Enabled {
		return "", fmt.Errorf("web_fetch tool is disabled in configuration")
	}

	timeout := time.Duration(t.cfg.Tools.Fetch.TimeoutSeconds) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest("GET", fetchArgs.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", t.cfg.Tools.Fetch.UserAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.ContentLength > t.cfg.Tools.Fetch.MaxResponseSize {
		return "", fmt.Errorf("response too large: %d bytes exceeds %d bytes limit",
			resp.ContentLength, t.cfg.Tools.Fetch.MaxResponseSize)
	}

	limitReader := io.LimitReader(resp.Body, t.cfg.Tools.Fetch.MaxResponseSize)
	body, err := io.ReadAll(limitReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if int64(len(body)) >= t.cfg.Tools.Fetch.MaxResponseSize {
		return "", fmt.Errorf("response truncated: exceeds %d bytes limit", t.cfg.Tools.Fetch.MaxResponseSize)
	}

	contentType := resp.Header.Get("Content-Type")
	content := string(body)

	if fetchArgs.Format == "text" && strings.Contains(contentType, "text/html") {
		content = t.stripHTML(content)
	}

	result := map[string]interface{}{
		"url":         fetchArgs.URL,
		"status":      resp.StatusCode,
		"contentType": contentType,
		"length":      len(content),
		"content":     content,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

func (t *FetchTool) stripHTML(html string) string {
	reScript := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	html = reScript.ReplaceAllString(html, "")

	reStyle := regexp.MustCompile(`(?i)<style[^>]*>.*?</style>`)
	html = reStyle.ReplaceAllString(html, "")

	reTags := regexp.MustCompile(`<[^>]+>`)
	html = reTags.ReplaceAllString(html, "\n")

	reSpace := regexp.MustCompile(`\s+`)
	html = reSpace.ReplaceAllString(html, " ")

	return strings.TrimSpace(html)
}
