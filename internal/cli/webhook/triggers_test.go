package webhook

import (
	"testing"
)

func TestCapitalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single lowercase letter",
			input:    "a",
			expected: "A",
		},
		{
			name:     "lowercase word",
			input:    "message",
			expected: "Message",
		},
		{
			name:     "already capitalized",
			input:    "Message",
			expected: "Message",
		},
		{
			name:     "all uppercase",
			input:    "MESSAGE",
			expected: "MESSAGE",
		},
		{
			name:     "mixed case",
			input:    "mESSAGE",
			expected: "MESSAGE",
		},
		{
			name:     "with numbers",
			input:    "message123",
			expected: "Message123",
		},
		{
			name:     "starts with number",
			input:    "123message",
			expected: "123message",
		},
		{
			name:     "unicode lowercase",
			input:    "über",
			expected: "Über",
		},
		{
			name:     "unicode uppercase start",
			input:    "Über",
			expected: "Über",
		},
		{
			name:     "special characters",
			input:    "_message",
			expected: "_message",
		},
		{
			name:     "space at start",
			input:    " message",
			expected: " message",
		},
		{
			name:     "hyphenated",
			input:    "message-type",
			expected: "Message-type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := capitalize(tt.input)
			if result != tt.expected {
				t.Errorf("capitalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
