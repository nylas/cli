//go:build !integration

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

func TestHTTPClient_SendTransactionalMessage(t *testing.T) {
	tests := []struct {
		name           string
		domainName     string
		request        *domain.SendMessageRequest
		expectedFields []string
		statusCode     int
		wantErr        bool
	}{
		{
			name:       "sends basic transactional message",
			domainName: "qasim.nylas.email",
			request: &domain.SendMessageRequest{
				Subject: "Transactional Email",
				Body:    "This is a transactional email body",
				To:      []domain.EmailParticipant{{Name: "Recipient", Email: "recipient@example.com"}},
				From:    []domain.EmailParticipant{{Email: "info@qasim.nylas.email"}},
			},
			expectedFields: []string{"subject", "body", "to", "from"},
			statusCode:     http.StatusOK,
			wantErr:        false,
		},
		{
			name:       "sends transactional message with CC and BCC",
			domainName: "test.nylas.email",
			request: &domain.SendMessageRequest{
				Subject: "With CC",
				Body:    "Body",
				To:      []domain.EmailParticipant{{Email: "to@example.com"}},
				From:    []domain.EmailParticipant{{Email: "sender@test.nylas.email"}},
				Cc:      []domain.EmailParticipant{{Email: "cc@example.com"}},
				Bcc:     []domain.EmailParticipant{{Email: "bcc@example.com"}},
			},
			expectedFields: []string{"subject", "body", "to", "from", "cc", "bcc"},
			statusCode:     http.StatusOK,
			wantErr:        false,
		},
		{
			name:       "sends scheduled transactional message",
			domainName: "scheduled.nylas.email",
			request: &domain.SendMessageRequest{
				Subject: "Scheduled Email",
				Body:    "Scheduled body",
				To:      []domain.EmailParticipant{{Email: "to@example.com"}},
				From:    []domain.EmailParticipant{{Email: "noreply@scheduled.nylas.email"}},
				SendAt:  1704153600,
			},
			expectedFields: []string{"subject", "body", "to", "from", "send_at"},
			statusCode:     http.StatusAccepted,
			wantErr:        false,
		},
		{
			name:       "sends transactional message with tracking",
			domainName: "track.nylas.email",
			request: &domain.SendMessageRequest{
				Subject: "Tracked Email",
				Body:    "Tracked body",
				To:      []domain.EmailParticipant{{Email: "to@example.com"}},
				From:    []domain.EmailParticipant{{Email: "marketing@track.nylas.email"}},
				TrackingOpts: &domain.TrackingOptions{
					Opens: true,
					Links: true,
					Label: "campaign-1",
				},
			},
			expectedFields: []string{"subject", "body", "to", "from", "tracking_options"},
			statusCode:     http.StatusOK,
			wantErr:        false,
		},
		{
			name:       "sends transactional message with metadata",
			domainName: "meta.nylas.email",
			request: &domain.SendMessageRequest{
				Subject:  "With Metadata",
				Body:     "Body",
				To:       []domain.EmailParticipant{{Email: "to@example.com"}},
				From:     []domain.EmailParticipant{{Email: "system@meta.nylas.email"}},
				Metadata: map[string]string{"campaign": "promo", "source": "cli"},
			},
			expectedFields: []string{"subject", "body", "to", "from", "metadata"},
			statusCode:     http.StatusOK,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v3/domains/"+tt.domainName+"/messages/send", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var body map[string]any
				err := json.NewDecoder(r.Body).Decode(&body)
				require.NoError(t, err)

				for _, field := range tt.expectedFields {
					assert.Contains(t, body, field, "Missing field: %s", field)
				}

				response := map[string]any{
					"data": map[string]any{
						"id":      "sent-transactional-msg-123",
						"subject": tt.request.Subject,
						"date":    1704067200,
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
			msg, err := client.SendTransactionalMessage(ctx, tt.domainName, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "sent-transactional-msg-123", msg.ID)
		})
	}
}

func TestHTTPClient_SendTransactionalMessage_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		response    map[string]any
		errContains string
	}{
		{
			name:       "handles invalid domain",
			statusCode: http.StatusNotFound,
			response: map[string]any{
				"error": map[string]string{"message": "Domain not found"},
			},
			errContains: "Domain not found",
		},
		{
			name:       "handles invalid recipient",
			statusCode: http.StatusBadRequest,
			response: map[string]any{
				"error": map[string]string{"message": "Invalid recipient email address"},
			},
			errContains: "Invalid recipient",
		},
		{
			name:       "handles rate limit",
			statusCode: http.StatusTooManyRequests,
			response: map[string]any{
				"error": map[string]string{"message": "Rate limit exceeded"},
			},
			errContains: "Rate limit",
		},
		{
			name:       "handles server error",
			statusCode: http.StatusInternalServerError,
			response: map[string]any{
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
			client.SetMaxRetries(0) // Disable retries for error handling tests

			ctx := context.Background()
			req := &domain.SendMessageRequest{
				Subject: "Test",
				Body:    "Body",
				To:      []domain.EmailParticipant{{Email: "to@example.com"}},
				From:    []domain.EmailParticipant{{Email: "sender@test.nylas.email"}},
			}
			_, err := client.SendTransactionalMessage(ctx, "test.nylas.email", req)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestMockClient_SendTransactionalMessage(t *testing.T) {
	mockClient := nylas.NewMockClient()

	ctx := context.Background()
	req := &domain.SendMessageRequest{
		Subject: "Test Subject",
		Body:    "Test Body",
		To:      []domain.EmailParticipant{{Email: "to@example.com"}},
		From:    []domain.EmailParticipant{{Email: "from@test.nylas.email"}},
	}

	msg, err := mockClient.SendTransactionalMessage(ctx, "test.nylas.email", req)
	require.NoError(t, err)
	assert.Equal(t, "sent-transactional-message-id", msg.ID)
	assert.Equal(t, "Test Subject", msg.Subject)
}
