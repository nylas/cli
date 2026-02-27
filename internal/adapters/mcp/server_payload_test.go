package mcp

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"
)

// ============================================================================
// TestReadRequestPayload_CRSkip — covers the '\r' peek+skip path (lines 101-105).
// ============================================================================

func TestReadRequestPayload_CRSkip(t *testing.T) {
	t.Parallel()

	jsonBody := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}`
	// Prepend CR bytes — these should be skipped before JSON is read.
	input := "\r\r" + jsonBody + "\n"
	reader := bufio.NewReader(strings.NewReader(input))

	got, isCL, err := readRequestPayload(reader)
	if err != nil {
		t.Fatalf("readRequestPayload() error = %v", err)
	}
	if isCL {
		t.Error("expected newline mode, got Content-Length mode")
	}
	if string(got) != jsonBody {
		t.Errorf("payload = %q, want %q", string(got), jsonBody)
	}
}

// ============================================================================
// TestReadRequestPayload_EOFAfterPartialJSON — covers EOF path at line 112-117.
// ============================================================================

func TestReadRequestPayload_EOFAfterPartialJSON(t *testing.T) {
	t.Parallel()

	// JSON body with no trailing newline — ReadBytes returns io.EOF with data.
	jsonBody := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}`
	reader := bufio.NewReader(strings.NewReader(jsonBody)) // no trailing newline

	got, isCL, err := readRequestPayload(reader)
	if err != nil {
		t.Fatalf("readRequestPayload() error = %v, want nil", err)
	}
	if isCL {
		t.Error("expected newline mode, got Content-Length mode")
	}
	if string(got) != jsonBody {
		t.Errorf("payload = %q, want %q", string(got), jsonBody)
	}
}

// ============================================================================
// TestReadRequestPayload_EOFEmptyLine — empty-line EOF path is unreachable
// given the peek guard (Peek sees '{', so line can never trim to empty).
// ============================================================================

func TestReadRequestPayload_EOFEmptyLine(t *testing.T) {
	t.Parallel()
	t.Skip("empty-line-after-JSON-peek branch is unreachable given peek guard")
}

// ============================================================================
// TestReadRequestPayload_ReadBytesError — covers non-EOF error from ReadBytes.
// Peek sees '{', then ReadBytes gets the custom close error from the pipe.
// ============================================================================

func TestReadRequestPayload_ReadBytesError(t *testing.T) {
	t.Parallel()

	// Construct a reader that: Peek returns '{' successfully, then Read fails.
	// We use a pipe: write the '{' byte, then close with error.
	pr, pw := io.Pipe()

	// Write just enough to satisfy Peek(1): write '{' then close with error.
	go func() {
		_, _ = pw.Write([]byte(`{`))
		pw.CloseWithError(errors.New("read error after peek"))
	}()

	reader := bufio.NewReader(pr)

	_, _, err := readRequestPayload(reader)
	if err == nil {
		t.Fatal("expected error from ReadBytes, got nil")
	}
}

// ============================================================================
// TestReadRequestPayload_ReadFullError — covers io.ReadFull error (line 169-171).
// Content-Length header is parsed successfully but the body read fails.
// ============================================================================

func TestReadRequestPayload_ReadFullError(t *testing.T) {
	t.Parallel()

	// Send valid Content-Length header claiming 50 bytes, but only send 5 bytes
	// of body before closing the pipe — io.ReadFull returns unexpected EOF.
	pr, pw := io.Pipe()
	go func() {
		header := "Content-Length: 50\r\n\r\n"
		_, _ = pw.Write([]byte(header))
		_, _ = pw.Write([]byte("hello")) // only 5 of the promised 50 bytes
		_ = pw.Close()
	}()

	reader := bufio.NewReader(pr)
	_, _, err := readRequestPayload(reader)
	if err == nil {
		t.Fatal("expected error from truncated body, got nil")
	}
}

// ============================================================================
// TestReadRequestPayload_HeaderReadError — covers ReadString error (line 136-138).
// ============================================================================

func TestReadRequestPayload_HeaderReadError(t *testing.T) {
	t.Parallel()

	// Construct a reader that returns a non-JSON, non-whitespace first byte (triggers
	// Content-Length mode), then errors on ReadString.
	pr, pw := io.Pipe()
	go func() {
		// Write 'C' to trigger content-length mode (not '{', '[', '\r', '\n').
		_, _ = pw.Write([]byte("C"))
		pw.CloseWithError(errors.New("header read error"))
	}()

	reader := bufio.NewReader(pr)
	_, _, err := readRequestPayload(reader)
	if err == nil {
		t.Fatal("expected error from ReadString in header mode, got nil")
	}
}

// ============================================================================
// TestReadRequestPayload_InvalidHeaderLine — covers the "no colon" error.
// ============================================================================

func TestReadRequestPayload_InvalidHeaderLine(t *testing.T) {
	t.Parallel()

	// A header line without ':' — triggers "invalid header line" error.
	input := "X-No-Colon-Header\r\n\r\n"
	reader := bufio.NewReader(strings.NewReader(input))

	_, _, err := readRequestPayload(reader)
	if err == nil {
		t.Fatal("expected error for invalid header line, got nil")
	}
	if !strings.Contains(err.Error(), "invalid header line") {
		t.Errorf("error = %q, want it to mention 'invalid header line'", err.Error())
	}
}

// ============================================================================
// TestReadRequestPayload_InvalidContentLength — covers bad Content-Length values.
// ============================================================================

func TestReadRequestPayload_InvalidContentLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{name: "non-numeric", value: "Content-Length: abc\r\n\r\n"},
		{name: "negative", value: "Content-Length: -1\r\n\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader := bufio.NewReader(strings.NewReader(tt.value))
			_, _, err := readRequestPayload(reader)
			if err == nil {
				t.Fatal("expected error for invalid Content-Length, got nil")
			}
			if !strings.Contains(err.Error(), "invalid Content-Length") {
				t.Errorf("error = %q, want it to mention 'invalid Content-Length'", err.Error())
			}
		})
	}
}
