package air

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// statusCancelledOnlyMIME pins the case where METHOD is REQUEST (so the
// invite is "live") but the VEVENT carries STATUS:CANCELLED. The handler
// must still 409 — accepting an RSVP on a cancelled event would create
// a confusing diverging UI state.
const statusCancelledOnlyMIME = "Content-Type: multipart/alternative; boundary=\"B\"\r\n" +
	"\r\n--B\r\n" +
	"Content-Type: text/calendar; charset=UTF-8; method=REQUEST\r\n\r\n" +
	"BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//T//EN\r\nMETHOD:REQUEST\r\n" +
	"BEGIN:VEVENT\r\nUID:status-cancelled-uid@example.com\r\nSUMMARY:CancelledStatusOnly\r\n" +
	"DTSTART:20260501T140000Z\r\nDTEND:20260501T150000Z\r\n" +
	"STATUS:CANCELLED\r\n" +
	"END:VEVENT\r\nEND:VCALENDAR\r\n" +
	"--B--\r\n"

// TestHandleEmailRSVP_StatusCancelledOnly_Rejected covers the case where
// only STATUS:CANCELLED (no METHOD:CANCEL) marks the event as dead.
// Without this branch the 409 guard would only fire on Outlook-style
// cancellations and silently accept Google's status-only path.
func TestHandleEmailRSVP_StatusCancelledOnly_Rejected(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, statusCancelledOnlyMIME)

	calledRSVP := false
	mock.SendRSVPFunc = func(context.Context, string, string, string, *domain.SendRSVPRequest) error {
		calledRSVP = true
		return nil
	}

	w := postRSVP(t, server, "email-1", `{"status":"yes"}`)
	if w.Code != http.StatusConflict {
		t.Errorf("status=%d body=%s, want 409 (cancelled)", w.Code, w.Body.String())
	}
	if calledRSVP {
		t.Error("SendRSVP was called for STATUS:CANCELLED event — guard is missing")
	}
}

// TestHandleEmailRSVP_GetCalendarsError pins the upstream-failure path
// for the calendars listing. The handler should NOT report "no calendars
// found" (which would mislead the user) — it must surface 502 so the
// frontend offers a retry.
func TestHandleEmailRSVP_GetCalendarsError(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)
	mock.GetCalendarsFunc = func(context.Context, string) ([]domain.Calendar, error) {
		return nil, errors.New("nylas listing failed: 503")
	}

	w := postRSVP(t, server, "email-1", `{"status":"yes"}`)
	if w.Code != http.StatusBadGateway {
		t.Errorf("status=%d body=%s, want 502 on calendar listing failure", w.Code, w.Body.String())
	}
}

// TestHandleEmailRSVP_NoCalendars covers the zero-calendar branch.
// If Nylas returns an empty calendars list (no error), the handler
// should not silently RSVP to an empty calendarID — it must 502.
func TestHandleEmailRSVP_NoCalendars(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)
	mock.GetCalendarsFunc = func(context.Context, string) ([]domain.Calendar, error) {
		return []domain.Calendar{}, nil
	}

	w := postRSVP(t, server, "email-1", `{"status":"yes"}`)
	if w.Code != http.StatusBadGateway {
		t.Errorf("status=%d body=%s, want 502 (no calendars)", w.Code, w.Body.String())
	}
}

// TestHandleEmailRSVP_MissingDefaultGrant exercises the auth path: when
// no default grant is configured, withAuthGrant writes a 400 envelope
// and the handler returns without calling Nylas. Without this test, a
// future refactor could break the "select an account first" UX.
func TestHandleEmailRSVP_MissingDefaultGrant(t *testing.T) {
	t.Parallel()
	server, mock, _ := newCachedTestServer(t)

	// Strip the default grant so requireDefaultGrant fails closed.
	if err := server.grantStore.ClearGrants(); err != nil {
		t.Fatalf("clear grants: %v", err)
	}

	calledNylas := false
	mock.SendRSVPFunc = func(context.Context, string, string, string, *domain.SendRSVPRequest) error {
		calledNylas = true
		return nil
	}

	w := postRSVP(t, server, "email-1", `{"status":"yes"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d body=%s, want 400 (no default grant)", w.Code, w.Body.String())
	}
	if calledNylas {
		t.Error("SendRSVP called without a default grant — auth gate failed")
	}
}

// TestHandleEmailRSVP_DemoMode_RejectsInvalidStatus pins that body
// validation runs BEFORE the demo-mode short-circuit, so a hostile or
// stale frontend can't get a green-path RSVP response with garbage data.
func TestHandleEmailRSVP_DemoMode_RejectsInvalidStatus(t *testing.T) {
	t.Parallel()
	server, _, _ := newCachedTestServer(t)
	server.demoMode = true

	w := postRSVP(t, server, "any-email", `{"status":"bogus"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d body=%s, want 400 even in demo mode", w.Code, w.Body.String())
	}
}

// TestHandleEmailRSVP_DemoMode_PinsLiteralEventID pins the demo response
// shape. The frontend reads event_id back to highlight the active button,
// so a regression in the demo literal would silently break the demo UX.
func TestHandleEmailRSVP_DemoMode_PinsLiteralEventID(t *testing.T) {
	t.Parallel()
	server, _, _ := newCachedTestServer(t)
	server.demoMode = true

	w := postRSVP(t, server, "any-email", `{"status":"maybe"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got rsvpResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.EventID != "demo-event-001" || got.CalendarID != "primary" || got.Status != "maybe" {
		t.Errorf("demo response=%+v, want {Status:maybe, EventID:demo-event-001, CalendarID:primary}", got)
	}
}

// TestHandleEmailRSVP_OversizedComment pins the comment-length cap.
// Without this guard a single user could submit megabytes of comment
// text against an organiser's RSVP — the LimitedBody (1MB) bound is too
// permissive, so we cap at the UI-meaningful rsvpCommentMaxBytes.
//
// Boundary cases are also pinned: exactly rsvpCommentMaxBytes is OK,
// rsvpCommentMaxBytes+1 is rejected. This keeps the cap interpretation
// frozen against an off-by-one refactor.
func TestHandleEmailRSVP_OversizedComment(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		size      int
		wantCode  int
		wantCalls bool
	}{
		{name: "at cap", size: rsvpCommentMaxBytes, wantCode: http.StatusOK, wantCalls: true},
		{name: "one over cap", size: rsvpCommentMaxBytes + 1, wantCode: http.StatusBadRequest, wantCalls: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)
			calledRSVP := false
			mock.SendRSVPFunc = func(context.Context, string, string, string, *domain.SendRSVPRequest) error {
				calledRSVP = true
				return nil
			}
			body := `{"status":"yes","comment":"` + strings.Repeat("x", tc.size) + `"}`
			w := postRSVP(t, server, "email-1", body)
			if w.Code != tc.wantCode {
				t.Errorf("status=%d body=%s, want %d", w.Code, w.Body.String(), tc.wantCode)
			}
			if calledRSVP != tc.wantCalls {
				t.Errorf("SendRSVP called=%v, want %v", calledRSVP, tc.wantCalls)
			}
		})
	}
}

// TestHandleEmailRSVP_TrimsCommentBeforeLengthCheck pins that surrounding
// whitespace is stripped before the comment cap is applied. Without this,
// a user pasting from a WYSIWYG editor (which often includes trailing
// newline/space) could trip the limit on a message they perceive as
// short. Also pins that the trimmed form is what gets forwarded — the
// Nylas organiser shouldn't see "  yes please   " in their notification.
func TestHandleEmailRSVP_TrimsCommentBeforeLengthCheck(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)

	var got string
	mock.SendRSVPFunc = func(_ context.Context, _, _, _ string, req *domain.SendRSVPRequest) error {
		got = req.Comment
		return nil
	}
	body := `{"status":"yes","comment":"  see you there  \n"}`
	w := postRSVP(t, server, "email-1", body)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got != "see you there" {
		t.Errorf("forwarded comment=%q, want %q (trimmed)", got, "see you there")
	}
}

// TestHandleEmailRSVP_OversizedBodyReturns413 pins that a request body
// exceeding the 1MB MaxRequestBodySize cap returns 413 (RequestEntityTooLarge),
// not 400 (BadRequest). Without this distinction, clients can't tell
// "your JSON is malformed" (caller bug) from "your payload is too big"
// (caller can shrink and retry).
func TestHandleEmailRSVP_OversizedBodyReturns413(t *testing.T) {
	t.Parallel()
	server, _ := rsvpHappyPathMock(t, rsvpInviteMIME)

	// 2MB body — well past the 1MB MaxRequestBodySize cap.
	huge := strings.Repeat("a", 2<<20)
	body := `{"status":"yes","comment":"` + huge + `"}`
	w := postRSVP(t, server, "email-1", body)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status=%d body=%s, want 413 on >1MB body", w.Code, w.Body.String())
	}
}

// TestHandleEmailRSVP_BodyParseErrorIsGeneric pins the privacy contract:
// the JSON-decode error must NOT echo the raw bytes back to the client.
// (Localhost-only mitigates blast radius, but echoing the input weakens
// defense-in-depth and complicates anti-XSS reasoning.)
func TestHandleEmailRSVP_BodyParseErrorIsGeneric(t *testing.T) {
	t.Parallel()
	server, _ := rsvpHappyPathMock(t, rsvpInviteMIME)

	probe := "<script>alert('xss')</script>"
	w := postRSVP(t, server, "email-1", probe)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", w.Code)
	}
	if strings.Contains(w.Body.String(), probe) {
		t.Errorf("error response echoed raw client bytes: %s", w.Body.String())
	}
}

// TestHandleEmailRSVP_RouteDispatch wires the test through handleEmailByID
// (the actual entry point bound to /api/emails/{id}/rsvp) instead of
// calling handleEmailRSVP directly. This pins the path-splitting logic
// in handlers_email.go so a future refactor of `parts[1] == "rsvp"`
// can't silently break the route while every direct-call test still
// passes.
func TestHandleEmailRSVP_RouteDispatch(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)

	hit := false
	mock.SendRSVPFunc = func(context.Context, string, string, string, *domain.SendRSVPRequest) error {
		hit = true
		return nil
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/emails/email-1/rsvp", strings.NewReader(`{"status":"yes"}`))
	r.Header.Set("Content-Type", "application/json")
	server.handleEmailByID(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s — route dispatch is broken", w.Code, w.Body.String())
	}
	if !hit {
		t.Error("SendRSVP not called via /rsvp route — handleEmailByID failed to dispatch")
	}
}

// (TestHandleEmailRSVP_LooksUpAcrossCalendars removed — duplicate of
// TestHandleEmailRSVP_FindsEventInSecondaryCalendar in
// handlers_email_rsvp_test.go, which has stricter assertions on the
// ical_uid filter and uses reflect.DeepEqual for the walk order.)

// TestHandleEmailRSVP_TransientCalendarLookupFailureSurfacedWhenAllMiss
// pins that a non-errEventNotImported error from a per-calendar lookup
// (e.g. Nylas 5xx) is surfaced as 502 when no other calendar resolves
// the event. Without this, repeated "calendar not imported" 404s would
// hide a real Nylas outage.
func TestHandleEmailRSVP_TransientCalendarLookupFailureSurfacedWhenAllMiss(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)
	mock.GetCalendarsFunc = func(context.Context, string) ([]domain.Calendar, error) {
		return []domain.Calendar{
			{ID: "cal-primary", IsPrimary: true},
		}, nil
	}
	mock.GetEventsWithCursorFunc = func(context.Context, string, string, *domain.EventQueryParams) (*domain.EventListResponse, error) {
		return nil, errors.New("nylas events listing returned 503")
	}

	w := postRSVP(t, server, "email-1", `{"status":"yes"}`)
	if w.Code != http.StatusBadGateway {
		t.Errorf("status=%d body=%s, want 502 (transient lookup error)", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "503") {
		t.Errorf("response leaked upstream status code: %s", w.Body.String())
	}
}
