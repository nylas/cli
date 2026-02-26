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

	got, err := readRequestPayload(reader)
	if err != nil {
		t.Fatalf("readRequestPayload() error = %v", err)
	}
	if string(got) != jsonBody {
		t.Errorf("payload = %q, want %q", string(got), jsonBody)
	}
}

func TestReadRequestPayload_NewlineFallback(t *testing.T) {
	t.Parallel()

	jsonBody := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}`
	reader := bufio.NewReader(bytes.NewBufferString(jsonBody + "\n"))

	got, err := readRequestPayload(reader)
	if err != nil {
		t.Fatalf("readRequestPayload() error = %v", err)
	}
	if string(got) != jsonBody {
		t.Errorf("payload = %q, want %q", string(got), jsonBody)
	}
}

func TestReadRequestPayload_MissingContentLength(t *testing.T) {
	t.Parallel()

	reader := bufio.NewReader(bytes.NewBufferString("Content-Type: application/json\r\n\r\n{}"))
	_, err := readRequestPayload(reader)
	if err == nil {
		t.Fatal("expected error, got nil")
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
