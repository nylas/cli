//go:build !integration

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFormat_AllFormats(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected OutputFormat
		hasError bool
	}{
		// Table format variants
		{"table lowercase", "table", FormatTable, false},
		{"table uppercase", "TABLE", FormatTable, false},
		{"table mixed case", "Table", FormatTable, false},
		{"empty defaults to table", "", FormatTable, false},

		// JSON format variants
		{"json lowercase", "json", FormatJSON, false},
		{"json uppercase", "JSON", FormatJSON, false},
		{"json mixed case", "Json", FormatJSON, false},

		// CSV format variants
		{"csv lowercase", "csv", FormatCSV, false},
		{"csv uppercase", "CSV", FormatCSV, false},
		{"csv mixed case", "Csv", FormatCSV, false},

		// YAML format variants
		{"yaml lowercase", "yaml", FormatYAML, false},
		{"yaml uppercase", "YAML", FormatYAML, false},
		{"yml shorthand", "yml", FormatYAML, false},
		{"YML uppercase", "YML", FormatYAML, false},

		// Invalid formats
		{"invalid format", "invalid", "", true},
		{"xml not supported", "xml", "", true},
		{"html not supported", "html", "", true},
		{"spaces not trimmed", " json ", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := ParseFormat(tt.input)

			if tt.hasError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid format")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, format)
			}
		})
	}
}
