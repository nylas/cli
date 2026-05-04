package output

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"github.com/nylas/cli/internal/ports"
)

const (
	maxColumnWidth = 50
	truncSuffix    = "..."
)

// TableWriter outputs data in human-readable table format
type TableWriter struct {
	w       io.Writer
	colored bool
}

// NewTableWriter creates a new table writer
func NewTableWriter(w io.Writer, colored bool) *TableWriter {
	return &TableWriter{w: w, colored: colored}
}

// Write outputs a single object
func (tw *TableWriter) Write(data any) error {
	// For single objects, output as key: value pairs
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		_, _ = fmt.Fprintf(tw.w, "%v\n", data)
		return nil
	}

	t := v.Type()
	maxKeyLen := 0
	for i := range t.NumField() {
		field := t.Field(i)
		if field.PkgPath != "" { // unexported
			continue
		}
		if len(field.Name) > maxKeyLen {
			maxKeyLen = len(field.Name)
		}
	}

	for i := range t.NumField() {
		field := t.Field(i)
		if field.PkgPath != "" { // unexported
			continue
		}
		value := v.Field(i).Interface()
		_, _ = fmt.Fprintf(tw.w, "%-*s  %v\n", maxKeyLen, field.Name+":", formatValue(value))
	}

	return nil
}

// WriteList outputs a list of objects as a table
func (tw *TableWriter) WriteList(data any, columns []ports.Column) error {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	// Handle empty data
	if v.Kind() != reflect.Slice || v.Len() == 0 {
		return nil
	}

	// Write to buffer first, then apply colors after tabwriter aligns columns
	// This avoids ANSI codes interfering with column width calculation
	var buf bytes.Buffer
	tabW := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

	// Write header
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = strings.ToUpper(col.Header)
	}
	_, _ = fmt.Fprintln(tabW, strings.Join(headers, "\t"))

	// Write rows
	for i := range v.Len() {
		row := v.Index(i)
		if row.Kind() == reflect.Pointer {
			row = row.Elem()
		}

		values := make([]string, len(columns))
		for j, col := range columns {
			val := getFieldValue(row, col.Field)
			formatted := formatValue(val)
			// Width: -1 = no truncation, 0 = auto (maxColumnWidth), >0 = fixed width
			if col.Width > 0 && utf8.RuneCountInString(formatted) > col.Width {
				formatted = truncate(formatted, col.Width)
			} else if col.Width == 0 && utf8.RuneCountInString(formatted) > maxColumnWidth {
				formatted = truncate(formatted, maxColumnWidth)
			}
			// Width == -1: no truncation
			values[j] = formatted
		}
		_, _ = fmt.Fprintln(tabW, strings.Join(values, "\t"))
	}

	if err := tabW.Flush(); err != nil {
		return err
	}

	// Apply colors to header line after tabwriter has aligned everything
	output := buf.String()
	if tw.colored {
		if idx := strings.Index(output, "\n"); idx != -1 {
			header := output[:idx]
			rest := output[idx:]
			_, _ = fmt.Fprintf(tw.w, "\033[1m%s\033[0m%s", header, rest)
			return nil
		}
	}

	_, _ = fmt.Fprint(tw.w, output)
	return nil
}

// WriteError outputs an error message
func (tw *TableWriter) WriteError(err error) error {
	if tw.colored {
		_, _ = fmt.Fprintf(tw.w, "\033[31mError:\033[0m %s\n", err.Error())
	} else {
		_, _ = fmt.Fprintf(tw.w, "Error: %s\n", err.Error())
	}
	return nil
}

// getFieldValue extracts a field value from a struct or map
func getFieldValue(v reflect.Value, field string) any {
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		f := v.FieldByName(field)
		if f.IsValid() {
			return f.Interface()
		}
		// Try case-insensitive match
		t := v.Type()
		for i := range t.NumField() {
			if strings.EqualFold(t.Field(i).Name, field) {
				return v.Field(i).Interface()
			}
		}
	case reflect.Map:
		key := reflect.ValueOf(field)
		val := v.MapIndex(key)
		if val.IsValid() {
			return val.Interface()
		}
	}

	return ""
}

// formatValue converts a value to a display string
func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case time.Time:
		return formatTime(val)
	case bool:
		if val {
			return "Yes"
		}
		return "No"
	case []string:
		if len(val) == 0 {
			return ""
		}
		return strings.Join(val, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatTime formats a time in a human-friendly way
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 02, 2006")
	}
}

// truncate shortens a string to max length with ellipsis
func truncate(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if maxLen <= len(truncSuffix) {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-len(truncSuffix)]) + truncSuffix
}
