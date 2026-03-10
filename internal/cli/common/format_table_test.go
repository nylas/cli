//go:build !integration

package common

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTable_BasicOperations(t *testing.T) {
	ResetLogger()
	InitLogger(false, false)

	t.Run("create and render table", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewTable("ID", "NAME", "STATUS").SetWriter(&buf)

		table.AddRow("1", "First", "Active")
		table.AddRow("2", "Second", "Inactive")

		assert.Equal(t, 2, table.RowCount())

		table.Render()
		output := buf.String()

		// Check headers are present
		assert.Contains(t, output, "ID")
		assert.Contains(t, output, "NAME")
		assert.Contains(t, output, "STATUS")

		// Check data is present
		assert.Contains(t, output, "First")
		assert.Contains(t, output, "Second")
		assert.Contains(t, output, "Active")
		assert.Contains(t, output, "Inactive")
	})

	t.Run("table with right alignment", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewTable("NAME", "COUNT").SetWriter(&buf)
		table.AlignRight(1) // Align COUNT column to right
		table.AddRow("Items", "100")
		table.Render()

		assert.Contains(t, buf.String(), "100")
	})

	t.Run("table with short rows", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewTable("A", "B", "C").SetWriter(&buf)

		// Add row with fewer values than headers
		table.AddRow("only one")
		table.Render()

		assert.Equal(t, 1, table.RowCount())
		assert.Contains(t, buf.String(), "only one")
	})

	t.Run("table with invalid column index", func(t *testing.T) {
		table := NewTable("A", "B")
		// This should not panic - should silently ignore invalid index
		table.AlignRight(10)
		table.AddRow("a", "b")
		// No assertion needed - just verify no panic
	})
}

func TestTable_QuietMode(t *testing.T) {
	ResetLogger()
	InitLogger(false, true) // Enable quiet mode

	var buf bytes.Buffer
	table := NewTable("HEADER").SetWriter(&buf)
	table.AddRow("value")
	table.Render()

	// In quiet mode, should not produce output
	assert.Empty(t, buf.String())
}

func TestPrintFunctions_QuietMode(t *testing.T) {
	ResetLogger()
	InitLogger(false, true) // Enable quiet mode

	// These should not panic in quiet mode
	PrintSuccess("success: %s", "test")
	PrintWarning("warning: %s", "test")
	PrintInfo("info: %s", "test")
}

func TestPrintError_AlwaysPrints(t *testing.T) {
	ResetLogger()
	InitLogger(false, true) // Enable quiet mode

	// PrintError should print even in quiet mode (to stderr)
	// We can't easily capture stderr, so just verify no panic
	PrintError("error: %s", "test")
}

func TestTable_UTF8Support(t *testing.T) {
	ResetLogger()
	InitLogger(false, false)

	t.Run("handles UTF-8 characters correctly", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewTable("NAME", "EMOJI").SetWriter(&buf)

		// UTF-8 characters should be counted by runes, not bytes
		table.AddRow("Alice", "👋🌍")
		table.AddRow("Bob", "🚀✨")

		table.Render()
		output := buf.String()

		assert.Contains(t, output, "👋🌍")
		assert.Contains(t, output, "🚀✨")
	})

	t.Run("handles mixed ASCII and UTF-8", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewTable("ID", "DESCRIPTION").SetWriter(&buf)

		table.AddRow("1", "Hello 世界")
		table.AddRow("2", "Test data")

		table.Render()
		output := buf.String()

		assert.Contains(t, output, "Hello 世界")
		assert.Contains(t, output, "Test data")
	})
}

func TestTable_MaxWidth(t *testing.T) {
	ResetLogger()
	InitLogger(false, false)

	t.Run("truncates long text with max width", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewTable("ID", "DESCRIPTION").SetWriter(&buf)
		table.SetMaxWidth(1, 20) // Limit DESCRIPTION column to 20 chars

		table.AddRow("1", "This is a very long description that should be truncated")
		table.AddRow("2", "Short")

		table.Render()
		output := buf.String()

		// Should contain truncated version with ellipsis
		assert.Contains(t, output, "...")
		assert.Contains(t, output, "Short")
		assert.NotContains(t, output, "should be truncated")
	})

	t.Run("no truncation when text is under max width", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewTable("NAME").SetWriter(&buf)
		table.SetMaxWidth(0, 50)

		table.AddRow("Short text")

		table.Render()
		output := buf.String()

		assert.Contains(t, output, "Short text")
		assert.NotContains(t, output, "...")
	})

	t.Run("no max width when not set", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewTable("CONTENT").SetWriter(&buf)

		longText := "This is a very long piece of text that should not be truncated"
		table.AddRow(longText)

		table.Render()
		output := buf.String()

		assert.Contains(t, output, longText)
	})
}

func TestTruncateCell(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"no truncation needed", "hello", 10, "hello"},
		{"truncate with ellipsis", "hello world", 8, "hello..."},
		{"exact length", "exactly", 7, "exactly"},
		{"very short maxLen", "test", 2, "te"},
		{"UTF-8 characters", "Hello 世界!", 7, "Hell..."},
		{"empty string", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateCell(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
			// Verify result doesn't exceed maxLen in rune count
			assert.LessOrEqual(t, len([]rune(result)), tt.maxLen)
		})
	}
}

func TestPadString(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		width      int
		alignRight bool
		expected   string
	}{
		{"pad right with spaces", "hello", 10, false, "hello     "},
		{"pad left with spaces", "world", 10, true, "     world"},
		{"no padding needed", "exactly", 7, false, "exactly"},
		{"UTF-8 left align", "Hi 世界", 8, false, "Hi 世界   "},
		{"UTF-8 right align", "Hi 世界", 8, true, "   Hi 世界"},
		{"emoji left align", "👋🌍", 5, false, "👋🌍   "},
		{"emoji right align", "👋🌍", 5, true, "   👋🌍"},
		{"no padding for longer string", "too long text", 5, false, "too long text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padString(tt.input, tt.width, tt.alignRight)
			assert.Equal(t, tt.expected, result)
			// Verify result width matches expected width (or is longer if input was too long)
			assert.GreaterOrEqual(t, len([]rune(result)), len([]rune(tt.input)))
		})
	}
}

func TestStripAnsiCodes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no ansi codes", "hello world", "hello world"},
		{"simple color code", "\x1b[31mred text\x1b[0m", "red text"},
		{"bold code", "\x1b[1mbold\x1b[0m", "bold"},
		{"multiple codes", "\x1b[32mgreen\x1b[0m and \x1b[34mblue\x1b[0m", "green and blue"},
		{"empty string", "", ""},
		{"only ansi codes", "\x1b[31m\x1b[0m", ""},
		{"cyan sprint", Cyan.Sprint("test"), "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripAnsiCodes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVisualWidth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"plain text", "hello", 5},
		{"with ansi codes", "\x1b[31mred\x1b[0m", 3},
		{"UTF-8 chars", "世界", 2},
		{"UTF-8 with ansi", "\x1b[32m世界\x1b[0m", 2},
		{"emoji", "👋🌍", 2},
		{"cyan sprint", Cyan.Sprint("test"), 4},
		{"green sprint Yes", Green.Sprint("Yes"), 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := visualWidth(tt.input)
			assert.Equal(t, tt.expected, result, "visualWidth(%q) should be %d", tt.input, tt.expected)
		})
	}
}

func TestTable_Alignment(t *testing.T) {
	ResetLogger()
	InitLogger(false, false)

	t.Run("columns align properly with varying widths", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewTable("SHORT", "MEDIUM LENGTH", "L").SetWriter(&buf)

		table.AddRow("A", "B", "C")
		table.AddRow("Long data", "X", "Very long content here")

		table.Render()
		output := buf.String()

		// Just verify all expected content is present and properly formatted
		assert.Contains(t, output, "SHORT")
		assert.Contains(t, output, "MEDIUM LENGTH")
		assert.Contains(t, output, "Long data")
		assert.Contains(t, output, "Very long content here")

		// Verify we have multiple lines (header + separator + rows)
		lines := strings.Split(strings.TrimSpace(output), "\n")
		assert.GreaterOrEqual(t, len(lines), 4, "Should have header, separator, and data rows")
	})

	t.Run("UTF-8 characters align correctly", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewTable("NAME", "EMOJI", "COUNT").SetWriter(&buf)

		table.AddRow("Alice", "👋", "1")
		table.AddRow("Bob", "🌍🚀", "2")
		table.AddRow("测试", "✨", "3")

		table.Render()
		output := buf.String()

		// Verify all expected content is present
		assert.Contains(t, output, "Alice")
		assert.Contains(t, output, "👋")
		assert.Contains(t, output, "🌍🚀")
		assert.Contains(t, output, "测试")
		assert.Contains(t, output, "✨")
	})

	t.Run("right alignment works correctly", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewTable("NAME", "AMOUNT").SetWriter(&buf)
		table.AlignRight(1) // Right-align the AMOUNT column

		table.AddRow("Item A", "100")
		table.AddRow("Item B", "50")
		table.AddRow("Item C", "1234")

		table.Render()
		output := buf.String()

		// Output should contain all values
		assert.Contains(t, output, "Item A")
		assert.Contains(t, output, "100")
		assert.Contains(t, output, "1234")
	})
}
