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

func TestHTTPClient_CreateVirtualCalendarGrant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connect/custom", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "virtual-calendar", body["provider"])

		settings, ok := body["settings"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "user@example.com", settings["email"])

		scope, ok := body["scope"].([]any)
		assert.True(t, ok)
		assert.Contains(t, scope, "calendar")

		response := map[string]any{
			"id":           "grant-123",
			"provider":     "virtual-calendar",
			"grant_status": "valid",
			"email":        "user@example.com",
			"scope":        []string{"calendar"},
			"user_id":      "user-456",
			"created_at":   int64(1609459200),
			"updated_at":   int64(1609459200),
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
	grant, err := client.CreateVirtualCalendarGrant(ctx, "user@example.com")

	require.NoError(t, err)
	assert.Equal(t, "grant-123", grant.ID)
	assert.Equal(t, "virtual-calendar", grant.Provider)
	assert.Equal(t, "user@example.com", grant.Email)
	assert.Equal(t, "valid", grant.GrantStatus)
}

func TestHTTPClient_ListVirtualCalendarGrants(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "virtual-calendar", r.URL.Query().Get("provider"))

		response := map[string]any{
			"data": []map[string]any{
				{
					"id":           "grant-1",
					"provider":     "virtual-calendar",
					"grant_status": "valid",
					"email":        "user1@example.com",
					"scope":        []string{"calendar"},
					"created_at":   int64(1609459200),
					"updated_at":   int64(1609459200),
				},
				{
					"id":           "grant-2",
					"provider":     "virtual-calendar",
					"grant_status": "valid",
					"email":        "user2@example.com",
					"scope":        []string{"calendar"},
					"created_at":   int64(1609459300),
					"updated_at":   int64(1609459300),
				},
			},
			"request_id": "req-123",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	grants, err := client.ListVirtualCalendarGrants(ctx)

	require.NoError(t, err)
	assert.Len(t, grants, 2)
	assert.Equal(t, "grant-1", grants[0].ID)
	assert.Equal(t, "user1@example.com", grants[0].Email)
	assert.Equal(t, "grant-2", grants[1].ID)
	assert.Equal(t, "user2@example.com", grants[1].Email)
}

func TestHTTPClient_GetVirtualCalendarGrant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-abc", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id":           "grant-abc",
				"provider":     "virtual-calendar",
				"grant_status": "valid",
				"email":        "admin@example.com",
				"scope":        []string{"calendar"},
				"user_id":      "user-789",
				"created_at":   int64(1609459200),
				"updated_at":   int64(1609459300),
			},
			"request_id": "req-456",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	grant, err := client.GetVirtualCalendarGrant(ctx, "grant-abc")

	require.NoError(t, err)
	assert.Equal(t, "grant-abc", grant.ID)
	assert.Equal(t, "virtual-calendar", grant.Provider)
	assert.Equal(t, "admin@example.com", grant.Email)
	assert.Equal(t, "valid", grant.GrantStatus)
}

func TestHTTPClient_DeleteVirtualCalendarGrant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-delete", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.DeleteVirtualCalendarGrant(ctx, "grant-delete")

	require.NoError(t, err)
}

func TestHTTPClient_GetRecurringEventInstances(t *testing.T) {
	now := int64(1609459200)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-recurring/events", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		// Check query params
		assert.Equal(t, "cal-123", r.URL.Query().Get("calendar_id"))
		assert.Equal(t, "event-master", r.URL.Query().Get("master_event_id"))
		assert.Equal(t, "true", r.URL.Query().Get("expand_recurring"))
		assert.Equal(t, "50", r.URL.Query().Get("limit"))

		response := map[string]any{
			"data": []map[string]any{
				{
					"id":          "event-instance-1",
					"grant_id":    "grant-recurring",
					"calendar_id": "cal-123",
					"title":       "Recurring Meeting - Instance 1",
					"when": map[string]any{
						"start_time": now,
						"end_time":   now + 3600,
						"object":     "timespan",
					},
					"created_at": now,
					"updated_at": now,
					"object":     "event",
				},
				{
					"id":          "event-instance-2",
					"grant_id":    "grant-recurring",
					"calendar_id": "cal-123",
					"title":       "Recurring Meeting - Instance 2",
					"when": map[string]any{
						"start_time": now + 86400,
						"end_time":   now + 86400 + 3600,
						"object":     "timespan",
					},
					"created_at": now,
					"updated_at": now,
					"object":     "event",
				},
			},
			"request_id": "req-recurring",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	events, err := client.GetRecurringEventInstances(ctx, "grant-recurring", "cal-123", "event-master", nil)

	require.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "event-instance-1", events[0].ID)
	assert.Equal(t, "Recurring Meeting - Instance 1", events[0].Title)
	assert.Equal(t, "event-instance-2", events[1].ID)
	assert.Equal(t, "Recurring Meeting - Instance 2", events[1].Title)
}

func TestHTTPClient_GetRecurringEventInstances_WithParams(t *testing.T) {
	now := int64(1609459200)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-params/events", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		// Check custom query params
		assert.Equal(t, "cal-456", r.URL.Query().Get("calendar_id"))
		assert.Equal(t, "event-custom", r.URL.Query().Get("master_event_id"))
		assert.Equal(t, "true", r.URL.Query().Get("expand_recurring"))
		assert.Equal(t, "100", r.URL.Query().Get("limit"))
		assert.Equal(t, "1609459200", r.URL.Query().Get("start"))
		assert.Equal(t, "1612051200", r.URL.Query().Get("end"))

		response := map[string]any{
			"data":       []map[string]any{},
			"request_id": "req-params",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	params := &domain.EventQueryParams{
		Limit: 100,
		Start: now,
		End:   now + 2592000, // 30 days later
	}
	events, err := client.GetRecurringEventInstances(ctx, "grant-params", "cal-456", "event-custom", params)

	require.NoError(t, err)
	assert.Len(t, events, 0)
}

func TestHTTPClient_UpdateRecurringEventInstance(t *testing.T) {
	now := int64(1609459200)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-update/events/event-instance-1", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "cal-update", r.URL.Query().Get("calendar_id"))

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Updated Recurring Meeting", body["title"])

		response := map[string]any{
			"data": map[string]any{
				"id":          "event-instance-1",
				"grant_id":    "grant-update",
				"calendar_id": "cal-update",
				"title":       "Updated Recurring Meeting",
				"when": map[string]any{
					"start_time": now,
					"end_time":   now + 3600,
					"object":     "timespan",
				},
				"created_at": now - 3600,
				"updated_at": now,
				"object":     "event",
			},
			"request_id": "req-update",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	title := "Updated Recurring Meeting"
	req := &domain.UpdateEventRequest{
		Title: &title,
	}

	event, err := client.UpdateRecurringEventInstance(ctx, "grant-update", "cal-update", "event-instance-1", req)

	require.NoError(t, err)
	assert.Equal(t, "event-instance-1", event.ID)
	assert.Equal(t, "Updated Recurring Meeting", event.Title)
	assert.Equal(t, "cal-update", event.CalendarID)
}

func TestHTTPClient_DeleteRecurringEventInstance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-del-instance/events/event-instance-3", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "cal-del", r.URL.Query().Get("calendar_id"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.DeleteRecurringEventInstance(ctx, "grant-del-instance", "cal-del", "event-instance-3")

	require.NoError(t, err)
}
