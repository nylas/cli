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

func TestHTTPClient_UpdateNotetaker(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Notetaker update is a PATCH (not PUT/POST) on the bare notetaker path.
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/v3/grants/grant-1/notetakers/nt-1", r.URL.Path)
		_ = json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": "nt-1", "state": "scheduled", "object": "notetaker"},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	video := false
	req := &domain.UpdateNotetakerRequest{
		JoinTime: 1893456000,
		Name:     "Recorder",
		MeetingSettings: &domain.NotetakerMeetingSettings{
			VideoRecording: &video,
		},
	}
	nt, err := client.UpdateNotetaker(context.Background(), "grant-1", "nt-1", req)

	require.NoError(t, err)
	assert.Equal(t, "nt-1", nt.ID)
	// The set fields must reach the wire; nested settings must be a sub-object,
	// and an explicit false must not be dropped.
	assert.Equal(t, "Recorder", body["name"])
	assert.EqualValues(t, 1893456000, body["join_time"])
	ms, ok := body["meeting_settings"].(map[string]any)
	require.True(t, ok, "meeting_settings should be a nested object")
	assert.Equal(t, false, ms["video_recording"])
}

func TestHTTPClient_UpdateNotetaker_Validation(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	t.Run("nil request", func(t *testing.T) {
		_, err := client.UpdateNotetaker(context.Background(), "grant-1", "nt-1", nil)
		assert.Error(t, err)
	})

	t.Run("empty request (no fields)", func(t *testing.T) {
		_, err := client.UpdateNotetaker(context.Background(), "grant-1", "nt-1", &domain.UpdateNotetakerRequest{})
		assert.Error(t, err)
	})
}
