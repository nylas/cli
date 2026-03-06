//go:build !integration
// +build !integration

package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	slack_api "github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/nylas/cli/internal/domain"
)

// createTestClient creates a Client with a mock server URL.
func createTestClient(t *testing.T, serverURL string) *Client {
	options := []slack_api.Option{
		slack_api.OptionAPIURL(serverURL + "/"),
	}
	api := slack_api.New("xoxp-test-token", options...)

	return &Client{
		api:          api,
		userToken:    "xoxp-test-token",
		rateLimiter:  rate.NewLimiter(rate.Limit(100), 10), // Fast for tests
		userCache:    make(map[string]*cachedUser),
		userCacheTTL: 5 * time.Minute,
	}
}

func TestClient_TestAuth_HTTP(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse any
		statusCode     int
		wantErr        bool
		wantUserID     string
	}{
		{
			name: "successful auth test",
			serverResponse: map[string]any{
				"ok":      true,
				"url":     "https://workspace.slack.com/",
				"team":    "Test Workspace",
				"user":    "testuser",
				"team_id": "T12345",
				"user_id": "U12345",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			wantUserID: "U12345",
		},
		{
			name: "auth failed - invalid token",
			serverResponse: map[string]any{
				"ok":    false,
				"error": "invalid_auth",
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name: "auth failed - token revoked",
			serverResponse: map[string]any{
				"ok":    false,
				"error": "token_revoked",
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "auth.test")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := createTestClient(t, server.URL)
			auth, err := client.TestAuth(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, auth)
			} else {
				require.NoError(t, err)
				require.NotNil(t, auth)
				assert.Equal(t, tt.wantUserID, auth.UserID)
			}
		})
	}
}

func TestClient_ListChannels_HTTP(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse any
		statusCode     int
		wantLen        int
		wantErr        bool
	}{
		{
			name: "returns channels successfully",
			serverResponse: map[string]any{
				"ok": true,
				"channels": []map[string]any{
					{
						"id":         "C12345",
						"name":       "general",
						"is_channel": true,
						"is_member":  true,
					},
					{
						"id":         "C23456",
						"name":       "random",
						"is_channel": true,
						"is_member":  true,
					},
				},
				"response_metadata": map[string]string{
					"next_cursor": "",
				},
			},
			statusCode: http.StatusOK,
			wantLen:    2,
			wantErr:    false,
		},
		{
			name: "returns empty when no channels",
			serverResponse: map[string]any{
				"ok":       true,
				"channels": []any{},
			},
			statusCode: http.StatusOK,
			wantLen:    0,
			wantErr:    false,
		},
		{
			name: "handles not_authed error",
			serverResponse: map[string]any{
				"ok":    false,
				"error": "not_authed",
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "conversations.list")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := createTestClient(t, server.URL)
			resp, err := client.ListChannels(context.Background(), nil)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Len(t, resp.Channels, tt.wantLen)
			}
		})
	}
}

func TestClient_GetChannel_HTTP(t *testing.T) {
	tests := []struct {
		name           string
		channelID      string
		serverResponse any
		statusCode     int
		wantErr        bool
		wantName       string
	}{
		{
			name:      "returns channel successfully",
			channelID: "C12345",
			serverResponse: map[string]any{
				"ok": true,
				"channel": map[string]any{
					"id":         "C12345",
					"name":       "general",
					"is_channel": true,
					"topic":      map[string]string{"value": "General discussion"},
					"purpose":    map[string]string{"value": "Company announcements"},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			wantName:   "general",
		},
		{
			name:      "channel not found",
			channelID: "C99999",
			serverResponse: map[string]any{
				"ok":    false,
				"error": "channel_not_found",
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "conversations.info")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := createTestClient(t, server.URL)
			ch, err := client.GetChannel(context.Background(), tt.channelID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, ch)
			} else {
				require.NoError(t, err)
				require.NotNil(t, ch)
				assert.Equal(t, tt.wantName, ch.Name)
			}
		})
	}
}

func TestClient_GetMessages_HTTP(t *testing.T) {
	tests := []struct {
		name           string
		channelID      string
		serverResponse any
		statusCode     int
		wantLen        int
		wantErr        bool
	}{
		{
			name:      "returns messages successfully",
			channelID: "C12345",
			serverResponse: map[string]any{
				"ok": true,
				"messages": []map[string]any{
					{
						"type": "message",
						"user": "U12345",
						"text": "Hello, world!",
						"ts":   "1234567890.123456",
					},
					{
						"type": "message",
						"user": "U23456",
						"text": "Hi there!",
						"ts":   "1234567891.123456",
					},
				},
				"has_more": false,
				"response_metadata": map[string]string{
					"next_cursor": "",
				},
			},
			statusCode: http.StatusOK,
			wantLen:    2,
			wantErr:    false,
		},
		{
			name:      "channel not found",
			channelID: "C99999",
			serverResponse: map[string]any{
				"ok":    false,
				"error": "channel_not_found",
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "conversations.history")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := createTestClient(t, server.URL)
			resp, err := client.GetMessages(context.Background(), &domain.SlackMessageQueryParams{
				ChannelID: tt.channelID,
			})

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Len(t, resp.Messages, tt.wantLen)
			}
		})
	}
}

func TestClient_SendMessage_HTTP(t *testing.T) {
	tests := []struct {
		name           string
		channelID      string
		text           string
		serverResponse any
		statusCode     int
		wantErr        bool
	}{
		{
			name:      "sends message successfully",
			channelID: "C12345",
			text:      "Hello!",
			serverResponse: map[string]any{
				"ok":      true,
				"channel": "C12345",
				"ts":      "1234567890.123456",
				"message": map[string]any{
					"text": "Hello!",
					"user": "U12345",
					"ts":   "1234567890.123456",
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:      "channel not found",
			channelID: "C99999",
			text:      "Test",
			serverResponse: map[string]any{
				"ok":    false,
				"error": "channel_not_found",
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "chat.postMessage")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := createTestClient(t, server.URL)
			msg, err := client.SendMessage(context.Background(), &domain.SlackSendMessageRequest{
				ChannelID: tt.channelID,
				Text:      tt.text,
			})

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, msg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, msg)
				assert.Equal(t, tt.channelID, msg.ChannelID)
			}
		})
	}
}

func TestClient_GetUser_HTTP(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		serverResponse any
		statusCode     int
		wantErr        bool
		wantName       string
	}{
		{
			name:   "returns user successfully",
			userID: "U12345",
			serverResponse: map[string]any{
				"ok": true,
				"user": map[string]any{
					"id":        "U12345",
					"name":      "testuser",
					"real_name": "Test User",
					"profile": map[string]any{
						"display_name": "Tester",
						"email":        "test@example.com",
					},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			wantName:   "testuser",
		},
		{
			name:   "user not found",
			userID: "U99999",
			serverResponse: map[string]any{
				"ok":    false,
				"error": "user_not_found",
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				// GetUser calls both users.info and users.profile.get
				if strings.Contains(r.URL.Path, "users.info") {
					w.WriteHeader(tt.statusCode)
					_ = json.NewEncoder(w).Encode(tt.serverResponse)
				} else if strings.Contains(r.URL.Path, "users.profile.get") {
					// Return empty profile for the profile fetch
					_ = json.NewEncoder(w).Encode(map[string]any{
						"ok":      true,
						"profile": map[string]any{},
					})
				} else {
					t.Errorf("unexpected endpoint: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			client := createTestClient(t, server.URL)
			user, err := client.GetUser(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				assert.Equal(t, tt.wantName, user.Name)
			}
		})
	}
}

func TestClient_SearchMessages_HTTP(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		serverResponse any
		statusCode     int
		wantLen        int
		wantErr        bool
	}{
		{
			name:  "returns search results",
			query: "important",
			serverResponse: map[string]any{
				"ok": true,
				"messages": map[string]any{
					"matches": []map[string]any{
						{
							"type":     "message",
							"user":     "U12345",
							"username": "testuser",
							"text":     "This is important",
							"ts":       "1234567890.123456",
							"channel": map[string]any{
								"id":   "C12345",
								"name": "general",
							},
						},
					},
				},
			},
			statusCode: http.StatusOK,
			wantLen:    1,
			wantErr:    false,
		},
		{
			name:  "no results",
			query: "nonexistent",
			serverResponse: map[string]any{
				"ok": true,
				"messages": map[string]any{
					"matches": []any{},
				},
			},
			statusCode: http.StatusOK,
			wantLen:    0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "search.messages")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := createTestClient(t, server.URL)
			msgs, err := client.SearchMessages(context.Background(), tt.query, 20)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, msgs)
			} else {
				require.NoError(t, err)
				assert.Len(t, msgs, tt.wantLen)
			}
		})
	}
}

func TestClient_DeleteMessage_HTTP(t *testing.T) {
	tests := []struct {
		name           string
		channelID      string
		messageTS      string
		serverResponse any
		statusCode     int
		wantErr        bool
	}{
		{
			name:      "deletes message successfully",
			channelID: "C12345",
			messageTS: "1234567890.123456",
			serverResponse: map[string]any{
				"ok":      true,
				"channel": "C12345",
				"ts":      "1234567890.123456",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:      "message not found",
			channelID: "C12345",
			messageTS: "0000000000.000000",
			serverResponse: map[string]any{
				"ok":    false,
				"error": "message_not_found",
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "chat.delete")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := createTestClient(t, server.URL)
			err := client.DeleteMessage(context.Background(), tt.channelID, tt.messageTS)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
