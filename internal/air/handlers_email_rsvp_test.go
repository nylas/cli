package air

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// MIME fixtures, rsvpHappyPathMock, and postRSVP live in
// handlers_email_rsvp_fixtures_test.go (extracted to keep this file
// under the 600-line ceiling).

func TestHandleEmailRSVP_HappyPath(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)

	var sentReq *domain.SendRSVPRequest
	var sentEventID, sentCalendarID string
	mock.SendRSVPFunc = func(_ context.Context, grantID, calendarID, eventID string, req *domain.SendRSVPRequest) error {
		if grantID != "grant-123" {
			// Use Fatalf — the rest of this assertion block (event ID,
			// calendar ID, request body) is meaningless if we ended up on
			// the wrong grant. Loud-fail at the source so debugging
			// points at the auth gate, not a downstream symptom.
			t.Fatalf("grantID=%q, want grant-123", grantID)
		}
		sentEventID = eventID
		sentCalendarID = calendarID
		sentReq = req
		return nil
	}

	w := postRSVP(t, server, "email-1", `{"status":"yes","comment":"see you there"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if sentEventID != "evt-resolved-1" {
		t.Errorf("eventID=%q, want evt-resolved-1 (resolved by ical_uid)", sentEventID)
	}
	if sentCalendarID != "cal-primary" {
		t.Errorf("calendarID=%q, want cal-primary", sentCalendarID)
	}
	if sentReq == nil || sentReq.Status != "yes" || sentReq.Comment != "see you there" {
		t.Errorf("request=%+v, want {Status:yes, Comment:see you there}", sentReq)
	}

	var got rsvpResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Status != "yes" || got.EventID != "evt-resolved-1" || got.CalendarID != "cal-primary" {
		t.Errorf("response=%+v", got)
	}
}

func TestHandleEmailRSVP_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	server, _ := rsvpHappyPathMock(t, rsvpInviteMIME)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/emails/email-1/rsvp", http.NoBody)
	server.handleEmailRSVP(w, r, "email-1")

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status=%d, want 405", w.Code)
	}
}

func TestHandleEmailRSVP_InvalidStatus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
	}{
		{"empty", `{"status":""}`},
		{"unknown", `{"status":"sure"}`},
		{"yes-with-typo", `{"status":"yess"}`},
		{"noreply rejected", `{"status":"noreply"}`}, // valid in Nylas API but no UI affordance
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server, _ := rsvpHappyPathMock(t, rsvpInviteMIME)
			w := postRSVP(t, server, "email-1", tc.body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status=%d, want 400, body=%s", w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleEmailRSVP_StatusCaseInsensitive pins that yes/no/maybe in
// any case (including with surrounding whitespace) all normalize to the
// lowercase Nylas vocabulary before being forwarded. Without exercising
// every branch a future refactor of the `strings.ToLower` chain could
// silently regress one variant.
func TestHandleEmailRSVP_StatusCaseInsensitive(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "uppercase yes", input: "YES", want: "yes"},
		{name: "titlecase maybe", input: "Maybe", want: "maybe"},
		{name: "uppercase no", input: "NO", want: "no"},
		{name: "padded yes", input: "  yes  ", want: "yes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)
			var got string
			mock.SendRSVPFunc = func(_ context.Context, _, _, _ string, req *domain.SendRSVPRequest) error {
				got = req.Status
				return nil
			}
			w := postRSVP(t, server, "email-1", `{"status":"`+tc.input+`"}`)
			if w.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if got != tc.want {
				t.Errorf("forwarded status=%q, want %q", got, tc.want)
			}
		})
	}
}

func TestHandleEmailRSVP_InvalidJSONBody(t *testing.T) {
	t.Parallel()
	server, _ := rsvpHappyPathMock(t, rsvpInviteMIME)

	w := postRSVP(t, server, "email-1", `not json at all`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400", w.Code)
	}
}

func TestHandleEmailRSVP_NoInviteOnEmail(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)

	mock.GetMessageWithFieldsFunc = func(context.Context, string, string, string) (*domain.Message, error) {
		return &domain.Message{RawMIME: "From: a@b.example\r\nSubject: Hi\r\n\r\nplain"}, nil
	}

	w := postRSVP(t, server, "email-1", `{"status":"yes"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("status=%d body=%s, want 404 (no calendar invitation)", w.Code, w.Body.String())
	}
}

func TestHandleEmailRSVP_CancelledInviteRejected(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, cancelledInviteMIME)

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
		t.Error("SendRSVP was called for a cancelled event — guard is missing")
	}
}

func TestHandleEmailRSVP_MissingUIDRejected(t *testing.T) {
	t.Parallel()
	server, _ := rsvpHappyPathMock(t, noUIDInviteMIME)

	w := postRSVP(t, server, "email-1", `{"status":"yes"}`)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d body=%s, want 422 (no UID)", w.Code, w.Body.String())
	}
}

func TestHandleEmailRSVP_NoMatchingEvent(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)

	mock.GetEventsWithCursorFunc = func(context.Context, string, string, *domain.EventQueryParams) (*domain.EventListResponse, error) {
		return &domain.EventListResponse{Data: nil}, nil
	}

	w := postRSVP(t, server, "email-1", `{"status":"yes"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("status=%d body=%s, want 404 (event not imported)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "imported") {
		t.Errorf("error message should mention import lag, got %s", w.Body.String())
	}
}

// TestHandleEmailRSVP_NoWritableCalendar pins the user-facing surface
// for "this account has only read-only calendars". This is a
// config-shaped failure — not transient — so the response must be 422
// with a descriptive message, not a generic 502 that would invite a
// useless retry. Asserting the message body too keeps a future refactor
// from silently broadening the user-visible string.
func TestHandleEmailRSVP_NoWritableCalendar(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)

	mock.GetCalendarsFunc = func(context.Context, string) ([]domain.Calendar, error) {
		return []domain.Calendar{
			{ID: "subscribed", Name: "US Holidays", IsPrimary: true, ReadOnly: true},
		}, nil
	}

	w := postRSVP(t, server, "email-1", `{"status":"yes"}`)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d body=%s, want 422 (no writable calendar — config issue, not transient)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "writable") {
		t.Errorf("response should mention 'writable' so users know what to fix; got %s", w.Body.String())
	}
}

// TestHandleEmailRSVP_FindsEventInSecondaryCalendar pins the multi-calendar
// search: when the invite landed in a writable calendar that ISN'T the
// primary (work + personal account, shared team calendar, etc.), the
// handler must keep looking instead of returning "not imported" off the
// first calendar's empty result. Also pins the lookup ORDER — primary
// must be queried first so we don't surprise users with a slower or
// less-likely-to-match calendar getting priority.
func TestHandleEmailRSVP_FindsEventInSecondaryCalendar(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)

	mock.GetCalendarsFunc = func(context.Context, string) ([]domain.Calendar, error) {
		return []domain.Calendar{
			{ID: "primary-cal", IsPrimary: true, ReadOnly: false},
			{ID: "team-cal", IsPrimary: false, ReadOnly: false},
		}, nil
	}
	// Record query order so we can pin "primary first, then secondaries"
	// — the docstring on findInviteEventAcrossCalendars promises this and
	// without an order assertion a future refactor could quietly invert it.
	var queryOrder []string
	mock.GetEventsWithCursorFunc = func(_ context.Context, _, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error) {
		if params == nil || params.ICalUID == "" {
			t.Errorf("ical_uid filter missing on calendar %q", calendarID)
		}
		queryOrder = append(queryOrder, calendarID)
		if calendarID == "team-cal" {
			return &domain.EventListResponse{Data: []domain.Event{{ID: "evt-team-1"}}}, nil
		}
		return &domain.EventListResponse{Data: nil}, nil
	}

	var sentCal, sentEvent string
	mock.SendRSVPFunc = func(_ context.Context, _, calendarID, eventID string, _ *domain.SendRSVPRequest) error {
		sentCal = calendarID
		sentEvent = eventID
		return nil
	}

	w := postRSVP(t, server, "email-1", `{"status":"yes"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200 (event must resolve via secondary calendar)", w.Code, w.Body.String())
	}
	if sentCal != "team-cal" || sentEvent != "evt-team-1" {
		t.Errorf("RSVP sent to (%q, %q), want (team-cal, evt-team-1)", sentCal, sentEvent)
	}
	wantOrder := []string{"primary-cal", "team-cal"}
	if !reflect.DeepEqual(queryOrder, wantOrder) {
		t.Errorf("calendar lookup order=%v, want %v (primary must be queried first)", queryOrder, wantOrder)
	}
}

// TestHandleEmailRSVP_TransientErrorOnPrimaryFallsThroughToSecondary
// pins the partial-failure walk: a flaky primary should NOT abort the
// search if a later calendar holds the event. Without this test, a
// future refactor could decide "tracked errors abort the loop" without
// breaking any existing assertion.
func TestHandleEmailRSVP_TransientErrorOnPrimaryFallsThroughToSecondary(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)

	mock.GetCalendarsFunc = func(context.Context, string) ([]domain.Calendar, error) {
		return []domain.Calendar{
			{ID: "primary-cal", IsPrimary: true, ReadOnly: false},
			{ID: "team-cal", IsPrimary: false, ReadOnly: false},
		}, nil
	}
	mock.GetEventsWithCursorFunc = func(_ context.Context, _, calendarID string, _ *domain.EventQueryParams) (*domain.EventListResponse, error) {
		if calendarID == "primary-cal" {
			// Transient blip — must NOT short-circuit the walk.
			return nil, errors.New("upstream 503 service unavailable")
		}
		return &domain.EventListResponse{Data: []domain.Event{{ID: "evt-team-2"}}}, nil
	}

	var sentCal, sentEvent string
	mock.SendRSVPFunc = func(_ context.Context, _, calendarID, eventID string, _ *domain.SendRSVPRequest) error {
		sentCal = calendarID
		sentEvent = eventID
		return nil
	}

	w := postRSVP(t, server, "email-1", `{"status":"yes"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200 (transient err on primary must fall through)", w.Code, w.Body.String())
	}
	if sentCal != "team-cal" || sentEvent != "evt-team-2" {
		t.Errorf("RSVP sent to (%q, %q), want (team-cal, evt-team-2)", sentCal, sentEvent)
	}
}

// TestHandleEmailRSVP_TransientLookupErrorSurfacedWhenAllFail pins that
// when every writable calendar errors on the lookup, the handler returns
// 502 (so the user can retry) — NOT 404. A transient Nylas blip on a
// secondary calendar shouldn't be reported as "this invite doesn't
// exist."
func TestHandleEmailRSVP_TransientLookupErrorSurfacedWhenAllFail(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)

	mock.GetCalendarsFunc = func(context.Context, string) ([]domain.Calendar, error) {
		return []domain.Calendar{
			{ID: "cal-a", IsPrimary: true, ReadOnly: false},
			{ID: "cal-b", IsPrimary: false, ReadOnly: false},
		}, nil
	}
	mock.GetEventsWithCursorFunc = func(context.Context, string, string, *domain.EventQueryParams) (*domain.EventListResponse, error) {
		return nil, errors.New("upstream 502")
	}

	w := postRSVP(t, server, "email-1", `{"status":"yes"}`)
	if w.Code != http.StatusBadGateway {
		t.Errorf("status=%d body=%s, want 502 (transient lookup error must not be 404)", w.Code, w.Body.String())
	}
}

func TestHandleEmailRSVP_FallsBackToFirstWritableCalendar(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)

	mock.GetCalendarsFunc = func(context.Context, string) ([]domain.Calendar, error) {
		// Primary is read-only (e.g. a "holidays" subscription marked
		// primary by mistake) — handler should fall through to the next
		// writable calendar instead of failing.
		return []domain.Calendar{
			{ID: "ro", IsPrimary: true, ReadOnly: true},
			{ID: "writable", IsPrimary: false, ReadOnly: false},
		}, nil
	}

	var seenCal string
	mock.SendRSVPFunc = func(_ context.Context, _, calendarID, _ string, _ *domain.SendRSVPRequest) error {
		seenCal = calendarID
		return nil
	}

	w := postRSVP(t, server, "email-1", `{"status":"maybe"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if seenCal != "writable" {
		t.Errorf("calendarID=%q, want fallback to writable", seenCal)
	}
}

func TestHandleEmailRSVP_UpstreamSendRSVPError(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)

	mock.SendRSVPFunc = func(context.Context, string, string, string, *domain.SendRSVPRequest) error {
		return errors.New("nylas API error: 503 service unavailable")
	}

	w := postRSVP(t, server, "email-1", `{"status":"no"}`)
	if w.Code != http.StatusBadGateway {
		t.Errorf("status=%d body=%s, want 502", w.Code, w.Body.String())
	}
}

func TestHandleEmailRSVP_UpstreamGetMessageError(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)

	mock.GetMessageFunc = func(context.Context, string, string) (*domain.Message, error) {
		return nil, errors.New("nylas API error: 500")
	}

	w := postRSVP(t, server, "email-1", `{"status":"yes"}`)
	if w.Code != http.StatusBadGateway {
		t.Errorf("status=%d body=%s, want 502", w.Code, w.Body.String())
	}
}

// TestHandleEmailRSVP_RawMimeFetchFails pins the regression where a
// transient raw_mime fetch failure was misclassified as "no invite" and
// returned 404. Now it must propagate as 502 so the frontend can offer
// a retry rather than telling the user the invite doesn't exist.
func TestHandleEmailRSVP_RawMimeFetchFails(t *testing.T) {
	t.Parallel()
	server, mock := rsvpHappyPathMock(t, rsvpInviteMIME)

	// GetMessage succeeds but exposes no parseable attachment, so the
	// resolver MUST fall through to GetMessageWithFields. Make that fail
	// with a transient-looking error.
	mock.GetMessageFunc = func(_ context.Context, _, messageID string) (*domain.Message, error) {
		return &domain.Message{ID: messageID, Subject: "Event Invitation"}, nil
	}
	mock.GetMessageWithFieldsFunc = func(context.Context, string, string, string) (*domain.Message, error) {
		return nil, errors.New("upstream timeout: 504")
	}

	w := postRSVP(t, server, "email-1", `{"status":"yes"}`)
	if w.Code != http.StatusBadGateway {
		t.Errorf("status=%d body=%s, want 502 (raw_mime fetch failure should not be 404)", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "does not contain a calendar invitation") {
		t.Errorf("raw_mime fetch failure must not be misreported as 'no invite'; body=%s", w.Body.String())
	}
}

// TestHandleEmailInvite_RawMimeFetchSilentlyDegrades pins the inverse
// for the preview endpoint: the same upstream failure should silently
// return HasInvite:false (so the email view doesn't error out) rather
// than crash the page with 500. Behavioural pair to the test above.
func TestHandleEmailInvite_RawMimeFetchSilentlyDegrades(t *testing.T) {
	t.Parallel()
	server, mock, _ := newCachedTestServer(t)

	mock.GetMessageFunc = func(_ context.Context, _, messageID string) (*domain.Message, error) {
		return &domain.Message{ID: messageID, Subject: "Event Invitation"}, nil
	}
	mock.GetMessageWithFieldsFunc = func(context.Context, string, string, string) (*domain.Message, error) {
		return nil, errors.New("upstream timeout: 504")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/emails/email-1/invite", http.NoBody)
	server.handleEmailInvite(w, r, "email-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200 (silent degrade)", w.Code, w.Body.String())
	}
	var got CalendarInviteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.HasInvite {
		t.Errorf("HasInvite=true on raw_mime fetch failure, want false; body=%s", w.Body.String())
	}
}

func TestHandleEmailRSVP_DemoMode(t *testing.T) {
	t.Parallel()
	server, _, _ := newCachedTestServer(t)
	server.demoMode = true

	w := postRSVP(t, server, "any-email", `{"status":"yes"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got rsvpResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Status != "yes" || got.EventID == "" {
		t.Errorf("demo response=%+v", got)
	}
}

func TestHandleEmailRSVP_AttachmentPathHasUID(t *testing.T) {
	// When the email carries the ICS as a real attachment (Microsoft's
	// shape), the parser should still surface the UID so RSVP works.
	t.Parallel()
	server, mock, _ := newCachedTestServer(t)

	icsBody := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//T//EN\r\nMETHOD:REQUEST\r\n" +
		"BEGIN:VEVENT\r\nUID:attached-uid@example.com\r\nSUMMARY:Attached\r\n" +
		"DTSTART:20260501T140000Z\r\nDTEND:20260501T150000Z\r\n" +
		"END:VEVENT\r\nEND:VCALENDAR\r\n"

	mock.GetMessageFunc = func(context.Context, string, string) (*domain.Message, error) {
		return &domain.Message{
			ID: "email-x",
			Attachments: []domain.Attachment{
				{ID: "att-1", Filename: "invite.ics", ContentType: "text/calendar"},
			},
		}, nil
	}
	mock.DownloadAttachmentFunc = func(context.Context, string, string, string) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(icsBody)), nil
	}
	mock.GetCalendarsFunc = func(context.Context, string) ([]domain.Calendar, error) {
		return []domain.Calendar{{ID: "primary", IsPrimary: true}}, nil
	}
	var seenUID string
	mock.GetEventsWithCursorFunc = func(_ context.Context, _, _ string, params *domain.EventQueryParams) (*domain.EventListResponse, error) {
		if params != nil {
			seenUID = params.ICalUID
		}
		return &domain.EventListResponse{Data: []domain.Event{{ID: "evt-attached-1"}}}, nil
	}
	mock.SendRSVPFunc = func(context.Context, string, string, string, *domain.SendRSVPRequest) error {
		return nil
	}

	w := postRSVP(t, server, "email-x", `{"status":"yes"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if seenUID != "attached-uid@example.com" {
		t.Errorf("ical_uid passed to events lookup=%q, want attached-uid@example.com", seenUID)
	}
}

// TestParseICS_SurfacesUID covers the parser change for the RSVP feature.
// Without the UID the RSVP handler can't resolve a Nylas event ID.
func TestParseICS_SurfacesUID(t *testing.T) {
	body := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//T//EN\r\nMETHOD:REQUEST\r\n" +
		"BEGIN:VEVENT\r\nUID:my-uid@example.com\r\nSUMMARY:Standup\r\n" +
		"DTSTART:20260501T140000Z\r\nDTEND:20260501T150000Z\r\n" +
		"END:VEVENT\r\nEND:VCALENDAR\r\n"
	got, err := parseICS(body)
	if err != nil {
		t.Fatalf("parseICS: %v", err)
	}
	if got.ICalUID != "my-uid@example.com" {
		t.Errorf("ICalUID=%q, want my-uid@example.com", got.ICalUID)
	}
}

// (postRSVP moved to handlers_email_rsvp_fixtures_test.go)
