//go:build !integration
// +build !integration

package nylas_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

				var body map[string]interface{}
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

			response := map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"email": "user@example.com",
						"time_slots": []map[string]interface{}{
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

			response := map[string]interface{}{
				"time_slots": []map[string]interface{}{
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
