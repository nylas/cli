package calendar

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEventsUpdateCmd_TimezoneWiring runs the real `events update` command
// against a fake Nylas API and asserts the PUT payload. It guards the update
// path's wiring of --timezone into parseEventTime — the create path got this
// fix first; this proves update did too — and the --lock-timezone metadata
// surviving through the UpdateEvent adapter.
func TestEventsUpdateCmd_TimezoneWiring(t *testing.T) {
	var captured struct {
		When struct {
			StartTime     int64  `json:"start_time"`
			EndTime       int64  `json:"end_time"`
			StartTimezone string `json:"start_timezone"`
			EndTimezone   string `json:"end_timezone"`
			Object        string `json:"object"`
		} `json:"when"`
		Metadata map[string]string `json:"metadata"`
	}
	requestSeen := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			// --lock-timezone fetches the event to merge existing metadata.
			_, _ = w.Write([]byte(`{"request_id":"req-g","data":{"id":"event-1","title":"Existing"}}`))
		case http.MethodPut:
			requestSeen = true
			assert.Equal(t, "/v3/grants/grant-123/events/event-1", r.URL.Path)
			assert.Equal(t, "cal-1", r.URL.Query().Get("calendar_id"))
			require.NoError(t, json.NewDecoder(r.Body).Decode(&captured))
			_, _ = w.Write([]byte(`{"request_id":"req-1","data":{"id":"event-1","title":"Updated"}}`))
		default:
			t.Errorf("unexpected %s request to %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	// Isolate from the developer's real config/keyring; env API key wins.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("NYLAS_API_KEY", "test-api-key")
	t.Setenv("NYLAS_API_BASE_URL", server.URL)

	cmd := newEventsUpdateCmd()
	cmd.SetArgs([]string{
		"event-1", "grant-123",
		"--calendar", "cal-1",
		"--start", "2024-01-15 14:00",
		"--end", "2024-01-15 15:00",
		"--timezone", "Asia/Tokyo",
		"--lock-timezone",
	})

	require.NoError(t, cmd.Execute())
	require.True(t, requestSeen, "update command never hit the API")

	// The timestamps must be the Tokyo wall clock, and the recorded zones
	// must match — otherwise --timezone on update silently falls back to the
	// system zone.
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)
	assert.Equal(t, "timespan", captured.When.Object)
	assert.Equal(t, time.Date(2024, 1, 15, 14, 0, 0, 0, tokyo).Unix(), captured.When.StartTime)
	assert.Equal(t, time.Date(2024, 1, 15, 15, 0, 0, 0, tokyo).Unix(), captured.When.EndTime)
	assert.Equal(t, "Asia/Tokyo", captured.When.StartTimezone)
	assert.Equal(t, "Asia/Tokyo", captured.When.EndTimezone)
	assert.Equal(t, "true", captured.Metadata["timezone_locked"],
		"--lock-timezone metadata must survive through the UpdateEvent adapter")
}

// TestEventsUpdateCmd_TimezoneWithoutStartFails verifies --timezone is rejected
// when --start is absent: the zone is only applied while parsing new times, so
// accepting it alone would silently do nothing.
func TestEventsUpdateCmd_TimezoneWithoutStartFails(t *testing.T) {
	apiCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		http.Error(w, "should not be called", http.StatusBadRequest)
	}))
	defer server.Close()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("NYLAS_API_KEY", "test-api-key")
	t.Setenv("NYLAS_API_BASE_URL", server.URL)

	cmd := newEventsUpdateCmd()
	cmd.SetArgs([]string{
		"event-1", "grant-123",
		"--calendar", "cal-1",
		"--timezone", "Asia/Tokyo",
	})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--timezone requires --start")
	assert.False(t, apiCalled, "--timezone without --start must be rejected before any API call")
}

// TestEventsUpdateCmd_LockTimezonePreservesMetadata verifies the lock/unlock
// update merges with the event's existing metadata instead of clobbering it:
// the Nylas update replaces the metadata object wholesale, so the command must
// send back every existing key alongside timezone_locked.
func TestEventsUpdateCmd_LockTimezonePreservesMetadata(t *testing.T) {
	var capturedMetadata map[string]string
	putSeen := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			// Existing event carries unrelated metadata that must survive.
			_, _ = w.Write([]byte(`{"request_id":"req-g","data":{"id":"event-1","title":"Existing","metadata":{"project":"apollo","priority":"high"}}}`))
		case http.MethodPut:
			putSeen = true
			var body struct {
				Metadata map[string]string `json:"metadata"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			capturedMetadata = body.Metadata
			_, _ = w.Write([]byte(`{"request_id":"req-p","data":{"id":"event-1","title":"Existing"}}`))
		default:
			t.Errorf("unexpected %s request to %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("NYLAS_API_KEY", "test-api-key")
	t.Setenv("NYLAS_API_BASE_URL", server.URL)

	cmd := newEventsUpdateCmd()
	cmd.SetArgs([]string{
		"event-1", "grant-123",
		"--calendar", "cal-1",
		"--lock-timezone",
	})

	require.NoError(t, cmd.Execute())
	require.True(t, putSeen, "update command never sent the PUT")
	assert.Equal(t, "true", capturedMetadata["timezone_locked"])
	assert.Equal(t, "apollo", capturedMetadata["project"],
		"existing metadata keys must be preserved when locking the timezone")
	assert.Equal(t, "high", capturedMetadata["priority"])
}

// TestEventsUpdateCmd_InvalidTimezoneFailsBeforeAPI verifies a bad --timezone
// is rejected locally: the update must not reach the API with a half-built
// payload.
func TestEventsUpdateCmd_InvalidTimezoneFailsBeforeAPI(t *testing.T) {
	apiCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		http.Error(w, "should not be called", http.StatusBadRequest)
	}))
	defer server.Close()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("NYLAS_API_KEY", "test-api-key")
	t.Setenv("NYLAS_API_BASE_URL", server.URL)

	cmd := newEventsUpdateCmd()
	cmd.SetArgs([]string{
		"event-1", "grant-123",
		"--calendar", "cal-1",
		"--start", "2024-01-15 14:00",
		"--timezone", "Not/AZone",
	})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timezone")
	assert.False(t, apiCalled, "invalid timezone must be rejected before any API call")
}
