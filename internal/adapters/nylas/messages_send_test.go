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

func TestHTTPClient_SendMessage(t *testing.T) {
	tests := []struct {
		name           string
		request        *domain.SendMessageRequest
		expectedFields []string
		statusCode     int
		wantErr        bool
	}{
		{
			name: "sends basic message",
			request: &domain.SendMessageRequest{
				Subject: "Test Subject",
				Body:    "Test body content",
				To:      []domain.EmailParticipant{{Name: "Recipient", Email: "recipient@example.com"}},
			},
			expectedFields: []string{"subject", "body", "to"},
			statusCode:     http.StatusOK,
			wantErr:        false,
		},
		{
			name: "sends message with CC and BCC",
			request: &domain.SendMessageRequest{
				Subject: "Test with CC",
				Body:    "Body",
				To:      []domain.EmailParticipant{{Email: "to@example.com"}},
				Cc:      []domain.EmailParticipant{{Email: "cc@example.com"}},
				Bcc:     []domain.EmailParticipant{{Email: "bcc@example.com"}},
			},
			expectedFields: []string{"subject", "body", "to", "cc", "bcc"},
			statusCode:     http.StatusOK,
			wantErr:        false,
		},
		{
			name: "sends message with From",
			request: &domain.SendMessageRequest{
				Subject: "From Alias",
				Body:    "Body",
				To:      []domain.EmailParticipant{{Email: "to@example.com"}},
				From:    []domain.EmailParticipant{{Name: "Alias", Email: "alias@example.com"}},
			},
			expectedFields: []string{"subject", "body", "to", "from"},
			statusCode:     http.StatusOK,
			wantErr:        false,
		},
		{
			name: "sends reply message",
			request: &domain.SendMessageRequest{
				Subject:      "Re: Original",
				Body:         "Reply body",
				To:           []domain.EmailParticipant{{Email: "to@example.com"}},
				ReplyToMsgID: "original-msg-123",
			},
			expectedFields: []string{"subject", "body", "to", "reply_to_message_id"},
			statusCode:     http.StatusOK,
			wantErr:        false,
		},
		{
			name: "sends scheduled message",
			request: &domain.SendMessageRequest{
				Subject: "Scheduled",
				Body:    "Scheduled body",
				To:      []domain.EmailParticipant{{Email: "to@example.com"}},
				SendAt:  1704153600,
			},
			expectedFields: []string{"subject", "body", "to", "send_at"},
			statusCode:     http.StatusAccepted,
			wantErr:        false,
		},
		{
			name: "sends message with tracking",
			request: &domain.SendMessageRequest{
				Subject: "Tracked",
				Body:    "Tracked body",
				To:      []domain.EmailParticipant{{Email: "to@example.com"}},
				TrackingOpts: &domain.TrackingOptions{
					Opens: true,
					Links: true,
					Label: "campaign-1",
				},
			},
			expectedFields: []string{"subject", "body", "to", "tracking_options"},
			statusCode:     http.StatusOK,
			wantErr:        false,
		},
		{
			name: "sends message with metadata",
			request: &domain.SendMessageRequest{
				Subject:  "With Metadata",
				Body:     "Body",
				To:       []domain.EmailParticipant{{Email: "to@example.com"}},
				Metadata: map[string]string{"campaign": "promo", "source": "cli"},
			},
			expectedFields: []string{"subject", "body", "to", "metadata"},
			statusCode:     http.StatusOK,
			wantErr:        false,
		},
		{
			name: "sends message with reply-to",
			request: &domain.SendMessageRequest{
				Subject: "With Reply-To",
				Body:    "Body",
				To:      []domain.EmailParticipant{{Email: "to@example.com"}},
				ReplyTo: []domain.EmailParticipant{{Email: "reply@example.com"}},
			},
			expectedFields: []string{"subject", "body", "to", "reply_to"},
			statusCode:     http.StatusOK,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v3/grants/grant-123/messages/send", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var body map[string]any
				err := json.NewDecoder(r.Body).Decode(&body)
				require.NoError(t, err)

				for _, field := range tt.expectedFields {
					assert.Contains(t, body, field, "Missing field: %s", field)
				}

				response := map[string]any{
					"data": map[string]any{
						"id":       "sent-msg-123",
						"grant_id": "grant-123",
						"subject":  tt.request.Subject,
						"date":     1704067200,
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
			msg, err := client.SendMessage(ctx, "grant-123", tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "sent-msg-123", msg.ID)
		})
	}
}

func TestHTTPClient_SendMessage_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		response    map[string]any
		errContains string
	}{
		{
			name:       "handles invalid recipient",
			statusCode: http.StatusBadRequest,
			response: map[string]any{
				"error": map[string]string{"message": "Invalid recipient email address"},
			},
			errContains: "Invalid recipient",
		},
		{
			name:       "handles quota exceeded",
			statusCode: http.StatusTooManyRequests,
			response: map[string]any{
				"error": map[string]string{"message": "Daily send limit exceeded"},
			},
			errContains: "Daily send limit",
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

			ctx := context.Background()
			req := &domain.SendMessageRequest{
				Subject: "Test",
				Body:    "Body",
				To:      []domain.EmailParticipant{{Email: "to@example.com"}},
			}
			_, err := client.SendMessage(ctx, "grant-123", req)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

// Note: Scheduled message tests are in scheduled_test.go

func TestHTTPClient_SmartCompose(t *testing.T) {
	tests := []struct {
		name           string
		request        *domain.SmartComposeRequest
		serverResponse map[string]any
		statusCode     int
		wantErr        bool
	}{
		{
			name: "generates compose suggestion",
			request: &domain.SmartComposeRequest{
				Prompt: "Write a follow-up email about the meeting",
			},
			serverResponse: map[string]any{
				"data": map[string]any{
					"suggestion": "Hi,\n\nThank you for attending the meeting...",
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "handles API error",
			request: &domain.SmartComposeRequest{
				Prompt: "Test",
			},
			serverResponse: map[string]any{
				"error": map[string]string{"message": "Smart compose not available"},
			},
			statusCode: http.StatusServiceUnavailable,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v3/grants/grant-123/messages/smart-compose", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				assert.Contains(t, body, "prompt")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			suggestion, err := client.SmartCompose(ctx, "grant-123", tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, suggestion.Suggestion)
		})
	}
}

func TestHTTPClient_SmartComposeReply(t *testing.T) {
	tests := []struct {
		name           string
		messageID      string
		request        *domain.SmartComposeRequest
		serverResponse map[string]any
		statusCode     int
		wantErr        bool
	}{
		{
			name:      "generates reply suggestion",
			messageID: "msg-123",
			request: &domain.SmartComposeRequest{
				Prompt: "Accept the meeting invitation politely",
			},
			serverResponse: map[string]any{
				"data": map[string]any{
					"suggestion": "Thank you for the invitation. I would be happy to attend...",
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:      "handles message not found",
			messageID: "nonexistent",
			request: &domain.SmartComposeRequest{
				Prompt: "Reply",
			},
			serverResponse: map[string]any{
				"error": map[string]string{"message": "Message not found"},
			},
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				expectedPath := "/v3/grants/grant-123/messages/" + tt.messageID + "/smart-compose"
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
			suggestion, err := client.SmartComposeReply(ctx, "grant-123", tt.messageID, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, suggestion.Suggestion)
		})
	}
}

func TestConvertContactsToAPI(t *testing.T) {
	// Test that contacts are properly converted through SendMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		// Verify 'to' field format
		to, ok := body["to"].([]any)
		require.True(t, ok)
		require.Len(t, to, 2)

		first := to[0].(map[string]any)
		assert.Equal(t, "Alice", first["name"])
		assert.Equal(t, "alice@example.com", first["email"])

		second := to[1].(map[string]any)
		assert.Equal(t, "Bob", second["name"])
		assert.Equal(t, "bob@example.com", second["email"])

		response := map[string]any{
			"data": map[string]any{
				"id":   "msg-123",
				"date": 1704067200,
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
	req := &domain.SendMessageRequest{
		Subject: "Test",
		Body:    "Body",
		To: []domain.EmailParticipant{
			{Name: "Alice", Email: "alice@example.com"},
			{Name: "Bob", Email: "bob@example.com"},
		},
	}
	_, err := client.SendMessage(ctx, "grant-123", req)
	require.NoError(t, err)
}
