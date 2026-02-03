package common

import "testing"

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "shorter than max",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "equal to max",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "longer than max",
			input:  "hello world",
			maxLen: 8,
			want:   "hello...",
		},
		{
			name:   "very long text",
			input:  "This is a very long string that needs to be truncated",
			maxLen: 20,
			want:   "This is a very lo...",
		},
		{
			name:   "max length 3",
			input:  "hello",
			maxLen: 3,
			want:   "hel",
		},
		{
			name:   "max length 2",
			input:  "hello",
			maxLen: 2,
			want:   "he",
		},
		{
			name:   "max length 1",
			input:  "hello",
			maxLen: 1,
			want:   "h",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "unicode text",
			input:  "Hello 世界",
			maxLen: 8,
			want:   "Hello...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "standard email",
			email: "user@example.com",
			want:  "example.com",
		},
		{
			name:  "nylas inbox email",
			email: "info@qasim.nylas.email",
			want:  "qasim.nylas.email",
		},
		{
			name:  "subdomain email",
			email: "test@subdomain.domain.com",
			want:  "subdomain.domain.com",
		},
		{
			name:  "no @ symbol",
			email: "invalidemail",
			want:  "",
		},
		{
			name:  "multiple @ symbols",
			email: "user@domain@extra.com",
			want:  "",
		},
		{
			name:  "empty string",
			email: "",
			want:  "",
		},
		{
			name:  "only @ symbol",
			email: "@",
			want:  "",
		},
		{
			name:  "@ at start",
			email: "@domain.com",
			want:  "domain.com",
		},
		{
			name:  "@ at end",
			email: "user@",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractDomain(tt.email)
			if got != tt.want {
				t.Errorf("ExtractDomain(%q) = %q, want %q", tt.email, got, tt.want)
			}
		})
	}
}
