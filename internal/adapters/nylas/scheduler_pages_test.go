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

// Scheduler Page Tests

func TestHTTPClient_GetSchedulerPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/pages/page-123", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id":   "page-123",
				"name": "Scheduling Page",
				"slug": "schedule-me",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	page, err := client.GetSchedulerPage(ctx, "page-123")

	require.NoError(t, err)
	assert.Equal(t, "page-123", page.ID)
	assert.Equal(t, "Scheduling Page", page.Name)
	assert.Equal(t, "schedule-me", page.Slug)
}

func TestHTTPClient_ListSchedulerPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/pages", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": []map[string]any{
				{
					"id":   "page-1",
					"name": "Page 1",
					"slug": "page-1",
				},
				{
					"id":   "page-2",
					"name": "Page 2",
					"slug": "page-2",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	pages, err := client.ListSchedulerPages(ctx)

	require.NoError(t, err)
	assert.Len(t, pages, 2)
	assert.Equal(t, "page-1", pages[0].ID)
}

func TestHTTPClient_CreateSchedulerPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/pages", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New Page", body["name"])

		response := map[string]any{
			"data": map[string]any{
				"id":   "page-new",
				"name": "New Page",
				"slug": "new-page",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	req := &domain.CreateSchedulerPageRequest{
		Name: "New Page",
	}
	page, err := client.CreateSchedulerPage(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "page-new", page.ID)
	assert.Equal(t, "New Page", page.Name)
}

func TestHTTPClient_UpdateSchedulerPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/pages/page-789", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Updated Page", body["name"])

		response := map[string]any{
			"data": map[string]any{
				"id":   "page-789",
				"name": "Updated Page",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	req := &domain.UpdateSchedulerPageRequest{
		Name: strPtr("Updated Page"),
	}
	page, err := client.UpdateSchedulerPage(ctx, "page-789", req)

	require.NoError(t, err)
	assert.Equal(t, "page-789", page.ID)
	assert.Equal(t, "Updated Page", page.Name)
}

func TestHTTPClient_DeleteSchedulerPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/pages/page-delete", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.DeleteSchedulerPage(ctx, "page-delete")

	require.NoError(t, err)
}

// URL Escaping Tests - ensures special characters are properly escaped

func TestHTTPClient_ListBookings_URLEscaping(t *testing.T) {
	tests := []struct {
		name          string
		configID      string
		expectedQuery string
	}{
		{
			name:          "simple ID",
			configID:      "config-123",
			expectedQuery: "config-123",
		},
		{
			name:          "ID with spaces",
			configID:      "config 123",
			expectedQuery: "config+123",
		},
		{
			name:          "ID with ampersand",
			configID:      "config&123",
			expectedQuery: "config%26123",
		},
		{
			name:          "ID with equals",
			configID:      "config=123",
			expectedQuery: "config%3D123",
		},
		{
			name:          "ID with special chars",
			configID:      "config/123?foo=bar",
			expectedQuery: "config%2F123%3Ffoo%3Dbar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the raw query string contains properly escaped value
				assert.Contains(t, r.URL.RawQuery, "configuration_id="+tt.expectedQuery)

				response := map[string]any{
					"data": []map[string]any{},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			_, err := client.ListBookings(ctx, tt.configID)
			require.NoError(t, err)
		})
	}
}

func TestHTTPClient_RescheduleBooking_URLEscaping(t *testing.T) {
	tests := []struct {
		name         string
		bookingID    string
		expectedPath string
	}{
		{
			name:         "simple ID",
			bookingID:    "booking-123",
			expectedPath: "/v3/scheduling/bookings/booking-123/reschedule",
		},
		{
			name:         "ID with slash",
			bookingID:    "booking/123",
			expectedPath: "/v3/scheduling/bookings/booking%2F123/reschedule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check the path - RawPath only set for escaped chars, Path always has decoded value
				if r.URL.RawPath != "" {
					assert.Equal(t, tt.expectedPath, r.URL.RawPath)
				} else {
					assert.Equal(t, tt.expectedPath, r.URL.Path)
				}

				response := map[string]any{
					"data": map[string]any{
						"booking_id": tt.bookingID,
						"status":     "confirmed",
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
			req := &domain.RescheduleBookingRequest{
				StartTime: 1704067200,
				EndTime:   1704070800,
			}
			_, err := client.RescheduleBooking(ctx, tt.bookingID, req)
			require.NoError(t, err)
		})
	}
}

func TestHTTPClient_CancelBooking_URLEscaping(t *testing.T) {
	tests := []struct {
		name          string
		bookingID     string
		reason        string
		expectedPath  string
		expectedQuery string
	}{
		{
			name:          "simple values",
			bookingID:     "booking-123",
			reason:        "cancelled",
			expectedPath:  "/v3/scheduling/bookings/booking-123",
			expectedQuery: "reason=cancelled",
		},
		{
			name:          "reason with spaces",
			bookingID:     "booking-123",
			reason:        "user requested cancellation",
			expectedPath:  "/v3/scheduling/bookings/booking-123",
			expectedQuery: "reason=user+requested+cancellation",
		},
		{
			name:          "reason with special chars",
			bookingID:     "booking-123",
			reason:        "conflict & reschedule",
			expectedPath:  "/v3/scheduling/bookings/booking-123",
			expectedQuery: "reason=conflict+%26+reschedule",
		},
		{
			name:          "ID with slash",
			bookingID:     "booking/123",
			reason:        "test",
			expectedPath:  "/v3/scheduling/bookings/booking%2F123",
			expectedQuery: "reason=test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check path escaping (use RawPath for escaped, Path for unescaped)
				if tt.expectedPath != r.URL.Path {
					assert.Equal(t, tt.expectedPath, r.URL.RawPath)
				}
				// Check query escaping
				assert.Equal(t, tt.expectedQuery, r.URL.RawQuery)

				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			err := client.CancelBooking(ctx, tt.bookingID, tt.reason)
			require.NoError(t, err)
		})
	}
}
