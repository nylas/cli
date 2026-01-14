//go:build !integration

package email

import (
	"testing"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/stretchr/testify/assert"
)

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "simple html",
			input:    "<p>Hello</p>",
			expected: "Hello",
		},
		{
			name:     "nested html",
			input:    "<div><p>Hello</p></div>",
			expected: "Hello",
		},
		{
			name:     "br tags",
			input:    "Line 1<br>Line 2<br/>Line 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "paragraph tags",
			input:    "<p>Para 1</p><p>Para 2</p>",
			expected: "Para 1\n\nPara 2",
		},
		{
			name:     "html entities",
			input:    "Hello &amp; World",
			expected: "Hello & World",
		},
		{
			name:     "nbsp entity",
			input:    "Hello&nbsp;World",
			expected: "Hello\u00a0World", // Non-breaking space character
		},
		{
			name:     "lt gt entities",
			input:    "&lt;tag&gt;",
			expected: "<tag>",
		},
		{
			name:     "style tag removal",
			input:    "<style>.class { color: red; }</style>Content",
			expected: "Content",
		},
		{
			name:     "script tag removal",
			input:    "<script>alert('hello');</script>Content",
			expected: "Content",
		},
		{
			name:     "head tag removal",
			input:    "<head><title>Title</title></head>Content",
			expected: "Content",
		},
		{
			name:     "multiple spaces collapsed",
			input:    "Hello    World",
			expected: "Hello World",
		},
		{
			name:     "multiple newlines collapsed",
			input:    "Line 1\n\n\n\nLine 2",
			expected: "Line 1\n\nLine 2",
		},
		{
			name:     "div tags add newlines",
			input:    "<div>Block 1</div><div>Block 2</div>",
			expected: "Block 1\n\nBlock 2",
		},
		{
			name:     "list items",
			input:    "<ul><li>Item 1</li><li>Item 2</li></ul>",
			expected: "Item 1\n\nItem 2",
		},
		{
			name:     "headings",
			input:    "<h1>Title</h1><h2>Subtitle</h2>",
			expected: "Title\n\nSubtitle",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   ",
			expected: "",
		},
		{
			name:     "complex html email",
			input:    `<html><head><style>.x{}</style></head><body><p>Hello,</p><p>This is a test.</p><br><p>Thanks</p></body></html>`,
			expected: "Hello,\n\nThis is a test.\n\nThanks",
		},
		{
			name:     "uppercase tags",
			input:    "<P>Hello</P><BR>World",
			expected: "Hello\n\nWorld",
		},
		{
			name:     "tr tags",
			input:    "<table><tr><td>Cell 1</td></tr><tr><td>Cell 2</td></tr></table>",
			expected: "Cell 1\n\nCell 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.StripHTML(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveTagWithContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		tag      string
		expected string
	}{
		{
			name:     "remove style",
			input:    "before<style>css</style>after",
			tag:      "style",
			expected: "beforeafter",
		},
		{
			name:     "remove script",
			input:    "before<script>js</script>after",
			tag:      "script",
			expected: "beforeafter",
		},
		{
			name:     "no matching tag",
			input:    "no tags here",
			tag:      "style",
			expected: "no tags here",
		},
		{
			name:     "multiple tags",
			input:    "<style>a</style>middle<style>b</style>",
			tag:      "style",
			expected: "middle",
		},
		{
			name:     "case insensitive",
			input:    "<STYLE>css</STYLE>content",
			tag:      "style",
			expected: "content",
		},
		{
			name:     "unclosed tag",
			input:    "<style>css",
			tag:      "style",
			expected: "css",
		},
		{
			name:     "empty tag",
			input:    "<style></style>content",
			tag:      "style",
			expected: "content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.RemoveTagWithContent(tt.input, tt.tag)
			assert.Equal(t, tt.expected, result)
		})
	}
}
