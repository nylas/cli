package mcp

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetMCPEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		region   string
		expected string
	}{
		{"us", NylasMCPEndpointUS},
		{"US", NylasMCPEndpointUS},
		{"eu", NylasMCPEndpointEU},
		{"EU", NylasMCPEndpointEU},
		{"", NylasMCPEndpointUS},      // default to US
		{"other", NylasMCPEndpointUS}, // unknown defaults to US
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			got := GetMCPEndpoint(tt.region)
			if got != tt.expected {
				t.Errorf("GetMCPEndpoint(%q) = %q, want %q", tt.region, got, tt.expected)
			}
		})
	}
}

func TestNewProxy(t *testing.T) {
	t.Parallel()

	t.Run("US region", func(t *testing.T) {
		proxy := NewProxy("test-api-key", "us")
		if proxy == nil {
			t.Fatal("NewProxy returned nil")
		}
		if proxy.apiKey != "test-api-key" {
			t.Errorf("expected apiKey 'test-api-key', got '%s'", proxy.apiKey)
		}
		if proxy.endpoint != NylasMCPEndpointUS {
			t.Errorf("expected endpoint '%s', got '%s'", NylasMCPEndpointUS, proxy.endpoint)
		}
		if proxy.httpClient == nil {
			t.Error("httpClient is nil")
		}
	})

	t.Run("EU region", func(t *testing.T) {
		proxy := NewProxy("test-api-key", "eu")
		if proxy.endpoint != NylasMCPEndpointEU {
			t.Errorf("expected endpoint '%s', got '%s'", NylasMCPEndpointEU, proxy.endpoint)
		}
	})

	t.Run("empty region defaults to US", func(t *testing.T) {
		proxy := NewProxy("test-api-key", "")
		if proxy.endpoint != NylasMCPEndpointUS {
			t.Errorf("expected endpoint '%s', got '%s'", NylasMCPEndpointUS, proxy.endpoint)
		}
	})
}

func TestProxy_SetDefaultGrant(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	// Initially empty
	if proxy.defaultGrant != "" {
		t.Errorf("expected empty defaultGrant, got '%s'", proxy.defaultGrant)
	}

	// Set grant
	proxy.SetDefaultGrant("grant-123")
	if proxy.defaultGrant != "grant-123" {
		t.Errorf("expected defaultGrant 'grant-123', got '%s'", proxy.defaultGrant)
	}
}

func TestProxy_createErrorResponse(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-key", "us")

	tests := []struct {
		name   string
		req    *rpcRequest
		err    error
		wantID any
	}{
		{
			name:   "with numeric id",
			req:    &rpcRequest{JSONRPC: "2.0", ID: float64(1), Method: "test"},
			err:    http.ErrNotSupported,
			wantID: float64(1),
		},
		{
			name:   "with string id",
			req:    &rpcRequest{JSONRPC: "2.0", ID: "abc", Method: "test"},
			err:    http.ErrNotSupported,
			wantID: "abc",
		},
		{
			name:   "without id",
			req:    nil,
			err:    http.ErrNotSupported,
			wantID: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := proxy.createErrorResponse(tt.req, tt.err)

			var resp struct {
				JSONRPC string `json:"jsonrpc"`
				ID      any    `json:"id"`
				Error   struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}

			if err := json.Unmarshal(result, &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if resp.JSONRPC != "2.0" {
				t.Errorf("expected jsonrpc '2.0', got '%s'", resp.JSONRPC)
			}
			if resp.ID != tt.wantID {
				t.Errorf("expected id %v, got %v", tt.wantID, resp.ID)
			}
			if resp.Error.Code != -32603 {
				t.Errorf("expected error code -32603, got %d", resp.Error.Code)
			}
		})
	}
}
