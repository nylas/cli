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

func TestHTTPClient_CreateSchedulerSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/sessions", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "config-123", body["configuration_id"])

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"session_id":       "session-abc",
				"configuration_id": "config-123",
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
	req := &domain.CreateSchedulerSessionRequest{
		ConfigurationID: "config-123",
	}
	session, err := client.CreateSchedulerSession(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "session-abc", session.SessionID)
	assert.Equal(t, "config-123", session.ConfigurationID)
}

func TestHTTPClient_GetSchedulerSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/sessions/session-123", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"session_id":       "session-123",
				"configuration_id": "config-456",
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
	session, err := client.GetSchedulerSession(ctx, "session-123")

	require.NoError(t, err)
	assert.Equal(t, "session-123", session.SessionID)
}

// Booking Tests

func TestHTTPClient_GetBooking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/bookings/booking-123", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"booking_id": "booking-123",
				"title":      "Interview with John Doe",
				"status":     "confirmed",
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
	booking, err := client.GetBooking(ctx, "booking-123")

	require.NoError(t, err)
	assert.Equal(t, "booking-123", booking.BookingID)
	assert.Equal(t, "Interview with John Doe", booking.Title)
	assert.Equal(t, "confirmed", booking.Status)
}

func TestHTTPClient_ListBookings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/bookings", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "config-123", r.URL.Query().Get("configuration_id"))

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"booking_id": "booking-1",
					"title":      "Meeting 1",
					"status":     "confirmed",
				},
				{
					"booking_id": "booking-2",
					"title":      "Meeting 2",
					"status":     "pending",
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
	bookings, err := client.ListBookings(ctx, "config-123")

	require.NoError(t, err)
	assert.Len(t, bookings, 2)
	assert.Equal(t, "booking-1", bookings[0].BookingID)
	assert.Equal(t, "confirmed", bookings[0].Status)
}

// Note: Bookings are created through the scheduler session flow, not directly

func TestHTTPClient_ConfirmBooking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/bookings/booking-123", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"booking_id": "booking-123",
				"status":     "confirmed",
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
	req := &domain.ConfirmBookingRequest{}
	booking, err := client.ConfirmBooking(ctx, "booking-123", req)

	require.NoError(t, err)
	assert.Equal(t, "booking-123", booking.BookingID)
	assert.Equal(t, "confirmed", booking.Status)
}

func TestHTTPClient_RescheduleBooking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/bookings/booking-456/reschedule", r.URL.Path)
		assert.Equal(t, "PATCH", r.Method)

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"booking_id": "booking-456",
				"status":     "confirmed",
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
	req := &domain.RescheduleBookingRequest{
		StartTime: 1704067200,
		EndTime:   1704070800,
	}
	booking, err := client.RescheduleBooking(ctx, "booking-456", req)

	require.NoError(t, err)
	assert.Equal(t, "booking-456", booking.BookingID)
}

func TestHTTPClient_CancelBooking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/bookings/booking-cancel", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "cancelled", r.URL.Query().Get("reason"))

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.CancelBooking(ctx, "booking-cancel", "cancelled")

	require.NoError(t, err)
}

// Scheduler Page Tests

func TestHTTPClient_GetSchedulerPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/pages/page-123", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]interface{}{
			"data": map[string]interface{}{
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

		response := map[string]interface{}{
			"data": []map[string]interface{}{
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

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New Page", body["name"])

		response := map[string]interface{}{
			"data": map[string]interface{}{
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

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Updated Page", body["name"])

		response := map[string]interface{}{
			"data": map[string]interface{}{
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

				response := map[string]interface{}{
					"data": []map[string]interface{}{},
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

				response := map[string]interface{}{
					"data": map[string]interface{}{
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

// Mock Client Tests
