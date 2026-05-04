package air

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

// upstreamErrorSentinel is a noise-laden upstream error string that
// resembles real Nylas API failures: it carries grant identifiers,
// endpoint paths, and token-shaped fragments. Each writeUpstreamError
// site is supposed to redact this entirely from the response body and
// log the raw err via slog only. Tests assert the response body does
// not echo any of these substrings.
const upstreamErrorSentinel = "nylas 503: grant_id=secret-grant-leak-XYZ refresh_token=ya29.zzCANARYzz endpoint=/v3/grants/g/leak"

func bodyDoesNotLeakUpstream(t *testing.T, body string) {
	t.Helper()
	for _, fragment := range []string{
		"secret-grant-leak-XYZ",
		"ya29.zzCANARYzz",
		"/v3/grants/g/leak",
		"503",
	} {
		assert.NotContains(t, body, fragment,
			"response body leaked upstream-error fragment %q; got %s",
			fragment, body)
	}
}

// TestHandleListEmails_APIError_NoCache_DoesNotLeakUpstream pins the
// privacy contract on handlers_email.go:123 — the writeUpstreamError
// site reached when the API call fails AND the cache fallback yields
// nothing. The user gets a 500; the body must be a generic "please
// try again" message; the upstream err lives in slog only.
//
// Lock-down: redaction is in place today. A regression that
// re-introduced `err.Error()` interpolation would surface here.
func TestHandleListEmails_APIError_NoCache_DoesNotLeakUpstream(t *testing.T) {
	t.Parallel()
	server, client, _ := newCachedTestServer(t)
	// Mock routes GetMessagesWithCursor through GetMessagesWithParamsFunc
	// (see mock_messages.go:32) — set the params variant so the cursor
	// call still hits our error path.
	client.GetMessagesWithParamsFunc = func(_ context.Context, _ string, _ *domain.MessageQueryParams) ([]domain.Message, error) {
		return nil, errors.New(upstreamErrorSentinel)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/emails?folder=inbox", nil)
	w := httptest.NewRecorder()
	server.handleListEmails(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s, want 500 (no cache → upstream error)",
			w.Code, w.Body.String())
	}
	bodyDoesNotLeakUpstream(t, w.Body.String())
}

// TestHandleEmailInvite_APIError_DoesNotLeakUpstream pins the privacy
// contract on handlers_email_invite.go:96 — the writeUpstreamError
// site reached on the initial GetMessage failure (a hard error,
// distinct from errInviteFetchFailed which yields a silent
// HasInvite:false 200).
func TestHandleEmailInvite_APIError_DoesNotLeakUpstream(t *testing.T) {
	t.Parallel()
	server, client, _ := newCachedTestServer(t)
	client.GetMessageFunc = func(context.Context, string, string) (*domain.Message, error) {
		return nil, errors.New(upstreamErrorSentinel)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/emails/email-1/invite", nil)
	w := httptest.NewRecorder()
	server.handleEmailInvite(w, req, "email-1")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s, want 500 (upstream GetMessage failure)",
			w.Code, w.Body.String())
	}
	bodyDoesNotLeakUpstream(t, w.Body.String())
}

// TestHandleAvailability_UpstreamError_DoesNotLeakUpstream pins
// handlers_availability.go:226 — Nylas GetAvailability failure.
func TestHandleAvailability_UpstreamError_DoesNotLeakUpstream(t *testing.T) {
	t.Parallel()
	server, client, _ := newCachedTestServer(t)
	client.GetAvailabilityFunc = func(context.Context, *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error) {
		return nil, errors.New(upstreamErrorSentinel)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/availability?start_time=1700000000&end_time=1700100000&participants=alice@example.com",
		nil)
	w := httptest.NewRecorder()
	server.handleAvailability(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s, want 500 (upstream GetAvailability failure)",
			w.Code, w.Body.String())
	}
	bodyDoesNotLeakUpstream(t, w.Body.String())
}

// TestHandleFreeBusy_UpstreamError_DoesNotLeakUpstream pins
// handlers_availability.go:336 — Nylas GetFreeBusy failure.
func TestHandleFreeBusy_UpstreamError_DoesNotLeakUpstream(t *testing.T) {
	t.Parallel()
	server, client, _ := newCachedTestServer(t)
	client.GetFreeBusyFunc = func(context.Context, string, *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error) {
		return nil, errors.New(upstreamErrorSentinel)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/calendars/freebusy?start_time=1700000000&end_time=1700100000&emails=alice@example.com",
		nil)
	w := httptest.NewRecorder()
	server.handleFreeBusy(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s, want 500 (upstream GetFreeBusy failure)",
			w.Code, w.Body.String())
	}
	bodyDoesNotLeakUpstream(t, w.Body.String())
}

// TestHandleConflicts_UpstreamError_DoesNotLeakUpstream pins
// handlers_availability.go:443 — Nylas GetEventsWithCursor failure.
func TestHandleConflicts_UpstreamError_DoesNotLeakUpstream(t *testing.T) {
	t.Parallel()
	server, client, _ := newCachedTestServer(t)
	client.GetEventsWithCursorFunc = func(context.Context, string, string, *domain.EventQueryParams) (*domain.EventListResponse, error) {
		return nil, errors.New(upstreamErrorSentinel)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/calendar/conflicts?start_time=1700000000&end_time=1700100000",
		nil)
	w := httptest.NewRecorder()
	server.handleConflicts(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s, want 500 (upstream GetEventsWithCursor failure)",
			w.Code, w.Body.String())
	}
	bodyDoesNotLeakUpstream(t, w.Body.String())
}

// TestHandleGetEmail_APIError_NoCache_DoesNotLeakUpstream pins the
// privacy contract on handlers_email.go:253 — the writeUpstreamError
// site reached when GetMessage fails AND the cache fallback yields
// nothing (cache miss or cache disabled). The previous sweep covered
// list/invite/availability/freebusy/conflicts but the single-message
// fetch path was left out, so a regression that re-introduced
// %v(err) interpolation here would not be caught.
//
// Lock-down: redaction is in place today. Mirrors the pattern of
// TestHandleListEmails_APIError_NoCache_DoesNotLeakUpstream — same
// helper, same sentinel.
func TestHandleGetEmail_APIError_NoCache_DoesNotLeakUpstream(t *testing.T) {
	t.Parallel()
	server, client, _ := newCachedTestServer(t)
	client.GetMessageFunc = func(context.Context, string, string) (*domain.Message, error) {
		return nil, errors.New(upstreamErrorSentinel)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/emails/email-1", nil)
	w := httptest.NewRecorder()
	server.handleGetEmail(w, req, "email-1")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s, want 500 (no cache → upstream error)",
			w.Code, w.Body.String())
	}
	bodyDoesNotLeakUpstream(t, w.Body.String())
}
