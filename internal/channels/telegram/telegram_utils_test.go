// Package telegram provides unit tests for telegram utility functions.
package telegram

import (
	"testing"
)

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected ContentType
	}{
		{
			name:     "plain text",
			text:     "Just plain text",
			expected: ContentTypePlain,
		},
		{
			name:     "empty string",
			text:     "",
			expected: ContentTypePlain,
		},
		{
			name:     "code block",
			text:     "```go\ncode here\n```",
			expected: ContentTypeCode,
		},
		{
			name:     "inline code",
			text:     "text with `inline code`",
			expected: ContentTypeCode,
		},
		{
			name:     "bold markdown",
			text:     "**bold text**",
			expected: ContentTypeMarkdown,
		},
		{
			name:     "italic markdown",
			text:     "*italic text*",
			expected: ContentTypeMarkdown,
		},
		{
			name:     "link markdown",
			text:     "[link](http://example.com)",
			expected: ContentTypeMarkdown,
		},
		{
			name:     "strikethrough",
			text:     "~~deleted~~",
			expected: ContentTypeMarkdown,
		},
		{
			name:     "underline",
			text:     "__underline__",
			expected: ContentTypeMarkdown,
		},
		{
			name:     "mixed formatting",
			text:     "**bold** and `code`",
			expected: ContentTypeCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectContentType(tt.text)
			if got != tt.expected {
				t.Errorf("DetectContentType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPreprocessMarkdownV2(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text with special chars",
			input:    "Hello *world*!",
			expected: "Hello \\*world\\*\\!",
		},
		{
			name:     "preserve code blocks",
			input:    "```go\nfunc test() *int { return nil }\n```",
			expected: "```go\nfunc test() *int { return nil }\n```",
		},
		{
			name:     "inline code preservation",
			input:    "text with `*code*` inside",
			expected: "text with `*code*` inside",
		},
		{
			name:     "multiple special chars",
			input:    "_*[]()~`>#+-=|{}.!",
			expected: "\\_\\*\\[\\]\\(\\)\\~\\`\\>\\#\\+\\-\\=\\|\\{\\}\\.\\!",
		},
		{
			name:     "mixed content",
			input:    "**bold** and `code`",
			expected: "\\*\\*bold\\*\\* and `code`",
		},
		{
			name:     "multiline with code block",
			input:    "Line 1\n```go\ncode()\n```\nLine 2",
			expected: "Line 1\n```go\ncode()\n```\nLine 2",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "preserve newlines",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PreprocessMarkdownV2(tt.input)
			if got != tt.expected {
				t.Errorf("PreprocessMarkdownV2() =\n%q\nwant\n%q", got, tt.expected)
			}
		})
	}
}

func TestMarkdownToHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "Plain text",
			expected: "Plain text",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "code block",
			input:    "```go\ncode\n```",
			expected: "<pre><code>code</code></pre>",
		},
		{
			name:     "inline code",
			input:    "`code`",
			expected: "<code>code</code>",
		},
		{
			name:     "bold",
			input:    "**bold**",
			expected: "<b>bold</b>",
		},
		{
			name:     "italic",
			input:    "*italic*",
			expected: "<i>italic</i>",
		},
		{
			name:     "strikethrough",
			input:    "~~strikethrough~~",
			expected: "<s>strikethrough</s>",
		},
		{
			name:     "link",
			input:    "[text](url)",
			expected: `<a href="url">text</a>`,
		},
		{
			name:     "mixed formatting",
			input:    "**bold** and *italic*",
			expected: "<b>bold</b> and <i>italic</i>",
		},
		{
			name:     "code block with language",
			input:    "```javascript\nconst x = 1;\n```",
			expected: "<pre><code>const x = 1;</code></pre>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MarkdownToHTML(tt.input)
			if got != tt.expected {
				t.Errorf("MarkdownToHTML() =\n%q\nwant\n%q", got, tt.expected)
			}
		})
	}
}

func TestStripFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "Plain text",
			expected: "Plain text",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "code block",
			input:    "```go\ncode\n```",
			expected: "code",
		},
		{
			name:     "inline code",
			input:    "`code`",
			expected: "code",
		},
		{
			name:     "bold",
			input:    "**bold**",
			expected: "bold",
		},
		{
			name:     "italic",
			input:    "*italic*",
			expected: "italic",
		},
		{
			name:     "strikethrough",
			input:    "~~strikethrough~~",
			expected: "strikethrough",
		},
		{
			name:     "link",
			input:    "[text](url)",
			expected: "text",
		},
		{
			name:     "underline",
			input:    "__underline__",
			expected: "underline",
		},
		{
			name:     "mixed formatting",
			input:    "**bold** and *italic* with `code`",
			expected: "bold and italic with code",
		},
		{
			name:     "complex mixed",
			input:    "Text with **bold**, *italic*, `code`, and [link](http://example.com)",
			expected: "Text with bold, italic, code, and link",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripFormatting(tt.input)
			if got != tt.expected {
				t.Errorf("StripFormatting() =\n%q\nwant\n%q", got, tt.expected)
			}
		})
	}
}

func TestIsMarkdownV2SpecialChar(t *testing.T) {
	tests := []struct {
		name     string
		char     rune
		expected bool
	}{
		{"underscore", '_', true},
		{"asterisk", '*', true},
		{"bracket_open", '[', true},
		{"bracket_close", ']', true},
		{"paren_open", '(', true},
		{"paren_close", ')', true},
		{"tilde", '~', true},
		{"backtick", '`', true},
		{"greater", '>', true},
		{"hash", '#', true},
		{"plus", '+', true},
		{"minus", '-', true},
		{"equal", '=', true},
		{"pipe", '|', true},
		{"brace_open", '{', true},
		{"brace_close", '}', true},
		{"dot", '.', true},
		{"exclamation", '!', true},
		{"regular char", 'a', false},
		{"space", ' ', false},
		{"digit", '1', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMarkdownV2SpecialChar(tt.char)
			if got != tt.expected {
				t.Errorf("isMarkdownV2SpecialChar(%q) = %v, want %v", tt.char, got, tt.expected)
			}
		})
	}
}

func TestHTMLEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "less than",
			input:    "a < b",
			expected: "a &lt; b",
		},
		{
			name:     "greater than",
			input:    "a > b",
			expected: "a &gt; b",
		},
		{
			name:     "ampersand",
			input:    "a & b",
			expected: "a &amp; b",
		},
		{
			name:     "quote",
			input:    `"quote"`,
			expected: "&quot;quote&quot;",
		},
		{
			name:     "apostrophe",
			input:    "it's",
			expected: "it&#39;s",
		},
		{
			name:     "mixed special chars",
			input:    "<tag>value & 'quote'",
			expected: "&lt;tag&gt;value &amp; &#39;quote&#39;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := htmlEscape(tt.input)
			if got != tt.expected {
				t.Errorf("htmlEscape() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestMarkdownV2Integration(t *testing.T) {
	t.Run("complete workflow", func(t *testing.T) {
		markdown := "**Bold** text with `code` and [link](http://example.com)"

		// Detect content type
		contentType := DetectContentType(markdown)
		if contentType != ContentTypeCode {
			t.Errorf("Expected ContentTypeCode, got %v", contentType)
		}

		// Strip formatting for fallback
		stripped := StripFormatting(markdown)
		expectedStripped := "Bold text with code and link"
		if stripped != expectedStripped {
			t.Errorf("StripFormatting() = %q, want %q", stripped, expectedStripped)
		}

		// Convert to HTML
		html := MarkdownToHTML(markdown)
		expectedHTML := "<b>Bold</b> text with <code>code</code> and <a href=\"http://example.com\">link</a>"
		if html != expectedHTML {
			t.Errorf("MarkdownToHTML() = %q, want %q", html, expectedHTML)
		}

		// Preprocess for MarkdownV2
		preprocessed := PreprocessMarkdownV2(markdown)
		expectedPreprocessed := "\\*\\*Bold\\*\\* text with `code` and \\[link\\]\\(http://example\\.com\\)"
		if preprocessed != expectedPreprocessed {
			t.Errorf("PreprocessMarkdownV2() = %q, want %q", preprocessed, expectedPreprocessed)
		}
	})
}

func TestContentTypeEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected ContentType
	}{
		{
			name:     "only asterisks",
			text:     "***",
			expected: ContentTypeMarkdown,
		},
		{
			name:     "escaped markdown",
			text:     "\\*not bold\\*",
			expected: ContentTypePlain,
		},
		{
			name:     "partial markdown",
			text:     "*bold",
			expected: ContentTypeMarkdown,
		},
		{
			name:     "whitespace only",
			text:     "   ",
			expected: ContentTypePlain,
		},
		{
			name:     "newlines only",
			text:     "\n\n",
			expected: ContentTypePlain,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectContentType(tt.text)
			if got != tt.expected {
				t.Errorf("DetectContentType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPreprocessMarkdownV2EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "consecutive backticks",
			input:    "````",
			expected: "````",
		},
		{
			name:     "nested backticks",
			input:    "```code `inside` ```",
			expected: "```code `inside` ```",
		},
		{
			name:     "escaped backtick",
			input:    "\\`",
			expected: "\\`",
		},
		{
			name:     "mixed escaping in code block",
			input:    "```\n*bold* text\n```",
			expected: "```\n*bold* text\n```",
		},
		{
			name:     "special char at start",
			input:    "*start",
			expected: "\\*start",
		},
		{
			name:     "special char at end",
			input:    "end!",
			expected: "end\\!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PreprocessMarkdownV2(tt.input)
			if got != tt.expected {
				t.Errorf("PreprocessMarkdownV2() =\n%q\nwant\n%q", got, tt.expected)
			}
		})
	}
}

func TestMarkdownToHTMLEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unclosed bold",
			input:    "**bold",
			expected: "**bold",
		},
		{
			name:     "empty inline code",
			input:    "``",
			expected: "``",
		},
		{
			name:     "link without url",
			input:    "[text]()",
			expected: `<a href="">text</a>`,
		},
		{
			name:     "multiple consecutive bold",
			input:    "**a****b**",
			expected: "<b>a</b><b>b</b>",
		},
		{
			name:     "nested formatting",
			input:    "**bold with *italic***",
			expected: "<b>bold with <i>italic</b></i>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MarkdownToHTML(tt.input)
			if got != tt.expected {
				t.Errorf("MarkdownToHTML() =\n%q\nwant\n%q", got, tt.expected)
			}
		})
	}
}
