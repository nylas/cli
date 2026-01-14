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

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "config-123", body["configuration_id"])

		response := map[string]any{
			"data": map[string]any{
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

		response := map[string]any{
			"data": map[string]any{
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

		response := map[string]any{
			"data": map[string]any{
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

		response := map[string]any{
			"data": []map[string]any{
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

		response := map[string]any{
			"data": map[string]any{
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

		response := map[string]any{
			"data": map[string]any{
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
