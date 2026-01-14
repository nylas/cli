package common

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/nylas/cli/internal/domain"
	"gopkg.in/yaml.v3"
)

// OutputFormat represents the output format type.
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
	FormatYAML  OutputFormat = "yaml"
)

// ParseFormat parses a format string into OutputFormat.
func ParseFormat(s string) (OutputFormat, error) {
	switch strings.ToLower(s) {
	case "table", "":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "csv":
		return FormatCSV, nil
	case "yaml", "yml":
		return FormatYAML, nil
	default:
		return "", NewInputError(fmt.Sprintf("invalid format: %s (valid: table, json, csv, yaml)", s))
	}
}

// Formatter handles output formatting.
type Formatter struct {
	format OutputFormat
	writer io.Writer
}

// NewFormatter creates a new formatter.
func NewFormatter(format OutputFormat) *Formatter {
	return &Formatter{
		format: format,
		writer: os.Stdout,
	}
}

// SetWriter sets the output writer.
func (f *Formatter) SetWriter(w io.Writer) *Formatter {
	f.writer = w
	return f
}

// Format formats and outputs data based on the configured format.
func (f *Formatter) Format(data any) error {
	switch f.format {
	case FormatJSON:
		return f.formatJSON(data)
	case FormatCSV:
		return f.formatCSV(data)
	case FormatYAML:
		return f.formatYAML(data)
	default:
		return f.formatTable(data)
	}
}

// formatJSON outputs data as JSON.
func (f *Formatter) formatJSON(data any) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// formatYAML outputs data as YAML.
func (f *Formatter) formatYAML(data any) error {
	encoder := yaml.NewEncoder(f.writer)
	encoder.SetIndent(2)
	defer func() { _ = encoder.Close() }()
	return encoder.Encode(data)
}

// formatCSV outputs data as CSV.
func (f *Formatter) formatCSV(data any) error {
	writer := csv.NewWriter(f.writer)
	defer writer.Flush()

	// Handle slice of structs
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice {
		// Single item - wrap in slice
		return f.formatCSVSingle(writer, data)
	}

	if v.Len() == 0 {
		return nil
	}

	// Get headers from first element
	first := v.Index(0)
	if first.Kind() == reflect.Ptr {
		first = first.Elem()
	}

	headers := getCSVHeaders(first)
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write rows
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		row := getCSVRow(elem)
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// formatCSVSingle formats a single item as CSV.
func (f *Formatter) formatCSVSingle(writer *csv.Writer, data any) error {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	headers := getCSVHeaders(v)
	if err := writer.Write(headers); err != nil {
		return err
	}

	row := getCSVRow(v)
	return writer.Write(row)
}

// getCSVHeaders extracts CSV headers from a struct.
func getCSVHeaders(v reflect.Value) []string {
	if v.Kind() != reflect.Struct {
		return []string{"value"}
	}

	t := v.Type()
	headers := make([]string, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		// Use json tag if available
		name := field.Name
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			parts := strings.Split(tag, ",")
			if parts[0] != "" {
				name = parts[0]
			}
		}
		headers = append(headers, name)
	}

	return headers
}

// getCSVRow extracts CSV row values from a struct.
func getCSVRow(v reflect.Value) []string {
	if v.Kind() != reflect.Struct {
		return []string{fmt.Sprintf("%v", v.Interface())}
	}

	t := v.Type()
	row := make([]string, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		// Skip fields with json:"-"
		if tag := field.Tag.Get("json"); tag == "-" {
			continue
		}

		value := v.Field(i)
		row = append(row, formatValue(value))
	}

	return row
}

// formatValue formats a reflect.Value as a string.
func formatValue(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return ""
		}
		return formatValue(v.Elem())
	case reflect.Slice, reflect.Array:
		if v.Len() == 0 {
			return ""
		}
		parts := make([]string, v.Len())
		for i := 0; i < v.Len(); i++ {
			parts[i] = formatValue(v.Index(i))
		}
		return strings.Join(parts, "; ")
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// formatTable outputs data as a formatted table.
func (f *Formatter) formatTable(data any) error {
	// This is a placeholder - actual table formatting is done by specific commands
	// for more control over display
	return f.formatJSON(data)
}

// Table provides a simple table builder.
type Table struct {
	headers    []string
	rows       [][]string
	alignRight []bool
	writer     io.Writer
}

// NewTable creates a new table with headers.
func NewTable(headers ...string) *Table {
	return &Table{
		headers:    headers,
		rows:       make([][]string, 0),
		alignRight: make([]bool, len(headers)),
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

	// Calculate column widths
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = len(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print headers
	for i, h := range t.headers {
		if t.alignRight[i] {
			_, _ = BoldCyan.Fprintf(t.writer, "%*s  ", widths[i], h)
		} else {
			_, _ = BoldCyan.Fprintf(t.writer, "%-*s  ", widths[i], h)
		}
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
			if t.alignRight[i] {
				_, _ = fmt.Fprintf(t.writer, "%*s  ", widths[i], cell)
			} else {
				_, _ = fmt.Fprintf(t.writer, "%-*s  ", widths[i], cell)
			}
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
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
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

// PrintListHeader prints a consistent "found N items" header.
func PrintListHeader(count int, resourceName string) {
	if IsQuiet() {
		return
	}
	if count == 1 {
		fmt.Printf("Found 1 %s:\n\n", resourceName)
	} else {
		fmt.Printf("Found %d %ss:\n\n", count, resourceName)
	}
}
