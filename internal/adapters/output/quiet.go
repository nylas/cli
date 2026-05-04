package output

import (
	"fmt"
	"io"
	"reflect"

	"github.com/nylas/cli/internal/ports"
)

// QuietWriter outputs minimal data (IDs only) for scripting.
type QuietWriter struct {
	w io.Writer
}

// NewQuietWriter creates a new quiet writer.
func NewQuietWriter(w io.Writer) *QuietWriter {
	return &QuietWriter{w: w}
}

// Write outputs the ID or quiet field of a single object.
func (qw *QuietWriter) Write(data any) error {
	id := extractQuietField(data)
	if id != "" {
		_, _ = fmt.Fprintln(qw.w, id)
	}
	return nil
}

// WriteList outputs IDs only, one per line.
func (qw *QuietWriter) WriteList(data any, _ []ports.Column) error {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice {
		return qw.Write(data)
	}

	for i := range v.Len() {
		item := v.Index(i).Interface()
		id := extractQuietField(item)
		if id != "" {
			_, _ = fmt.Fprintln(qw.w, id)
		}
	}

	return nil
}

// WriteError outputs nothing in quiet mode (errors go to stderr in CLI).
func (qw *QuietWriter) WriteError(_ error) error {
	// In quiet mode, we don't output errors to stdout
	// The CLI layer should handle error output to stderr
	return nil
}

// extractQuietField extracts the ID or quiet field from an object.
func extractQuietField(data any) string {
	// Check if it implements QuietFielder interface
	if qf, ok := data.(ports.QuietFielder); ok {
		return qf.QuietField()
	}

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Sprintf("%v", data)
	}

	// Try common ID field names
	for _, fieldName := range []string{"ID", "Id", "id"} {
		f := v.FieldByName(fieldName)
		if f.IsValid() {
			return fmt.Sprintf("%v", f.Interface())
		}
	}

	return ""
}
