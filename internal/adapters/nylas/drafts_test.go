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

func TestHTTPClient_CreateDraft_WithoutAttachments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/drafts", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		assert.Equal(t, "Test Subject", body["subject"])
		assert.Equal(t, "Test Body", body["body"])

		response := map[string]any{
			"data": map[string]any{
				"id":       "draft-001",
				"grant_id": "grant-123",
				"subject":  "Test Subject",
				"body":     "Test Body",
				"to": []map[string]string{
					{"email": "test@example.com", "name": "Test"},
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
	req := &domain.CreateDraftRequest{
		Subject: "Test Subject",
		Body:    "Test Body",
		To:      []domain.EmailParticipant{{Email: "test@example.com", Name: "Test"}},
	}

	draft, err := client.CreateDraft(ctx, "grant-123", req)

	require.NoError(t, err)
	assert.Equal(t, "draft-001", draft.ID)
	assert.Equal(t, "Test Subject", draft.Subject)
}

func TestHTTPClient_CreateDraft_WithAttachments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/drafts", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.True(t, strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data"))

		// Parse multipart form
		err := r.ParseMultipartForm(10 << 20) // 10MB max
		require.NoError(t, err)

		// Check message field
		message := r.FormValue("message")
		assert.Contains(t, message, "Test Subject")
		assert.Contains(t, message, "Test Body")

		// Check for file field
		_, fileHeader, err := r.FormFile("file0")
		require.NoError(t, err)
		assert.Equal(t, "test.txt", fileHeader.Filename)

		response := map[string]any{
			"data": map[string]any{
				"id":       "draft-002",
				"grant_id": "grant-123",
				"subject":  "Test Subject",
				"body":     "Test Body",
				"attachments": []map[string]any{
					{
						"id":           "attach-001",
						"filename":     "test.txt",
						"content_type": "text/plain",
						"size":         12,
					},
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
	req := &domain.CreateDraftRequest{
		Subject: "Test Subject",
		Body:    "Test Body",
		To:      []domain.EmailParticipant{{Email: "test@example.com"}},
		Attachments: []domain.Attachment{
			{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Content:     []byte("Test content"),
				Size:        12,
			},
		},
	}

	draft, err := client.CreateDraft(ctx, "grant-123", req)

	require.NoError(t, err)
	assert.Equal(t, "draft-002", draft.ID)
	assert.Len(t, draft.Attachments, 1)
	assert.Equal(t, "test.txt", draft.Attachments[0].Filename)
}

func TestHTTPClient_CreateDraft_MultipleAttachments(t *testing.T) {
	attachmentCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(10 << 20)
		require.NoError(t, err)

		// Count file fields
		for key := range r.MultipartForm.File {
			if strings.HasPrefix(key, "file") {
				attachmentCount++
			}
		}

		response := map[string]any{
			"data": map[string]any{
				"id":       "draft-003",
				"grant_id": "grant-123",
				"subject":  "Multi Attachment Test",
				"attachments": []map[string]any{
					{"id": "attach-001", "filename": "file1.pdf"},
					{"id": "attach-002", "filename": "file2.jpg"},
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
	req := &domain.CreateDraftRequest{
		Subject: "Multi Attachment Test",
		Body:    "Body",
		Attachments: []domain.Attachment{
			{Filename: "file1.pdf", ContentType: "application/pdf", Content: []byte("PDF content")},
			{Filename: "file2.jpg", ContentType: "image/jpeg", Content: []byte("JPG content")},
		},
	}

	draft, err := client.CreateDraft(ctx, "grant-123", req)

	require.NoError(t, err)
	assert.Equal(t, 2, attachmentCount)
	assert.Len(t, draft.Attachments, 2)
}

func TestMockClient_CreateDraft_WithAttachments(t *testing.T) {
	ctx := context.Background()
	mock := nylas.NewMockClient()

	req := &domain.CreateDraftRequest{
		Subject: "Test Draft",
		Body:    "Draft body",
		To:      []domain.EmailParticipant{{Email: "test@example.com"}},
		Attachments: []domain.Attachment{
			{
				Filename:    "report.pdf",
				ContentType: "application/pdf",
				Content:     []byte("PDF data"),
				Size:        8,
			},
		},
	}

	draft, err := mock.CreateDraft(ctx, "grant-123", req)

	require.NoError(t, err)
	assert.True(t, mock.CreateDraftCalled)
	assert.Equal(t, "grant-123", mock.LastGrantID)
	assert.Equal(t, "Test Draft", draft.Subject)
	assert.Len(t, draft.Attachments, 1)
	assert.Equal(t, "report.pdf", draft.Attachments[0].Filename)
	assert.Equal(t, "attach-1", draft.Attachments[0].ID)
}

func TestMockClient_UpdateDraft_WithAttachments(t *testing.T) {
	ctx := context.Background()
	mock := nylas.NewMockClient()

	req := &domain.CreateDraftRequest{
		Subject: "Updated Draft",
		Body:    "Updated body",
		Attachments: []domain.Attachment{
			{Filename: "new-file.doc", ContentType: "application/msword", Size: 100},
		},
	}

	draft, err := mock.UpdateDraft(ctx, "grant-123", "draft-456", req)

	require.NoError(t, err)
	assert.True(t, mock.UpdateDraftCalled)
	assert.Equal(t, "draft-456", draft.ID)
	assert.Len(t, draft.Attachments, 1)
	assert.Equal(t, "new-file.doc", draft.Attachments[0].Filename)
}

func TestDemoClient_CreateDraft_WithAttachments(t *testing.T) {
	ctx := context.Background()
	demo := nylas.NewDemoClient()

	req := &domain.CreateDraftRequest{
		Subject: "Demo Draft",
		Body:    "Demo body",
		To:      []domain.EmailParticipant{{Email: "demo@example.com"}},
		Attachments: []domain.Attachment{
			{Filename: "demo.pdf", ContentType: "application/pdf", Content: []byte("Demo PDF")},
		},
	}

	draft, err := demo.CreateDraft(ctx, "demo-grant", req)

	require.NoError(t, err)
	assert.NotEmpty(t, draft.ID)
	// Demo client may not preserve attachments, just verify no error
}

func TestHTTPClient_CreateDraftWithAttachmentFromReader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.True(t, strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data"))

		err := r.ParseMultipartForm(10 << 20)
		require.NoError(t, err)

		// Verify file was uploaded
		file, header, err := r.FormFile("file")
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		assert.Equal(t, "stream.txt", header.Filename)

		content, _ := io.ReadAll(file)
		assert.Equal(t, "streamed content", string(content))

		response := map[string]any{
			"data": map[string]any{
				"id":       "draft-stream",
				"grant_id": "grant-123",
				"subject":  "Stream Test",
				"attachments": []map[string]any{
					{"id": "attach-stream", "filename": "stream.txt"},
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
	req := &domain.CreateDraftRequest{
		Subject: "Stream Test",
		Body:    "Body",
	}

	reader := strings.NewReader("streamed content")
	draft, err := client.CreateDraftWithAttachmentFromReader(ctx, "grant-123", req, "stream.txt", "text/plain", reader)

	require.NoError(t, err)
	assert.Equal(t, "draft-stream", draft.ID)
	assert.Len(t, draft.Attachments, 1)
	assert.Equal(t, "stream.txt", draft.Attachments[0].Filename)
}
