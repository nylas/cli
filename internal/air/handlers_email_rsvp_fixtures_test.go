package air

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nylasmock "github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
)

// Shared MIME fixtures and helpers for the RSVP handler test suites.
// Lifted out of handlers_email_rsvp_test.go so the main test file stays
// under the 600-line cap.

// rsvpInviteMIME is a Gmail-style invite shaped like the production
// payload the RSVP path needs to handle: inline text/calendar with a
// stable UID we can resolve to a Nylas event.
const rsvpInviteMIME = "From: organizer@example.com\r\n" +
	"To: user@example.com\r\n" +
	"Subject: Event Invitation: Standup\r\n" +
	"Content-Type: multipart/alternative; boundary=\"BOUNDARY1\"\r\n" +
	"\r\n" +
	"--BOUNDARY1\r\n" +
	"Content-Type: text/calendar; charset=UTF-8; method=REQUEST\r\n" +
	"\r\n" +
	"BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Test//EN\r\n" +
	"METHOD:REQUEST\r\n" +
	"BEGIN:VEVENT\r\nUID:rsvp-event-uid@example.com\r\nSUMMARY:Standup\r\n" +
	"DTSTART:20260501T140000Z\r\nDTEND:20260501T143000Z\r\n" +
	"END:VEVENT\r\nEND:VCALENDAR\r\n" +
	"--BOUNDARY1--\r\n"

// cancelledInviteMIME exercises the METHOD:CANCEL guard.
const cancelledInviteMIME = "Content-Type: multipart/alternative; boundary=\"B\"\r\n" +
	"\r\n--B\r\n" +
	"Content-Type: text/calendar; charset=UTF-8; method=CANCEL\r\n\r\n" +
	"BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//T//EN\r\nMETHOD:CANCEL\r\n" +
	"BEGIN:VEVENT\r\nUID:cancelled-uid@example.com\r\nSUMMARY:Killed\r\n" +
	"DTSTART:20260501T140000Z\r\nDTEND:20260501T150000Z\r\n" +
	"STATUS:CANCELLED\r\n" +
	"END:VEVENT\r\nEND:VCALENDAR\r\n" +
	"--B--\r\n"

// noUIDInviteMIME pins the "Microsoft sent us an invite without a UID"
// edge case — handler should fail loudly, not silently RSVP to nothing.
const noUIDInviteMIME = "Content-Type: multipart/alternative; boundary=\"B\"\r\n" +
	"\r\n--B\r\n" +
	"Content-Type: text/calendar; charset=UTF-8; method=REQUEST\r\n\r\n" +
	"BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//T//EN\r\nMETHOD:REQUEST\r\n" +
	"BEGIN:VEVENT\r\nSUMMARY:UID-less\r\n" +
	"DTSTART:20260501T140000Z\r\nDTEND:20260501T150000Z\r\n" +
	"END:VEVENT\r\nEND:VCALENDAR\r\n" +
	"--B--\r\n"

// rsvpHappyPathMock returns a server + mock pre-wired for the standard
// "everything works" path: the email exposes mime, the user has a primary
// writable calendar, and the iCal UID resolves to a Nylas event.
// Individual tests override fields on the mock to exercise failure modes.
func rsvpHappyPathMock(t *testing.T, mime string) (*Server, *nylasmock.MockClient) {
	t.Helper()
	server, mock, _ := newCachedTestServer(t)

	mock.GetMessageFunc = func(_ context.Context, _, messageID string) (*domain.Message, error) {
		return &domain.Message{ID: messageID, Subject: "Event Invitation: Standup"}, nil
	}
	mock.GetMessageWithFieldsFunc = func(_ context.Context, _, messageID, _ string) (*domain.Message, error) {
		return &domain.Message{ID: messageID, RawMIME: mime}, nil
	}
	mock.GetCalendarsFunc = func(context.Context, string) ([]domain.Calendar, error) {
		return []domain.Calendar{
			{ID: "cal-primary", Name: "Primary", IsPrimary: true, ReadOnly: false},
		}, nil
	}
	mock.GetEventsWithCursorFunc = func(_ context.Context, _, _ string, params *domain.EventQueryParams) (*domain.EventListResponse, error) {
		// Pin the contract: handler must filter by ical_uid, not scan
		// the whole calendar — without this the lookup could return the
		// first random event and we'd RSVP to the wrong meeting.
		if params == nil || params.ICalUID == "" {
			t.Errorf("GetEventsWithCursor called without ICalUID filter; params=%+v", params)
		}
		return &domain.EventListResponse{Data: []domain.Event{{ID: "evt-resolved-1"}}}, nil
	}
	return server, mock
}

// postRSVP sends a POST to the RSVP endpoint and returns the recorder.
func postRSVP(t *testing.T, server *Server, emailID, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/emails/"+emailID+"/rsvp", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	server.handleEmailRSVP(w, r, emailID)
	return w
}
