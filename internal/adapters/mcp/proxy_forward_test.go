package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProxy_forward(t *testing.T) {
	t.Parallel()

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Authorization header 'Bearer test-api-key', got '%s'", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		// Return a response
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Mcp-Session-Id", "test-session-123")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"test":"ok"}}`))
	}))
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL

	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
	response, err := proxy.forward(t.Context(), request, nil)

	if err != nil {
		t.Fatalf("forward failed: %v", err)
	}

	// Verify response
	var resp map[string]any
	if err := json.Unmarshal(response, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["jsonrpc"] != "2.0" {
		t.Errorf("expected jsonrpc '2.0', got '%v'", resp["jsonrpc"])
	}

	// Verify session ID was stored
	if proxy.sessionID != "test-session-123" {
		t.Errorf("expected sessionID 'test-session-123', got '%s'", proxy.sessionID)
	}
}

func TestProxy_forward_WithDefaultGrant(t *testing.T) {
	t.Parallel()

	// Create a mock server that verifies the grant_id is injected into tool calls
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the request body to verify grant_id was injected
		body, _ := io.ReadAll(r.Body)

		var req struct {
			Method string `json:"method"`
			Params struct {
				Arguments map[string]any `json:"arguments"`
			} `json:"params"`
		}
		if err := json.Unmarshal(body, &req); err == nil {
			if req.Method == "tools/call" {
				if grantID, ok := req.Params.Arguments["grant_id"].(string); !ok || grantID != "test-grant-456" {
					t.Errorf("expected grant_id 'test-grant-456' in arguments, got '%v'", req.Params.Arguments["grant_id"])
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"grant":"ok"}}`))
	}))
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL
	proxy.SetDefaultGrant("test-grant-456")

	// Test with a tools/call request
	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_messages","arguments":{}}}`)
	response, err := proxy.forward(t.Context(), request, nil)

	if err != nil {
		t.Fatalf("forward failed: %v", err)
	}

	if response == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestProxy_injectDefaultGrant(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")
	proxy.SetDefaultGrant("my-grant-id")

	tests := []struct {
		name       string
		input      string
		wantGrant  bool
		grantValue string
	}{
		{
			name:       "injects grant_id into tools/call for list_messages",
			input:      `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_messages","arguments":{}}}`,
			wantGrant:  true,
			grantValue: "my-grant-id",
		},
		{
			name:       "does not override existing grant_id",
			input:      `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_messages","arguments":{"grant_id":"existing"}}}`,
			wantGrant:  true,
			grantValue: "existing",
		},
		{
			name:       "does not override existing identifier",
			input:      `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_messages","arguments":{"identifier":"user@example.com"}}}`,
			wantGrant:  false,
			grantValue: "",
		},
		{
			name:      "ignores non-tools/call methods",
			input:     `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
			wantGrant: false,
		},
		{
			name:      "does not inject grant_id for epoch_to_datetime utility tool",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"epoch_to_datetime","arguments":{"batch":[{"epoch_time":1735063516,"timezone":"America/Los_Angeles"}]}}}`,
			wantGrant: false,
		},
		{
			name:      "does not inject grant_id for current_time utility tool",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"current_time","arguments":{"timezone":"America/Los_Angeles"}}}`,
			wantGrant: false,
		},
		{
			name:      "does not inject grant_id for datetime_to_epoch utility tool",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"datetime_to_epoch","arguments":{"batch":[{"date":"2024-12-24","time":"10:00:00","timezone":"America/Los_Angeles"}]}}}`,
			wantGrant: false,
		},
		{
			name:      "does not inject grant_id for availability (grant_id goes in participants)",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"participants":[]}}}}`,
			wantGrant: false,
		},
		{
			name:      "does not inject grant_id for confirm_send_message (validates message content only)",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"confirm_send_message","arguments":{"message_request":{"to":[{"email":"test@example.com"}]}}}}`,
			wantGrant: false,
		},
		{
			name:      "does not inject grant_id for confirm_send_draft (validates draft content only)",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"confirm_send_draft","arguments":{"grant_id":"abc","draft_id":"123"}}}`,
			wantGrant: false,
		},
		{
			name:       "injects grant_id for create_event",
			input:      `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"create_event","arguments":{}}}`,
			wantGrant:  true,
			grantValue: "my-grant-id",
		},
		{
			name:       "injects grant_id for list_calendars",
			input:      `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_calendars","arguments":{}}}`,
			wantGrant:  true,
			grantValue: "my-grant-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := proxy.injectDefaultGrant([]byte(tt.input), nil)

			var parsed struct {
				Params struct {
					Arguments map[string]any `json:"arguments"`
				} `json:"params"`
			}
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Fatalf("failed to parse result: %v", err)
			}

			grantID, hasGrant := parsed.Params.Arguments["grant_id"].(string)
			if tt.wantGrant {
				if !hasGrant {
					t.Error("expected grant_id in arguments")
				} else if grantID != tt.grantValue {
					t.Errorf("expected grant_id '%s', got '%s'", tt.grantValue, grantID)
				}
			}
		})
	}
}

func TestProxy_forward_Error(t *testing.T) {
	t.Parallel()

	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Bad request"))
	}))
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL

	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
	_, err := proxy.forward(t.Context(), request, nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProxy_forward_ModifiesToolsList(t *testing.T) {
	t.Parallel()

	// Create a mock server that returns a tools/list response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": 1,
			"result": {
				"tools": [
					{
						"name": "get_grant",
						"description": "Look up grant by email.",
						"inputSchema": {
							"type": "object",
							"properties": {"email": {"type": "string"}},
							"required": ["email"]
						}
					}
				]
			}
		}`))
	}))
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL

	// Send a tools/list request
	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	var req rpcRequest
	_ = json.Unmarshal(request, &req)
	response, err := proxy.forward(t.Context(), request, &req)
	if err != nil {
		t.Fatalf("forward failed: %v", err)
	}

	// Parse response and verify get_grant was modified
	var resp map[string]any
	if err := json.Unmarshal(response, &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	result := resp["result"].(map[string]any)
	tools := result["tools"].([]any)
	getGrantTool := tools[0].(map[string]any)

	// Verify email is no longer required
	inputSchema := getGrantTool["inputSchema"].(map[string]any)
	required, _ := inputSchema["required"].([]any)
	for _, r := range required {
		if r == "email" {
			t.Error("expected email to be removed from required, but it's still there")
		}
	}

	// Verify description was modified
	desc := getGrantTool["description"].(string)
	if !strings.Contains(desc, "default authenticated grant") {
		t.Error("expected description to be modified")
	}
}
