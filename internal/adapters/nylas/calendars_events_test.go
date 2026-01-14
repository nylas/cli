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
		serverResponse map[string]interface{}
		statusCode     int
		wantCount      int
		wantErr        bool
	}{
		{
			name:       "returns events successfully",
			calendarID: "cal-123",
			params:     nil,
			serverResponse: map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":          "event-1",
						"calendar_id": "cal-123",
						"title":       "Team Meeting",
						"status":      "confirmed",
						"busy":        true,
						"when": map[string]interface{}{
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
						"when": map[string]interface{}{
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
			serverResponse: map[string]interface{}{
				"data": []interface{}{},
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

				response := map[string]interface{}{
					"data": []interface{}{},
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
			response := map[string]interface{}{
				"data": []map[string]interface{}{
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
		serverResponse map[string]interface{}
		statusCode     int
		wantErr        bool
		errContains    string
	}{
		{
			name:       "returns event successfully",
			eventID:    "event-123",
			calendarID: "cal-123",
			serverResponse: map[string]interface{}{
				"data": map[string]interface{}{
					"id":          "event-123",
					"calendar_id": "cal-123",
					"title":       "Important Meeting",
					"description": "Discuss Q1 goals",
					"location":    "Room 101",
					"status":      "confirmed",
					"busy":        true,
					"visibility":  "public",
					"when": map[string]interface{}{
						"start_time": 1704067200,
						"end_time":   1704070800,
						"object":     "timespan",
					},
					"participants": []map[string]interface{}{
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
			serverResponse: map[string]interface{}{
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

func TestHTTPClient_CreateEvent(t *testing.T) {
	tests := []struct {
		name           string
		request        *domain.CreateEventRequest
		expectedFields []string
		statusCode     int
		wantErr        bool
	}{
		{
			name: "creates basic event",
			request: &domain.CreateEventRequest{
				Title: "New Event",
				When: domain.EventWhen{
					StartTime: 1704067200,
					EndTime:   1704070800,
				},
			},
			expectedFields: []string{"title", "when"},
			statusCode:     http.StatusCreated,
			wantErr:        false,
		},
		{
			name: "creates event with all fields",
			request: &domain.CreateEventRequest{
				Title:       "Complete Event",
				Description: "A fully specified event",
				Location:    "Main Office",
				When: domain.EventWhen{
					StartTime: 1704067200,
					EndTime:   1704070800,
				},
				Busy:       true,
				Visibility: "private",
				Participants: []domain.Participant{
					{Person: domain.Person{Name: "Bob", Email: "bob@example.com"}},
				},
				Recurrence: []string{"RRULE:FREQ=WEEKLY;COUNT=10"},
				Conferencing: &domain.Conferencing{
					Provider: "Google Meet",
					Details: &domain.ConferencingDetails{
						MeetingCode: "abc-123",
					},
				},
				Reminders: &domain.Reminders{
					UseDefault: false,
					Overrides: []domain.Reminder{
						{ReminderMinutes: 30, ReminderMethod: "display"},
					},
				},
				Metadata: map[string]string{"source": "cli"},
			},
			expectedFields: []string{
				"title", "description", "location", "when",
				"busy", "visibility", "participants", "recurrence",
				"conferencing", "reminders", "metadata",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v3/grants/grant-123/events", r.URL.Path)
				assert.Equal(t, "cal-123", r.URL.Query().Get("calendar_id"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)

				for _, field := range tt.expectedFields {
					assert.Contains(t, body, field, "Missing field: %s", field)
				}

				response := map[string]interface{}{
					"data": map[string]interface{}{
						"id":          "new-event-123",
						"calendar_id": "cal-123",
						"title":       tt.request.Title,
					},
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			event, err := client.CreateEvent(ctx, "grant-123", "cal-123", tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "new-event-123", event.ID)
		})
	}
}

func TestHTTPClient_UpdateEvent(t *testing.T) {
	tests := []struct {
		name       string
		request    *domain.UpdateEventRequest
		wantFields []string
	}{
		{
			name: "updates title",
			request: func() *domain.UpdateEventRequest {
				title := "Updated Title"
				return &domain.UpdateEventRequest{Title: &title}
			}(),
			wantFields: []string{"title"},
		},
		{
			name: "updates description",
			request: func() *domain.UpdateEventRequest {
				desc := "New description"
				return &domain.UpdateEventRequest{Description: &desc}
			}(),
			wantFields: []string{"description"},
		},
		{
			name: "updates location",
			request: func() *domain.UpdateEventRequest {
				loc := "New Location"
				return &domain.UpdateEventRequest{Location: &loc}
			}(),
			wantFields: []string{"location"},
		},
		{
			name: "updates time",
			request: &domain.UpdateEventRequest{
				When: &domain.EventWhen{
					StartTime: 1704153600,
					EndTime:   1704157200,
				},
			},
			wantFields: []string{"when"},
		},
		{
			name: "updates busy status",
			request: func() *domain.UpdateEventRequest {
				busy := false
				return &domain.UpdateEventRequest{Busy: &busy}
			}(),
			wantFields: []string{"busy"},
		},
		{
			name: "updates visibility",
			request: func() *domain.UpdateEventRequest {
				vis := "private"
				return &domain.UpdateEventRequest{Visibility: &vis}
			}(),
			wantFields: []string{"visibility"},
		},
		{
			name: "updates participants",
			request: &domain.UpdateEventRequest{
				Participants: []domain.Participant{
					{Person: domain.Person{Name: "New Person", Email: "new@example.com"}},
				},
			},
			wantFields: []string{"participants"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PUT", r.Method)
				assert.Equal(t, "/v3/grants/grant-123/events/event-456", r.URL.Path)
				assert.Equal(t, "cal-123", r.URL.Query().Get("calendar_id"))

				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)

				for _, field := range tt.wantFields {
					assert.Contains(t, body, field, "Missing field: %s", field)
				}

				response := map[string]interface{}{
					"data": map[string]interface{}{
						"id":          "event-456",
						"calendar_id": "cal-123",
						"title":       "Updated",
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			event, err := client.UpdateEvent(ctx, "grant-123", "cal-123", "event-456", tt.request)

			require.NoError(t, err)
			assert.Equal(t, "event-456", event.ID)
		})
	}
}

func TestHTTPClient_DeleteEvent(t *testing.T) {
	tests := []struct {
		name       string
		eventID    string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "deletes with 200",
			eventID:    "event-123",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "deletes with 204",
			eventID:    "event-456",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "returns error for not found",
			eventID:    "nonexistent",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				expectedPath := "/v3/grants/grant-123/events/" + tt.eventID
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, "cal-123", r.URL.Query().Get("calendar_id"))

				w.WriteHeader(tt.statusCode)
				if tt.statusCode >= 400 {
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"error": map[string]string{"message": "not found"},
					})
				}
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			err := client.DeleteEvent(ctx, "grant-123", "cal-123", tt.eventID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
