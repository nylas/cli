package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDispatcher_Dispatch(t *testing.T) {
	tests := []struct {
		name       string
		raw        []byte
		register   func(*Dispatcher)
		wantNil    bool
		wantID     string
		wantResult string
		wantCode   int
		wantMsg    string
		wantData   string
		wantLog    string
	}{
		{
			name: "valid request routes to handler and returns result",
			raw:  []byte(`{"jsonrpc":"2.0","id":1,"method":"ping","params":{"name":"Ada"}}`),
			register: func(d *Dispatcher) {
				d.Register("ping", func(ctx context.Context, params json.RawMessage) (any, error) {
					return map[string]string{"status": "ok"}, nil
				})
			},
			wantID:     "1",
			wantResult: `{"status":"ok"}`,
		},
		{
			name:    "unknown notification returns nil",
			raw:     []byte(`{"jsonrpc":"2.0","method":"ping","params":{"name":"Ada"}}`),
			wantNil: true,
		},
		{
			name:     "unknown method returns method not found",
			raw:      []byte(`{"jsonrpc":"2.0","id":"abc","method":"missing"}`),
			wantID:   `"abc"`,
			wantCode: MethodNotFound,
			wantMsg:  "method not found",
		},
		{
			name:     "malformed JSON returns parse error",
			raw:      []byte(`{"jsonrpc":"2.0","id":1,"method":`),
			wantID:   "null",
			wantCode: ParseError,
			wantMsg:  "parse error",
		},
		{
			name:     "empty method returns invalid request",
			raw:      []byte(`{"jsonrpc":"2.0","id":1,"method":""}`),
			wantID:   "null",
			wantCode: InvalidRequest,
			wantMsg:  "invalid request",
		},
		{
			name:     "missing jsonrpc version returns invalid request",
			raw:      []byte(`{"id":1,"method":"ping"}`),
			wantID:   "null",
			wantCode: InvalidRequest,
			wantMsg:  "invalid request",
		},
		{
			name:     "wrong jsonrpc version returns invalid request",
			raw:      []byte(`{"jsonrpc":"1.0","id":1,"method":"ping"}`),
			wantID:   "null",
			wantCode: InvalidRequest,
			wantMsg:  "invalid request",
		},
		{
			name: "plain handler error maps to internal error",
			raw:  []byte(`{"jsonrpc":"2.0","id":1,"method":"boom"}`),
			register: func(d *Dispatcher) {
				d.Register("boom", func(ctx context.Context, params json.RawMessage) (any, error) {
					return nil, errors.New("database unavailable")
				})
			},
			wantID:   "1",
			wantCode: InternalError,
			wantMsg:  "internal error",
			wantLog:  "database unavailable",
		},
		{
			name: "RPCError handler error passes through",
			raw:  []byte(`{"jsonrpc":"2.0","id":1,"method":"badParams"}`),
			register: func(d *Dispatcher) {
				d.Register("badParams", func(ctx context.Context, params json.RawMessage) (any, error) {
					return nil, fmt.Errorf("checking params: %w", NewRPCError(InvalidParams, "bad params", map[string]string{"field": "email"}))
				})
			},
			wantID:   "1",
			wantCode: InvalidParams,
			wantMsg:  "bad params",
			wantData: `{"field":"email"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			var loggedErr error
			if tt.wantLog != "" {
				d.LogError = func(err error) {
					loggedErr = err
				}
			}
			if tt.register != nil {
				tt.register(d)
			}

			got := d.Dispatch(context.Background(), tt.raw)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("Dispatch() = %s, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("Dispatch() = nil, want response")
			}

			var resp Response
			if err := json.Unmarshal(got, &resp); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			if resp.JSONRPC != "2.0" {
				t.Errorf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
			}

			if tt.wantID != "" {
				wantID := `"id":` + tt.wantID
				if !strings.Contains(string(got), wantID) {
					t.Errorf("response = %s, want %s", got, wantID)
				}
			}

			if tt.wantCode != 0 {
				if resp.Error == nil {
					t.Fatal("Error = nil, want RPC error")
				}
				if resp.Error.Code != tt.wantCode {
					t.Errorf("Error.Code = %d, want %d", resp.Error.Code, tt.wantCode)
				}
				if resp.Error.Message != tt.wantMsg {
					t.Errorf("Error.Message = %q, want %q", resp.Error.Message, tt.wantMsg)
				}
				if tt.wantData != "" {
					gotData, err := json.Marshal(resp.Error.Data)
					if err != nil {
						t.Fatalf("marshal error data: %v", err)
					}
					if string(gotData) != tt.wantData {
						t.Errorf("Error.Data = %s, want %s", gotData, tt.wantData)
					}
				}
				if tt.wantLog != "" {
					if loggedErr == nil {
						t.Fatal("LogError was not called")
					}
					if !strings.Contains(loggedErr.Error(), tt.wantLog) {
						t.Errorf("logged error = %q, want it to contain %q", loggedErr, tt.wantLog)
					}
				}
				return
			}

			if resp.Error != nil {
				t.Fatalf("Error = %#v, want nil", resp.Error)
			}
			gotResult, err := json.Marshal(resp.Result)
			if err != nil {
				t.Fatalf("marshal result: %v", err)
			}
			if string(gotResult) != tt.wantResult {
				t.Errorf("Result = %s, want %s", gotResult, tt.wantResult)
			}
		})
	}
}

func TestDispatcher_Dispatch_NotificationRunsHandler(t *testing.T) {
	d := NewDispatcher()
	called := false

	d.Register("ping", func(ctx context.Context, params json.RawMessage) (any, error) {
		called = true
		return "ignored", nil
	})

	got := d.Dispatch(context.Background(), []byte(`{"jsonrpc":"2.0","method":"ping","params":{"name":"Ada"}}`))
	if got != nil {
		t.Fatalf("Dispatch() = %s, want nil", got)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestDispatcher_Dispatch_ParseErrorSerializesNullID(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
	}{
		{name: "malformed object", raw: []byte(`{"jsonrpc":"2.0","id":1,"method":`)},
		{name: "not json", raw: []byte(`nope`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewDispatcher().Dispatch(context.Background(), tt.raw)
			if !strings.Contains(string(got), `"id":null`) {
				t.Fatalf("Dispatch() = %s, want serialized null id", got)
			}
		})
	}
}

func TestDispatcher_Dispatch_ForwardsRawParams(t *testing.T) {
	d := NewDispatcher()
	wantParams := `{"email":"ada@example.com","limit":5}`
	called := false

	d.Register("inspect", func(ctx context.Context, params json.RawMessage) (any, error) {
		called = true
		if string(params) != wantParams {
			t.Errorf("params = %s, want %s", params, wantParams)
		}
		return "ok", nil
	})

	got := d.Dispatch(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"inspect","params":`+wantParams+`}`))
	if got == nil {
		t.Fatal("Dispatch() = nil, want response")
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestNewNotification(t *testing.T) {
	got, err := NewNotification("progress", map[string]int{"percent": 50})
	if err != nil {
		t.Fatalf("NewNotification() error = %v", err)
	}

	var notif Notification
	if err := json.Unmarshal(got, &notif); err != nil {
		t.Fatalf("unmarshal notification: %v", err)
	}
	if notif.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", notif.JSONRPC, "2.0")
	}
	if notif.Method != "progress" {
		t.Errorf("Method = %q, want %q", notif.Method, "progress")
	}
	if string(got) != `{"jsonrpc":"2.0","method":"progress","params":{"percent":50}}` {
		t.Errorf("NewNotification() = %s", got)
	}
}
