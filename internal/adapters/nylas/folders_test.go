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

func TestHTTPClient_GetFolders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/folders", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": []map[string]any{
				{
					"id":            "folder-inbox",
					"name":          "INBOX",
					"system_folder": "inbox",
					"total_count":   100,
					"unread_count":  5,
				},
				{
					"id":            "folder-sent",
					"name":          "Sent",
					"system_folder": "sent",
					"total_count":   50,
					"unread_count":  0,
				},
				{
					"id":          "folder-custom",
					"name":        "Projects",
					"total_count": 25,
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
	folders, err := client.GetFolders(ctx, "grant-123")

	require.NoError(t, err)
	assert.Len(t, folders, 3)
	assert.Equal(t, "folder-inbox", folders[0].ID)
	assert.Equal(t, "INBOX", folders[0].Name)
	assert.Equal(t, "inbox", folders[0].SystemFolder)
	assert.Equal(t, 100, folders[0].TotalCount)
	assert.Equal(t, 5, folders[0].UnreadCount)
	assert.Equal(t, "Projects", folders[2].Name)
}

func TestHTTPClient_GetFolder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/folders/folder-456", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id":               "folder-456",
				"name":             "Work",
				"total_count":      150,
				"unread_count":     10,
				"background_color": "#FF0000",
				"text_color":       "#FFFFFF",
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
	folder, err := client.GetFolder(ctx, "grant-123", "folder-456")

	require.NoError(t, err)
	assert.Equal(t, "folder-456", folder.ID)
	assert.Equal(t, "Work", folder.Name)
	assert.Equal(t, 150, folder.TotalCount)
	assert.Equal(t, 10, folder.UnreadCount)
	assert.Equal(t, "#FF0000", folder.BackgroundColor)
	assert.Equal(t, "#FFFFFF", folder.TextColor)
}

func TestHTTPClient_CreateFolder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/folders", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New Folder", body["name"])

		response := map[string]any{
			"data": map[string]any{
				"id":          "folder-new",
				"name":        "New Folder",
				"total_count": 0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	req := &domain.CreateFolderRequest{
		Name: "New Folder",
	}
	folder, err := client.CreateFolder(ctx, "grant-123", req)

	require.NoError(t, err)
	assert.Equal(t, "folder-new", folder.ID)
	assert.Equal(t, "New Folder", folder.Name)
}

func TestHTTPClient_UpdateFolder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/folders/folder-789", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Renamed Folder", body["name"])

		response := map[string]any{
			"data": map[string]any{
				"id":          "folder-789",
				"name":        "Renamed Folder",
				"total_count": 25,
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
	req := &domain.UpdateFolderRequest{
		Name: "Renamed Folder",
	}
	folder, err := client.UpdateFolder(ctx, "grant-123", "folder-789", req)

	require.NoError(t, err)
	assert.Equal(t, "folder-789", folder.ID)
	assert.Equal(t, "Renamed Folder", folder.Name)
}

func TestHTTPClient_UpdateFolder_WithColors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Colorful", body["name"])
		assert.Equal(t, "#00FF00", body["background_color"])
		assert.Equal(t, "#000000", body["text_color"])

		response := map[string]any{
			"data": map[string]any{
				"id":               "folder-color",
				"name":             "Colorful",
				"background_color": "#00FF00",
				"text_color":       "#000000",
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
	req := &domain.UpdateFolderRequest{
		Name:            "Colorful",
		BackgroundColor: "#00FF00",
		TextColor:       "#000000",
	}
	folder, err := client.UpdateFolder(ctx, "grant-123", "folder-color", req)

	require.NoError(t, err)
	assert.Equal(t, "#00FF00", folder.BackgroundColor)
	assert.Equal(t, "#000000", folder.TextColor)
}

func TestHTTPClient_DeleteFolder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/folders/folder-delete", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.DeleteFolder(ctx, "grant-123", "folder-delete")

	require.NoError(t, err)
}

func TestHTTPClient_DeleteFolder_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			map[string]any{
				"error": map[string]string{
					"message": "Folder not found",
				},
			})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.DeleteFolder(ctx, "grant-123", "nonexistent")

	require.Error(t, err)
}

func TestMockClient_FolderOperations(t *testing.T) {
	ctx := context.Background()
	mock := nylas.NewMockClient()

	// Test GetFolders
	folders, err := mock.GetFolders(ctx, "grant-123")
	require.NoError(t, err)
	assert.NotEmpty(t, folders)
	assert.Equal(t, "grant-123", mock.LastGrantID)

	// Test GetFolder
	folder, err := mock.GetFolder(ctx, "grant-456", "folder-xyz")
	require.NoError(t, err)
	assert.Equal(t, "folder-xyz", folder.ID)
	assert.Equal(t, "grant-456", mock.LastGrantID)

	// Test CreateFolder
	req := &domain.CreateFolderRequest{Name: "Test Folder"}
	created, err := mock.CreateFolder(ctx, "grant-789", req)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Test Folder", created.Name)

	// Test UpdateFolder
	updateReq := &domain.UpdateFolderRequest{Name: "Updated Name"}
	updated, err := mock.UpdateFolder(ctx, "grant-abc", "folder-123", updateReq)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)

	// Test DeleteFolder
	err = mock.DeleteFolder(ctx, "grant-def", "folder-456")
	require.NoError(t, err)
}

func TestDemoClient_Folders(t *testing.T) {
	ctx := context.Background()
	demo := nylas.NewDemoClient()

	// Test GetFolders
	folders, err := demo.GetFolders(ctx, "demo-grant")
	require.NoError(t, err)
	assert.NotEmpty(t, folders)
	// Demo client should return realistic folders like Inbox, Sent, etc.
	foundInbox := false
	for _, f := range folders {
		if f.Name == "Inbox" {
			foundInbox = true
			break
		}
	}
	assert.True(t, foundInbox, "Demo client should return Inbox folder")

	// Test GetFolder
	folder, err := demo.GetFolder(ctx, "demo-grant", "demo-folder")
	require.NoError(t, err)
	assert.NotEmpty(t, folder.ID)

	// Test CreateFolder
	req := &domain.CreateFolderRequest{Name: "Demo New Folder"}
	created, err := demo.CreateFolder(ctx, "demo-grant", req)
	require.NoError(t, err)
	assert.Equal(t, "Demo New Folder", created.Name)

	// Test UpdateFolder
	updateReq := &domain.UpdateFolderRequest{Name: "Demo Renamed"}
	updated, err := demo.UpdateFolder(ctx, "demo-grant", "folder-id", updateReq)
	require.NoError(t, err)
	assert.Equal(t, "Demo Renamed", updated.Name)

	// Test DeleteFolder
	err = demo.DeleteFolder(ctx, "demo-grant", "folder-id")
	require.NoError(t, err)
}

// Validation tests
func TestHTTPClient_GetFolders_EmptyGrantID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	folders, err := client.GetFolders(ctx, "")

	require.Error(t, err)
	assert.Nil(t, folders)
	assert.Contains(t, err.Error(), "grant ID")
}

func TestHTTPClient_GetFolder_EmptyGrantID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	folder, err := client.GetFolder(ctx, "", "folder-123")

	require.Error(t, err)
	assert.Nil(t, folder)
	assert.Contains(t, err.Error(), "grant ID")
}

func TestHTTPClient_GetFolder_EmptyFolderID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	folder, err := client.GetFolder(ctx, "grant-123", "")

	require.Error(t, err)
	assert.Nil(t, folder)
	assert.Contains(t, err.Error(), "folder ID")
}

func TestHTTPClient_CreateFolder_EmptyGrantID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	req := &domain.CreateFolderRequest{Name: "Test"}
	folder, err := client.CreateFolder(ctx, "", req)

	require.Error(t, err)
	assert.Nil(t, folder)
	assert.Contains(t, err.Error(), "grant ID")
}

func TestHTTPClient_CreateFolder_WithAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-full/folders", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Full Folder", body["name"])
		assert.Equal(t, "parent-123", body["parent_id"])
		assert.Equal(t, "#0000FF", body["background_color"])
		assert.Equal(t, "#FFFF00", body["text_color"])

		response := map[string]any{
			"data": map[string]any{
				"id":               "folder-full",
				"name":             "Full Folder",
				"parent_id":        "parent-123",
				"background_color": "#0000FF",
				"text_color":       "#FFFF00",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	req := &domain.CreateFolderRequest{
		Name:            "Full Folder",
		ParentID:        "parent-123",
		BackgroundColor: "#0000FF",
		TextColor:       "#FFFF00",
	}
	folder, err := client.CreateFolder(ctx, "grant-full", req)

	require.NoError(t, err)
	assert.Equal(t, "folder-full", folder.ID)
	assert.Equal(t, "Full Folder", folder.Name)
	assert.Equal(t, "parent-123", folder.ParentID)
	assert.Equal(t, "#0000FF", folder.BackgroundColor)
	assert.Equal(t, "#FFFF00", folder.TextColor)
}

func TestHTTPClient_UpdateFolder_EmptyGrantID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	req := &domain.UpdateFolderRequest{Name: "Test"}
	folder, err := client.UpdateFolder(ctx, "", "folder-123", req)

	require.Error(t, err)
	assert.Nil(t, folder)
	assert.Contains(t, err.Error(), "grant ID")
}

func TestHTTPClient_UpdateFolder_EmptyFolderID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	req := &domain.UpdateFolderRequest{Name: "Test"}
	folder, err := client.UpdateFolder(ctx, "grant-123", "", req)

	require.Error(t, err)
	assert.Nil(t, folder)
	assert.Contains(t, err.Error(), "folder ID")
}

func TestHTTPClient_UpdateFolder_WithParentID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-parent/folders/folder-move", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Moved Folder", body["name"])
		assert.Equal(t, "new-parent-456", body["parent_id"])

		response := map[string]any{
			"data": map[string]any{
				"id":        "folder-move",
				"name":      "Moved Folder",
				"parent_id": "new-parent-456",
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
	req := &domain.UpdateFolderRequest{
		Name:     "Moved Folder",
		ParentID: "new-parent-456",
	}
	folder, err := client.UpdateFolder(ctx, "grant-parent", "folder-move", req)

	require.NoError(t, err)
	assert.Equal(t, "folder-move", folder.ID)
	assert.Equal(t, "Moved Folder", folder.Name)
	assert.Equal(t, "new-parent-456", folder.ParentID)
}

func TestHTTPClient_DeleteFolder_EmptyGrantID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	err := client.DeleteFolder(ctx, "", "folder-123")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "grant ID")
}

func TestHTTPClient_DeleteFolder_EmptyFolderID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	err := client.DeleteFolder(ctx, "grant-123", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "folder ID")
}

// SystemFolder handling tests
func TestHTTPClient_GetFolders_WithBoolSystemFolder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"data": []map[string]any{
				{
					"id":            "folder-bool-true",
					"name":          "Bool True Folder",
					"system_folder": true, // Boolean true
				},
				{
					"id":            "folder-bool-false",
					"name":          "Bool False Folder",
					"system_folder": false, // Boolean false
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
	folders, err := client.GetFolders(ctx, "grant-bool")

	require.NoError(t, err)
	assert.Len(t, folders, 2)

	// Boolean true should be converted to "true"
	assert.Equal(t, "folder-bool-true", folders[0].ID)
	assert.Equal(t, "true", folders[0].SystemFolder)

	// Boolean false should be converted to empty string
	assert.Equal(t, "folder-bool-false", folders[1].ID)
	assert.Equal(t, "", folders[1].SystemFolder)
}
