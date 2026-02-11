package fetch

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
)

type FetchTool struct {
	cfg       *config.Config
	logger    *logger.Logger
	resolver  func(string, string) string
	sessionID string
}

type FetchArgs struct {
	URL             string            `json:"url"`
	Format          string            `json:"format"`
	Headers         map[string]string `json:"headers"`
	Method          string            `json:"method"`
	Body            string            `json:"body"`
	BasicAuth       *BasicAuth        `json:"basicAuth"`
	Cookies         map[string]string `json:"cookies"`
	FollowRedirects *bool             `json:"followRedirects"`
	Timeout         *int              `json:"timeout"`
}

type BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
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

func (t *FetchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch. Must start with http:// or https://",
			},
			"format": map[string]any{
				"type":        "string",
				"enum":        []string{"text", "html", "markdown", "json"},
				"default":     "text",
				"description": "Output format: 'text' (strips HTML tags), 'html' (raw HTML), 'markdown' (converts HTML to Markdown), or 'json' (parse JSON response)",
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "Optional HTTP headers. Use $SECRET_NAME to reference secrets. Example: {\"Authorization\": \"Bearer $APIKEY\"}",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
			"basicAuth": map[string]any{
				"type":        "object",
				"description": "Optional Basic Authentication. Use $SECRET_NAME for password to reference secrets. Example: {\"username\": \"user\", \"password\": \"$MYPASS\"}",
				"properties": map[string]any{
					"username": map[string]any{
						"type":        "string",
						"description": "Username for Basic Auth",
					},
					"password": map[string]any{
						"type":        "string",
						"description": "Password for Basic Auth. Supports $SECRET_NAME reference",
					},
				},
			},
			"cookies": map[string]any{
				"type":        "object",
				"description": "Optional cookies to send. Example: {\"sessionid\": \"abc123\", \"user_pref\": \"dark\"}",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
			"followRedirects": map[string]any{
				"type":        "boolean",
				"default":     true,
				"description": "Follow HTTP redirects. Set to false to stop at the first redirect and return the redirect URL",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout in seconds (1-120). Overrides the default configuration. Omit to use default timeout",
				"minimum":     1,
				"maximum":     120,
			},
			"method": map[string]any{
				"type":        "string",
				"enum":        []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
				"default":     "GET",
				"description": "HTTP method to use",
			},
			"body": map[string]any{
				"type":        "string",
				"description": "Request body (for POST, PUT, PATCH methods)",
			},
		},
		"required": []any{"url"},
	}
}

func (t *FetchTool) SetSecretResolver(resolver func(string, string) string) {
	t.resolver = resolver
}

func (t *FetchTool) SetSessionID(sessionID string) {
	t.sessionID = sessionID
}

func (t *FetchTool) Execute(args string) (string, error) {
	var fetchArgs FetchArgs
	if err := json.Unmarshal([]byte(args), &fetchArgs); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if fetchArgs.URL == "" {
		return "", fmt.Errorf("url is required")
	}
	if fetchArgs.Format == "" {
		fetchArgs.Format = "text"
	}
	if fetchArgs.Method == "" {
		fetchArgs.Method = "GET"
	}
	if fetchArgs.Body != "" && (fetchArgs.Method == "GET" || fetchArgs.Method == "HEAD" || fetchArgs.Method == "DELETE") {
		fetchArgs.Body = ""
	}

	if !t.cfg.Tools.Fetch.Enabled {
		return "", fmt.Errorf("web_fetch tool is disabled in configuration")
	}

	timeout := time.Duration(t.cfg.Tools.Fetch.TimeoutSeconds) * time.Second
	if fetchArgs.Timeout != nil {
		if *fetchArgs.Timeout < 1 {
			return "", fmt.Errorf("timeout must be at least 1 second")
		}
		if *fetchArgs.Timeout > 120 {
			return "", fmt.Errorf("timeout cannot exceed 120 seconds")
		}
		timeout = time.Duration(*fetchArgs.Timeout) * time.Second
	}

	client := &http.Client{
		Timeout: timeout,
	}

	if fetchArgs.FollowRedirects != nil && !*fetchArgs.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	var bodyReader io.Reader
	if fetchArgs.Body != "" {
		bodyReader = strings.NewReader(fetchArgs.Body)
	}

	url := fetchArgs.URL
	if t.resolver != nil && t.sessionID != "" {
		url = t.resolver(t.sessionID, url)
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "", fmt.Errorf("url must start with http:// or https://")
	}

	req, err := http.NewRequest(fetchArgs.Method, url, bodyReader)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", t.cfg.Tools.Fetch.UserAgent)
	req.Header.Set("Accept", "*/*")
	if fetchArgs.Body != "" {
		contentTypeSet := false
		for name := range fetchArgs.Headers {
			if strings.ToLower(name) == "content-type" {
				contentTypeSet = true
				break
			}
		}
		if !contentTypeSet {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	for name, value := range fetchArgs.Headers {
		if t.resolver != nil && t.sessionID != "" {
			value = t.resolver(t.sessionID, value)
		}
		req.Header.Set(name, value)
	}

	if fetchArgs.BasicAuth != nil && fetchArgs.BasicAuth.Username != "" {
		password := fetchArgs.BasicAuth.Password
		if t.resolver != nil && t.sessionID != "" && strings.HasPrefix(password, "$") {
			password = t.resolver(t.sessionID, password)
		}
		authValue := fetchArgs.BasicAuth.Username + ":" + password
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(authValue))
		req.Header.Set("Authorization", "Basic "+encodedAuth)
	}

	if len(fetchArgs.Cookies) > 0 {
		cookiePairs := make([]string, 0, len(fetchArgs.Cookies))
		for key, value := range fetchArgs.Cookies {
			cookiePairs = append(cookiePairs, key+"="+value)
		}
		req.Header.Set("Cookie", strings.Join(cookiePairs, "; "))
	}

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

	if fetchArgs.Format == "markdown" && strings.Contains(contentType, "text/html") {
		content = t.htmlToMarkdown(content)
	}

	result := map[string]any{
		"url":         fetchArgs.URL,
		"status":      resp.StatusCode,
		"statusText":  resp.Status,
		"contentType": contentType,
		"length":      len(content),
		"content":     content,
	}

	if fetchArgs.Format == "json" {
		var jsonData any
		if err := json.Unmarshal(body, &jsonData); err != nil {
			return "", fmt.Errorf("failed to parse JSON response: %w", err)
		}
		result["json"] = jsonData
	}

	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	result["headers"] = headers

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

func (t *FetchTool) htmlToMarkdown(html string) string {
	opts := &md.Options{
		HeadingStyle:    "atx",
		CodeBlockStyle:  "fenced",
		EmDelimiter:     "*",
		StrongDelimiter: "**",
	}

	converter := md.NewConverter("", true, opts)

	converter.Keep("a", "img")

	converter.AddRules(md.Rule{
		Filter: []string{"nav", "footer", "aside", "script", "style"},
		Replacement: func(content string, selec *goquery.Selection, opt *md.Options) *string {
			return new("")
		},
	})

	markdown, err := converter.ConvertString(html)
	if err != nil {
		t.logger.Error("Failed to convert HTML to Markdown", err)
		return ""
	}

	reSpace := regexp.MustCompile(`\s+`)
	markdown = reSpace.ReplaceAllString(markdown, " ")

	reCleanNewlines := regexp.MustCompile(`\n{3,}`)
	markdown = reCleanNewlines.ReplaceAllString(markdown, "\n\n")

	return strings.TrimSpace(markdown)
}
