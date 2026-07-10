package common

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nylas/cli/internal/domain"
)

// Table provides a simple table builder.
type Table struct {
	headers    []string
	rows       [][]string
	alignRight []bool
	maxWidths  []int // max width per column (0 = no limit)
	writer     io.Writer
}

// NewTable creates a new table with headers.
func NewTable(headers ...string) *Table {
	return &Table{
		headers:    headers,
		rows:       make([][]string, 0),
		alignRight: make([]bool, len(headers)),
		maxWidths:  make([]int, len(headers)),
		writer:     os.Stdout,
	}
}

// SetWriter sets the output writer.
func (t *Table) SetWriter(w io.Writer) *Table {
	t.writer = w
	return t
}

// AlignRight sets right alignment for a column.
func (t *Table) AlignRight(col int) *Table {
	if col < len(t.alignRight) {
		t.alignRight[col] = true
	}
	return t
}

// SetMaxWidth sets the maximum width for a column (truncates with ellipsis if exceeded).
func (t *Table) SetMaxWidth(col int, maxWidth int) *Table {
	if col < len(t.maxWidths) {
		t.maxWidths[col] = maxWidth
	}
	return t
}

// AddRow adds a row to the table.
func (t *Table) AddRow(values ...string) *Table {
	// Pad row to match headers
	row := make([]string, len(t.headers))
	for i := 0; i < len(t.headers) && i < len(values); i++ {
		row[i] = values[i]
	}
	t.rows = append(t.rows, row)
	return t
}

// Render renders the table to the writer.
func (t *Table) Render() {
	if IsQuiet() {
		return
	}

	// Calculate column widths using visual width (excludes ANSI codes)
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = visualWidth(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			cellWidth := visualWidth(cell)
			if cellWidth > widths[i] {
				widths[i] = cellWidth
			}
		}
	}

	// Apply max width limits
	for i := range widths {
		if t.maxWidths[i] > 0 && widths[i] > t.maxWidths[i] {
			widths[i] = t.maxWidths[i]
		}
	}

	// Print headers
	for i, h := range t.headers {
		padded := padString(h, widths[i], t.alignRight[i])
		_, _ = BoldCyan.Fprint(t.writer, padded)
		_, _ = fmt.Fprint(t.writer, "  ")
	}
	_, _ = fmt.Fprintln(t.writer)

	// Print separator
	for i, w := range widths {
		_, _ = Dim.Fprint(t.writer, strings.Repeat("─", w))
		if i < len(widths)-1 {
			_, _ = Dim.Fprint(t.writer, "──")
		}
	}
	_, _ = fmt.Fprintln(t.writer)

	// Print rows
	for _, row := range t.rows {
		for i, cell := range row {
			// Truncate cell if it exceeds column width
			displayCell := cell
			if t.maxWidths[i] > 0 && visualWidth(cell) > widths[i] {
				displayCell = truncateCell(cell, widths[i])
			}

			padded := padString(displayCell, widths[i], t.alignRight[i])
			_, _ = fmt.Fprint(t.writer, padded)
			_, _ = fmt.Fprint(t.writer, "  ")
		}
		_, _ = fmt.Fprintln(t.writer)
	}
}

// RowCount returns the number of rows.
func (t *Table) RowCount() int {
	return len(t.rows)
}

// PrintSuccess prints a success message.
func PrintSuccess(format string, args ...any) {
	if IsQuiet() {
		return
	}
	_, _ = Green.Printf("✓ "+format+"\n", args...)
}

// PrintError prints an error message.
func PrintError(format string, args ...any) {
	_, _ = Red.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
}

// PrintWarning prints a warning message.
func PrintWarning(format string, args ...any) {
	if IsQuiet() {
		return
	}
	_, _ = Yellow.Printf("⚠ "+format+"\n", args...)
}

// PrintWarningStderr prints a warning message to stderr, keeping stdout clean
// for structured output (e.g. --json).
func PrintWarningStderr(format string, args ...any) {
	if IsQuiet() {
		return
	}
	_, _ = Yellow.Fprintf(os.Stderr, "⚠ "+format+"\n", args...)
}

// PrintInfo prints an info message.
func PrintInfo(format string, args ...any) {
	if IsQuiet() {
		return
	}
	_, _ = Cyan.Printf("ℹ "+format+"\n", args...)
}

// Confirm prompts for user confirmation.
func Confirm(prompt string, defaultYes bool) bool {
	if IsQuiet() {
		return defaultYes
	}

	suffix := " [y/N]: "
	if defaultYes {
		suffix = " [Y/n]: "
	}

	fmt.Print(prompt + suffix)

	var response string
	_, _ = fmt.Scanln(&response) // Ignore error - empty string treated as default

	response = strings.ToLower(strings.TrimSpace(response))

	if response == "" {
		return defaultYes
	}

	return response == "y" || response == "yes"
}

// FormatParticipant formats an email participant for display.
func FormatParticipant(p domain.EmailParticipant) string {
	if p.Name != "" {
		return p.Name
	}
	return p.Email
}

// FormatParticipants formats a slice of email participants.
func FormatParticipants(participants []domain.EmailParticipant) string {
	names := make([]string, len(participants))
	for i, p := range participants {
		names[i] = FormatParticipant(p)
	}
	return strings.Join(names, ", ")
}

// FormatSize formats a file size in bytes to a human-readable string.
// Whole-number results drop the trailing ".0" (e.g. "1 KB" not "1.0 KB").
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	num := strings.TrimSuffix(fmt.Sprintf("%.1f", float64(bytes)/float64(div)), ".0")
	return fmt.Sprintf("%s %cB", num, "KMGTPE"[exp])
}

// PrintEmptyState prints a consistent "no items found" message.
func PrintEmptyState(resourceName string) {
	if IsQuiet() {
		return
	}
	fmt.Printf("No %s found.\n", resourceName)
}

// PrintEmptyStateWithHint prints empty state with a helpful hint.
func PrintEmptyStateWithHint(resourceName, hint string) {
	if IsQuiet() {
		return
	}
	fmt.Printf("No %s found.\n", resourceName)
	if hint != "" {
		PrintInfo(hint)
	}
}

// PrintJSON writes data to stdout as pretty-printed JSON.
// This is a convenience function for commands that need simple JSON output.
func PrintJSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// truncateCell truncates a table cell to maxLen characters with ellipsis using proper UTF-8 rune counting
func truncateCell(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

// stripAnsiCodes removes ANSI escape sequences from a string.
// Fast path: returns original string if no escape sequences present.
func stripAnsiCodes(s string) string {
	// Fast path: no escape sequences present
	if !strings.ContainsRune(s, '\x1b') {
		return s
	}

	// ANSI codes start with ESC[ and end with a letter
	var result strings.Builder
	result.Grow(len(s)) // Pre-allocate for efficiency
	inEscape := false
	escStart := false

	for _, r := range s {
		if r == '\x1b' { // ESC character
			inEscape = true
			escStart = true
			continue
		}
		if inEscape {
			if escStart && r == '[' {
				escStart = false
				continue
			}
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == 'm' {
				inEscape = false
				escStart = false
				continue
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

// visualWidth returns the display width of a string (excluding ANSI codes)
func visualWidth(s string) int {
	return len([]rune(stripAnsiCodes(s)))
}

// padString pads a string to the specified width (in runes) with spaces
// alignRight=true pads on the left, false pads on the right
func padString(s string, width int, alignRight bool) string {
	runeCount := visualWidth(s)
	if runeCount >= width {
		return s
	}
	padding := strings.Repeat(" ", width-runeCount)
	if alignRight {
		return padding + s
	}
	return s + padding
}
