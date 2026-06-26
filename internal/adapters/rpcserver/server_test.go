package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestServer_WebSocketAuthDispatchAndBroadcast(t *testing.T) {
	d := NewDispatcher()
	d.Register("echo", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return map[string]string{"message": p.Message}, nil
	})

	srv := NewServer(Config{Token: "secret-token"}, d)
	httpSrv := newHTTPTestServer(t, srv.handler())
	t.Cleanup(httpSrv.Close)
	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http") + "/ws"

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "missing token rejected", wantStatus: http.StatusUnauthorized},
		{name: "wrong token rejected", token: "wrong-token", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, resp, err := websocket.DefaultDialer.Dial(wsURL, authHeader(tt.token))
			if err == nil {
				t.Fatal("Dial() error = nil, want handshake failure")
			}
			if resp == nil {
				t.Fatal("Dial() response = nil, want HTTP response")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
		})
	}

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, authHeader("secret-token"))
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"jsonrpc":"2.0","id":1,"method":"echo","params":{"message":"hi"}}`)); err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}

	var response struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			Message string `json:"message"`
		} `json:"result"`
	}
	readJSON(t, conn, &response)
	if response.JSONRPC != "2.0" || response.ID != 1 || response.Result.Message != "hi" {
		t.Fatalf("response = %+v, want echo result", response)
	}

	if err := srv.Broadcast("message.received", map[string]string{"id": "msg-1"}); err != nil {
		t.Fatalf("Broadcast() error = %v", err)
	}

	var notification struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  struct {
			ID string `json:"id"`
		} `json:"params"`
	}
	readJSON(t, conn, &notification)
	if notification.JSONRPC != "2.0" || notification.Method != "message.received" || notification.Params.ID != "msg-1" {
		t.Fatalf("notification = %+v, want message.received msg-1", notification)
	}
}

func TestServer_ConcurrentClientWritesAndBroadcast(t *testing.T) {
	d := NewDispatcher()
	d.Register("echo", func(ctx context.Context, params json.RawMessage) (any, error) {
		time.Sleep(2 * time.Millisecond)
		return json.RawMessage(params), nil
	})

	srv := NewServer(Config{Token: "secret-token"}, d)
	httpSrv := newHTTPTestServer(t, srv.handler())
	t.Cleanup(httpSrv.Close)
	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http") + "/ws"

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, authHeader("secret-token"))
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
	if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}

	broadcastReceived := make(chan struct{})
	readErr := make(chan error, 1)
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				readErr <- err
				return
			}
			var envelope struct {
				Method string `json:"method"`
			}
			if err := json.Unmarshal(msg, &envelope); err != nil {
				readErr <- fmt.Errorf("unmarshal %s: %w", msg, err)
				return
			}
			if envelope.Method == "message.received" {
				close(broadcastReceived)
				return
			}
		}
	}()

	const writes = 25
	var wg sync.WaitGroup
	wg.Add(2)
	errs := make(chan error, 2)

	go func() {
		defer wg.Done()
		for i := range writes {
			msg := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"echo","params":{"i":%d}}`, i, i)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				errs <- fmt.Errorf("write request %d: %w", i, err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := range writes {
			if err := srv.Broadcast("message.received", map[string]int{"i": i}); err != nil {
				errs <- fmt.Errorf("broadcast %d: %w", i, err)
				return
			}
		}
	}()

	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	select {
	case <-broadcastReceived:
	case err := <-readErr:
		t.Fatalf("read message: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func authHeader(token string) http.Header {
	h := http.Header{}
	if token != "" {
		h.Set("Authorization", "Bearer "+token)
	}
	return h
}

func newHTTPTestServer(t *testing.T, h http.Handler) *httptest.Server {
	t.Helper()

	var srv *httptest.Server
	func() {
		defer func() {
			if r := recover(); r != nil {
				msg := fmt.Sprint(r)
				if strings.Contains(msg, "httptest: failed to listen on a port") && strings.Contains(msg, "operation not permitted") {
					t.Skipf("local TCP listener unavailable in this sandbox: %v", r)
				}
				panic(r)
			}
		}()
		srv = httptest.NewServer(h)
	}()
	return srv
}

func readJSON(t *testing.T, conn *websocket.Conn, v any) {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	if err := json.Unmarshal(msg, v); err != nil {
		t.Fatalf("unmarshal %s: %v", msg, err)
	}
}
