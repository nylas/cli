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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestCLI_RPCServeHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("rpc", "serve", "--help")
	if err != nil {
		t.Fatalf("rpc serve --help failed: %v\nstderr: %s", err, stderr)
	}

	output := stdout + stderr
	for _, want := range []string{"JSON-RPC", "serve"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected rpc serve help to contain %q, got stdout: %s\nstderr: %s", want, stdout, stderr)
		}
	}
}

func TestCLI_RPCServe_AuthAndEmailList(t *testing.T) {
	skipIfMissingCreds(t)

	port := freeTCPPort(t)
	addr := "127.0.0.1:" + strconv.Itoa(port)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, testBinary, "rpc", "serve", "--addr", addr)
	cmd.Env = cliTestEnv(map[string]string{"NYLAS_WS_TOKEN": "integration-rpc-token"})
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start rpc server: %v", err)
	}

	var stopOnce sync.Once
	stopServer := func() {
		stopOnce.Do(func() {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(os.Interrupt)
			}
			_ = cmd.Wait()
		})
	}
	t.Cleanup(stopServer)

	wsURL := "ws://" + addr + "/ws"
	goodHeader := http.Header{"Authorization": {"Bearer integration-rpc-token"}}
	var conn *websocket.Conn
	var lastErr error
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var resp *http.Response
		conn, resp, lastErr = websocket.DefaultDialer.Dial(wsURL, goodHeader)
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if lastErr == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if lastErr != nil {
		stopServer()
		t.Fatalf("rpc server did not become ready: %v\nstdout: %s\nstderr: %s", lastErr, stdout.String(), stderr.String())
	}
	defer func() { _ = conn.Close() }()

	badConn, resp, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Authorization": {"Bearer wrong-token"}})
	if badConn != nil {
		_ = badConn.Close()
	}
	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err == nil {
		stopServer()
		t.Fatalf("expected websocket dial with wrong bearer token to fail\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		stopServer()
		t.Fatalf("wrong-token websocket status = %d, want %d\nstdout: %s\nstderr: %s", resp.StatusCode, http.StatusUnauthorized, stdout.String(), stderr.String())
	}

	acquireRateLimit(t)
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "email.list",
		"params": map[string]any{
			"grant_id": testGrantID,
			"limit":    2,
		},
	}
	if err := conn.WriteJSON(request); err != nil {
		stopServer()
		t.Fatalf("failed to write email.list request: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		stopServer()
		t.Fatalf("failed to set websocket read deadline: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	for {
		var response struct {
			JSONRPC string                     `json:"jsonrpc"`
			ID      json.RawMessage            `json:"id"`
			Result  map[string]json.RawMessage `json:"result"`
			Error   *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := conn.ReadJSON(&response); err != nil {
			stopServer()
			t.Fatalf("failed to read email.list response: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
		}
		if string(response.ID) != "1" {
			continue
		}
		if response.Error != nil {
			stopServer()
			t.Fatalf("email.list returned RPC error %d %q\nstdout: %s\nstderr: %s", response.Error.Code, response.Error.Message, stdout.String(), stderr.String())
		}
		if _, ok := response.Result["messages"]; !ok {
			stopServer()
			t.Fatalf("email.list result missing messages key: %s\nstdout: %s\nstderr: %s", string(mustMarshalJSON(response.Result)), stdout.String(), stderr.String())
		}
		break
	}
}

func mustMarshalJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("<invalid json>")
	}
	return b
}
