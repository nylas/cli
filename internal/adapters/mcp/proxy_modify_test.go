package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProxy_SetGrantStore(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	// Initially nil
	if proxy.grantStore != nil {
		t.Error("expected grantStore to be nil initially")
	}

	// Set grant store
	store := &mockGrantStore{}
	proxy.SetGrantStore(store)

	if proxy.grantStore == nil {
		t.Error("expected grantStore to be set")
	}
}

func TestProxy_modifyInitializeResponse(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	tests := []struct {
		name                 string
		response             string
		wantTimezoneGuidance bool
	}{
		{
			name: "adds timezone guidance to initialize response",
			response: `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"serverInfo": {"name": "nylas"},
					"instructions": "Nylas MCP server instructions."
				}
			}`,
			wantTimezoneGuidance: true,
		},
		{
			name: "handles empty instructions",
			response: `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"serverInfo": {"name": "nylas"}
				}
			}`,
			wantTimezoneGuidance: true,
		},
		{
			name:                 "handles invalid JSON",
			response:             `not valid json`,
			wantTimezoneGuidance: false,
		},
		{
			name: "handles missing result",
			response: `{
				"jsonrpc": "2.0",
				"id": 1,
				"error": {"code": -1, "message": "error"}
			}`,
			wantTimezoneGuidance: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := proxy.modifyInitializeResponse([]byte(tt.response))

			hasGuidance := strings.Contains(string(result), "Timezone Consistency")
			if hasGuidance != tt.wantTimezoneGuidance {
				t.Errorf("modifyInitializeResponse() timezone guidance = %v, want %v", hasGuidance, tt.wantTimezoneGuidance)
			}

			if tt.wantTimezoneGuidance {
				// Verify key guidance points are present
				if !strings.Contains(string(result), "epoch_to_datetime") {
					t.Error("expected guidance to mention epoch_to_datetime tool")
				}
				// Should contain the detected timezone
				if !strings.Contains(string(result), "user's local timezone is") {
					t.Error("expected guidance to include detected timezone")
				}
			}
		})
	}
}

func TestProxy_modifyToolsListResponse(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	tests := []struct {
		name              string
		response          string
		wantEmailOptional bool
		wantDescModified  bool
	}{
		{
			name: "modifies get_grant to make email optional",
			response: `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"tools": [
						{
							"name": "get_grant",
							"description": "Look up grant by email address.",
							"inputSchema": {
								"type": "object",
								"properties": {
									"email": {"type": "string", "description": "the email address"}
								},
								"required": ["email"]
							}
						},
						{
							"name": "list_messages",
							"description": "List messages",
							"inputSchema": {
								"type": "object",
								"properties": {},
								"required": ["grant_id"]
							}
						}
					]
				}
			}`,
			wantEmailOptional: true,
			wantDescModified:  true,
		},
		{
			name: "handles empty required array",
			response: `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"tools": [
						{
							"name": "get_grant",
							"description": "Look up grant",
							"inputSchema": {
								"type": "object",
								"properties": {"email": {"type": "string"}},
								"required": []
							}
						}
					]
				}
			}`,
			wantEmailOptional: true,
			wantDescModified:  true,
		},
		{
			name: "preserves other tools unchanged",
			response: `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"tools": [
						{
							"name": "list_messages",
							"description": "List messages",
							"inputSchema": {
								"type": "object",
								"required": ["grant_id"]
							}
						}
					]
				}
			}`,
			wantEmailOptional: false,
			wantDescModified:  false,
		},
		{
			name:              "handles invalid JSON",
			response:          `not json`,
			wantEmailOptional: false,
			wantDescModified:  false,
		},
		{
			name:              "handles missing result",
			response:          `{"jsonrpc":"2.0","id":1}`,
			wantEmailOptional: false,
			wantDescModified:  false,
		},
		{
			name:              "handles missing tools",
			response:          `{"jsonrpc":"2.0","id":1,"result":{}}`,
			wantEmailOptional: false,
			wantDescModified:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := proxy.modifyToolsListResponse([]byte(tt.response))

			// Parse result
			var resp map[string]any
			if err := json.Unmarshal(result, &resp); err != nil {
				if tt.wantEmailOptional || tt.wantDescModified {
					t.Fatalf("failed to parse result: %v", err)
				}
				return // Expected for invalid JSON test
			}

			// Find get_grant tool if it exists
			resultObj, ok := resp["result"].(map[string]any)
			if !ok {
				return
			}

			tools, ok := resultObj["tools"].([]any)
			if !ok {
				return
			}

			for _, tool := range tools {
				toolMap, ok := tool.(map[string]any)
				if !ok {
					continue
				}

				name, _ := toolMap["name"].(string)
				if name != "get_grant" {
					continue
				}

				// Check if email is optional (not in required)
				inputSchema, ok := toolMap["inputSchema"].(map[string]any)
				if ok {
					required, _ := inputSchema["required"].([]any)
					emailRequired := false
					for _, r := range required {
						if r == "email" {
							emailRequired = true
							break
						}
					}
					if tt.wantEmailOptional && emailRequired {
						t.Error("expected email to be optional, but it's still required")
					}
				}

				// Check if description was modified
				desc, _ := toolMap["description"].(string)
				hasModifiedDesc := strings.Contains(desc, "default authenticated grant")
				if tt.wantDescModified && !hasModifiedDesc {
					t.Error("expected description to be modified")
				}
			}
		})
	}
}
