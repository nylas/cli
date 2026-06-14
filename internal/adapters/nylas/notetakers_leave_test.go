//go:build !integration
// +build !integration

package nylas_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_LeaveNotetaker(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"accepts 200 OK", http.StatusOK},
		{"accepts 202 Accepted", http.StatusAccepted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// leave is a POST to the /leave sub-path, distinct from delete.
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/v3/grants/grant-1/notetakers/nt-1/leave", r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"request_id": "req-1",
					"data":       map[string]any{"id": "nt-1", "message": "left meeting"},
				})
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			err := client.LeaveNotetaker(context.Background(), "grant-1", "nt-1")
			require.NoError(t, err)
		})
	}
}

func TestHTTPClient_LeaveNotetaker_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req-1",
			"error":      map[string]any{"type": "not_found", "message": "notetaker not found"},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	err := client.LeaveNotetaker(context.Background(), "grant-1", "missing")
	assert.Error(t, err)
}
