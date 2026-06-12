package nylas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListLists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/lists", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"items": []map[string]any{
					{"id": "list-001", "name": "Blocked domains", "type": "domain", "items_count": 2},
					{"id": "list-002", "name": "VIP addresses", "type": "address", "items_count": 5},
				},
			},
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	lists, err := client.ListLists(context.Background())
	require.NoError(t, err)
	require.Len(t, lists, 2)
	assert.Equal(t, "list-001", lists[0].ID)
	assert.Equal(t, "domain", lists[0].Type)
	assert.Equal(t, 2, lists[0].ItemsCount)
	assert.Equal(t, "list-002", lists[1].ID)
}

func TestListLists_Pagination(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			assert.Empty(t, r.URL.Query().Get("page_token"))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":        map[string]any{"items": []map[string]any{{"id": "list-001"}}},
				"next_cursor": "cursor-2",
			})
			return
		}
		assert.Equal(t, "cursor-2", r.URL.Query().Get("page_token"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"items": []map[string]any{{"id": "list-002"}}},
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	lists, err := client.ListLists(context.Background())
	require.NoError(t, err)
	require.Len(t, lists, 2)
	assert.Equal(t, 2, calls)
}

func TestGetList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/lists/list-001", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": "list-001", "name": "Blocked domains", "type": "domain"},
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	list, err := client.GetList(context.Background(), "list-001")
	require.NoError(t, err)
	assert.Equal(t, "list-001", list.ID)
	assert.Equal(t, "Blocked domains", list.Name)
}

func TestGetList_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"type": "not_found"}})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	_, err := client.GetList(context.Background(), "missing")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrListNotFound)
}

func TestCreateList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/lists", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "Blocked domains", payload["name"])
		assert.Equal(t, "domain", payload["type"])

		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": "list-new", "name": "Blocked domains", "type": "domain", "items_count": 0},
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	list, err := client.CreateList(context.Background(), map[string]any{"name": "Blocked domains", "type": "domain"})
	require.NoError(t, err)
	assert.Equal(t, "list-new", list.ID)
	assert.Equal(t, 0, list.ItemsCount)
}

func TestUpdateList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/lists/list-001", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"name": "Renamed"},
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	list, err := client.UpdateList(context.Background(), "list-001", map[string]any{"name": "Renamed"})
	require.NoError(t, err)
	// ID is backfilled when the API omits it from the update response.
	assert.Equal(t, "list-001", list.ID)
	assert.Equal(t, "Renamed", list.Name)
}

func TestDeleteList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/lists/list-001", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req-1"})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	require.NoError(t, client.DeleteList(context.Background(), "list-001"))
}

func TestGetListItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/lists/list-001/items", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"items": []string{"spam.com", "junk.net"}},
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	items, err := client.GetListItems(context.Background(), "list-001")
	require.NoError(t, err)
	assert.Equal(t, []string{"spam.com", "junk.net"}, items)
}

func TestGetListItems_Pagination(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, "/v3/lists/list-001/items", r.URL.Path)
		if calls == 1 {
			assert.Empty(t, r.URL.Query().Get("page_token"))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":        map[string]any{"items": []string{"spam.com"}},
				"next_cursor": "cursor-2",
			})
			return
		}
		assert.Equal(t, "cursor-2", r.URL.Query().Get("page_token"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"items": []string{"junk.net"}},
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	items, err := client.GetListItems(context.Background(), "list-001")
	require.NoError(t, err)
	assert.Equal(t, []string{"spam.com", "junk.net"}, items)
	assert.Equal(t, 2, calls)
}

func TestAddListItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/lists/list-001/items", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, []any{"spam.com", "junk.net"}, payload["items"])

		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": "list-001", "items_count": 2},
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	list, err := client.AddListItems(context.Background(), "list-001", []string{"spam.com", "junk.net"})
	require.NoError(t, err)
	assert.Equal(t, 2, list.ItemsCount)
}

func TestRemoveListItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/lists/list-001/items", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, []any{"spam.com"}, payload["items"])

		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": "list-001", "items_count": 1},
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	list, err := client.RemoveListItems(context.Background(), "list-001", []string{"spam.com"})
	require.NoError(t, err)
	assert.Equal(t, 1, list.ItemsCount)
}
