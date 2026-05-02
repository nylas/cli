//go:build !integration

package air

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseInt64Param pins the contract that empty values are treated as
// "not provided" (zero, no error) but malformed values surface a clear
// error — the previous handler-level `_ = strconv.ParseInt(...)` form
// silently coerced both into zero, which masked client bugs.
func TestParseInt64Param(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		value   string
		want    int64
		wantErr bool
	}{
		{"empty is zero, no error", "", 0, false},
		{"valid integer", "1700000000", 1700000000, false},
		{"negative valid", "-42", -42, false},
		{"non-numeric", "tomorrow", 0, true},
		{"trailing junk", "12abc", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			q := url.Values{}
			if tc.value != "" {
				q.Set("k", tc.value)
			}
			got, err := parseInt64Param(q, "k")
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "k")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

// TestHandleAvailability_BadIntegerParams verifies the regression: bad
// query params now produce 400 Bad Request instead of being silently
// substituted by the "next 7 days" default. We use a fully configured
// server (mock client + a default grant) so the request reaches the
// param-validation stage instead of bouncing on requireConfig.
func TestHandleAvailability_BadIntegerParams(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		url  string
	}{
		{"bad start_time", "/api/availability?start_time=tomorrow"},
		{"bad end_time", "/api/availability?end_time=never"},
		{"bad duration_minutes", "/api/availability?duration_minutes=halfhour"},
		{"bad interval_minutes", "/api/availability?interval_minutes=fifteenish"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s, _, _ := newCachedTestServer(t)
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			w := httptest.NewRecorder()
			s.handleAvailability(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
		})
	}
}

func TestHandleFreeBusy_BadIntegerParams(t *testing.T) {
	t.Parallel()
	s, _, _ := newCachedTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/calendars/freebusy?start_time=oops", nil)
	w := httptest.NewRecorder()
	s.handleFreeBusy(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
}

// TestHandleConflicts_BadIntegerParams covers the parity gap left by the
// air-i003 review-pass: handleAvailability and handleFreeBusy got
// parseInt64Param-based validation AND BadIntegerParams tests, but
// handleConflicts only got the validation. A future refactor that
// re-introduces `_, _ = strconv.ParseInt` here would silently fall
// through to the default "current week" window with no test failure.
func TestHandleConflicts_BadIntegerParams(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		url  string
	}{
		{"bad start_time", "/api/calendar/conflicts?start_time=tomorrow"},
		{"bad end_time", "/api/calendar/conflicts?end_time=never"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s, _, _ := newCachedTestServer(t)
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			w := httptest.NewRecorder()
			s.handleConflicts(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
		})
	}
}

// TestHandleAvailability_BadParams_DoesNotEchoRawValue pins the privacy
// contract on the query-param error path. `parseInt64Param` builds
//
//	fmt.Errorf("invalid %s: %q is not a valid integer", key, raw)
//
// and the handler hands that error string straight to the client via
// `writeError(w, http.StatusBadRequest, perr.Error())`. The %q-encoded
// `raw` is the attacker's input echoed back into a JSON response —
// reflective surface that contradicts the writeUpstreamError
// redaction discipline this PR introduced one file over.
//
// EXPECTED FAILURE today: the response body contains the literal raw
// query value. After the fix the body should be a generic message
// (e.g., "invalid start_time") and the raw value should appear only in
// slog attrs.
func TestHandleAvailability_BadParams_DoesNotEchoRawValue(t *testing.T) {
	t.Parallel()
	const sentinel = "tomorrow-XYZ-canary"
	s, _, _ := newCachedTestServer(t)
	req := httptest.NewRequest(http.MethodGet,
		"/api/availability?start_time="+sentinel, nil)
	w := httptest.NewRecorder()
	s.handleAvailability(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NotContains(t, w.Body.String(), sentinel,
		"response must not reflect raw query value back to client")
}

func TestHandleFreeBusy_BadParams_DoesNotEchoRawValue(t *testing.T) {
	t.Parallel()
	const sentinel = "never-XYZ-canary"
	s, _, _ := newCachedTestServer(t)
	req := httptest.NewRequest(http.MethodGet,
		"/api/calendars/freebusy?start_time="+sentinel, nil)
	w := httptest.NewRecorder()
	s.handleFreeBusy(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NotContains(t, w.Body.String(), sentinel,
		"response must not reflect raw query value back to client")
}

func TestHandleConflicts_BadParams_DoesNotEchoRawValue(t *testing.T) {
	t.Parallel()
	const sentinel = "yesterday-XYZ-canary"
	s, _, _ := newCachedTestServer(t)
	req := httptest.NewRequest(http.MethodGet,
		"/api/calendar/conflicts?start_time="+sentinel, nil)
	w := httptest.NewRecorder()
	s.handleConflicts(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NotContains(t, w.Body.String(), sentinel,
		"response must not reflect raw query value back to client")
}

// TestHandleAvailability_BadDurationMinutes_DoesNotEchoRawValue closes
// the parity gap in the privacy sweep. Today the sweep covers
// start_time/end_time but not duration_minutes/interval_minutes —
// handlers_availability.go:160-169 routes all four through the same
// writeBadParamError helper, so a regression that inlines
// `writeError(perr.Error())` at one of the inner branches would escape
// the existing tests. Lock-down: same code path, same canary.
func TestHandleAvailability_BadDurationMinutes_DoesNotEchoRawValue(t *testing.T) {
	t.Parallel()
	const sentinel = "halfhour-XYZ-duration-canary"
	s, _, _ := newCachedTestServer(t)
	req := httptest.NewRequest(http.MethodGet,
		"/api/availability?duration_minutes="+sentinel, nil)
	w := httptest.NewRecorder()
	s.handleAvailability(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NotContains(t, w.Body.String(), sentinel,
		"duration_minutes raw value must not be reflected back to the client")
}

func TestHandleAvailability_BadIntervalMinutes_DoesNotEchoRawValue(t *testing.T) {
	t.Parallel()
	const sentinel = "fifteenish-XYZ-interval-canary"
	s, _, _ := newCachedTestServer(t)
	// IntervalMinutes is parsed AFTER start/end/duration/participants;
	// supply valid values for those so we reach the interval branch.
	req := httptest.NewRequest(http.MethodGet,
		"/api/availability?start_time=1700000000&end_time=1700100000&duration_minutes=30&interval_minutes="+sentinel, nil)
	w := httptest.NewRecorder()
	s.handleAvailability(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NotContains(t, w.Body.String(), sentinel,
		"interval_minutes raw value must not be reflected back to the client")
}

// Test handler types
func TestAvailabilityRequest_Fields(t *testing.T) {
	req := AvailabilityRequest{
		StartTime:       1704067200,
		EndTime:         1704153600,
		DurationMinutes: 30,
		Participants:    []string{"user1@example.com", "user2@example.com"},
		IntervalMinutes: 15,
	}

	assert.Equal(t, int64(1704067200), req.StartTime)
	assert.Equal(t, int64(1704153600), req.EndTime)
	assert.Equal(t, 30, req.DurationMinutes)
	assert.Equal(t, 15, req.IntervalMinutes)
	assert.Len(t, req.Participants, 2)
}

func TestAvailabilityResponse_Fields(t *testing.T) {
	resp := AvailabilityResponse{
		Slots: []AvailableSlotResponse{
			{StartTime: 1704067200, EndTime: 1704070800, Emails: []string{"test@example.com"}},
		},
		Message: "Test message",
	}

	assert.Len(t, resp.Slots, 1)
	assert.Equal(t, "Test message", resp.Message)
}

func TestAvailableSlotResponse_Fields(t *testing.T) {
	slot := AvailableSlotResponse{
		StartTime: 1704067200,
		EndTime:   1704070800,
		Emails:    []string{"user1@example.com", "user2@example.com"},
	}

	assert.Equal(t, int64(1704067200), slot.StartTime)
	assert.Equal(t, int64(1704070800), slot.EndTime)
	assert.Len(t, slot.Emails, 2)
}

func TestFreeBusyRequest_Fields(t *testing.T) {
	req := FreeBusyRequest{
		StartTime: 1704067200,
		EndTime:   1704153600,
		Emails:    []string{"user@example.com"},
	}

	assert.Equal(t, int64(1704067200), req.StartTime)
	assert.Equal(t, int64(1704153600), req.EndTime)
	assert.Len(t, req.Emails, 1)
}

func TestFreeBusyResponse_Fields(t *testing.T) {
	resp := FreeBusyResponse{
		Data: []FreeBusyCalendarResponse{
			{
				Email: "user@example.com",
				TimeSlots: []TimeSlotResponse{
					{StartTime: 1704067200, EndTime: 1704070800, Status: "busy"},
				},
			},
		},
	}

	assert.Len(t, resp.Data, 1)
	assert.Equal(t, "user@example.com", resp.Data[0].Email)
	assert.Len(t, resp.Data[0].TimeSlots, 1)
	assert.Equal(t, "busy", resp.Data[0].TimeSlots[0].Status)
}

func TestFreeBusyCalendarResponse_Fields(t *testing.T) {
	calResp := FreeBusyCalendarResponse{
		Email: "test@example.com",
		TimeSlots: []TimeSlotResponse{
			{StartTime: 1704067200, EndTime: 1704070800, Status: "busy"},
			{StartTime: 1704074400, EndTime: 1704078000, Status: "free"},
		},
	}

	assert.Equal(t, "test@example.com", calResp.Email)
	assert.Len(t, calResp.TimeSlots, 2)
}

func TestTimeSlotResponse_Fields(t *testing.T) {
	slot := TimeSlotResponse{
		StartTime: 1704067200,
		EndTime:   1704070800,
		Status:    "free",
	}

	assert.Equal(t, int64(1704067200), slot.StartTime)
	assert.Equal(t, int64(1704070800), slot.EndTime)
	assert.Equal(t, "free", slot.Status)
}

func TestConflictsResponse_Fields(t *testing.T) {
	resp := ConflictsResponse{
		Conflicts: []EventConflict{
			{
				Event1: EventResponse{ID: "e1", Title: "Event 1"},
				Event2: EventResponse{ID: "e2", Title: "Event 2"},
			},
		},
		HasMore: false,
	}

	assert.Len(t, resp.Conflicts, 1)
	assert.False(t, resp.HasMore)
}

func TestEventConflict_Fields(t *testing.T) {
	conflict := EventConflict{
		Event1: EventResponse{ID: "e1", Title: "Meeting 1"},
		Event2: EventResponse{ID: "e2", Title: "Meeting 2"},
	}

	assert.Equal(t, "e1", conflict.Event1.ID)
	assert.Equal(t, "e2", conflict.Event2.ID)
}

// Demo mode handler tests
func TestHandleAvailability_DemoMode(t *testing.T) {
	s := &Server{demoMode: true}

	req := httptest.NewRequest(http.MethodGet, "/api/availability", nil)
	w := httptest.NewRecorder()

	s.handleAvailability(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result AvailabilityResponse
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Slots)
	assert.Contains(t, result.Message, "Demo mode")
}

func TestHandleAvailability_DemoMode_POST(t *testing.T) {
	s := &Server{demoMode: true}

	req := httptest.NewRequest(http.MethodPost, "/api/availability", nil)
	w := httptest.NewRecorder()

	s.handleAvailability(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result AvailabilityResponse
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Slots)
}

func TestHandleAvailability_MethodNotAllowed(t *testing.T) {
	s := &Server{}

	req := httptest.NewRequest(http.MethodDelete, "/api/availability", nil)
	w := httptest.NewRecorder()

	s.handleAvailability(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestHandleFreeBusy_DemoMode(t *testing.T) {
	s := &Server{demoMode: true}

	req := httptest.NewRequest(http.MethodGet, "/api/calendars/freebusy", nil)
	w := httptest.NewRecorder()

	s.handleFreeBusy(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result FreeBusyResponse
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Data)
	assert.Equal(t, "demo@example.com", result.Data[0].Email)
}

func TestHandleFreeBusy_DemoMode_POST(t *testing.T) {
	s := &Server{demoMode: true}

	req := httptest.NewRequest(http.MethodPost, "/api/calendars/freebusy", nil)
	w := httptest.NewRecorder()

	s.handleFreeBusy(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandleFreeBusy_MethodNotAllowed(t *testing.T) {
	s := &Server{}

	req := httptest.NewRequest(http.MethodDelete, "/api/calendars/freebusy", nil)
	w := httptest.NewRecorder()

	s.handleFreeBusy(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestHandleConflicts_DemoMode(t *testing.T) {
	s := &Server{demoMode: true}

	req := httptest.NewRequest(http.MethodGet, "/api/calendar/conflicts", nil)
	w := httptest.NewRecorder()

	s.handleConflicts(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result ConflictsResponse
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Conflicts)
	assert.False(t, result.HasMore)
}

func TestHandleConflicts_MethodNotAllowed(t *testing.T) {
	s := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/calendar/conflicts", nil)
	w := httptest.NewRecorder()

	s.handleConflicts(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestHandleAvailability_NotConfigured(t *testing.T) {
	s := &Server{demoMode: false, nylasClient: nil}

	req := httptest.NewRequest(http.MethodGet, "/api/availability", nil)
	w := httptest.NewRecorder()

	s.handleAvailability(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandleFreeBusy_NotConfigured(t *testing.T) {
	s := &Server{demoMode: false, nylasClient: nil}

	req := httptest.NewRequest(http.MethodGet, "/api/calendars/freebusy", nil)
	w := httptest.NewRecorder()

	s.handleFreeBusy(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandleConflicts_NotConfigured(t *testing.T) {
	s := &Server{demoMode: false, nylasClient: nil}

	req := httptest.NewRequest(http.MethodGet, "/api/calendar/conflicts", nil)
	w := httptest.NewRecorder()

	s.handleConflicts(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}
