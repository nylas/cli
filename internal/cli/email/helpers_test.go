//go:build !integration

package email

import (
	"testing"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatParticipant(t *testing.T) {
	tests := []struct {
		name     string
		contact  domain.EmailParticipant
		expected string
	}{
		{
			name:     "with name and email",
			contact:  domain.EmailParticipant{Name: "John Doe", Email: "john@example.com"},
			expected: "John Doe",
		},
		{
			name:     "email only",
			contact:  domain.EmailParticipant{Email: "john@example.com"},
			expected: "john@example.com",
		},
		{
			name:     "name only",
			contact:  domain.EmailParticipant{Name: "John Doe"},
			expected: "John Doe",
		},
		{
			name:     "empty contact",
			contact:  domain.EmailParticipant{},
			expected: "",
		},
		{
			name:     "empty name returns email",
			contact:  domain.EmailParticipant{Name: "", Email: "test@test.com"},
			expected: "test@test.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.FormatParticipant(tt.contact)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatParticipants(t *testing.T) {
	tests := []struct {
		name     string
		contacts []domain.EmailParticipant
		expected string
	}{
		{
			name: "multiple contacts with names",
			contacts: []domain.EmailParticipant{
				{Name: "Alice", Email: "alice@example.com"},
				{Name: "Bob", Email: "bob@example.com"},
			},
			expected: "Alice, Bob",
		},
		{
			name: "mixed names and emails",
			contacts: []domain.EmailParticipant{
				{Name: "Alice"},
				{Email: "bob@example.com"},
			},
			expected: "Alice, bob@example.com",
		},
		{
			name: "single contact",
			contacts: []domain.EmailParticipant{
				{Name: "Alice", Email: "alice@example.com"},
			},
			expected: "Alice",
		},
		{
			name:     "empty list",
			contacts: []domain.EmailParticipant{},
			expected: "",
		},
		{
			name:     "nil list",
			contacts: nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.FormatParticipants(tt.contacts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"bytes under 1KB", 500, "500 B"},
		{"exactly 1KB", 1024, "1.0 KB"},
		{"kilobytes", 2048, "2.0 KB"},
		{"kilobytes with decimal", 1536, "1.5 KB"},
		{"megabytes", 1048576, "1.0 MB"},
		{"megabytes with decimal", 2621440, "2.5 MB"},
		{"gigabytes", 1073741824, "1.0 GB"},
		{"large file", 5368709120, "5.0 GB"},
		{"terabytes", 1099511627776, "1.0 TB"},
		{"small value", 1, "1 B"},
		{"1023 bytes", 1023, "1023 B"},
		{"1025 bytes", 1025, "1.0 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.FormatSize(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// HTML stripping tests are in helpers_html_test.go

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item present",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: true,
		},
		{
			name:     "item not present",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "a",
			expected: false,
		},
		{
			name:     "nil slice",
			slice:    nil,
			item:     "a",
			expected: false,
		},
		{
			name:     "first item",
			slice:    []string{"first", "second"},
			item:     "first",
			expected: true,
		},
		{
			name:     "last item",
			slice:    []string{"first", "last"},
			item:     "last",
			expected: true,
		},
		{
			name:     "case sensitive",
			slice:    []string{"Hello"},
			item:     "hello",
			expected: false,
		},
		{
			name:     "empty string in slice",
			slice:    []string{"", "a"},
			item:     "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPrintMetadataInfo(t *testing.T) {
	// Just verify it doesn't panic
	printMetadataInfo()
}

func TestParseEmails(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single email",
			input:    "test@example.com",
			expected: []string{"test@example.com"},
		},
		{
			name:     "multiple emails comma separated",
			input:    "a@b.com, c@d.com",
			expected: []string{"a@b.com", "c@d.com"},
		},
		{
			name:     "emails with extra spaces",
			input:    "  spaced@test.com  ,  other@test.com  ",
			expected: []string{"spaced@test.com", "other@test.com"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseEmails(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseContacts(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []domain.EmailParticipant
		wantErr  bool
	}{
		{
			name:  "simple email",
			input: []string{"test@example.com"},
			expected: []domain.EmailParticipant{
				{Email: "test@example.com"},
			},
		},
		{
			name:  "email with name",
			input: []string{"John Doe <john@example.com>"},
			expected: []domain.EmailParticipant{
				{Name: "John Doe", Email: "john@example.com"},
			},
		},
		{
			name:  "mixed formats",
			input: []string{"plain@test.com", "Named <named@test.com>"},
			expected: []domain.EmailParticipant{
				{Email: "plain@test.com"},
				{Name: "Named", Email: "named@test.com"},
			},
		},
		{
			name:    "invalid email",
			input:   []string{"invalid-email"},
			wantErr: true,
		},
		{
			name:    "empty email",
			input:   []string{""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseContacts(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid date", "2024-01-15", false},
		{"valid date 2", "2025-12-31", false},
		{"invalid format", "01-15-2024", true},
		{"invalid format 2", "2024/01/15", true},
		{"invalid string", "invalid", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDate(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotZero(t, result)
			}
		})
	}
}
