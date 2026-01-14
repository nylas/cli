package mcp

import (
	"encoding/json"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestProxy_handleLocalToolCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		request      string
		grantStore   *mockGrantStore
		defaultGrant string
		wantHandled  bool
		wantGrantID  string
		wantEmail    string
		wantError    bool
	}{
		{
			name:    "returns default grant when no email provided",
			request: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_grant","arguments":{}}}`,
			grantStore: &mockGrantStore{
				grants: []domain.GrantInfo{
					{ID: "grant-123", Email: "user@example.com", Provider: "google"},
				},
			},
			defaultGrant: "grant-123",
			wantHandled:  true,
			wantGrantID:  "grant-123",
			wantEmail:    "user@example.com",
		},
		{
			name:    "returns first grant when no default set",
			request: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_grant","arguments":{}}}`,
			grantStore: &mockGrantStore{
				grants: []domain.GrantInfo{
					{ID: "first-grant", Email: "first@example.com", Provider: "google"},
					{ID: "second-grant", Email: "second@example.com", Provider: "microsoft"},
				},
			},
			defaultGrant: "",
			wantHandled:  true,
			wantGrantID:  "first-grant",
			wantEmail:    "first@example.com",
		},
		{
			name:    "passes through when email is provided",
			request: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_grant","arguments":{"email":"other@example.com"}}}`,
			grantStore: &mockGrantStore{
				grants: []domain.GrantInfo{
					{ID: "grant-123", Email: "user@example.com", Provider: "google"},
				},
			},
			defaultGrant: "grant-123",
			wantHandled:  false,
		},
		{
			name:         "passes through for non-get_grant tools",
			request:      `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_messages","arguments":{}}}`,
			grantStore:   &mockGrantStore{},
			defaultGrant: "",
			wantHandled:  false,
		},
		{
			name:         "passes through for non-tools/call methods",
			request:      `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
			grantStore:   &mockGrantStore{},
			defaultGrant: "",
			wantHandled:  false,
		},
		{
			name:         "returns error when no grants exist",
			request:      `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_grant","arguments":{}}}`,
			grantStore:   &mockGrantStore{},
			defaultGrant: "",
			wantHandled:  true,
			wantError:    true,
		},
		{
			name:         "passes through when no grant store configured",
			request:      `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_grant","arguments":{}}}`,
			grantStore:   nil,
			defaultGrant: "",
			wantHandled:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy := NewProxy("test-api-key", "us")
			if tt.grantStore != nil {
				proxy.SetGrantStore(tt.grantStore)
			}
			if tt.defaultGrant != "" {
				proxy.SetDefaultGrant(tt.defaultGrant)
			}

			// Parse the request to pass as *rpcRequest
			var req rpcRequest
			_ = json.Unmarshal([]byte(tt.request), &req)
			response, handled := proxy.handleLocalToolCall(&req)

			if handled != tt.wantHandled {
				t.Errorf("expected handled=%v, got %v", tt.wantHandled, handled)
			}

			if !tt.wantHandled {
				return
			}

			// Parse the response
			var resp struct {
				JSONRPC string `json:"jsonrpc"`
				ID      any    `json:"id"`
				Result  struct {
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
					IsError bool `json:"isError"`
				} `json:"result"`
			}
			if err := json.Unmarshal(response, &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			if resp.JSONRPC != "2.0" {
				t.Errorf("expected jsonrpc '2.0', got '%s'", resp.JSONRPC)
			}

			if tt.wantError {
				if !resp.Result.IsError {
					t.Error("expected isError=true")
				}
				return
			}

			if len(resp.Result.Content) == 0 {
				t.Fatal("expected content in response")
			}

			// Parse the text content as JSON
			var grantResult struct {
				GrantID  string `json:"grant_id"`
				Email    string `json:"email"`
				Provider string `json:"provider"`
			}
			if err := json.Unmarshal([]byte(resp.Result.Content[0].Text), &grantResult); err != nil {
				t.Fatalf("failed to parse grant result: %v", err)
			}

			if grantResult.GrantID != tt.wantGrantID {
				t.Errorf("expected grant_id '%s', got '%s'", tt.wantGrantID, grantResult.GrantID)
			}
			if grantResult.Email != tt.wantEmail {
				t.Errorf("expected email '%s', got '%s'", tt.wantEmail, grantResult.Email)
			}
		})
	}
}
