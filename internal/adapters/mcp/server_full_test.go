package mcp

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// ============================================================================
// errWriter — a writer that always fails, used to trigger write error paths.
// ============================================================================

type errWriter struct{}

func (errWriter) Write(_ []byte) (int, error) { return 0, errors.New("write failed") }

// ============================================================================
// mockGrantStore — minimal GrantStore implementation for SetGrantStore tests.
// ============================================================================

type mockGrantStore struct{}

func (mockGrantStore) SaveGrant(_ domain.GrantInfo) error              { return nil }
func (mockGrantStore) GetGrant(_ string) (*domain.GrantInfo, error)    { return nil, nil }
func (mockGrantStore) GetGrantByEmail(_ string) (*domain.GrantInfo, error) {
	return nil, nil
}
func (mockGrantStore) ListGrants() ([]domain.GrantInfo, error)  { return nil, nil }
func (mockGrantStore) DeleteGrant(_ string) error               { return nil }
func (mockGrantStore) SetDefaultGrant(_ string) error           { return nil }
func (mockGrantStore) GetDefaultGrant() (string, error)         { return "", nil }
func (mockGrantStore) ClearGrants() error                       { return nil }

// Ensure mockGrantStore satisfies ports.GrantStore at compile time.
var _ ports.GrantStore = mockGrantStore{}

// ============================================================================
// TestNewServer — covers NewServer constructor.
// ============================================================================

func TestNewServer(t *testing.T) {
	t.Parallel()

	client := &mockNylasClient{}
	s := NewServer(client, "my-grant")

	if s == nil {
		t.Fatal("NewServer() returned nil")
	}
	if s.client != client {
		t.Error("NewServer() did not store client")
	}
	if s.defaultGrant != "my-grant" {
		t.Errorf("defaultGrant = %q, want %q", s.defaultGrant, "my-grant")
	}
	if s.grantStore != nil {
		t.Error("grantStore should be nil after NewServer()")
	}
}

// ============================================================================
// TestSetGrantStore — covers SetGrantStore.
// ============================================================================

func TestSetGrantStore(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "grant-1")

	// Initially nil.
	if s.grantStore != nil {
		t.Error("grantStore should be nil before SetGrantStore")
	}

	store := mockGrantStore{}
	s.SetGrantStore(store)

	if s.grantStore == nil {
		t.Error("grantStore should be set after SetGrantStore")
	}

	// Setting to nil clears it.
	s.SetGrantStore(nil)
	if s.grantStore != nil {
		t.Error("grantStore should be nil after SetGrantStore(nil)")
	}
}

// ============================================================================
// TestRunWithIO — covers RunWithIO code paths.
// ============================================================================

// pingNewlineJSON builds a newline-delimited JSON ping request.
func pingNewlineJSON(id int) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"ping","params":{}}`, id) + "\n"
}

// pingContentLength builds a Content-Length framed ping request.
func pingContentLength(id int) string {
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"ping","params":{}}`, id)
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

func TestRunWithIO_EOF(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// Empty reader → EOF → returns nil.
	var out bytes.Buffer
	err := s.RunWithIO(ctx, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("RunWithIO() with EOF reader = %v, want nil", err)
	}
}

func TestRunWithIO_ContextCancelled(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	var out bytes.Buffer
	err := s.RunWithIO(ctx, strings.NewReader(""), &out)
	if err == nil {
		t.Fatal("RunWithIO() with cancelled context should return error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("RunWithIO() error = %v, want context.Canceled", err)
	}
}

func TestRunWithIO_ReadError(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// A reader that immediately returns a non-EOF error.
	errReader := &alwaysErrReader{err: errors.New("disk read failure")}
	var out bytes.Buffer
	err := s.RunWithIO(ctx, errReader, &out)
	if err == nil {
		t.Fatal("RunWithIO() with read error should return error")
	}
	if !strings.Contains(err.Error(), "reading stdin") {
		t.Errorf("error = %q, want it to mention 'reading stdin'", err.Error())
	}
}

// alwaysErrReader is a reader that returns an error on every read.
type alwaysErrReader struct {
	err error
}

func (r *alwaysErrReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

func TestRunWithIO_InvalidJSON(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// First line: starts with '{' but is not valid JSON → parse error response sent.
	// Second line: valid ping → normal response.
	// EOF → loop exits with nil.
	input := "{not-valid-json}\n" + pingNewlineJSON(1)
	var out bytes.Buffer
	err := s.RunWithIO(ctx, strings.NewReader(input), &out)
	if err != nil {
		t.Fatalf("RunWithIO() = %v, want nil", err)
	}

	// Output should contain the parse error response.
	outStr := out.String()
	if !strings.Contains(outStr, fmt.Sprintf("%d", codeParseError)) {
		t.Errorf("output missing parse error code %d; got: %s", codeParseError, outStr)
	}
}

func TestRunWithIO_InvalidJSON_ContentLength(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// Content-Length framed invalid JSON → parse error, then EOF.
	body := `not json at all`
	input := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	var out bytes.Buffer
	err := s.RunWithIO(ctx, strings.NewReader(input), &out)
	if err != nil {
		t.Fatalf("RunWithIO() = %v, want nil", err)
	}

	outStr := out.String()
	if !strings.Contains(outStr, fmt.Sprintf("%d", codeParseError)) {
		t.Errorf("output missing parse error code; got: %s", outStr)
	}
}

func TestRunWithIO_Notification(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// notifications/initialized → dispatch returns nil → no response written, loop continues.
	notif := `{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}` + "\n"
	var out bytes.Buffer
	err := s.RunWithIO(ctx, strings.NewReader(notif), &out)
	if err != nil {
		t.Fatalf("RunWithIO() = %v, want nil", err)
	}

	// No output should have been written for the notification.
	if out.Len() != 0 {
		t.Errorf("expected no output for notification, got %q", out.String())
	}
}

func TestRunWithIO_NormalRequest(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	var out bytes.Buffer
	err := s.RunWithIO(ctx, strings.NewReader(pingNewlineJSON(42)), &out)
	if err != nil {
		t.Fatalf("RunWithIO() = %v, want nil", err)
	}

	outStr := out.String()
	if !strings.Contains(outStr, `"id":42`) {
		t.Errorf("response missing id=42; got: %s", outStr)
	}
}

func TestRunWithIO_ContentLengthRequest(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	var out bytes.Buffer
	err := s.RunWithIO(ctx, strings.NewReader(pingContentLength(7)), &out)
	if err != nil {
		t.Fatalf("RunWithIO() = %v, want nil", err)
	}

	outStr := out.String()
	// Content-Length mode response should include the framing header.
	if !strings.Contains(outStr, "Content-Length:") {
		t.Errorf("response missing Content-Length header; got: %s", outStr)
	}
	if !strings.Contains(outStr, `"id":7`) {
		t.Errorf("response missing id=7; got: %s", outStr)
	}
}

func TestRunWithIO_WriteError(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// A writer that always fails → writePayload should return an error.
	err := s.RunWithIO(ctx, strings.NewReader(pingNewlineJSON(1)), errWriter{})
	if err == nil {
		t.Fatal("RunWithIO() with failing writer should return error")
	}
}

func TestRunWithIO_WriteErrorOnParseError(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// Invalid JSON (starts with '{' to enter JSON mode) triggers the error response write path.
	// The writer fails, so RunWithIO should return that write error.
	err := s.RunWithIO(ctx, strings.NewReader("{bad-json}\n"), errWriter{})
	if err == nil {
		t.Fatal("RunWithIO() with failing writer on parse error should return error")
	}
}

// ============================================================================
// TestWritePayload — covers writePayload both branches.
// ============================================================================

func TestWritePayload_ContentLengthMode(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	writer := bufio.NewWriter(&out)
	payload := []byte(`{"jsonrpc":"2.0","id":1,"result":{}}`)

	err := writePayload(writer, payload, true)
	if err != nil {
		t.Fatalf("writePayload(contentLength=true) = %v", err)
	}

	outStr := out.String()
	if !strings.Contains(outStr, "Content-Length:") {
		t.Errorf("expected Content-Length header, got: %q", outStr)
	}
	if !strings.Contains(outStr, string(payload)) {
		t.Errorf("expected payload in output, got: %q", outStr)
	}
}

func TestWritePayload_NewlineMode(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	writer := bufio.NewWriter(&out)
	payload := []byte(`{"jsonrpc":"2.0","id":2,"result":{}}`)

	err := writePayload(writer, payload, false)
	if err != nil {
		t.Fatalf("writePayload(contentLength=false) = %v", err)
	}

	want := string(payload) + "\n"
	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}

// ============================================================================
// TestWriteResponsePayload_WriteError — covers error paths in writeResponsePayload.
// A 1-byte buffer forces every multi-byte write to flush immediately, triggering
// the underlying errWriter error on the first write (header line).
// ============================================================================

func TestWriteResponsePayload_HeaderWriteError(t *testing.T) {
	t.Parallel()

	// 1-byte buffer forces immediate flush on any write > 1 byte → header write fails.
	writer := bufio.NewWriterSize(errWriter{}, 1)
	payload := []byte(`{"jsonrpc":"2.0","id":1,"result":{}}`)

	err := writeResponsePayload(writer, payload)
	if err == nil {
		t.Fatal("writeResponsePayload() with failing writer should return error")
	}
	if !strings.Contains(err.Error(), "writing response") {
		t.Errorf("error = %q, want it to mention 'writing response'", err.Error())
	}
}

func TestWriteResponsePayload_BodyWriteError(t *testing.T) {
	t.Parallel()

	// Use a countWriter that fails after the first successful write (header),
	// causing the body write to fail.
	cw := &countFailWriter{failAfter: 1}
	writer := bufio.NewWriterSize(cw, 1)
	payload := []byte(`{"jsonrpc":"2.0","id":1,"result":{}}`)

	err := writeResponsePayload(writer, payload)
	if err == nil {
		t.Fatal("writeResponsePayload() body write error should return error")
	}
}

func TestWriteResponsePayload_FlushError(t *testing.T) {
	t.Parallel()

	// Use a buffer exactly as large as header+payload so that neither fmt.Fprintf
	// nor writer.Write triggers an intermediate flush — both writes fit in the buffer.
	// When writer.Flush() is called, it sends all buffered bytes to the underlying
	// errWriter, which fails → "flushing response" error.
	payload := []byte(`{"jsonrpc":"2.0","id":1,"result":{}}`)
	headerStr := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))
	totalSize := len(headerStr) + len(payload)
	// bufio buffer must be >= totalSize so no intermediate flush occurs.
	writer := bufio.NewWriterSize(errWriter{}, totalSize)

	err := writeResponsePayload(writer, payload)
	if err == nil {
		t.Fatal("writeResponsePayload() flush error should return error")
	}
	if !strings.Contains(err.Error(), "flushing response") {
		t.Errorf("error = %q, want it to mention 'flushing response'", err.Error())
	}
}

// ============================================================================
// TestWriteNewlinePayload_WriteError — covers error paths in writeNewlinePayload.
// ============================================================================

func TestWriteNewlinePayload_WriteError(t *testing.T) {
	t.Parallel()

	// 1-byte buffer forces immediate flush → body write fails immediately.
	writer := bufio.NewWriterSize(errWriter{}, 1)
	payload := []byte(`{"jsonrpc":"2.0","id":1,"result":{}}`)

	err := writeNewlinePayload(writer, payload)
	if err == nil {
		t.Fatal("writeNewlinePayload() with failing writer should return error")
	}
	if !strings.Contains(err.Error(), "writing response") {
		t.Errorf("error = %q, want it to mention 'writing response'", err.Error())
	}
}

func TestWriteNewlinePayload_NewlineWriteError(t *testing.T) {
	t.Parallel()

	// Use a buffer exactly as large as the payload so that writer.Write(payload)
	// fits without flushing. When writer.WriteByte('\n') is called, the buffer is
	// full so bufio forces a flush that sends everything to errWriter — errWriter
	// fails, surfacing as "writing response newline" error.
	payload := []byte(`{"jsonrpc":"2.0","id":1,"result":{}}`)
	writer := bufio.NewWriterSize(errWriter{}, len(payload))

	err := writeNewlinePayload(writer, payload)
	if err == nil {
		t.Fatal("writeNewlinePayload() newline write error should return error")
	}
	if !strings.Contains(err.Error(), "writing response") {
		t.Errorf("error = %q, want it to mention 'writing response'", err.Error())
	}
}

// countFailWriter succeeds for the first failAfter writes, then always fails.
type countFailWriter struct {
	writes    int
	failAfter int
}

func (w *countFailWriter) Write(p []byte) (int, error) {
	if w.writes >= w.failAfter {
		return 0, errors.New("write failed")
	}
	w.writes++
	return len(p), nil
}

// ============================================================================
// TestDispatch_ToolsCall — covers the tools/call case in dispatch.
// ============================================================================

func TestDispatch_ToolsCall(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	ctx := context.Background()

	req := &Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		ID:      float64(10),
	}
	req.Params.Name = "ping" // unknown tool — returns tool-level error
	req.Params.Arguments = map[string]any{}

	resp := s.dispatch(ctx, req)
	if resp == nil {
		t.Fatal("dispatch(tools/call) returned nil, want non-nil response")
	}

	got := unmarshalResponse(t, resp)
	if got["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", got["jsonrpc"])
	}
	// tools/call always returns a result (tool errors are inside result.isError).
	if _, hasResult := got["result"]; !hasResult {
		t.Error("dispatch(tools/call) response missing result field")
	}
}

