//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// rpcTestToken is the fixed WS session token injected into the server subprocess for tests.
const rpcTestToken = "integration-rpc-token"

// startRPCServer launches `nylas rpc serve` on a free loopback port with the test credentials and a
// known token, waits until it accepts WebSocket connections, and returns its addr + token.
// extraEnv overrides/augments the default test env (e.g. set NYLAS_GRANT_ID to drive the pollers).
// Server shutdown + process reap is registered via t.Cleanup.
func startRPCServer(t *testing.T, extraEnv map[string]string) (addr, token string) {
	t.Helper()
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	addr = "127.0.0.1:" + strconv.Itoa(freeTCPPort(t))
	ctx, cancel := context.WithCancel(context.Background())

	env := map[string]string{"NYLAS_WS_TOKEN": rpcTestToken}
	for k, v := range extraEnv {
		env[k] = v
	}

	cmd := exec.CommandContext(ctx, testBinary, "rpc", "serve", "--addr", addr)
	cmd.Env = cliTestEnv(env)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("start rpc server: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(os.Interrupt)
		}
		cancel()
		_ = cmd.Wait()
	})

	// Readiness: retry-dial until the WebSocket handshake succeeds.
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		c, resp, err := websocket.DefaultDialer.Dial("ws://"+addr+"/ws",
			http.Header{"Authorization": {"Bearer " + rpcTestToken}})
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err == nil {
			_ = c.Close()
			return addr, rpcTestToken
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("rpc server did not become ready\nstderr: %s", stderr.String())
	return "", ""
}

// dialRPC opens an authenticated WebSocket connection (Authorization: Bearer). Close is registered.
func dialRPC(t *testing.T, addr, token string) *websocket.Conn {
	t.Helper()
	conn, resp, err := websocket.DefaultDialer.Dial("ws://"+addr+"/ws",
		http.Header{"Authorization": {"Bearer " + token}})
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("dial rpc: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// rpcResult is the parsed outcome of a JSON-RPC call: either Result is set, or IsError with code/message.
type rpcResult struct {
	Result  map[string]json.RawMessage
	IsError bool
	ErrCode int
	ErrMsg  string
}

// rpcCall sends a request, reads until the response with the matching id arrives (skipping any
// interleaved notifications), and returns the parsed result or error.
func rpcCall(t *testing.T, conn *websocket.Conn, id int, method string, params map[string]any) rpcResult {
	t.Helper()
	req := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		req["params"] = params
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("write %s: %v", method, err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	for {
		var resp struct {
			ID     json.RawMessage            `json:"id"`
			Result map[string]json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := conn.ReadJSON(&resp); err != nil {
			t.Fatalf("read %s response: %v", method, err)
		}
		if string(resp.ID) != strconv.Itoa(id) {
			continue // notification or a different id
		}
		if resp.Error != nil {
			return rpcResult{IsError: true, ErrCode: resp.Error.Code, ErrMsg: resp.Error.Message}
		}
		return rpcResult{Result: resp.Result}
	}
}

// rpcID extracts a string "id" field from a result object (most create/get results carry one).
func rpcID(t *testing.T, result map[string]json.RawMessage) string {
	t.Helper()
	var s string
	if raw, ok := result["id"]; ok {
		_ = json.Unmarshal(raw, &s)
	}
	return s
}

// waitForNotification reads frames until a server->client notification (no id) with the given method
// (and optional matching predicate) arrives, or the timeout elapses.
func waitForNotification(t *testing.T, conn *websocket.Conn, method string, timeout time.Duration, match func(json.RawMessage) bool) (json.RawMessage, bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := conn.SetReadDeadline(deadline); err != nil {
			return nil, false
		}
		var msg struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			return nil, false
		}
		if msg.ID != nil {
			continue // a response, not a notification
		}
		if msg.Method == method && (match == nil || match(msg.Params)) {
			return msg.Params, true
		}
	}
	return nil, false
}
