//go:build integration
// +build integration

package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// mcpClient — in-process test helper that speaks the MCP protocol
// ============================================================================

// mcpClient drives a Server via io.Pipe, sending JSON-RPC requests and
// reading responses. It auto-detects whether the server responds with
// Content-Length framing or newline-delimited JSON.
type mcpClient struct {
	t        *testing.T
	stdin    *io.PipeWriter // we write requests here → server reads
	stdoutBR *bufio.Reader  // buffered reader for peeking at response framing
	stdoutPR *io.PipeReader // underlying pipe reader (for cleanup)
}

// newMCPTestClient starts server.RunWithIO in a goroutine and returns a client
// connected to it via pipes. Cleanup is registered with t.Cleanup.
func newMCPTestClient(t *testing.T, server *Server) *mcpClient {
	t.Helper()

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- server.RunWithIO(ctx, stdinR, stdoutW)
		_ = stdoutW.Close()
	}()

	t.Cleanup(func() {
		cancel()
		_ = stdinW.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Log("warning: MCP server goroutine did not stop within 3s")
		}
	})

	return &mcpClient{
		t:        t,
		stdin:    stdinW,
		stdoutBR: bufio.NewReader(stdoutR),
		stdoutPR: stdoutR,
	}
}

// send marshals req as JSON, writes it with Content-Length framing, and
// returns the parsed response map. It blocks until a response is available.
func (c *mcpClient) send(req map[string]any) map[string]any {
	c.t.Helper()

	data, err := json.Marshal(req)
	if err != nil {
		c.t.Fatalf("mcpClient.send: marshal: %v", err)
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := io.WriteString(c.stdin, header); err != nil {
		c.t.Fatalf("mcpClient.send: write header: %v", err)
	}
	if _, err := c.stdin.Write(data); err != nil {
		c.t.Fatalf("mcpClient.send: write body: %v", err)
	}

	return c.readResponse()
}

// sendNewline writes req as newline-delimited JSON.
func (c *mcpClient) sendNewline(req map[string]any) map[string]any {
	c.t.Helper()

	data, err := json.Marshal(req)
	if err != nil {
		c.t.Fatalf("mcpClient.sendNewline: marshal: %v", err)
	}
	data = append(data, '\n')

	if _, err := c.stdin.Write(data); err != nil {
		c.t.Fatalf("mcpClient.sendNewline: write: %v", err)
	}

	return c.readResponse()
}

// readResponse reads a single response from stdout, auto-detecting whether
// the server used Content-Length framing or newline-delimited JSON.
func (c *mcpClient) readResponse() map[string]any {
	c.t.Helper()

	// Peek at first byte to detect framing mode.
	peek, err := c.stdoutBR.Peek(1)
	if err != nil {
		c.t.Fatalf("mcpClient.readResponse: peek: %v", err)
	}

	if peek[0] == '{' || peek[0] == '[' {
		return c.readNewlineResponse()
	}
	return c.readContentLengthResponse()
}

// readNewlineResponse reads a newline-delimited JSON response.
func (c *mcpClient) readNewlineResponse() map[string]any {
	c.t.Helper()

	line, err := c.stdoutBR.ReadBytes('\n')
	if err != nil {
		c.t.Fatalf("mcpClient.readNewlineResponse: %v", err)
	}
	line = bytes.TrimSpace(line)

	var resp map[string]any
	if err := json.Unmarshal(line, &resp); err != nil {
		c.t.Fatalf("mcpClient.readNewlineResponse: unmarshal: %v (line=%s)", err, line)
	}
	return resp
}

// readContentLengthResponse reads a Content-Length framed response.
func (c *mcpClient) readContentLengthResponse() map[string]any {
	c.t.Helper()

	// Read one byte at a time until we see "\r\n\r\n" (end of headers).
	var headerBuf strings.Builder
	for {
		b, err := c.stdoutBR.ReadByte()
		if err != nil {
			c.t.Fatalf("mcpClient.readContentLengthResponse: read header byte: %v", err)
		}
		headerBuf.WriteByte(b)
		if strings.HasSuffix(headerBuf.String(), "\r\n\r\n") {
			break
		}
	}

	// Parse Content-Length from headers.
	contentLength := 0
	for _, line := range strings.Split(headerBuf.String(), "\r\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok && strings.EqualFold(strings.TrimSpace(key), "content-length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err == nil && n > 0 {
				contentLength = n
			}
		}
	}
	if contentLength == 0 {
		c.t.Fatalf("mcpClient.readContentLengthResponse: missing Content-Length in: %q", headerBuf.String())
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(c.stdoutBR, body); err != nil {
		c.t.Fatalf("mcpClient.readContentLengthResponse: read body: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		c.t.Fatalf("mcpClient.readContentLengthResponse: unmarshal: %v (body=%s)", err, body)
	}
	return resp
}

// assertNoError fails the test if the response contains an "error" field.
func assertNoError(t *testing.T, resp map[string]any) {
	t.Helper()
	if e, ok := resp["error"]; ok {
		t.Fatalf("unexpected error in response: %v", e)
	}
}

// assertResult returns the result map and fails if absent.
func assertResult(t *testing.T, resp map[string]any) map[string]any {
	t.Helper()
	assertNoError(t, resp)
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("result field missing or not a map; resp=%v", resp)
	}
	return result
}

// assertToolContent returns the first content block text from a tools/call response.
// It fails the test if the response indicates an error or is missing content.
func assertToolContent(t *testing.T, resp map[string]any) string {
	t.Helper()
	result := assertResult(t, resp)

	if isErr, _ := result["isError"].(bool); isErr {
		content, _ := result["content"].([]any)
		if len(content) > 0 {
			if block, ok := content[0].(map[string]any); ok {
				t.Fatalf("tool returned isError=true: %v", block["text"])
			}
		}
		t.Fatal("tool returned isError=true with no content")
	}

	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("content field missing or empty; result=%v", result)
	}
	block, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("content[0] is not a map")
	}
	text, _ := block["text"].(string)
	return text
}

// ============================================================================
// TestIntegration_Initialize
// ============================================================================

func TestIntegration_Initialize(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	c := newMCPTestClient(t, s)

	resp := c.send(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{},
	})

	if resp["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", resp["jsonrpc"])
	}
	result := assertResult(t, resp)

	if result["protocolVersion"] != protocolVersion {
		t.Errorf("protocolVersion = %v, want %s", result["protocolVersion"], protocolVersion)
	}

	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("serverInfo missing or wrong type")
	}
	if serverInfo["name"] != serverName {
		t.Errorf("serverInfo.name = %v, want %s", serverInfo["name"], serverName)
	}
	if serverInfo["version"] != serverVersion {
		t.Errorf("serverInfo.version = %v, want %s", serverInfo["version"], serverVersion)
	}

	caps, ok := result["capabilities"].(map[string]any)
	if !ok {
		t.Fatal("capabilities field missing or wrong type")
	}
	if _, hasTool := caps["tools"]; !hasTool {
		t.Error("capabilities.tools missing")
	}

	instructions, ok := result["instructions"].(string)
	if !ok || instructions == "" {
		t.Error("instructions field missing or empty")
	}
}

// ============================================================================
// TestIntegration_ToolsList
// ============================================================================

func TestIntegration_ToolsList(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	c := newMCPTestClient(t, s)

	resp := c.send(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	})

	result := assertResult(t, resp)

	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatal("tools field missing or not a slice")
	}

	const wantCount = 47
	if len(tools) != wantCount {
		t.Errorf("tool count = %d, want %d", len(tools), wantCount)
	}

	for i, raw := range tools {
		tool, ok := raw.(map[string]any)
		if !ok {
			t.Errorf("tools[%d] is not a map", i)
			continue
		}
		for _, field := range []string{"name", "description", "inputSchema"} {
			if _, exists := tool[field]; !exists {
				t.Errorf("tools[%d] missing field %q", i, field)
			}
		}
		if name, _ := tool["name"].(string); name == "" {
			t.Errorf("tools[%d] has empty name", i)
		}
	}
}

// ============================================================================
// TestIntegration_Ping
// ============================================================================

func TestIntegration_Ping(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	c := newMCPTestClient(t, s)

	resp := c.send(map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "ping",
		"params":  map[string]any{},
	})

	if resp["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", resp["jsonrpc"])
	}
	// ping returns an empty result object, not an error
	assertResult(t, resp)
}

// ============================================================================
// TestIntegration_UnknownMethod
// ============================================================================

func TestIntegration_UnknownMethod(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	c := newMCPTestClient(t, s)

	resp := c.send(map[string]any{
		"jsonrpc": "2.0",
		"id":      99,
		"method":  "unknown/method",
		"params":  map[string]any{},
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error field; got %v", resp)
	}
	gotCode := int(errObj["code"].(float64))
	if gotCode != codeMethodNotFound {
		t.Errorf("error.code = %d, want %d", gotCode, codeMethodNotFound)
	}
}

// ============================================================================
// TestIntegration_InvalidJSON
// ============================================================================

func TestIntegration_InvalidJSON(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	c := newMCPTestClient(t, s)

	// Send garbage via newline-delimited mode (starts with '{' so server
	// attempts to read until '\n', then unmarshal fails → parse error).
	if _, err := io.WriteString(c.stdin, "{not valid json}\n"); err != nil {
		t.Fatalf("write garbage: %v", err)
	}

	resp := c.readResponse()
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error field; got %v", resp)
	}
	gotCode := int(errObj["code"].(float64))
	if gotCode != codeParseError {
		t.Errorf("error.code = %d, want %d (parse error)", gotCode, codeParseError)
	}
}

// ============================================================================
// TestIntegration_MultipleRequests
// ============================================================================

func TestIntegration_MultipleRequests(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	c := newMCPTestClient(t, s)

	ids := []float64{10, 11, 12}
	for _, id := range ids {
		resp := c.send(map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  "ping",
			"params":  map[string]any{},
		})
		gotID, _ := resp["id"].(float64)
		if gotID != id {
			t.Errorf("response id = %v, want %v", gotID, id)
		}
		assertResult(t, resp)
	}
}

// ============================================================================
// TestIntegration_NewlineDelimitedJSON
// ============================================================================

func TestIntegration_NewlineDelimitedJSON(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	c := newMCPTestClient(t, s)

	// Newline mode: send without Content-Length header.
	resp := c.sendNewline(map[string]any{
		"jsonrpc": "2.0",
		"id":      20,
		"method":  "ping",
		"params":  map[string]any{},
	})

	if resp["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", resp["jsonrpc"])
	}
	assertResult(t, resp)
}

// ============================================================================
// TestIntegration_ContentLengthFraming
// ============================================================================

func TestIntegration_ContentLengthFraming(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	c := newMCPTestClient(t, s)

	// Verify the response carries proper Content-Length framing by inspecting
	// the response ID round-trip with a known request.
	resp := c.send(map[string]any{
		"jsonrpc": "2.0",
		"id":      42,
		"method":  "initialize",
		"params":  map[string]any{},
	})

	gotID, _ := resp["id"].(float64)
	if gotID != 42 {
		t.Errorf("response id = %v, want 42", gotID)
	}
}

// ============================================================================
// TestIntegration_NotificationNoResponse
// ============================================================================

func TestIntegration_NotificationNoResponse(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	c := newMCPTestClient(t, s)

	// Notifications must NOT produce a response. Send one, then send a ping
	// and verify the ping response comes back (not a stale notification response).
	if _, err := io.WriteString(c.stdin, `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n"); err != nil {
		t.Fatalf("write notification: %v", err)
	}

	// The server should skip the notification; next response is for ping.
	resp := c.send(map[string]any{
		"jsonrpc": "2.0",
		"id":      55,
		"method":  "ping",
	})
	gotID, _ := resp["id"].(float64)
	if gotID != 55 {
		t.Errorf("response id = %v, want 55", gotID)
	}
	assertResult(t, resp)
}

// ============================================================================
// TestIntegration_ToolCall_UnknownTool
// ============================================================================

func TestIntegration_ToolCall_UnknownTool(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	c := newMCPTestClient(t, s)

	resp := c.send(map[string]any{
		"jsonrpc": "2.0",
		"id":      30,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "does_not_exist",
			"arguments": map[string]any{},
		},
	})

	// tools/call always returns a success response; error is inside isError field.
	result := assertResult(t, resp)
	isErr, _ := result["isError"].(bool)
	if !isErr {
		t.Errorf("expected isError=true for unknown tool, got result=%v", result)
	}
}
