//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestCLI_RPC_Auth_MissingToken(t *testing.T) {
	skipIfMissingCreds(t)
	addr, _ := startRPCServer(t, map[string]string{"NYLAS_GRANT_ID": ""})

	conn, resp, err := websocket.DefaultDialer.Dial("ws://"+addr+"/ws", nil)
	if conn != nil {
		_ = conn.Close()
	}
	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err == nil {
		t.Fatal("expected websocket dial without token to fail")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestCLI_RPC_Auth_WrongToken(t *testing.T) {
	skipIfMissingCreds(t)
	addr, _ := startRPCServer(t, map[string]string{"NYLAS_GRANT_ID": ""})

	conn, resp, err := websocket.DefaultDialer.Dial("ws://"+addr+"/ws",
		http.Header{"Authorization": {"Bearer wrong"}})
	if conn != nil {
		_ = conn.Close()
	}
	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err == nil {
		t.Fatal("expected websocket dial with wrong token to fail")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestCLI_RPC_Auth_QueryParamToken(t *testing.T) {
	skipIfMissingCreds(t)
	addr, _ := startRPCServer(t, map[string]string{"NYLAS_GRANT_ID": ""})

	conn, resp, err := websocket.DefaultDialer.Dial("ws://"+addr+"/ws?token="+rpcTestToken, nil)
	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		t.Fatalf("dial with query token: %v", err)
	}
	_ = conn.Close()
}

func TestCLI_RPC_Auth_OriginRejected(t *testing.T) {
	skipIfMissingCreds(t)
	addr, tok := startRPCServer(t, map[string]string{"NYLAS_GRANT_ID": ""})

	conn, resp, err := websocket.DefaultDialer.Dial("ws://"+addr+"/ws", http.Header{
		"Authorization": {"Bearer " + tok},
		"Origin":        {"http://evil.example"},
	})
	if conn != nil {
		_ = conn.Close()
	}
	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err == nil {
		t.Fatal("expected websocket dial with cross-origin header to fail")
	}
	if resp == nil {
		t.Fatal("expected HTTP response for rejected origin")
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestCLI_RPC_NonLoopbackRefused(t *testing.T) {
	if testBinary == "" {
		t.Skip("no binary")
	}

	_, stderr, err := runCLI("rpc", "serve", "--addr", "0.0.0.0:12345")
	if err == nil {
		t.Fatal("expected non-loopback bind without --allow-remote to fail")
	}
	if !strings.Contains(stderr, "refusing to bind") {
		t.Fatalf("stderr = %q, want substring %q", stderr, "refusing to bind")
	}
}

func TestCLI_RPC_UnknownMethod(t *testing.T) {
	skipIfMissingCreds(t)
	addr, tok := startRPCServer(t, map[string]string{"NYLAS_GRANT_ID": ""})
	conn := dialRPC(t, addr, tok)

	result := rpcCall(t, conn, 1, "does.not.exist", nil)
	if !result.IsError || result.ErrCode != -32601 {
		t.Fatalf("rpc error = (%v, %d), want (true, -32601)", result.IsError, result.ErrCode)
	}
}

func TestCLI_RPC_MissingRequiredParam(t *testing.T) {
	skipIfMissingCreds(t)
	addr, tok := startRPCServer(t, map[string]string{"NYLAS_GRANT_ID": ""})
	conn := dialRPC(t, addr, tok)

	result := rpcCall(t, conn, 2, "email.get", map[string]any{"grant_id": "x"})
	if !result.IsError || result.ErrCode != -32602 {
		t.Fatalf("rpc error = (%v, %d), want (true, -32602)", result.IsError, result.ErrCode)
	}
}

func TestCLI_RPC_MalformedJSON(t *testing.T) {
	skipIfMissingCreds(t)
	addr, tok := startRPCServer(t, map[string]string{"NYLAS_GRANT_ID": ""})
	conn := dialRPC(t, addr, tok)

	if err := conn.WriteMessage(websocket.TextMessage, []byte("{not valid json")); err != nil {
		t.Fatalf("write malformed json: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read parse error response: %v", err)
	}
	var resp struct {
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(msg, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error.Code != -32700 {
		t.Fatalf("error.code = %d, want -32700", resp.Error.Code)
	}
	if string(resp.ID) != "null" {
		t.Fatalf("id = %s, want null", resp.ID)
	}
}

func TestCLI_RPC_BadVersion(t *testing.T) {
	skipIfMissingCreds(t)
	addr, tok := startRPCServer(t, map[string]string{"NYLAS_GRANT_ID": ""})
	conn := dialRPC(t, addr, tok)

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"id":1,"method":"email.list"}`)); err != nil {
		t.Fatalf("write bad-version request: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read bad-version response: %v", err)
	}
	var resp struct {
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(msg, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error.Code != -32600 {
		t.Fatalf("error.code = %d, want -32600", resp.Error.Code)
	}
}

func TestCLI_RPC_Notification_NoReply(t *testing.T) {
	skipIfMissingCreds(t)
	addr, tok := startRPCServer(t, map[string]string{"NYLAS_GRANT_ID": ""})
	conn := dialRPC(t, addr, tok)

	if err := conn.WriteJSON(map[string]any{
		"jsonrpc": "2.0",
		"method":  "client.focus",
		"params":  map[string]any{"focused": true},
	}); err != nil {
		t.Fatalf("write notification: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, _, err := conn.ReadMessage()
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("ReadMessage error = %v, want timeout", err)
	}
}
