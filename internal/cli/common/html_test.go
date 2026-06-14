package common

import "testing"

// TestStripHTML_BlockTagsWithAttributes pins the fix for block/void tags that
// carry attributes. The original implementation only converted *bare* block
// tags (<p>, <br/>) to newlines; a tag with attributes (<br class="x"/>,
// <p style="...">) fell through to the generic tag stripper and was removed
// with no separator, silently joining adjacent lines (e.g. "Line1Line2").
//
// Real-world HTML email (Gmail, Outlook) almost always emits attributed tags,
// so this is the common case, not an edge case.
func TestStripHTML_BlockTagsWithAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "self-closing br with attributes still breaks the line",
			input:    `Line 1<br class="gmail_br"/>Line 2`,
			expected: "Line 1\nLine 2",
		},
		{
			name:     "void br with attributes and no slash",
			input:    `Line 1<br data-x="y">Line 2`,
			expected: "Line 1\nLine 2",
		},
		{
			name:     "paragraphs with attributes separate like bare paragraphs",
			input:    `<p class="MsoNormal">Para 1</p><p class="MsoNormal">Para 2</p>`,
			expected: "Para 1\n\nPara 2",
		},
		{
			name:     "div with style attribute separates blocks",
			input:    `<div style="color:red">Block 1</div><div style="color:blue">Block 2</div>`,
			expected: "Block 1\n\nBlock 2",
		},
		{
			name:     "uppercase tag with attributes",
			input:    `Line 1<BR CLASS="x"/>Line 2`,
			expected: "Line 1\nLine 2",
		},
		{
			name:     "list items with attributes",
			input:    `<ul><li class="a">Item 1</li><li class="a">Item 2</li></ul>`,
			expected: "Item 1\n\nItem 2",
		},
		{
			// Guard against over-matching: a non-block tag whose name merely
			// starts with a block tag's letter (pre vs p) must NOT be treated
			// as a block separator.
			name:     "non-block tag sharing a prefix is not a block separator",
			input:    `<pre>code stays inline</pre>`,
			expected: "code stays inline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripHTML(tt.input); got != tt.expected {
				t.Errorf("StripHTML(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
