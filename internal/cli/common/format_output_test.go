//go:build !integration

package common

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatter_JSON_Output(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		contains []string
	}{
		{
			name:     "simple map",
			data:     map[string]string{"key": "value"},
			contains: []string{`"key"`, `"value"`},
		},
		{
			name:     "slice of maps",
			data:     []map[string]int{{"a": 1}, {"b": 2}},
			contains: []string{`"a"`, `"b"`, "1", "2"},
		},
		{
			name: "struct",
			data: struct {
				Name  string `json:"name"`
				Count int    `json:"count"`
			}{Name: "test", Count: 42},
			contains: []string{`"name"`, `"test"`, `"count"`, "42"},
		},
		{
			name:     "nested struct",
			data:     map[string]any{"outer": map[string]string{"inner": "value"}},
			contains: []string{`"outer"`, `"inner"`, `"value"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(FormatJSON).SetWriter(&buf)

			err := formatter.Format(tt.data)
			require.NoError(t, err)

			output := buf.String()
			for _, s := range tt.contains {
				assert.Contains(t, output, s)
			}
		})
	}
}

func TestFormatter_YAML_Output(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		contains []string
	}{
		{
			name:     "simple map",
			data:     map[string]string{"key": "value"},
			contains: []string{"key:", "value"},
		},
		{
			name:     "multiple fields",
			data:     map[string]int{"count": 10, "total": 100},
			contains: []string{"count:", "10", "total:", "100"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(FormatYAML).SetWriter(&buf)

			err := formatter.Format(tt.data)
			require.NoError(t, err)

			output := buf.String()
			for _, s := range tt.contains {
				assert.Contains(t, output, s)
			}
		})
	}
}

func TestFormatter_CSV_Slice(t *testing.T) {
	type Item struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
		Tag   string `json:"tag"`
	}

	tests := []struct {
		name     string
		data     []Item
		contains []string
	}{
		{
			name: "multiple items",
			data: []Item{
				{Name: "item1", Value: 1, Tag: "a"},
				{Name: "item2", Value: 2, Tag: "b"},
			},
			contains: []string{"name", "value", "tag", "item1", "item2", "1", "2", "a", "b"},
		},
		{
			name:     "single item",
			data:     []Item{{Name: "only", Value: 99, Tag: "x"}},
			contains: []string{"name", "value", "only", "99", "x"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(FormatCSV).SetWriter(&buf)

			err := formatter.Format(tt.data)
			require.NoError(t, err)

			output := buf.String()
			for _, s := range tt.contains {
				assert.Contains(t, output, s)
			}
		})
	}
}

func TestFormatter_CSV_EmptySlice(t *testing.T) {
	type Item struct {
		Name string `json:"name"`
	}

	var buf bytes.Buffer
	formatter := NewFormatter(FormatCSV).SetWriter(&buf)

	err := formatter.Format([]Item{})
	require.NoError(t, err)

	// Empty slice should produce no output
	assert.Empty(t, buf.String())
}

func TestFormatter_CSV_SingleItem(t *testing.T) {
	type Item struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	var buf bytes.Buffer
	formatter := NewFormatter(FormatCSV).SetWriter(&buf)

	// Test single item (not in slice)
	err := formatter.Format(Item{ID: "123", Name: "test"})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "id")
	assert.Contains(t, output, "name")
	assert.Contains(t, output, "123")
	assert.Contains(t, output, "test")
}

func TestFormatter_CSV_NonStructTypes(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewFormatter(FormatCSV).SetWriter(&buf)

	// Non-struct types should fall back to "value" header
	err := formatter.Format("simple string")
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "value")
	assert.Contains(t, output, "simple string")
}

func TestGetCSVHeaders(t *testing.T) {
	type TestStruct struct {
		Public     string `json:"public_field"`
		NoTag      string
		SkipField  string `json:"-"`
		unexported string //nolint:unused
	}

	tests := []struct {
		name     string
		data     any
		expected []string
	}{
		{
			name:     "struct with json tags",
			data:     TestStruct{Public: "val", NoTag: "val2"},
			expected: []string{"public_field", "NoTag"},
		},
		{
			name:     "non-struct returns value",
			data:     "string",
			expected: []string{"value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need to use reflection to test this internal function
			// Test through Format instead
			var buf bytes.Buffer
			formatter := NewFormatter(FormatCSV).SetWriter(&buf)

			switch v := tt.data.(type) {
			case TestStruct:
				err := formatter.Format(v)
				require.NoError(t, err)
				output := buf.String()
				for _, exp := range tt.expected {
					assert.Contains(t, output, exp)
				}
			case string:
				err := formatter.Format(v)
				require.NoError(t, err)
				output := buf.String()
				assert.Contains(t, output, "value")
			}
		})
	}
}

func TestFormatValue_SpecialTypes(t *testing.T) {
	type ItemWithSlice struct {
		Tags []string `json:"tags"`
	}

	tests := []struct {
		name     string
		data     any
		contains string
	}{
		{
			name:     "slice field",
			data:     []ItemWithSlice{{Tags: []string{"a", "b", "c"}}},
			contains: "a; b; c",
		},
		{
			name:     "empty slice field",
			data:     []ItemWithSlice{{Tags: []string{}}},
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(FormatCSV).SetWriter(&buf)

			err := formatter.Format(tt.data)
			require.NoError(t, err)

			output := buf.String()
			if tt.contains != "" {
				assert.Contains(t, output, tt.contains)
			}
		})
	}
}
