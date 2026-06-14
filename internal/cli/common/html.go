package common

import (
	"html"
	"regexp"
	"strings"
)

// blockTagRe matches opening, closing, and self-closing block-level tags —
// with or without attributes — so they can be turned into newlines before the
// generic tag stripper runs. The optional `(?:\s[^>]*)?` consumes any
// attributes, which is what bare-string matching missed: a tag like
// <br class="x"/> or <p style="..."> would otherwise be stripped with no
// separator, silently joining adjacent lines.
//
// Table cell elements (table, td, th, tbody, thead, tfoot) are intentionally
// excluded because they're typically layout; tr is included to separate rows.
// The `\s`/`/`/`>` boundary after the tag name prevents over-matching names
// that merely share a prefix (e.g. <pre> must not match <p>).
var blockTagRe = regexp.MustCompile(`(?i)</?(?:br|p|div|tr|li|h[1-6])(?:\s[^>]*)?/?>`)

// StripHTML removes HTML tags from a string and decodes HTML entities.
func StripHTML(s string) string {
	// Remove style and script tags and their contents
	s = RemoveTagWithContent(s, "style")
	s = RemoveTagWithContent(s, "script")
	s = RemoveTagWithContent(s, "head")

	// Replace block-level elements (including attributed/self-closing forms)
	// with newlines before stripping the remaining tags.
	s = blockTagRe.ReplaceAllString(s, "\n")

	// Strip remaining HTML tags
	var result strings.Builder
	inTag := false

	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}

	// Decode HTML entities (&nbsp;, &lt;, &gt;, etc.)
	text := html.UnescapeString(result.String())

	// Clean up whitespace
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Collapse multiple spaces on the same line
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	// Trim spaces from each line first
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}

	// Remove consecutive empty lines, keeping at most one blank line
	var cleanedLines []string
	prevEmpty := false
	for _, line := range lines {
		isEmpty := line == ""
		if isEmpty && prevEmpty {
			continue // Skip consecutive empty lines
		}
		cleanedLines = append(cleanedLines, line)
		prevEmpty = isEmpty
	}

	text = strings.Join(cleanedLines, "\n")

	// Remove leading/trailing empty lines
	return strings.TrimSpace(text)
}

// RemoveTagWithContent removes a tag and all its content.
func RemoveTagWithContent(s, tag string) string {
	result := s
	for {
		lower := strings.ToLower(result)
		startIdx := strings.Index(lower, "<"+tag)
		if startIdx == -1 {
			break
		}
		endTag := "</" + tag + ">"
		endIdx := strings.Index(lower[startIdx:], endTag)
		if endIdx == -1 {
			// No closing tag, just remove opening tag
			closeIdx := strings.Index(result[startIdx:], ">")
			if closeIdx == -1 {
				break
			}
			result = result[:startIdx] + result[startIdx+closeIdx+1:]
		} else {
			result = result[:startIdx] + result[startIdx+endIdx+len(endTag):]
		}
	}
	return result
}
