package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDPoP implements ports.DPoP for testing.
type mockDPoP struct {
	proof string
	err   error
}

func (m *mockDPoP) GenerateProof(method, url, accessToken string) (string, error) {
	return m.proof, m.err
}

func (m *mockDPoP) Thumbprint() string {
	return "test-thumbprint"
}

func TestSetDPoPProof(t *testing.T) {
	tests := []struct {
		name       string
		proof      string
		proofErr   error
		wantHeader string
		wantErr    bool
	}{
		{
			name:       "sets DPoP header on success",
			proof:      "test-proof-jwt",
			wantHeader: "test-proof-jwt",
		},
		{
			name:     "returns error on failure",
			proofErr: errTestDPoP,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &AccountClient{
				dpop: &mockDPoP{proof: tt.proof, err: tt.proofErr},
			}

			req := httptest.NewRequest(http.MethodPost, "https://example.com/test", nil)
			err := client.setDPoPProof(req, http.MethodPost, "https://example.com/test", "token")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := req.Header.Get("DPoP"); got != tt.wantHeader {
				t.Errorf("DPoP header = %q, want %q", got, tt.wantHeader)
			}
		})
	}
}

var errTestDPoP = &testError{msg: "dpop generation failed"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

func TestParseErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantMsg    string
		wantErrIs  error
	}{
		{
			name:       "parses error with code and message",
			statusCode: 400,
			body:       `{"error":{"code":"invalid_request","message":"bad input"}}`,
			wantMsg:    "invalid_request: bad input",
		},
		{
			name:       "parses error with message only",
			statusCode: 500,
			body:       `{"error":{"message":"internal error"}}`,
			wantMsg:    "internal error",
		},
		{
			name:       "falls back to raw body",
			statusCode: 502,
			body:       "Bad Gateway",
			wantMsg:    "Bad Gateway",
		},
		{
			name:       "truncates long body",
			statusCode: 500,
			body:       string(make([]byte, 300)),
			wantMsg:    "", // truncated to 200 chars
		},
		{
			name:       "classifies invalid session",
			statusCode: 401,
			body:       `{"error":{"code":"INVALID_SESSION","message":"Invalid or expired session"}}`,
			wantMsg:    "INVALID_SESSION: Invalid or expired session",
			wantErrIs:  domain.ErrDashboardSessionExpired,
		},
		{
			name:       "classifies code-only invalid session",
			statusCode: 401,
			body:       `{"error":{"code":"INVALID_SESSION"}}`,
			wantMsg:    "INVALID_SESSION",
			wantErrIs:  domain.ErrDashboardSessionExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseErrorResponse(tt.statusCode, []byte(tt.body))
			dashErr, ok := err.(*domain.DashboardAPIError)
			if !ok {
				t.Fatalf("expected *domain.DashboardAPIError, got %T", err)
			}
			if dashErr.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", dashErr.StatusCode, tt.statusCode)
			}
			if tt.wantMsg != "" && dashErr.ServerMsg != tt.wantMsg {
				t.Errorf("ServerMsg = %q, want %q", dashErr.ServerMsg, tt.wantMsg)
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("expected errors.Is(%v), got %v", tt.wantErrIs, err)
			}
		})
	}
}

func TestUnwrapEnvelope(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantKey string
		wantErr bool
	}{
		{
			name:    "unwraps data field",
			body:    `{"request_id":"abc","success":true,"data":{"name":"test"}}`,
			wantKey: "name",
		},
		{
			name:    "returns body as-is when no data field",
			body:    `{"name":"test"}`,
			wantKey: "name",
		},
		{
			name:    "returns error on invalid JSON",
			body:    "not json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := unwrapEnvelope([]byte(tt.body))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var parsed map[string]any
			if jsonErr := json.Unmarshal(result, &parsed); jsonErr != nil {
				t.Fatalf("result is not valid JSON: %v", jsonErr)
			}
			if _, ok := parsed[tt.wantKey]; !ok {
				t.Errorf("result missing key %q: %s", tt.wantKey, string(result))
			}
		})
	}
}

func TestDashboardAPIError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     domain.DashboardAPIError
		wantStr string
	}{
		{
			name:    "with message",
			err:     domain.DashboardAPIError{StatusCode: 400, ServerMsg: "bad request"},
			wantStr: "dashboard API error (HTTP 400): bad request",
		},
		{
			name:    "without message",
			err:     domain.DashboardAPIError{StatusCode: 500},
			wantStr: "dashboard API error (HTTP 500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantStr {
				t.Errorf("Error() = %q, want %q", got, tt.wantStr)
			}
		})
	}
}

func TestDoPostAndGet_Integration(t *testing.T) {
	// Set up a test server that returns a valid envelope response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify DPoP header was set
		if dpop := r.Header.Get("DPoP"); dpop == "" {
			t.Error("DPoP header not set")
		}

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"request_id": "test-123",
			"success":    true,
			"data":       map[string]string{"id": "app-1", "name": "Test App"},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &AccountClient{
		baseURL:    server.URL,
		httpClient: server.Client(),
		dpop:       &mockDPoP{proof: "test-proof"},
	}

	t.Run("doPost decodes response", func(t *testing.T) {
		var result map[string]string
		err := client.doPost(context.Background(), "/test", nil, nil, "token", &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result["id"] != "app-1" {
			t.Errorf("result[id] = %q, want %q", result["id"], "app-1")
		}
	})

	t.Run("doGet decodes response", func(t *testing.T) {
		var result map[string]string
		err := client.doGet(context.Background(), "/test", nil, "token", &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result["name"] != "Test App" {
			t.Errorf("result[name] = %q, want %q", result["name"], "Test App")
		}
	})
}

func TestDoPost_PreservesNestedDataField(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"request_id": "test-456",
			"success":    true,
			"data": map[string]any{
				"id": "app-1",
				"data": map[string]string{
					"inner": "value",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &AccountClient{
		baseURL:    server.URL,
		httpClient: server.Client(),
		dpop:       &mockDPoP{proof: "test-proof"},
	}

	var result struct {
		ID   string            `json:"id"`
		Data map[string]string `json:"data"`
	}

	err := client.doPost(context.Background(), "/test", nil, nil, "token", &result)
	require.NoError(t, err)
	assert.Equal(t, "app-1", result.ID)
	assert.Equal(t, map[string]string{"inner": "value"}, result.Data)
}
