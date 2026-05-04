//go:build !integration
// +build !integration

package nylas_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_SendRSVP(t *testing.T) {
	tests := []struct {
		name           string
		request        *domain.SendRSVPRequest
		expectedStatus string
		statusCode     int
		wantErr        bool
	}{
		{
			name: "sends accept response",
			request: &domain.SendRSVPRequest{
				Status: "yes",
			},
			expectedStatus: "yes",
			statusCode:     http.StatusOK,
			wantErr:        false,
		},
		{
			name: "sends decline response with comment",
			request: &domain.SendRSVPRequest{
				Status:  "no",
				Comment: "I have a conflict",
			},
			expectedStatus: "no",
			statusCode:     http.StatusAccepted,
			wantErr:        false,
		},
		{
			name: "sends maybe response",
			request: &domain.SendRSVPRequest{
				Status: "maybe",
			},
			expectedStatus: "maybe",
			statusCode:     http.StatusOK,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v3/grants/grant-123/events/event-456/send-rsvp", r.URL.Path)
				assert.Equal(t, "cal-123", r.URL.Query().Get("calendar_id"))

				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				assert.Equal(t, tt.expectedStatus, body["status"])

				if tt.request.Comment != "" {
					assert.Equal(t, tt.request.Comment, body["comment"])
				}

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			err := client.SendRSVP(ctx, "grant-123", "cal-123", "event-456", tt.request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHTTPClient_GetFreeBusy(t *testing.T) {
	t.Run("returns free/busy information", func(t *testing.T) {
		now := time.Now()
		startTime := now.Unix()
		endTime := now.Add(24 * time.Hour).Unix()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/v3/grants/grant-123/calendars/free-busy", r.URL.Path)

			response := map[string]any{
				"data": []map[string]any{
					{
						"email": "user@example.com",
						"time_slots": []map[string]any{
							{
								"start_time": startTime,
								"end_time":   startTime + 3600,
								"status":     "busy",
							},
						},
					},
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
		req := &domain.FreeBusyRequest{
			StartTime: startTime,
			EndTime:   endTime,
			Emails:    []string{"user@example.com"},
		}
		result, err := client.GetFreeBusy(ctx, "grant-123", req)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestHTTPClient_GetAvailability(t *testing.T) {
	t.Run("returns availability slots", func(t *testing.T) {
		now := time.Now()
		startTime := now.Unix()
		endTime := now.Add(24 * time.Hour).Unix()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/v3/calendars/availability", r.URL.Path)

			response := map[string]any{
				"time_slots": []map[string]any{
					{
						"start_time": startTime + 7200,
						"end_time":   startTime + 10800,
						"emails":     []string{"user@example.com"},
					},
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
		req := &domain.AvailabilityRequest{
			StartTime:       startTime,
			EndTime:         endTime,
			DurationMinutes: 30,
			Participants: []domain.AvailabilityParticipant{
				{Email: "user@example.com"},
			},
		}
		result, err := client.GetAvailability(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// TestHTTPClient_SendRSVP_RejectsEmptyArgs pins the input-validation
// invariant on SendRSVP. Every other Get/Update method on this client
// validates required fields up-front; without these checks an empty
// grantID would form `/v3/grants//events//send-rsvp` and 404 silently
// upstream — a confusing failure for any future caller (e.g. CLI) that
// doesn't validate at its own boundary.
func TestHTTPClient_SendRSVP_RejectsEmptyArgs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		grantID    string
		calendarID string
		eventID    string
		req        *domain.SendRSVPRequest
		wantSubstr string
	}{
		{
			name:       "empty grant ID",
			grantID:    "",
			calendarID: "cal-1",
			eventID:    "evt-1",
			req:        &domain.SendRSVPRequest{Status: "yes"},
			wantSubstr: "grant ID",
		},
		{
			name:       "empty calendar ID",
			grantID:    "grant-1",
			calendarID: "",
			eventID:    "evt-1",
			req:        &domain.SendRSVPRequest{Status: "yes"},
			wantSubstr: "calendar ID",
		},
		{
			name:       "empty event ID",
			grantID:    "grant-1",
			calendarID: "cal-1",
			eventID:    "",
			req:        &domain.SendRSVPRequest{Status: "yes"},
			wantSubstr: "event ID",
		},
		{
			name:       "nil request",
			grantID:    "grant-1",
			calendarID: "cal-1",
			eventID:    "evt-1",
			req:        nil,
			wantSubstr: "nil",
		},
		{
			name:       "invalid status",
			grantID:    "grant-1",
			calendarID: "cal-1",
			eventID:    "evt-1",
			req:        &domain.SendRSVPRequest{Status: "bogus"},
			wantSubstr: "yes, no, maybe",
		},
		{
			name:       "empty status",
			grantID:    "grant-1",
			calendarID: "cal-1",
			eventID:    "evt-1",
			req:        &domain.SendRSVPRequest{Status: ""},
			wantSubstr: "yes, no, maybe",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			called := false
			server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				called = true
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			err := client.SendRSVP(context.Background(), tc.grantID, tc.calendarID, tc.eventID, tc.req)
			require.Error(t, err, "validation must fail closed before any HTTP request")
			assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tc.wantSubstr),
				"error %q must mention %q so callers can diagnose", err.Error(), tc.wantSubstr)
			assert.False(t, called, "SendRSVP must not issue any HTTP request when validation fails")
		})
	}
}

// TestHTTPClient_SendRSVP_OmitsEmptyComment pins the contract that an
// empty comment is NOT serialized into the JSON body. Nylas v3 treats a
// "comment":"" field as a literal empty comment-line, which Gmail then
// renders as a blank line under the attendee name in the organiser's
// notification email — visible UX rot from the user's perspective.
func TestHTTPClient_SendRSVP_OmitsEmptyComment(t *testing.T) {
	t.Parallel()

	var rawBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// io.ReadAll is required because (a) r.Body is a stream — a
		// single Read() is not guaranteed to fill the buffer, and (b)
		// r.ContentLength can be -1 for chunked requests, which would
		// panic make([]byte, -1).
		body, _ := io.ReadAll(r.Body)
		rawBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	err := client.SendRSVP(context.Background(), "grant-1", "cal-1", "evt-1", &domain.SendRSVPRequest{
		Status: "yes",
	})
	require.NoError(t, err)
	assert.NotContains(t, rawBody, `"comment"`,
		"empty comment must be omitted from the request body to avoid rendering a blank attendee comment in the organiser's notification; raw=%s", rawBody)
}

// TestHTTPClient_SendRSVP_PropagatesUpstreamError pins the 4xx/5xx
// surface area: a Nylas error during send-rsvp must surface as a real
// error (not nil). Air's handler relies on this to return 502 to the
// browser instead of a misleading 200.
func TestHTTPClient_SendRSVP_PropagatesUpstreamError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"calendar permission denied"}`))
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	err := client.SendRSVP(context.Background(), "grant-1", "cal-1", "evt-1", &domain.SendRSVPRequest{
		Status: "yes",
	})
	require.Error(t, err, "adapter must surface non-2xx responses as errors")
}

// TestHTTPClient_SendRSVP_ValidationErrorsWrapErrInvalidInput pins that
// every input-validation failure wraps domain.ErrInvalidInput so callers
// can classify the failure with errors.Is. Without the wrap, the rest of
// this client (which IS consistent) becomes impossible to align against —
// CLI / Air handlers can no longer return a uniform 4xx envelope for
// validation failures coming out of any adapter method.
func TestHTTPClient_SendRSVP_ValidationErrorsWrapErrInvalidInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		grantID    string
		calendarID string
		eventID    string
		req        *domain.SendRSVPRequest
	}{
		{name: "empty grant", grantID: "", calendarID: "c", eventID: "e", req: &domain.SendRSVPRequest{Status: "yes"}},
		{name: "empty calendar", grantID: "g", calendarID: "", eventID: "e", req: &domain.SendRSVPRequest{Status: "yes"}},
		{name: "empty event", grantID: "g", calendarID: "c", eventID: "", req: &domain.SendRSVPRequest{Status: "yes"}},
		{name: "nil request", grantID: "g", calendarID: "c", eventID: "e", req: nil},
		{name: "bad status", grantID: "g", calendarID: "c", eventID: "e", req: &domain.SendRSVPRequest{Status: "definitely"}},
		{name: "oversized comment", grantID: "g", calendarID: "c", eventID: "e", req: &domain.SendRSVPRequest{
			Status:  "yes",
			Comment: strings.Repeat("x", 2000),
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL("http://invalid.example") // never reached

			err := client.SendRSVP(context.Background(), tc.grantID, tc.calendarID, tc.eventID, tc.req)
			require.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrInvalidInput,
				"validation error %q must wrap domain.ErrInvalidInput so callers can classify it", err)
		})
	}
}

// TestHTTPClient_SendRSVP_NormalizesStatusCase pins that adapter-level
// validation accepts "YES" / "Yes" / " maybe " — a CLI or future SDK
// caller that doesn't normalize at its own layer must not be rejected
// for a cosmetic difference. Nylas itself only accepts lowercase; the
// adapter must lowercase before forwarding.
func TestHTTPClient_SendRSVP_NormalizesStatusCase(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		input       string
		wantPayload string
	}{
		{name: "uppercase", input: "YES", wantPayload: "yes"},
		{name: "titlecase", input: "Maybe", wantPayload: "maybe"},
		{name: "padded", input: "  no  ", wantPayload: "no"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var rawBody string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				rawBody = string(body)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			err := client.SendRSVP(context.Background(), "g", "c", "e", &domain.SendRSVPRequest{Status: tc.input})
			require.NoError(t, err)
			assert.Contains(t, rawBody, `"status":"`+tc.wantPayload+`"`,
				"expected normalized status %q in body, got %s", tc.wantPayload, rawBody)
		})
	}
}

// TestHTTPClient_SendRSVP_ForwardsCommentExactly pins that a non-empty
// comment is forwarded byte-for-byte (modulo JSON encoding) — quoting,
// special characters, multi-byte unicode all round-trip. Without this
// pin a future refactor that "cleans up" the comment (e.g. stripping
// quotes, ASCII-only) would silently corrupt user-typed messages on
// the way to the organiser's inbox.
func TestHTTPClient_SendRSVP_ForwardsCommentExactly(t *testing.T) {
	t.Parallel()
	tricky := `He said "hi" — and \\ then 🎉`

	var rawBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rawBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	err := client.SendRSVP(context.Background(), "g", "c", "e", &domain.SendRSVPRequest{
		Status:  "yes",
		Comment: tricky,
	})
	require.NoError(t, err)

	// Decode the wire body to compare against the input post-JSON.
	var got struct {
		Status  string `json:"status"`
		Comment string `json:"comment"`
	}
	require.NoError(t, json.Unmarshal([]byte(rawBody), &got),
		"server-side body must be valid JSON, got %q", rawBody)
	assert.Equal(t, tricky, got.Comment,
		"adapter must forward comment exactly; quotes/backslash/emoji should round-trip")
	assert.Equal(t, "yes", got.Status)
}

// TestHTTPClient_SendRSVP_RejectsOversizedComment pins the
// defense-in-depth comment cap at the adapter boundary. The Air handler
// has a matching cap, but CLI / SDK consumers go straight through the
// adapter — without this the cap is single-layer.
func TestHTTPClient_SendRSVP_RejectsOversizedComment(t *testing.T) {
	t.Parallel()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	err := client.SendRSVP(context.Background(), "g", "c", "e", &domain.SendRSVPRequest{
		Status:  "yes",
		Comment: strings.Repeat("x", 2000),
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput),
		"oversized comment must surface as ErrInvalidInput")
	assert.False(t, called, "adapter must short-circuit before issuing any HTTP request")
}
