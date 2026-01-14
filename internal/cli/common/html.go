package common

import (
	"html"
	"strings"
)

// StripHTML removes HTML tags from a string and decodes HTML entities.
func StripHTML(s string) string {
	// Remove style and script tags and their contents
	s = RemoveTagWithContent(s, "style")
	s = RemoveTagWithContent(s, "script")
	s = RemoveTagWithContent(s, "head")

	// Replace block-level elements with newlines before stripping tags
	blockTags := []string{"br", "p", "div", "tr", "li", "h1", "h2", "h3", "h4", "h5", "h6"}
	for _, tag := range blockTags {
		// Handle <br>, <br/>, <br />
		s = strings.ReplaceAll(s, "<"+tag+">", "\n")
		s = strings.ReplaceAll(s, "<"+tag+"/>", "\n")
		s = strings.ReplaceAll(s, "<"+tag+" />", "\n")
		s = strings.ReplaceAll(s, "</"+tag+">", "\n")
		// Case insensitive
		s = strings.ReplaceAll(s, "<"+strings.ToUpper(tag)+">", "\n")
		s = strings.ReplaceAll(s, "</"+strings.ToUpper(tag)+">", "\n")
	}

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

	// Collapse multiple newlines
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	// Trim spaces from each line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	text = strings.Join(lines, "\n")

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
