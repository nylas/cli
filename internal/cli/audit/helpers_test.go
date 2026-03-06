package audit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// FormatDuration Tests
// =============================================================================

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{
			name:     "sub-millisecond microseconds",
			input:    500 * time.Microsecond,
			expected: "500µs",
		},
		{
			name:     "sub-millisecond nanoseconds",
			input:    100 * time.Nanosecond,
			expected: "100ns",
		},
		{
			name:     "exactly one millisecond",
			input:    time.Millisecond,
			expected: "1ms",
		},
		{
			name:     "milliseconds range",
			input:    250 * time.Millisecond,
			expected: "250ms",
		},
		{
			name:     "milliseconds with sub-ms component rounds to ms",
			input:    250*time.Millisecond + 123*time.Microsecond,
			expected: "250ms",
		},
		{
			name:  "exactly one second",
			input: time.Second,
			// Rounds to 10ms precision: 1s
			expected: "1s",
		},
		{
			name:     "seconds range",
			input:    3 * time.Second,
			expected: "3s",
		},
		{
			name:     "seconds with ms component rounds to 10ms",
			input:    3*time.Second + 456*time.Millisecond,
			expected: "3.46s",
		},
		{
			name:  "zero duration",
			input: 0,
			// Zero is < time.Millisecond → falls through to d.String()
			expected: "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// =============================================================================
// FormatSize Tests
// =============================================================================

func TestFormatSize(t *testing.T) {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		// Bytes range
		{name: "zero bytes", input: 0, expected: "0 B"},
		{name: "one byte", input: 1, expected: "1 B"},
		{name: "512 bytes", input: 512, expected: "512 B"},
		{name: "1023 bytes", input: 1023, expected: "1023 B"},
		// KB range
		{name: "exactly 1 KB", input: KB, expected: "1 KB"},
		{name: "1.5 KB", input: KB + KB/2, expected: "1.5 KB"},
		{name: "exactly 1023 KB", input: 1023 * KB, expected: "1023 KB"},
		// MB range
		{name: "exactly 1 MB", input: MB, expected: "1 MB"},
		{name: "1.5 MB", input: MB + MB/2, expected: "1.5 MB"},
		{name: "10 MB", input: 10 * MB, expected: "10 MB"},
		{name: "exactly 1023 MB", input: 1023 * MB, expected: "1023 MB"},
		// GB range
		{name: "exactly 1 GB", input: GB, expected: "1 GB"},
		{name: "1.5 GB", input: GB + GB/2, expected: "1.5 GB"},
		{name: "10 GB", input: 10 * GB, expected: "10 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSize(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// =============================================================================
// parseBool Tests
// =============================================================================

func TestParseBool(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Truthy values
		{name: "true lowercase", input: "true", expected: true},
		{name: "true uppercase", input: "TRUE", expected: true},
		{name: "true mixed case", input: "True", expected: true},
		{name: "yes lowercase", input: "yes", expected: true},
		{name: "yes uppercase", input: "YES", expected: true},
		{name: "1", input: "1", expected: true},
		{name: "on lowercase", input: "on", expected: true},
		{name: "on uppercase", input: "ON", expected: true},
		// Falsy values
		{name: "false", input: "false", expected: false},
		{name: "no", input: "no", expected: false},
		{name: "0", input: "0", expected: false},
		{name: "off", input: "off", expected: false},
		{name: "empty string", input: "", expected: false},
		{name: "random string", input: "maybe", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBool(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// =============================================================================
// parseDate Tests
// =============================================================================

func TestParseDate(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expectYear  int
		expectMonth time.Month
		expectDay   int
	}{
		{
			name:        "YYYY-MM-DD",
			input:       "2024-01-15",
			expectError: false,
			expectYear:  2024,
			expectMonth: time.January,
			expectDay:   15,
		},
		{
			name:        "YYYY-MM-DDTHH:MM:SS",
			input:       "2024-06-20T14:30:00",
			expectError: false,
			expectYear:  2024,
			expectMonth: time.June,
			expectDay:   20,
		},
		{
			name:        "YYYY-MM-DD HH:MM:SS",
			input:       "2024-12-31 23:59:59",
			expectError: false,
			expectYear:  2024,
			expectMonth: time.December,
			expectDay:   31,
		},
		{
			name:        "RFC3339",
			input:       "2024-03-01T00:00:00Z",
			expectError: false,
			expectYear:  2024,
			expectMonth: time.March,
			expectDay:   1,
		},
		{
			name:        "invalid format",
			input:       "01/15/2024",
			expectError: true,
		},
		{
			name:        "garbage string",
			input:       "not-a-date",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDate(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unrecognized date format")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectYear, got.Year())
				assert.Equal(t, tt.expectMonth, got.Month())
				assert.Equal(t, tt.expectDay, got.Day())
			}
		})
	}
}

// =============================================================================
// orDash Tests
// =============================================================================

func TestOrDash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty string returns dash", input: "", expected: "-"},
		{name: "non-empty string returns itself", input: "alice", expected: "alice"},
		{name: "whitespace string is non-empty", input: " ", expected: " "},
		{name: "dash string returns dash", input: "-", expected: "-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := orDash(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// =============================================================================
// yesNo Tests
// =============================================================================

func TestYesNo(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected string
	}{
		{name: "true returns Yes", input: true, expected: "Yes"},
		{name: "false returns No", input: false, expected: "No"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := yesNo(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// =============================================================================
// enabledDisabled Tests
// =============================================================================

func TestEnabledDisabled(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected string
	}{
		{name: "true returns enabled", input: true, expected: "enabled"},
		{name: "false returns disabled", input: false, expected: "disabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enabledDisabled(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// =============================================================================
// enabledSuffix Tests
// =============================================================================

func TestEnabledSuffix(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected string
	}{
		{name: "true returns ' and enabled'", input: true, expected: " and enabled"},
		{name: "false returns empty string", input: false, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enabledSuffix(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
