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

func TestHTTPClient_GetEvents(t *testing.T) {
	tests := []struct {
		name           string
		calendarID     string
		params         *domain.EventQueryParams
		serverResponse map[string]any
		statusCode     int
		wantCount      int
		wantErr        bool
	}{
		{
			name:       "returns events successfully",
			calendarID: "cal-123",
			params:     nil,
			serverResponse: map[string]any{
				"data": []map[string]any{
					{
						"id":          "event-1",
						"calendar_id": "cal-123",
						"title":       "Team Meeting",
						"status":      "confirmed",
						"busy":        true,
						"when": map[string]any{
							"start_time": 1704067200,
							"end_time":   1704070800,
							"object":     "timespan",
						},
					},
					{
						"id":          "event-2",
						"calendar_id": "cal-123",
						"title":       "Lunch",
						"status":      "confirmed",
						"busy":        false,
						"when": map[string]any{
							"start_time": 1704081600,
							"end_time":   1704085200,
							"object":     "timespan",
						},
					},
				},
			},
			statusCode: http.StatusOK,
			wantCount:  2,
			wantErr:    false,
		},
		{
			name:       "returns empty list",
			calendarID: "cal-456",
			params:     &domain.EventQueryParams{Limit: 10},
			serverResponse: map[string]any{
				"data": []any{},
			},
			statusCode: http.StatusOK,
			wantCount:  0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Contains(t, r.URL.Path, "/v3/grants/grant-123/events")
				assert.Equal(t, tt.calendarID, r.URL.Query().Get("calendar_id"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			events, err := client.GetEvents(ctx, "grant-123", tt.calendarID, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, events, tt.wantCount)
		})
	}
}

func TestHTTPClient_GetEventsWithCursor(t *testing.T) {
	tests := []struct {
		name          string
		params        *domain.EventQueryParams
		wantQueryKeys []string
	}{
		{
			name: "includes time range params",
			params: &domain.EventQueryParams{
				Limit: 50,
				Start: 1704067200,
				End:   1704153600,
			},
			wantQueryKeys: []string{"limit", "start", "end"},
		},
		{
			name: "includes title filter",
			params: &domain.EventQueryParams{
				Limit: 10,
				Title: "meeting",
			},
			wantQueryKeys: []string{"title"},
		},
		{
			name: "includes location filter",
			params: &domain.EventQueryParams{
				Limit:    10,
				Location: "Conference Room A",
			},
			wantQueryKeys: []string{"location"},
		},
		{
			name: "includes show_cancelled flag",
			params: &domain.EventQueryParams{
				Limit:         10,
				ShowCancelled: true,
			},
			wantQueryKeys: []string{"show_cancelled"},
		},
		{
			name: "includes expand_recurring flag",
			params: &domain.EventQueryParams{
				Limit:           10,
				ExpandRecurring: true,
			},
			wantQueryKeys: []string{"expand_recurring"},
		},
		{
			name: "includes busy filter",
			params: func() *domain.EventQueryParams {
				busy := true
				return &domain.EventQueryParams{
					Limit: 10,
					Busy:  &busy,
				}
			}(),
			wantQueryKeys: []string{"busy"},
		},
		{
			name: "includes order_by param",
			params: &domain.EventQueryParams{
				Limit:   10,
				OrderBy: "start",
			},
			wantQueryKeys: []string{"order_by"},
		},
		{
			name: "includes page token",
			params: &domain.EventQueryParams{
				Limit:     10,
				PageToken: "next-page-token",
			},
			wantQueryKeys: []string{"page_token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for _, key := range tt.wantQueryKeys {
					assert.NotEmpty(t, r.URL.Query().Get(key), "Missing query param: %s", key)
				}

				response := map[string]any{
					"data": []any{},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			_, _ = client.GetEventsWithCursor(ctx, "grant-123", "cal-123", tt.params)
		})
	}

	t.Run("returns pagination info", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]any{
				"data": []map[string]any{
					{"id": "event-1", "title": "Event"},
				},
				"next_cursor": "eyJsYXN0X2lkIjoiZXZlbnQtMSJ9",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		ctx := context.Background()
		result, err := client.GetEventsWithCursor(ctx, "grant-123", "cal-123", nil)

		require.NoError(t, err)
		assert.Equal(t, "eyJsYXN0X2lkIjoiZXZlbnQtMSJ9", result.Pagination.NextCursor)
		assert.True(t, result.Pagination.HasMore)
	})
}

func TestHTTPClient_GetEvent(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		calendarID     string
		serverResponse map[string]any
		statusCode     int
		wantErr        bool
		errContains    string
	}{
		{
			name:       "returns event successfully",
			eventID:    "event-123",
			calendarID: "cal-123",
			serverResponse: map[string]any{
				"data": map[string]any{
					"id":          "event-123",
					"calendar_id": "cal-123",
					"title":       "Important Meeting",
					"description": "Discuss Q1 goals",
					"location":    "Room 101",
					"status":      "confirmed",
					"busy":        true,
					"visibility":  "public",
					"when": map[string]any{
						"start_time": 1704067200,
						"end_time":   1704070800,
						"object":     "timespan",
					},
					"participants": []map[string]any{
						{
							"email":  "alice@example.com",
							"name":   "Alice",
							"status": "yes",
						},
					},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "returns error for not found",
			eventID:    "nonexistent",
			calendarID: "cal-123",
			serverResponse: map[string]any{
				"error": map[string]string{"message": "event not found"},
			},
			statusCode:  http.StatusNotFound,
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				expectedPath := "/v3/grants/grant-123/events/" + tt.eventID
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, tt.calendarID, r.URL.Query().Get("calendar_id"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			event, err := client.GetEvent(ctx, "grant-123", tt.calendarID, tt.eventID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.eventID, event.ID)
		})
	}
}
