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

func newGroupEventTestClient(t *testing.T, handler http.HandlerFunc) (*nylas.HTTPClient, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)
	return client, server.Close
}

func TestHTTPClient_ListGroupEvents(t *testing.T) {
	var gotQuery string
	client, closeFn := newGroupEventTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		// Group events are nested under grant + configuration.
		assert.Equal(t, "/v3/grants/grant-1/scheduling/configurations/cfg-1/group-events", r.URL.Path)
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": "ge-1", "title": "Workshop", "capacity": 50}},
		})
	})
	defer closeFn()

	events, err := client.ListGroupEvents(context.Background(), "grant-1", "cfg-1", "primary", 1735689600, 1738368000)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "ge-1", events[0].ID)
	assert.Equal(t, "Workshop", events[0].Title)
	// The list endpoint requires calendar_id + start_time + end_time; all must
	// be forwarded as query params (the API 400s without them).
	assert.Contains(t, gotQuery, "calendar_id=primary")
	assert.Contains(t, gotQuery, "start_time=1735689600")
	assert.Contains(t, gotQuery, "end_time=1738368000")
}

func TestHTTPClient_CreateGroupEvent(t *testing.T) {
	var body map[string]any
	client, closeFn := newGroupEventTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v3/grants/grant-1/scheduling/configurations/cfg-1/group-events", r.URL.Path)
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": "ge-new", "title": "Workshop"}},
		})
	})
	defer closeFn()

	req := &domain.CreateGroupEventRequest{
		CalendarID:   "primary",
		Title:        "Workshop",
		Capacity:     50,
		Participants: []domain.GroupEventParticipant{{Email: "nyla@example.com", IsOrganizer: true}},
		When:         &domain.GroupEventWhen{StartTime: 1744286400, EndTime: 1744290000, StartTimezone: "America/New_York"},
	}
	events, err := client.CreateGroupEvent(context.Background(), "grant-1", "cfg-1", req)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "ge-new", events[0].ID)

	// Required fields and the nested when object must reach the wire.
	assert.Equal(t, "primary", body["calendar_id"])
	assert.EqualValues(t, 50, body["capacity"])
	when, ok := body["when"].(map[string]any)
	require.True(t, ok, "when must be a nested object")
	assert.EqualValues(t, 1744286400, when["start_time"])

	// Participants must serialize as an array of objects carrying email and the
	// organizer flag — not be flattened or dropped.
	parts, ok := body["participants"].([]any)
	require.True(t, ok, "participants must be a JSON array")
	require.Len(t, parts, 1)
	p0, ok := parts[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "nyla@example.com", p0["email"])
	assert.Equal(t, true, p0["is_organizer"])
}

func TestHTTPClient_CreateGroupEvent_OmitsNilParticipants(t *testing.T) {
	var body map[string]any
	client, closeFn := newGroupEventTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"id": "ge"}}})
	})
	defer closeFn()

	// No participants supplied: the field must be OMITTED (so the API falls back
	// to the organizer), never sent as participants:null.
	_, err := client.CreateGroupEvent(context.Background(), "grant-1", "cfg-1", &domain.CreateGroupEventRequest{
		CalendarID: "primary", Title: "Solo", Capacity: 10,
		When: &domain.GroupEventWhen{StartTime: 1, EndTime: 2},
	})
	require.NoError(t, err)
	assert.NotContains(t, body, "participants", "nil participants must be omitted, not sent as null")
}

func TestHTTPClient_UpdateGroupEvent(t *testing.T) {
	var body map[string]any
	client, closeFn := newGroupEventTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v3/grants/grant-1/scheduling/configurations/cfg-1/group-events/ge-1", r.URL.Path)
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"id": "ge-1", "title": "New Title"}}})
	})
	defer closeFn()

	events, err := client.UpdateGroupEvent(context.Background(), "grant-1", "cfg-1", "ge-1", &domain.UpdateGroupEventRequest{
		Title:    "New Title",
		Capacity: 80,
	})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "New Title", events[0].Title)
	// The changed fields must actually be in the PUT body.
	assert.Equal(t, "New Title", body["title"])
	assert.EqualValues(t, 80, body["capacity"])
}

func TestHTTPClient_DeleteGroupEvent(t *testing.T) {
	client, closeFn := newGroupEventTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/v3/grants/grant-1/scheduling/configurations/cfg-1/group-events/ge-1", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"request_id":"r"}`))
	})
	defer closeFn()

	err := client.DeleteGroupEvent(context.Background(), "grant-1", "cfg-1", "ge-1")
	assert.NoError(t, err)
}

func TestHTTPClient_ImportGroupEvents(t *testing.T) {
	var body []map[string]any
	client, closeFn := newGroupEventTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		// Import is configuration-scoped, NOT grant-scoped.
		assert.Equal(t, "/v3/scheduling/configurations/cfg-1/import-group-events", r.URL.Path)
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"id": "ge-imp"}}})
	})
	defer closeFn()

	events, err := client.ImportGroupEvents(context.Background(), "cfg-1", []domain.ImportGroupEventItem{
		{CalendarID: "primary", EventID: "evt-9", Capacity: 30},
	})
	require.NoError(t, err)
	require.Len(t, events, 1)
	// Body must be a JSON array carrying the import item's required fields.
	require.Len(t, body, 1)
	assert.Equal(t, "primary", body[0]["calendar_id"])
	assert.Equal(t, "evt-9", body[0]["event_id"])
}

func TestHTTPClient_GroupEvents_Validation(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	_, err := client.ListGroupEvents(context.Background(), "", "cfg-1", "primary", 1, 2)
	assert.Error(t, err, "missing grant")
	_, err = client.ListGroupEvents(context.Background(), "grant-1", "", "primary", 1, 2)
	assert.Error(t, err, "missing config")
	_, err = client.ListGroupEvents(context.Background(), "grant-1", "cfg-1", "", 1, 2)
	assert.Error(t, err, "missing calendar")
	_, err = client.ListGroupEvents(context.Background(), "grant-1", "cfg-1", "primary", 0, 0)
	assert.Error(t, err, "missing time window")
	err = client.DeleteGroupEvent(context.Background(), "grant-1", "cfg-1", "")
	assert.Error(t, err, "missing event id")
	_, err = client.ImportGroupEvents(context.Background(), "cfg-1", nil)
	assert.Error(t, err, "no items")
}
