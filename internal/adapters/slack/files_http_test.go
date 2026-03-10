//go:build !integration
// +build !integration

package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nylas/cli/internal/domain"
)

func TestClient_ListFiles_HTTP(t *testing.T) {
	tests := []struct {
		name           string
		params         *domain.SlackFileQueryParams
		serverResponse any
		statusCode     int
		wantLen        int
		wantErr        bool
	}{
		{
			name:   "returns files successfully",
			params: nil,
			serverResponse: map[string]any{
				"ok": true,
				"files": []map[string]any{
					{
						"id":       "F12345",
						"name":     "document.pdf",
						"title":    "Important Document",
						"mimetype": "application/pdf",
						"filetype": "pdf",
						"size":     1024,
						"user":     "U12345",
					},
					{
						"id":       "F23456",
						"name":     "image.png",
						"title":    "Screenshot",
						"mimetype": "image/png",
						"filetype": "png",
						"size":     2048,
						"user":     "U12345",
					},
				},
			},
			statusCode: http.StatusOK,
			wantLen:    2,
			wantErr:    false,
		},
		{
			name: "with channel filter",
			params: &domain.SlackFileQueryParams{
				ChannelID: "C12345",
			},
			serverResponse: map[string]any{
				"ok": true,
				"files": []map[string]any{
					{
						"id":   "F12345",
						"name": "channel-file.txt",
					},
				},
			},
			statusCode: http.StatusOK,
			wantLen:    1,
			wantErr:    false,
		},
		{
			name: "with user filter",
			params: &domain.SlackFileQueryParams{
				UserID: "U12345",
			},
			serverResponse: map[string]any{
				"ok": true,
				"files": []map[string]any{
					{
						"id":   "F12345",
						"name": "user-file.doc",
					},
				},
			},
			statusCode: http.StatusOK,
			wantLen:    1,
			wantErr:    false,
		},
		{
			name: "with types filter",
			params: &domain.SlackFileQueryParams{
				Types: []string{"images", "pdfs"},
			},
			serverResponse: map[string]any{
				"ok":    true,
				"files": []any{},
			},
			statusCode: http.StatusOK,
			wantLen:    0,
			wantErr:    false,
		},
		{
			name:   "auth failed",
			params: nil,
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
				assert.Contains(t, r.URL.Path, "files.list")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := createTestClient(t, server.URL)
			resp, err := client.ListFiles(context.Background(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Len(t, resp.Files, tt.wantLen)
			}
		})
	}
}

func TestClient_GetFileInfo_HTTP(t *testing.T) {
	tests := []struct {
		name           string
		fileID         string
		serverResponse any
		statusCode     int
		wantErr        bool
		wantName       string
	}{
		{
			name:   "returns file info successfully",
			fileID: "F12345",
			serverResponse: map[string]any{
				"ok": true,
				"file": map[string]any{
					"id":       "F12345",
					"name":     "document.pdf",
					"title":    "Important Document",
					"mimetype": "application/pdf",
					"filetype": "pdf",
					"size":     1024,
					"user":     "U12345",
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			wantName:   "document.pdf",
		},
		{
			name:   "file not found",
			fileID: "F99999",
			serverResponse: map[string]any{
				"ok":    false,
				"error": "file_not_found",
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name:    "empty file ID returns error",
			fileID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fileID == "" {
				// Test without server for empty file ID case
				client := &Client{
					rateLimiter: createTestClient(t, "http://localhost").rateLimiter,
				}
				file, err := client.GetFileInfo(context.Background(), tt.fileID)
				assert.Error(t, err)
				assert.Nil(t, file)
				return
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "files.info")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := createTestClient(t, server.URL)
			file, err := client.GetFileInfo(context.Background(), tt.fileID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, file)
			} else {
				require.NoError(t, err)
				require.NotNil(t, file)
				assert.Equal(t, tt.wantName, file.Name)
			}
		})
	}
}

func TestClient_UpdateMessage_HTTP(t *testing.T) {
	tests := []struct {
		name           string
		channelID      string
		messageTS      string
		newText        string
		serverResponse any
		statusCode     int
		wantErr        bool
	}{
		{
			name:      "updates message successfully",
			channelID: "C12345",
			messageTS: "1234567890.123456",
			newText:   "Updated text",
			serverResponse: map[string]any{
				"ok":      true,
				"channel": "C12345",
				"ts":      "1234567890.123456",
				"text":    "Updated text",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:      "message not found",
			channelID: "C12345",
			messageTS: "0000000000.000000",
			newText:   "Won't update",
			serverResponse: map[string]any{
				"ok":    false,
				"error": "message_not_found",
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name:      "cant edit message",
			channelID: "C12345",
			messageTS: "1234567890.123456",
			newText:   "Can't edit",
			serverResponse: map[string]any{
				"ok":    false,
				"error": "cant_update_message",
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "chat.update")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := createTestClient(t, server.URL)
			msg, err := client.UpdateMessage(context.Background(), tt.channelID, tt.messageTS, tt.newText)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, msg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, msg)
				assert.True(t, msg.Edited)
				assert.Equal(t, tt.newText, msg.Text)
			}
		})
	}
}

func TestClient_GetThreadReplies_HTTP(t *testing.T) {
	tests := []struct {
		name           string
		channelID      string
		threadTS       string
		serverResponse any
		statusCode     int
		wantLen        int
		wantErr        bool
	}{
		{
			name:      "returns thread replies",
			channelID: "C12345",
			threadTS:  "1234567890.123456",
			serverResponse: map[string]any{
				"ok": true,
				"messages": []map[string]any{
					{
						"type":        "message",
						"user":        "U12345",
						"text":        "Parent message",
						"ts":          "1234567890.123456",
						"thread_ts":   "1234567890.123456",
						"reply_count": 2,
					},
					{
						"type":      "message",
						"user":      "U23456",
						"text":      "First reply",
						"ts":        "1234567891.123456",
						"thread_ts": "1234567890.123456",
					},
					{
						"type":      "message",
						"user":      "U34567",
						"text":      "Second reply",
						"ts":        "1234567892.123456",
						"thread_ts": "1234567890.123456",
					},
				},
				"has_more": false,
			},
			statusCode: http.StatusOK,
			wantLen:    3,
			wantErr:    false,
		},
		{
			name:      "thread not found",
			channelID: "C12345",
			threadTS:  "0000000000.000000",
			serverResponse: map[string]any{
				"ok":    false,
				"error": "thread_not_found",
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "conversations.replies")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := createTestClient(t, server.URL)
			replies, err := client.GetThreadReplies(context.Background(), tt.channelID, tt.threadTS, 10)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, replies)
			} else {
				require.NoError(t, err)
				assert.Len(t, replies, tt.wantLen)
			}
		})
	}
}

func TestClient_ListMyChannels_HTTP(t *testing.T) {
	tests := []struct {
		name           string
		params         *domain.SlackChannelQueryParams
		serverResponse any
		statusCode     int
		wantLen        int
		wantErr        bool
	}{
		{
			name:   "returns user channels",
			params: nil,
			serverResponse: map[string]any{
				"ok": true,
				"channels": []map[string]any{
					{
						"id":         "C12345",
						"name":       "my-channel",
						"is_channel": true,
						"is_member":  true,
					},
				},
				"response_metadata": map[string]string{
					"next_cursor": "",
				},
			},
			statusCode: http.StatusOK,
			wantLen:    1,
			wantErr:    false,
		},
		{
			name:   "rate limited",
			params: nil,
			serverResponse: map[string]any{
				"ok":    false,
				"error": "ratelimited",
			},
			statusCode: http.StatusTooManyRequests,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "users.conversations")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := createTestClient(t, server.URL)
			resp, err := client.ListMyChannels(context.Background(), tt.params)

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
