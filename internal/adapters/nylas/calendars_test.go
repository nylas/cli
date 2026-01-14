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

func TestHTTPClient_GetCalendars(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse map[string]interface{}
		statusCode     int
		wantCount      int
		wantErr        bool
	}{
		{
			name: "returns calendars successfully",
			serverResponse: map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":          "cal-primary",
						"grant_id":    "grant-123",
						"name":        "Personal Calendar",
						"description": "My primary calendar",
						"is_primary":  true,
						"timezone":    "America/New_York",
						"hex_color":   "#0066FF",
						"read_only":   false,
					},
					{
						"id":         "cal-work",
						"grant_id":   "grant-123",
						"name":       "Work Calendar",
						"is_primary": false,
						"timezone":   "America/Chicago",
						"read_only":  false,
					},
				},
			},
			statusCode: http.StatusOK,
			wantCount:  2,
			wantErr:    false,
		},
		{
			name: "returns empty list",
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
				assert.Equal(t, "/v3/grants/grant-123/calendars", r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			calendars, err := client.GetCalendars(ctx, "grant-123")

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, calendars, tt.wantCount)
		})
	}
}

func TestHTTPClient_GetCalendar(t *testing.T) {
	tests := []struct {
		name           string
		calendarID     string
		serverResponse map[string]interface{}
		statusCode     int
		wantErr        bool
		errContains    string
	}{
		{
			name:       "returns calendar successfully",
			calendarID: "cal-123",
			serverResponse: map[string]interface{}{
				"data": map[string]interface{}{
					"id":          "cal-123",
					"grant_id":    "grant-123",
					"name":        "Test Calendar",
					"description": "A test calendar",
					"location":    "Room 101",
					"timezone":    "UTC",
					"hex_color":   "#FF0000",
					"is_primary":  false,
					"read_only":   true,
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "returns error for not found",
			calendarID: "nonexistent",
			serverResponse: map[string]interface{}{
				"error": map[string]string{"message": "calendar not found"},
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
				expectedPath := "/v3/grants/grant-123/calendars/" + tt.calendarID
				assert.Equal(t, expectedPath, r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			calendar, err := client.GetCalendar(ctx, "grant-123", tt.calendarID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.calendarID, calendar.ID)
		})
	}
}

func TestHTTPClient_CreateCalendar(t *testing.T) {
	tests := []struct {
		name           string
		request        *domain.CreateCalendarRequest
		expectedFields map[string]string
		statusCode     int
		wantErr        bool
	}{
		{
			name: "creates calendar with name only",
			request: &domain.CreateCalendarRequest{
				Name: "New Calendar",
			},
			expectedFields: map[string]string{"name": "New Calendar"},
			statusCode:     http.StatusCreated,
			wantErr:        false,
		},
		{
			name: "creates calendar with all fields",
			request: &domain.CreateCalendarRequest{
				Name:        "Full Calendar",
				Description: "A calendar with all fields",
				Location:    "Conference Room A",
				Timezone:    "America/Los_Angeles",
			},
			expectedFields: map[string]string{
				"name":        "Full Calendar",
				"description": "A calendar with all fields",
				"location":    "Conference Room A",
				"timezone":    "America/Los_Angeles",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v3/grants/grant-123/calendars", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)

				for key, expectedValue := range tt.expectedFields {
					assert.Equal(t, expectedValue, body[key], "Field %s mismatch", key)
				}

				response := map[string]interface{}{
					"data": map[string]interface{}{
						"id":       "new-cal-123",
						"grant_id": "grant-123",
						"name":     tt.request.Name,
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
			calendar, err := client.CreateCalendar(ctx, "grant-123", tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "new-cal-123", calendar.ID)
			assert.Equal(t, tt.request.Name, calendar.Name)
		})
	}
}

func TestHTTPClient_UpdateCalendar(t *testing.T) {
	tests := []struct {
		name       string
		calendarID string
		request    *domain.UpdateCalendarRequest
		wantFields []string
	}{
		{
			name:       "updates name",
			calendarID: "cal-123",
			request: func() *domain.UpdateCalendarRequest {
				name := "Updated Name"
				return &domain.UpdateCalendarRequest{Name: &name}
			}(),
			wantFields: []string{"name"},
		},
		{
			name:       "updates description",
			calendarID: "cal-123",
			request: func() *domain.UpdateCalendarRequest {
				desc := "New description"
				return &domain.UpdateCalendarRequest{Description: &desc}
			}(),
			wantFields: []string{"description"},
		},
		{
			name:       "updates multiple fields",
			calendarID: "cal-123",
			request: func() *domain.UpdateCalendarRequest {
				name := "Updated"
				desc := "Updated desc"
				loc := "New Location"
				tz := "Europe/London"
				color := "#00FF00"
				return &domain.UpdateCalendarRequest{
					Name:        &name,
					Description: &desc,
					Location:    &loc,
					Timezone:    &tz,
					HexColor:    &color,
				}
			}(),
			wantFields: []string{"name", "description", "location", "timezone", "hex_color"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PUT", r.Method)
				expectedPath := "/v3/grants/grant-123/calendars/" + tt.calendarID
				assert.Equal(t, expectedPath, r.URL.Path)

				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)

				for _, field := range tt.wantFields {
					assert.Contains(t, body, field, "Missing field: %s", field)
				}

				response := map[string]interface{}{
					"data": map[string]interface{}{
						"id":       tt.calendarID,
						"grant_id": "grant-123",
						"name":     "Updated",
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
			calendar, err := client.UpdateCalendar(ctx, "grant-123", tt.calendarID, tt.request)

			require.NoError(t, err)
			assert.Equal(t, tt.calendarID, calendar.ID)
		})
	}
}

func TestHTTPClient_DeleteCalendar(t *testing.T) {
	tests := []struct {
		name       string
		calendarID string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "deletes with 200",
			calendarID: "cal-123",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "deletes with 204",
			calendarID: "cal-456",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "returns error for not found",
			calendarID: "nonexistent",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				expectedPath := "/v3/grants/grant-123/calendars/" + tt.calendarID
				assert.Equal(t, expectedPath, r.URL.Path)

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
			err := client.DeleteCalendar(ctx, "grant-123", tt.calendarID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHTTPClient_GetCalendars_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		response    map[string]interface{}
		errContains string
	}{
		{
			name:       "handles 401 unauthorized",
			statusCode: http.StatusUnauthorized,
			response: map[string]interface{}{
				"error": map[string]string{"message": "Invalid API key"},
			},
			errContains: "Invalid API key",
		},
		{
			name:       "handles 403 forbidden",
			statusCode: http.StatusForbidden,
			response: map[string]interface{}{
				"error": map[string]string{"message": "Access denied to calendar"},
			},
			errContains: "Access denied",
		},
		{
			name:       "handles 500 server error",
			statusCode: http.StatusInternalServerError,
			response: map[string]interface{}{
				"error": map[string]string{"message": "Internal server error"},
			},
			errContains: "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			_, err := client.GetCalendars(ctx, "grant-123")

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}
