package nylas_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_ListAttachments(t *testing.T) {
	t.Run("returns attachments from message", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v3/grants/grant-123/messages/msg-456", r.URL.Path)
			assert.Equal(t, "GET", r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"id":       "msg-456",
					"grant_id": "grant-123",
					"subject":  "Test with attachments",
					"attachments": []map[string]interface{}{
						{
							"id":           "attach-1",
							"filename":     "report.pdf",
							"content_type": "application/pdf",
							"size":         12345,
							"is_inline":    false,
						},
						{
							"id":           "attach-2",
							"filename":     "image.png",
							"content_type": "image/png",
							"size":         67890,
							"is_inline":    true,
							"content_id":   "img001",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response) // Test helper, encode error not actionable
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		ctx := context.Background()
		attachments, err := client.ListAttachments(ctx, "grant-123", "msg-456")

		require.NoError(t, err)
		assert.Len(t, attachments, 2)

		assert.Equal(t, "attach-1", attachments[0].ID)
		assert.Equal(t, "report.pdf", attachments[0].Filename)
		assert.Equal(t, "application/pdf", attachments[0].ContentType)
		assert.Equal(t, int64(12345), attachments[0].Size)
		assert.False(t, attachments[0].IsInline)

		assert.Equal(t, "attach-2", attachments[1].ID)
		assert.Equal(t, "image.png", attachments[1].Filename)
		assert.True(t, attachments[1].IsInline)
		assert.Equal(t, "img001", attachments[1].ContentID)
	})

	t.Run("returns empty list for message without attachments", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"id":          "msg-456",
					"grant_id":    "grant-123",
					"subject":     "No attachments",
					"attachments": []interface{}{},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response) // Test helper, encode error not actionable
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		ctx := context.Background()
		attachments, err := client.ListAttachments(ctx, "grant-123", "msg-456")

		require.NoError(t, err)
		assert.Len(t, attachments, 0)
	})
}

func TestHTTPClient_GetAttachment(t *testing.T) {
	t.Run("returns attachment metadata", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v3/grants/grant-123/messages/msg-456/attachments/attach-789", r.URL.Path)
			assert.Equal(t, "GET", r.Method)

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"id":           "attach-789",
					"grant_id":     "grant-123",
					"filename":     "document.pdf",
					"content_type": "application/pdf",
					"size":         54321,
					"is_inline":    false,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response) // Test helper, encode error not actionable
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		ctx := context.Background()
		attachment, err := client.GetAttachment(ctx, "grant-123", "msg-456", "attach-789")

		require.NoError(t, err)
		assert.Equal(t, "attach-789", attachment.ID)
		assert.Equal(t, "document.pdf", attachment.Filename)
		assert.Equal(t, "application/pdf", attachment.ContentType)
		assert.Equal(t, int64(54321), attachment.Size)
	})

	t.Run("returns error for non-existent attachment", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{ // Test helper, encode error not actionable
				"error": map[string]string{
					"message": "attachment not found",
					"type":    "not_found",
				},
			})
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		ctx := context.Background()
		attachment, err := client.GetAttachment(ctx, "grant-123", "msg-456", "nonexistent")

		assert.Error(t, err)
		assert.Nil(t, attachment)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestHTTPClient_DownloadAttachment(t *testing.T) {
	t.Run("downloads attachment content", func(t *testing.T) {
		expectedContent := "This is the attachment content"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v3/grants/grant-123/messages/msg-456/attachments/attach-789/download", r.URL.Path)
			assert.Equal(t, "GET", r.Method)

			w.Header().Set("Content-Type", "application/pdf")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(expectedContent))
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		ctx := context.Background()
		reader, err := client.DownloadAttachment(ctx, "grant-123", "msg-456", "attach-789")

		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, expectedContent, string(content))
	})

	t.Run("returns error for non-existent attachment", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		ctx := context.Background()
		reader, err := client.DownloadAttachment(ctx, "grant-123", "msg-456", "nonexistent")

		assert.Error(t, err)
		assert.Nil(t, reader)
	})
}

func TestMockClient_ListAttachments(t *testing.T) {
	ctx := context.Background()

	t.Run("default behavior", func(t *testing.T) {
		mock := nylas.NewMockClient()
		attachments, err := mock.ListAttachments(ctx, "grant-123", "msg-456")
		require.NoError(t, err)
		assert.Len(t, attachments, 2)
		assert.True(t, mock.ListAttachmentsCalled)
		assert.Equal(t, "grant-123", mock.LastGrantID)
		assert.Equal(t, "msg-456", mock.LastMessageID)
	})

	t.Run("with custom function", func(t *testing.T) {
		mock := nylas.NewMockClient()
		mock.ListAttachmentsFunc = func(ctx context.Context, grantID, messageID string) ([]domain.Attachment, error) {
			return []domain.Attachment{
				{ID: "custom-attach", Filename: "custom.txt"},
			}, nil
		}

		attachments, err := mock.ListAttachments(ctx, "grant-123", "msg-456")
		require.NoError(t, err)
		assert.Len(t, attachments, 1)
		assert.Equal(t, "custom-attach", attachments[0].ID)
	})
}

func TestDemoClient_Attachments(t *testing.T) {
	ctx := context.Background()
	demo := nylas.NewDemoClient()

	t.Run("ListAttachments returns demo data", func(t *testing.T) {
		attachments, err := demo.ListAttachments(ctx, "grant-123", "msg-456")
		require.NoError(t, err)
		assert.Len(t, attachments, 2)
		assert.Equal(t, "quarterly-report.pdf", attachments[0].Filename)
	})

	t.Run("GetAttachment returns demo data", func(t *testing.T) {
		attachment, err := demo.GetAttachment(ctx, "grant-123", "msg-456", "attach-123")
		require.NoError(t, err)
		assert.Equal(t, "attach-123", attachment.ID)
		assert.Equal(t, "quarterly-report.pdf", attachment.Filename)
	})

	t.Run("DownloadAttachment returns demo content", func(t *testing.T) {
		reader, err := demo.DownloadAttachment(ctx, "grant-123", "msg-456", "attach-123")
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.True(t, strings.Contains(string(content), "demo"))
	})
}
