package mcp

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestCreateToolSuccessResponse(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	tests := []struct {
		name   string
		id     any
		result map[string]any
	}{
		{
			name: "simple result",
			id:   1,
			result: map[string]any{
				"status": "success",
				"data":   "test",
			},
		},
		{
			name: "nested result",
			id:   "request-123",
			result: map[string]any{
				"grant_id": "grant-abc",
				"email":    "user@example.com",
				"provider": "google",
			},
		},
		{
			name:   "nil id",
			id:     nil,
			result: map[string]any{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := proxy.createToolSuccessResponse(tt.id, tt.result)

			if response == nil {
				t.Fatal("expected non-nil response")
			}

			// Parse the response to verify structure
			var parsed map[string]any
			if err := json.Unmarshal(response, &parsed); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			// Verify JSON-RPC structure
			if parsed["jsonrpc"] != "2.0" {
				t.Errorf("expected jsonrpc 2.0, got %v", parsed["jsonrpc"])
			}

			// Verify result contains content array
			result, ok := parsed["result"].(map[string]any)
			if !ok {
				t.Fatal("expected result to be a map")
			}

			content, ok := result["content"].([]any)
			if !ok {
				t.Fatal("expected content to be an array")
			}

			if len(content) == 0 {
				t.Fatal("expected at least one content item")
			}

			contentItem, ok := content[0].(map[string]any)
			if !ok {
				t.Fatal("expected content item to be a map")
			}

			if contentItem["type"] != "text" {
				t.Errorf("expected content type 'text', got %v", contentItem["type"])
			}
		})
	}
}

func TestCreateToolErrorResponse(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	tests := []struct {
		name    string
		id      any
		message string
	}{
		{
			name:    "simple error",
			id:      1,
			message: "Something went wrong",
		},
		{
			name:    "auth error",
			id:      "request-456",
			message: "No authenticated grants found. Please run 'nylas auth login' first.",
		},
		{
			name:    "nil id",
			id:      nil,
			message: "Unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := proxy.createToolErrorResponse(tt.id, tt.message)

			if response == nil {
				t.Fatal("expected non-nil response")
			}

			// Parse the response
			var parsed map[string]any
			if err := json.Unmarshal(response, &parsed); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			// Verify JSON-RPC structure
			if parsed["jsonrpc"] != "2.0" {
				t.Errorf("expected jsonrpc 2.0, got %v", parsed["jsonrpc"])
			}

			// Verify result contains isError: true
			result, ok := parsed["result"].(map[string]any)
			if !ok {
				t.Fatal("expected result to be a map")
			}

			if result["isError"] != true {
				t.Error("expected isError to be true")
			}

			// Verify content contains the error message
			content, ok := result["content"].([]any)
			if !ok || len(content) == 0 {
				t.Fatal("expected content array with items")
			}

			contentItem, ok := content[0].(map[string]any)
			if !ok {
				t.Fatal("expected content item to be a map")
			}

			if contentItem["text"] != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, contentItem["text"])
			}
		})
	}
}

func TestCreateErrorResponse(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	tests := []struct {
		name string
		req  *rpcRequest
		err  error
	}{
		{
			name: "with request",
			req: &rpcRequest{
				ID:     1,
				Method: "tools/call",
			},
			err: errors.New("upstream error"),
		},
		{
			name: "nil request",
			req:  nil,
			err:  errors.New("connection failed"),
		},
		{
			name: "string id",
			req: &rpcRequest{
				ID:     "req-789",
				Method: "initialize",
			},
			err: errors.New("initialization failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := proxy.createErrorResponse(tt.req, tt.err)

			if response == nil {
				t.Fatal("expected non-nil response")
			}

			// Parse the response
			var parsed map[string]any
			if err := json.Unmarshal(response, &parsed); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			// Verify JSON-RPC structure
			if parsed["jsonrpc"] != "2.0" {
				t.Errorf("expected jsonrpc 2.0, got %v", parsed["jsonrpc"])
			}

			// Verify error structure
			errorObj, ok := parsed["error"].(map[string]any)
			if !ok {
				t.Fatal("expected error to be a map")
			}

			if errorObj["code"].(float64) != -32603 {
				t.Errorf("expected error code -32603, got %v", errorObj["code"])
			}

			if errorObj["message"] != tt.err.Error() {
				t.Errorf("expected message %q, got %q", tt.err.Error(), errorObj["message"])
			}
		})
	}
}

func TestModifyToolsListResponse_NoGetGrantTool(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	// Response without get_grant tool
	response := []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"result": {
			"tools": [
				{
					"name": "list_messages",
					"description": "List messages"
				}
			]
		}
	}`)

	modified := proxy.modifyToolsListResponse(response)

	// Should return the response unchanged
	var parsed map[string]any
	if err := json.Unmarshal(modified, &parsed); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	result := parsed["result"].(map[string]any)
	tools := result["tools"].([]any)

	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}

	tool := tools[0].(map[string]any)
	if tool["name"] != "list_messages" {
		t.Errorf("expected tool name 'list_messages', got %v", tool["name"])
	}
}

func TestModifyToolsListResponse_InvalidJSON(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	// Invalid JSON should return original
	response := []byte(`not valid json`)
	modified := proxy.modifyToolsListResponse(response)

	if string(modified) != string(response) {
		t.Error("expected invalid JSON to be returned unchanged")
	}
}

func TestModifyToolsListResponse_NoResult(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	// Response without result field
	response := []byte(`{"jsonrpc": "2.0", "id": 1}`)
	modified := proxy.modifyToolsListResponse(response)

	if string(modified) != string(response) {
		t.Error("expected response without result to be returned unchanged")
	}
}

func TestModifyInitializeResponse_AddsTimezoneGuidance(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	response := []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"result": {
			"protocolVersion": "2024-11-05",
			"capabilities": {},
			"serverInfo": {"name": "nylas-mcp"},
			"instructions": "You are a helpful assistant."
		}
	}`)

	modified := proxy.modifyInitializeResponse(response)

	var parsed map[string]any
	if err := json.Unmarshal(modified, &parsed); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	result := parsed["result"].(map[string]any)
	instructions := result["instructions"].(string)

	// Verify timezone guidance was added
	if len(instructions) <= len("You are a helpful assistant.") {
		t.Error("expected instructions to be extended with timezone guidance")
	}

	// Should contain timezone-related content
	if !containsSubstring(instructions, "Timezone") {
		t.Error("expected instructions to contain 'Timezone'")
	}
}

func TestModifyInitializeResponse_InvalidJSON(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	response := []byte(`invalid json`)
	modified := proxy.modifyInitializeResponse(response)

	if string(modified) != string(response) {
		t.Error("expected invalid JSON to be returned unchanged")
	}
}

func TestModifyInitializeResponse_NoResult(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	response := []byte(`{"jsonrpc": "2.0", "id": 1}`)
	modified := proxy.modifyInitializeResponse(response)

	if string(modified) != string(response) {
		t.Error("expected response without result to be returned unchanged")
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
