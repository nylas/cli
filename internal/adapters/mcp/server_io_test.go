package mcp

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"
)

func TestReadRequestPayload_ContentLength(t *testing.T) {
	t.Parallel()

	jsonBody := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}`
	input := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(jsonBody), jsonBody)
	reader := bufio.NewReader(bytes.NewBufferString(input))

	got, isCL, err := readRequestPayload(reader)
	if err != nil {
		t.Fatalf("readRequestPayload() error = %v", err)
	}
	if !isCL {
		t.Error("expected Content-Length mode, got newline mode")
	}
	if string(got) != jsonBody {
		t.Errorf("payload = %q, want %q", string(got), jsonBody)
	}
}

func TestReadRequestPayload_NewlineFallback(t *testing.T) {
	t.Parallel()

	jsonBody := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}`
	reader := bufio.NewReader(bytes.NewBufferString(jsonBody + "\n"))

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

func TestReadRequestPayload_MissingContentLength(t *testing.T) {
	t.Parallel()

	reader := bufio.NewReader(bytes.NewBufferString("Content-Type: application/json\r\n\r\n{}"))
	_, _, err := readRequestPayload(reader)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestReadRequestPayload_ContentLengthExceedsMax(t *testing.T) {
	t.Parallel()

	// Claim a Content-Length larger than maxContentLength (10 MB).
	input := fmt.Sprintf("Content-Length: %d\r\n\r\n{}", maxContentLength+1)
	reader := bufio.NewReader(bytes.NewBufferString(input))

	_, _, err := readRequestPayload(reader)
	if err == nil {
		t.Fatal("expected error for oversized Content-Length, got nil")
	}
	if got := err.Error(); !bytes.Contains([]byte(got), []byte("exceeds maximum")) {
		t.Errorf("error = %q, want it to mention 'exceeds maximum'", got)
	}
}

func TestReadRequestPayload_EmptyLineRetry(t *testing.T) {
	t.Parallel()

	// Send an empty line followed by a valid JSON line — tests the non-recursive retry path.
	jsonBody := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}`
	input := "\n\n" + jsonBody + "\n"
	reader := bufio.NewReader(bytes.NewBufferString(input))

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

func TestWriteResponsePayload(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	writer := bufio.NewWriter(&out)
	payload := []byte(`{"jsonrpc":"2.0","id":1,"result":{}}`)

	if err := writeResponsePayload(writer, payload); err != nil {
		t.Fatalf("writeResponsePayload() error = %v", err)
	}

	wantPrefix := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))
	if got := out.String(); got != wantPrefix+string(payload) {
		t.Errorf("output = %q, want %q", got, wantPrefix+string(payload))
	}
}

func TestWriteNewlinePayload(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	writer := bufio.NewWriter(&out)
	payload := []byte(`{"jsonrpc":"2.0","id":1,"result":{}}`)

	if err := writeNewlinePayload(writer, payload); err != nil {
		t.Fatalf("writeNewlinePayload() error = %v", err)
	}

	want := string(payload) + "\n"
	if got := out.String(); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}
