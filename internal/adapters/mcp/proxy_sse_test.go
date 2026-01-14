package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestProxy_forward_SSE(t *testing.T) {
	t.Parallel()

	// Create a mock server that returns SSE
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"sse\":true}}\n\n"))
	}))
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL

	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
	response, err := proxy.forward(t.Context(), request, nil)

	if err != nil {
		t.Fatalf("forward failed: %v", err)
	}

	// Verify SSE response was parsed
	var resp map[string]any
	if err := json.Unmarshal(response, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result to be a map, got %T", resp["result"])
	}
	if result["sse"] != true {
		t.Errorf("expected sse true, got %v", result["sse"])
	}
}

func TestProxy_readSSE(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-key", "us")

	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantJSON bool
	}{
		{
			name:     "single message",
			input:    "data: {\"id\":1}\n\n",
			wantLen:  1,
			wantJSON: true,
		},
		{
			name:     "multiple messages",
			input:    "data: {\"id\":1}\n\ndata: {\"id\":2}\n\n",
			wantLen:  2,
			wantJSON: true,
		},
		{
			name:     "empty",
			input:    "",
			wantLen:  0,
			wantJSON: false,
		},
		{
			name:     "with comments",
			input:    ": comment\ndata: {\"id\":1}\n\n",
			wantLen:  1,
			wantJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := proxy.readSSE(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("readSSE failed: %v", err)
			}

			if tt.wantLen == 0 {
				if result != nil {
					t.Errorf("expected nil result, got %s", string(result))
				}
				return
			}

			if tt.wantJSON {
				if !json.Valid(result) {
					t.Errorf("expected valid JSON, got %s", string(result))
				}
			}
		})
	}
}

// mockGrantStore implements ports.GrantStore for testing.
type mockGrantStore struct {
	grants       []domain.GrantInfo
	defaultGrant string
}

func (m *mockGrantStore) SaveGrant(info domain.GrantInfo) error {
	m.grants = append(m.grants, info)
	return nil
}

func (m *mockGrantStore) GetGrant(grantID string) (*domain.GrantInfo, error) {
	for _, g := range m.grants {
		if g.ID == grantID {
			return &g, nil
		}
	}
	return nil, domain.ErrGrantNotFound
}

func (m *mockGrantStore) GetGrantByEmail(email string) (*domain.GrantInfo, error) {
	for _, g := range m.grants {
		if g.Email == email {
			return &g, nil
		}
	}
	return nil, domain.ErrGrantNotFound
}

func (m *mockGrantStore) ListGrants() ([]domain.GrantInfo, error) {
	return m.grants, nil
}

func (m *mockGrantStore) DeleteGrant(grantID string) error {
	return nil
}

func (m *mockGrantStore) SetDefaultGrant(grantID string) error {
	m.defaultGrant = grantID
	return nil
}

func (m *mockGrantStore) GetDefaultGrant() (string, error) {
	if m.defaultGrant == "" {
		return "", domain.ErrNoDefaultGrant
	}
	return m.defaultGrant, nil
}

func (m *mockGrantStore) ClearGrants() error {
	m.grants = nil
	m.defaultGrant = ""
	return nil
}
