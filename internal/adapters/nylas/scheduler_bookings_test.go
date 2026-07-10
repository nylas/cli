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

// bookingTestServer answers the session-mint POST /v3/scheduling/sessions that
// precedes every booking call (returning sessionID), then delegates all other
// requests to bookingHandler. It asserts the booking request is authorized by
// the minted session token, not the application API key — the auth model the
// Nylas v3 booking endpoints actually require.
func bookingTestServer(t *testing.T, sessionID string, bookingHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v3/scheduling/sessions" {
			assert.Equal(t, http.MethodPost, r.Method)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.NotEmpty(t, body["configuration_id"], "session mint must send configuration_id")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"session_id": sessionID},
			})
			return
		}
		assert.Equal(t, "Bearer "+sessionID, r.Header.Get("Authorization"),
			"booking requests must use the session token, not the API key")
		bookingHandler(w, r)
	}))
}

func TestHTTPClient_GetBooking(t *testing.T) {
	server := bookingTestServer(t, "session-abc", func(w http.ResponseWriter, r *http.Request) {
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
	})
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	booking, err := client.GetBooking(ctx, "config-1", "booking-123")

	require.NoError(t, err)
	assert.Equal(t, "booking-123", booking.BookingID)
	assert.Equal(t, "Interview with John Doe", booking.Title)
	assert.Equal(t, "confirmed", booking.Status)
}

// Note: Bookings are created through the scheduler session flow, not directly

func TestHTTPClient_ConfirmBooking(t *testing.T) {
	server := bookingTestServer(t, "session-abc", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/bookings/booking-123", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)

		// The spec requires salt + status in the body; cancellation_reason is
		// optional. Verify they are sent (regression for the missing salt/
		// mislabelled reason bug).
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "s4lt", body["salt"])
		assert.Equal(t, "confirmed", body["status"])
		assert.Equal(t, "changed my mind", body["cancellation_reason"])
		_, hasReason := body["reason"]
		assert.False(t, hasReason, "must not send the non-spec 'reason' field")

		response := map[string]any{
			"data": map[string]any{
				"booking_id": "booking-123",
				"status":     "confirmed",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			response)
	})
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	req := &domain.ConfirmBookingRequest{
		Salt:               "s4lt",
		Status:             "confirmed",
		CancellationReason: "changed my mind",
	}
	booking, err := client.ConfirmBooking(ctx, "config-1", "booking-123", req)

	require.NoError(t, err)
	assert.Equal(t, "booking-123", booking.BookingID)
	assert.Equal(t, "confirmed", booking.Status)
}

func TestHTTPClient_ConfirmBooking_CancelledReturnsNoDataEnvelope(t *testing.T) {
	// Declining a pending booking (status "cancelled") returns the no-data delete
	// envelope, not a booking body; the adapter must not try to decode a booking.
	server := bookingTestServer(t, "session-abc", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req-1"}) // no "data"
	})
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	booking, err := client.ConfirmBooking(context.Background(), "config-1", "booking-9",
		&domain.ConfirmBookingRequest{Salt: "s4lt", Status: "cancelled"})

	require.NoError(t, err)
	assert.Equal(t, "booking-9", booking.BookingID)
	assert.Equal(t, "cancelled", booking.Status)
}

func TestHTTPClient_ConfirmBooking_Validation(t *testing.T) {
	// The adapter is the API boundary; it must reject spec-invalid confirm
	// requests (nil, missing salt, bad status) before hitting the network,
	// regardless of what the CLI/RPC callers validate.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s — invalid confirm should not reach the API", r.URL.Path)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)
	ctx := context.Background()

	tests := []struct {
		name string
		req  *domain.ConfirmBookingRequest
	}{
		{name: "nil request", req: nil},
		{name: "missing salt", req: &domain.ConfirmBookingRequest{Status: "confirmed"}},
		{name: "empty status", req: &domain.ConfirmBookingRequest{Salt: "s4lt"}},
		{name: "invalid status", req: &domain.ConfirmBookingRequest{Salt: "s4lt", Status: "maybe"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.ConfirmBooking(ctx, "config-1", "booking-1", tt.req)
			require.Error(t, err)
		})
	}
}

func TestHTTPClient_CreateSchedulerSession_RejectsInvalidRequest(t *testing.T) {
	// The adapter must reject spec-invalid session requests before any HTTP
	// call: nil, missing configuration_id/slug, and out-of-range TTL. The
	// substring assertions pin the failure to Validate() — a transport error
	// from the unreachable base URL must not satisfy the test.
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL("http://127.0.0.1:0") // any request would fail loudly

	ctx := context.Background()
	for name, tc := range map[string]struct {
		req     *domain.CreateSchedulerSessionRequest
		wantErr string
	}{
		"nil request":       {req: nil, wantErr: "session request is required"},
		"no config or slug": {req: &domain.CreateSchedulerSessionRequest{TimeToLive: 10}, wantErr: "configuration_id or slug is required"},
		"ttl above cap":     {req: &domain.CreateSchedulerSessionRequest{ConfigurationID: "config-1", TimeToLive: 31}, wantErr: "time_to_live"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := client.CreateSchedulerSession(ctx, tc.req)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestHTTPClient_CancelBooking_EmptyReasonOmitsField(t *testing.T) {
	// With an empty reason the cancellation_reason field must be omitted from
	// the body (omitempty), matching the spec where it is optional.
	server := bookingTestServer(t, "session-abc", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		_, present := body["cancellation_reason"]
		assert.False(t, present, "cancellation_reason must be absent when reason is empty")
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	require.NoError(t, client.CancelBooking(context.Background(), "config-1", "booking-1", ""))
}

func TestHTTPClient_RescheduleBooking(t *testing.T) {
	// PATCH returns only a request_id; the adapter reads the booking back for the
	// full record (status/title/etc.).
	server := bookingTestServer(t, "session-abc", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/bookings/booking-456", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPatch:
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req-1"})
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"booking_id": "booking-456", "status": "confirmed", "title": "Interview"},
			})
		default:
			t.Errorf("unexpected method %s on reschedule", r.Method)
		}
	})
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	req := &domain.RescheduleBookingRequest{StartTime: 1704067200, EndTime: 1704070800}
	booking, err := client.RescheduleBooking(ctx, "config-1", "booking-456", req)

	require.NoError(t, err)
	assert.Equal(t, "booking-456", booking.BookingID)
	// Full record comes from the read-back...
	assert.Equal(t, "confirmed", booking.Status)
	assert.Equal(t, "Interview", booking.Title)
	// ...but the booking model has no times, so they come from the request.
	assert.Equal(t, int64(1704067200), booking.StartTime.Unix())
	assert.Equal(t, int64(1704070800), booking.EndTime.Unix())
}

func TestHTTPClient_RescheduleBooking_FallsBackWhenReadBackNotFound(t *testing.T) {
	// If the read-back GET races ahead of propagation (404), a successful
	// reschedule must still succeed, reflecting the requested times.
	server := bookingTestServer(t, "session-abc", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req-1"})
		case http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	req := &domain.RescheduleBookingRequest{StartTime: 1704067200, EndTime: 1704070800}
	booking, err := client.RescheduleBooking(context.Background(), "config-1", "booking-456", req)

	require.NoError(t, err)
	assert.Equal(t, "booking-456", booking.BookingID)
	assert.Equal(t, int64(1704067200), booking.StartTime.Unix())
	assert.Equal(t, int64(1704070800), booking.EndTime.Unix())
}

func TestHTTPClient_RescheduleBooking_SurfacesReadBackError(t *testing.T) {
	// Only a propagation 404 may silently fall back to the requested times; any
	// other read-back failure (auth, 5xx, rate limit) must surface the typed
	// partial success so a possibly-diverged server state is never reported as
	// a clean, verified result. 403 keeps the test out of the 5xx retry loop.
	server := bookingTestServer(t, "session-abc", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req-1"})
		case http.MethodGet:
			w.WriteHeader(http.StatusForbidden)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	req := &domain.RescheduleBookingRequest{StartTime: 1704067200, EndTime: 1704070800}
	booking, err := client.RescheduleBooking(context.Background(), "config-1", "booking-456", req)

	require.Error(t, err)
	// The typed sentinel lets the CLI/RPC boundaries report the reschedule as
	// applied while surfacing the verification failure...
	assert.ErrorIs(t, err, domain.ErrBookingReadBackFailed)
	// ...using the partial record the port contract promises alongside it.
	require.NotNil(t, booking)
	assert.Equal(t, "booking-456", booking.BookingID)
	assert.Equal(t, int64(1704067200), booking.StartTime.Unix())
	assert.Equal(t, int64(1704070800), booking.EndTime.Unix())
}

func TestHTTPClient_CancelBooking(t *testing.T) {
	server := bookingTestServer(t, "session-abc", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/bookings/booking-cancel", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		// The spec carries the reason in the JSON body as cancellation_reason,
		// not as a query param (regression for the dropped-reason bug).
		assert.Empty(t, r.URL.Query().Get("reason"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "user cancelled", body["cancellation_reason"])

		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.CancelBooking(ctx, "config-1", "booking-cancel", "user cancelled")

	require.NoError(t, err)
}

// URL Escaping Tests - ensures special characters are properly escaped

func TestHTTPClient_RescheduleBooking_URLEscaping(t *testing.T) {
	tests := []struct {
		name         string
		bookingID    string
		expectedPath string
	}{
		{
			name:         "simple ID",
			bookingID:    "booking-123",
			expectedPath: "/v3/scheduling/bookings/booking-123",
		},
		{
			name:         "ID with slash",
			bookingID:    "booking/123",
			expectedPath: "/v3/scheduling/bookings/booking%2F123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := bookingTestServer(t, "session-abc", func(w http.ResponseWriter, r *http.Request) {
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
			})
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			req := &domain.RescheduleBookingRequest{
				StartTime: 1704067200,
				EndTime:   1704070800,
			}
			_, err := client.RescheduleBooking(ctx, "config-1", tt.bookingID, req)
			require.NoError(t, err)
		})
	}
}

func TestHTTPClient_CancelBooking_URLEscaping(t *testing.T) {
	tests := []struct {
		name         string
		bookingID    string
		reason       string
		expectedPath string
	}{
		{
			name:         "simple values",
			bookingID:    "booking-123",
			reason:       "cancelled",
			expectedPath: "/v3/scheduling/bookings/booking-123",
		},
		{
			name:         "reason with special chars",
			bookingID:    "booking-123",
			reason:       "conflict & reschedule",
			expectedPath: "/v3/scheduling/bookings/booking-123",
		},
		{
			name:         "ID with slash",
			bookingID:    "booking/123",
			reason:       "test",
			expectedPath: "/v3/scheduling/bookings/booking%2F123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := bookingTestServer(t, "session-abc", func(w http.ResponseWriter, r *http.Request) {
				// Check path escaping (use RawPath for escaped, Path for unescaped)
				if tt.expectedPath != r.URL.Path {
					assert.Equal(t, tt.expectedPath, r.URL.RawPath)
				}
				// The reason travels in the JSON body, not the query string.
				assert.Empty(t, r.URL.RawQuery)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, tt.reason, body["cancellation_reason"])

				w.WriteHeader(http.StatusNoContent)
			})
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			err := client.CancelBooking(ctx, "config-1", tt.bookingID, tt.reason)
			require.NoError(t, err)
		})
	}
}

// Mock Client Tests
