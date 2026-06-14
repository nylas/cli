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
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_ImportEvents(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Import hits the dedicated /events/import path, not /events.
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v3/grants/grant-1/events/import", r.URL.Path)
		gotQuery = r.URL.RawQuery

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "evt-1", "calendar_id": "primary", "title": "Kickoff"},
			},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	events, err := client.ImportEvents(context.Background(), "grant-1", &domain.EventQueryParams{
		CalendarID: "primary",
		Start:      1735689600,
		End:        1767225600,
	})

	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "evt-1", events[0].ID)
	// calendar_id is required by the API and must be forwarded as a query param.
	assert.Contains(t, gotQuery, "calendar_id=primary")
	assert.Contains(t, gotQuery, "start=1735689600")
}

func TestHTTPClient_ImportEvents_RequiresCalendar(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	t.Run("missing calendar id", func(t *testing.T) {
		_, err := client.ImportEvents(context.Background(), "grant-1", &domain.EventQueryParams{})
		assert.Error(t, err)
	})

	t.Run("nil params", func(t *testing.T) {
		_, err := client.ImportEvents(context.Background(), "grant-1", nil)
		assert.Error(t, err)
	})

	t.Run("missing grant", func(t *testing.T) {
		_, err := client.ImportEvents(context.Background(), "", &domain.EventQueryParams{CalendarID: "primary"})
		assert.Error(t, err)
	})
}
