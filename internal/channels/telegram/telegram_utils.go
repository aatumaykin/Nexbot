// Package telegram provides utility functions for handling Telegram markdown formatting.
package telegram

import (
	"strings"
	"unicode"
)

// ContentType represents the type of text content.
type ContentType int

const (
	// ContentTypePlain represents plain text without formatting.
	ContentTypePlain ContentType = iota
	// ContentTypeMarkdown represents text with markdown formatting.
	ContentTypeMarkdown
	// ContentTypeCode represents text with code blocks.
	ContentTypeCode
)

// MarkdownV2SpecialChars are characters that need escaping in Telegram MarkdownV2.
var MarkdownV2SpecialChars = []rune{
	'_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!',
}

// DetectContentType determines the content type of the text.
// It checks for code blocks, inline code, and markdown patterns.
func DetectContentType(text string) ContentType {
	// Check for code blocks first (```)
	if containsNonEscaped(text, "```") {
		return ContentTypeCode
	}

	// Check for inline code (`)
	if containsNonEscaped(text, "`") {
		return ContentTypeCode
	}

	// Check for markdown patterns
	markdownPatterns := []string{
		"**", // bold
		"__", // bold (alternative)
		"~~", // strikethrough
	}

	for _, pattern := range markdownPatterns {
		if containsNonEscaped(text, pattern) {
			return ContentTypeMarkdown
		}
	}

	// Check for single char markdown (italic, link, etc.)
	singleCharPatterns := []rune{'*', '_', '[', '~'}
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		// Skip escaped characters
		if r == '\\' && i+1 < len(runes) {
			i++
			continue
		}

		for _, pattern := range singleCharPatterns {
			if r == pattern {
				// For * and _, check if it's part of a pair (bold/underline)
				if r == '*' || r == '_' {
					if i+1 < len(runes) && runes[i+1] == r {
						continue // Skip as it's handled by multi-char patterns
					}
				}
				return ContentTypeMarkdown
			}
		}
	}

	return ContentTypePlain
}

// containsNonEscaped checks if pattern exists in text without being escaped.
func containsNonEscaped(text, pattern string) bool {
	idx := strings.Index(text, pattern)
	for idx != -1 {
		// Check if this pattern is escaped
		backslashes := 0
		for i := idx - 1; i >= 0 && text[i] == '\\'; i-- {
			backslashes++
		}
		// Odd number of backslashes means the pattern is escaped
		if backslashes%2 == 0 {
			return true // Found non-escaped pattern
		}
		// Look for next occurrence
		idx = strings.Index(text[idx+len(pattern):], pattern)
		if idx != -1 {
			idx += len(pattern)
		}
	}
	return false
}

// PreprocessMarkdownV2 preprocesses text for Telegram MarkdownV2 format.
// It escapes special characters while preserving code blocks.
func PreprocessMarkdownV2(text string) string {
	var result strings.Builder
	lines := strings.Split(text, "\n")
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect code block start/end
		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock {
				inCodeBlock = true
				result.WriteString("```")
				// Extract language if present
				lang := strings.TrimPrefix(trimmed, "```")
				if lang != "" {
					result.WriteString(lang)
				}
				result.WriteString("\n")
			} else {
				inCodeBlock = false
				result.WriteString("```\n")
			}
			continue
		}

		if inCodeBlock {
			// Inside code block - no escaping
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		// Outside code block - process inline code and escape
		processedLine := processInlineCodeAndEscape(line)
		result.WriteString(processedLine)
		result.WriteString("\n")
	}

	return strings.TrimSuffix(result.String(), "\n")
}

// processInlineCodeAndEscape processes inline code blocks and escapes special characters.
func processInlineCodeAndEscape(line string) string {
	var result strings.Builder
	runes := []rune(line)
	i := 0

	for i < len(runes) {
		r := runes[i]

		// Handle inline code
		if r == '`' {
			// Check if this is an escaped backtick (preceded by backslash)
			if i > 0 && runes[i-1] == '\\' {
				// Remove one backslash and keep the literal backtick
				result.WriteString("`")
				i++
				continue
			}

			// Look for closing backtick
			nextIdx := i + 1
			for nextIdx < len(runes) && runes[nextIdx] != '`' {
				nextIdx++
			}

			if nextIdx < len(runes) {
				// Found closing backtick - this is inline code, preserve it
				result.WriteRune(r)
				for j := i + 1; j <= nextIdx; j++ {
					result.WriteRune(runes[j])
				}
				i = nextIdx + 1
				continue
			}

			// No closing backtick found - escape this one
			result.WriteString("\\`")
			i++
			continue
		}

		// Handle backslash (escape it if it's not escaping a special char)
		if r == '\\' {
			// Check if next character is a special character that should be escaped
			if i+1 < len(runes) && isMarkdownV2SpecialChar(runes[i+1]) {
				// This is an escape sequence, keep it as is
				result.WriteRune(r)
				i++
				result.WriteRune(runes[i])
				i++
				continue
			}
			// Otherwise, escape the backslash itself
			result.WriteString("\\\\")
			i++
			continue
		}

		// Escape special characters
		if isMarkdownV2SpecialChar(r) {
			result.WriteRune('\\')
		}
		result.WriteRune(r)
		i++
	}

	return result.String()
}

// isMarkdownV2SpecialChar checks if a rune needs escaping in MarkdownV2.
func isMarkdownV2SpecialChar(r rune) bool {
	for _, char := range MarkdownV2SpecialChars {
		if r == char {
			return true
		}
	}
	return false
}

// MarkdownToHTML converts markdown text to HTML format for Telegram.
// It handles bold, italic, code blocks, links, and other common markdown patterns.
func MarkdownToHTML(markdown string) string {
	if markdown == "" {
		return ""
	}

	html := markdown

	// Process code blocks first (```)
	html = processCodeBlocks(html, true)

	// Process inline code (`)
	html = processInlineCode(html, true)

	// Process bold (**text**)
	html = processBold(html, true)

	// Process italic (*text* or _text_)
	html = processItalic(html, true)

	// Process strikethrough (~~text~~)
	html = processStrikethrough(html, true)

	// Process links [text](url)
	html = processLinks(html, true)

	// Process underline (__text__)
	html = processUnderline(html, true)

	return html
}

// StripFormatting removes all markdown formatting from text.
// Useful as a fallback when formatting fails.
func StripFormatting(text string) string {
	if text == "" {
		return ""
	}

	plain := text

	// Process code blocks first (```)
	plain = processCodeBlocks(plain, false)

	// Process inline code (`)
	plain = processInlineCode(plain, false)

	// Process bold (**text**)
	plain = processBold(plain, false)

	// Process italic (*text* or _text_)
	plain = processItalic(plain, false)

	// Process strikethrough (~~text~~)
	plain = processStrikethrough(plain, false)

	// Process links [text](url)
	plain = processLinks(plain, false)

	// Process underline (__text__)
	plain = processUnderline(plain, false)

	return plain
}

// processCodeBlocks processes code blocks (```).
// If htmlMode is true, converts to <pre><code>; otherwise removes formatting.
func processCodeBlocks(text string, htmlMode bool) string {
	var result strings.Builder
	runes := []rune(text)
	i := 0

	for i < len(runes) {
		// Look for code block start
		if i+2 < len(runes) && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
			start := i + 3

			// Skip language identifier and whitespace
			for start < len(runes) && !unicode.IsSpace(runes[start]) {
				start++
			}
			for start < len(runes) && unicode.IsSpace(runes[start]) {
				start++
			}

			// Find code block end
			end := start
			for end < len(runes)-2 {
				if runes[end] == '`' && runes[end+1] == '`' && runes[end+2] == '`' {
					break
				}
				end++
			}

			// Extract code content (skip trailing newline)
			codeContent := string(runes[start:end])
			if len(codeContent) > 0 && codeContent[len(codeContent)-1] == '\n' {
				codeContent = codeContent[:len(codeContent)-1]
			}

			if htmlMode {
				result.WriteString("<pre><code>")
				result.WriteString(htmlEscape(codeContent))
				result.WriteString("</code></pre>")
			} else {
				result.WriteString(codeContent)
			}

			i = end + 3
			continue
		}

		result.WriteRune(runes[i])
		i++
	}

	return result.String()
}

// processInlineCode processes inline code (`).
// If htmlMode is true, converts to <code>; otherwise removes formatting.
func processInlineCode(text string, htmlMode bool) string {
	var result strings.Builder
	inCode := false
	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		if runes[i] == '`' {
			inCode = !inCode
			if htmlMode {
				if inCode {
					// Check if this is an empty inline code (next char is also backtick)
					if i+1 < len(runes) && runes[i+1] == '`' {
						result.WriteString("``")
						i++
						continue
					}
					result.WriteString("<code>")
				} else {
					result.WriteString("</code>")
				}
			}
			continue
		}

		// Always write the character, only add HTML tags if htmlMode is true
		result.WriteRune(runes[i])
	}

	return result.String()
}

// processBold processes bold text (**text** or __text__).
// If htmlMode is true, converts to <b>; otherwise removes formatting.
func processBold(text string, htmlMode bool) string {
	// Process **text** first
	var result1 strings.Builder
	runes := []rune(text)
	i := 0

	for i < len(runes) {
		// Look for ** pattern
		if i+1 < len(runes) && runes[i] == '*' && runes[i+1] == '*' {
			start := i + 2

			// Find closing **
			end := start
			for end < len(runes)-1 {
				if runes[end] == '*' && runes[end+1] == '*' {
					break
				}
				end++
			}

			if end < len(runes)-1 {
				content := string(runes[start:end])
				if htmlMode {
					result1.WriteString("<b>")
					result1.WriteString(content)
					result1.WriteString("</b>")
				} else {
					result1.WriteString(content)
				}
				i = end + 2
				continue
			}
		}

		result1.WriteRune(runes[i])
		i++
	}

	// Process __text__ (alternative bold)
	var result2 strings.Builder
	runes = []rune(result1.String())
	i = 0

	for i < len(runes) {
		// Look for __ pattern
		if i+1 < len(runes) && runes[i] == '_' && runes[i+1] == '_' {
			start := i + 2

			// Find closing __
			end := start
			for end < len(runes)-1 {
				if runes[end] == '_' && runes[end+1] == '_' {
					break
				}
				end++
			}

			if end < len(runes)-1 {
				content := string(runes[start:end])
				if htmlMode {
					result2.WriteString("<b>")
					result2.WriteString(content)
					result2.WriteString("</b>")
				} else {
					result2.WriteString(content)
				}
				i = end + 2
				continue
			}
		}

		result2.WriteRune(runes[i])
		i++
	}

	return result2.String()
}

// processItalic processes italic text (*text* or _text_).
// If htmlMode is true, converts to <i>; otherwise removes formatting.
func processItalic(text string, htmlMode bool) string {
	var result strings.Builder
	runes := []rune(text)
	i := 0

	for i < len(runes) {
		// Look for * or _ pattern (single character)
		if runes[i] == '*' || runes[i] == '_' {
			char := runes[i]
			start := i + 1

			// Find closing * or _
			end := start
			for end < len(runes) {
				if runes[end] == char {
					break
				}
				end++
			}

			if end < len(runes) && start < end {
				content := string(runes[start:end])
				if htmlMode {
					result.WriteString("<i>")
					result.WriteString(content)
					result.WriteString("</i>")
				} else {
					result.WriteString(content)
				}
				i = end + 1
				continue
			}
		}

		result.WriteRune(runes[i])
		i++
	}

	return result.String()
}

// processStrikethrough processes strikethrough text (~~text~~).
// If htmlMode is true, converts to <s>; otherwise removes formatting.
func processStrikethrough(text string, htmlMode bool) string {
	var result strings.Builder
	runes := []rune(text)
	i := 0

	for i < len(runes) {
		// Look for ~~ pattern
		if i+1 < len(runes) && runes[i] == '~' && runes[i+1] == '~' {
			start := i + 2

			// Find closing ~~
			end := start
			for end < len(runes)-1 {
				if runes[end] == '~' && runes[end+1] == '~' {
					break
				}
				end++
			}

			if end < len(runes)-1 {
				content := string(runes[start:end])
				if htmlMode {
					result.WriteString("<s>")
					result.WriteString(content)
					result.WriteString("</s>")
				} else {
					result.WriteString(content)
				}
				i = end + 2
				continue
			}
		}

		result.WriteRune(runes[i])
		i++
	}

	return result.String()
}

// processLinks processes links [text](url).
// If htmlMode is true, converts to <a href="url">text</a>; otherwise returns text only.
func processLinks(text string, htmlMode bool) string {
	var result strings.Builder
	runes := []rune(text)
	i := 0

	for i < len(runes) {
		// Look for [ pattern
		if runes[i] == '[' {
			start := i + 1

			// Find closing ]
			end := start
			for end < len(runes) {
				if runes[end] == ']' {
					break
				}
				end++
			}

			if end < len(runes) && end+1 < len(runes) && runes[end+1] == '(' {
				// Extract link text
				linkText := string(runes[start:end])

				// Extract URL
				urlStart := end + 2
				urlEnd := urlStart
				for urlEnd < len(runes) {
					if runes[urlEnd] == ')' {
						break
					}
					urlEnd++
				}

				if urlEnd < len(runes) {
					url := string(runes[urlStart:urlEnd])
					if htmlMode {
						result.WriteString(`<a href="`)
						result.WriteString(htmlEscape(url))
						result.WriteString(`">`)
						result.WriteString(linkText)
						result.WriteString(`</a>`)
					} else {
						result.WriteString(linkText)
					}
					i = urlEnd + 1
					continue
				}
			}
		}

		result.WriteRune(runes[i])
		i++
	}

	return result.String()
}

// processUnderline processes underline text (__text__).
// If htmlMode is true, converts to <u>; otherwise removes formatting.
func processUnderline(text string, htmlMode bool) string {
	var result strings.Builder
	runes := []rune(text)
	i := 0

	for i < len(runes) {
		// Look for __ pattern
		if i+1 < len(runes) && runes[i] == '_' && runes[i+1] == '_' {
			start := i + 2

			// Find closing __
			end := start
			for end < len(runes)-1 {
				if runes[end] == '_' && runes[end+1] == '_' {
					break
				}
				end++
			}

			if end < len(runes)-1 {
				content := string(runes[start:end])
				if htmlMode {
					result.WriteString("<u>")
					result.WriteString(content)
					result.WriteString("</u>")
				} else {
					result.WriteString(content)
				}
				i = end + 2
				continue
			}
		}

		result.WriteRune(runes[i])
		i++
	}

	return result.String()
}

// htmlEscape escapes HTML special characters.
func htmlEscape(text string) string {
	result := strings.Builder{}

	for _, r := range text {
		switch r {
		case '<':
			result.WriteString("&lt;")
		case '>':
			result.WriteString("&gt;")
		case '&':
			result.WriteString("&amp;")
		case '"':
			result.WriteString("&quot;")
		case '\'':
			result.WriteString("&#39;")
		default:
			result.WriteRune(r)
		}
	}

	return result.String()
}
