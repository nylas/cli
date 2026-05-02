package air

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/nylas/cli/internal/domain"
)

// rsvpCommentMaxBytes caps the free-form RSVP comment forwarded to Nylas.
// 1MB JSON body limits already protect against DoS, but a UI-meaningful
// cap surfaces a friendlier error and prevents accidentally pasting an
// entire email body into the comment field.
const rsvpCommentMaxBytes = 1024

// rsvpRequest is the JSON body accepted by POST /api/emails/{id}/rsvp.
// Mirrors the shape the CLI's `nylas calendar events rsvp` command uses
// so frontend and CLI converge on the same vocabulary.
type rsvpRequest struct {
	Status  string `json:"status"`
	Comment string `json:"comment,omitempty"`
}

// rsvpResponse is the success body — small on purpose so the frontend
// can update local state (button highlight, attendee count) without a
// follow-up fetch.
type rsvpResponse struct {
	Status     string `json:"status"`
	EventID    string `json:"event_id"`
	CalendarID string `json:"calendar_id"`
}

// validRSVPStatuses pins the Nylas v3 send-rsvp vocabulary.
// "noreply" exists in the API but isn't a meaningful UI choice — the
// user just doesn't click anything — so we don't accept it here.
var validRSVPStatuses = map[string]struct{}{
	"yes":   {},
	"no":    {},
	"maybe": {},
}

// errEventNotImported is returned when the iCalendar UID does not match
// any event Nylas has imported for the grant. Most commonly happens when
// a Gmail/Outlook user receives an invite seconds before the calendar
// auto-importer catches up — the right UX is "try again in a moment",
// not "permanent failure".
var errEventNotImported = errors.New("invite has not been imported into your calendar yet")

// errNoWritableCalendar is returned when the grant has zero writable
// calendars (e.g. the only calendar is a read-only "US Holidays"
// subscription). We surface this distinctly from a transient lookup
// failure so the user-facing message can be specific instead of the
// generic "Failed to look up event — please try again", which would
// recommend a retry that can never succeed.
var errNoWritableCalendar = errors.New("no writable calendar available")

// handleEmailRSVP forwards the user's RSVP choice to Nylas. The pipeline:
//
//  1. Re-parse the email's invite (sharing logic with /invite) to recover
//     the VEVENT UID. We do not trust a UID submitted by the client —
//     that would let the page RSVP to events on the user's behalf.
//  2. Look up the Nylas event by ical_uid in the user's primary writable
//     calendar.
//  3. Call the Nylas send-rsvp endpoint.
//
// Refusing to accept a client-supplied event ID is the security
// invariant: the invite-card UI never sends an event ID across the wire,
// and the handler always re-resolves from the email to prevent CSRF-like
// "RSVP to arbitrary event" attacks via a forged frontend.
func (s *Server) handleEmailRSVP(w http.ResponseWriter, r *http.Request, emailID string) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var body rsvpRequest
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&body); err != nil {
		// json.Decoder error messages are benign here (syntax errors,
		// unexpected EOF) but echoing them verbatim leaks the raw bytes
		// the client sent back through the error string. Keep the message
		// generic; the breadcrumb lives in the server log.
		//
		// MaxBytesReader signals body-too-large via *http.MaxBytesError; map
		// that to 413 so clients can distinguish "your payload is too big"
		// from "your JSON is malformed" — both currently land here.
		slog.Warn("RSVP request body decode failed", "emailID", emailID, "err", err)
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "Request body too large")
			return
		}
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	status := strings.ToLower(strings.TrimSpace(body.Status))
	if _, ok := validRSVPStatuses[status]; !ok {
		writeError(w, http.StatusBadRequest, "status must be one of: yes, no, maybe")
		return
	}
	// Trim before the length check so leading/trailing whitespace doesn't
	// eat the user's budget. The trimmed value is what gets forwarded to
	// Nylas — surrounding whitespace was never useful anyway.
	body.Comment = strings.TrimSpace(body.Comment)
	if len(body.Comment) > rsvpCommentMaxBytes {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("comment must be %d bytes or fewer", rsvpCommentMaxBytes))
		return
	}

	// Demo mode: skip all upstream calls so the canned-data UI keeps
	// working without API credentials. The frontend gets the same shape
	// the real path returns.
	if s.demoMode {
		writeJSON(w, http.StatusOK, rsvpResponse{
			Status:     status,
			EventID:    "demo-event-001",
			CalendarID: "primary",
		})
		return
	}

	grantID := s.withAuthGrant(w, nil)
	if grantID == "" {
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	invite, err := s.resolveEmailInvite(ctx, grantID, emailID)
	if err != nil {
		// Both the initial GetMessage failure AND the raw_mime fallback
		// failure (errInviteFetchFailed) land here. Either is a transient
		// upstream condition the user can retry — distinct from "this
		// email has no invite" (which returns no error and HasInvite:false
		// further down). Surface as 502 so the frontend can suggest a
		// retry instead of permanently disabling the RSVP buttons. The
		// raw error stays in server logs (avoid leaking upstream details).
		slog.Error("RSVP failed to fetch email", "emailID", emailID, "err", err)
		writeError(w, http.StatusBadGateway, "Failed to fetch email — please try again")
		return
	}
	if !invite.HasInvite {
		writeError(w, http.StatusNotFound, "This email does not contain a calendar invitation")
		return
	}
	if strings.EqualFold(invite.Method, "CANCEL") || strings.EqualFold(invite.Status, "CANCELLED") {
		writeError(w, http.StatusConflict, "This event has been cancelled — RSVP is no longer accepted")
		return
	}
	if invite.ICalUID == "" {
		// Some Microsoft senders ship invites without a UID. The user
		// would have to RSVP via the underlying calendar UI directly.
		writeError(w, http.StatusUnprocessableEntity, "Invite has no UID — open the event in your calendar to RSVP")
		return
	}

	// Resolve the (calendar, event) pair across ALL writable calendars,
	// not just the primary. Users with multiple writable calendars
	// (work + personal under one Google account, shared team calendars,
	// etc.) often have invites land in a non-primary one — searching
	// only the primary returned a misleading "not imported yet" 404 even
	// though the event was sitting in the next calendar over.
	calendarID, eventID, err := s.findInviteEventAcrossCalendars(ctx, grantID, invite.ICalUID)
	if err != nil {
		if errors.Is(err, errEventNotImported) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		// "No writable calendar" is a config-shaped failure (only
		// read-only subscriptions on this grant). Retrying won't help —
		// surface a 422 with a clear message so the user knows where to
		// look instead of being told to retry forever.
		if errors.Is(err, errNoWritableCalendar) {
			writeError(w, http.StatusUnprocessableEntity,
				"No writable calendar on this account — RSVP requires a calendar you can edit")
			return
		}
		slog.Error("RSVP calendar lookup failed", "emailID", emailID, "icalUID", invite.ICalUID, "err", err)
		writeError(w, http.StatusBadGateway, "Failed to look up event — please try again")
		return
	}

	rsvpReq := &domain.SendRSVPRequest{Status: status, Comment: body.Comment}
	if err := s.nylasClient.SendRSVP(ctx, grantID, calendarID, eventID, rsvpReq); err != nil {
		slog.Error("RSVP send failed",
			"emailID", emailID,
			"calendarID", calendarID,
			"eventID", eventID,
			"status", status,
			"err", err,
		)
		writeError(w, http.StatusBadGateway, "Failed to send RSVP — please try again")
		return
	}

	writeJSON(w, http.StatusOK, rsvpResponse{
		Status:     status,
		EventID:    eventID,
		CalendarID: calendarID,
	})
}

// findInviteEventAcrossCalendars searches every writable calendar on the
// grant for an event matching icalUID, returning the (calendarID, eventID)
// pair. Order:
//
//  1. Primary writable (most invites land here).
//  2. Other writable calendars in list order.
//
// Read-only calendars are skipped because RSVP can't be sent on them
// anyway — Nylas needs to update the attendee record on the event itself.
//
// Returns errEventNotImported when no writable calendar contains the UID —
// the typical cause is that the calendar auto-importer hasn't ingested the
// invite yet, or the invite was filed into a calendar Nylas can't see.
//
// Note: this scans calendars in order. For accounts with many writable
// calendars (rare; most users have 1-3), each calendar costs one
// listEvents call. We bail at the first match.
func (s *Server) findInviteEventAcrossCalendars(ctx context.Context, grantID, icalUID string) (string, string, error) {
	calendars, err := s.nylasClient.GetCalendars(ctx, grantID)
	if err != nil {
		return "", "", fmt.Errorf("failed to list calendars: %w", err)
	}
	if len(calendars) == 0 {
		return "", "", errors.New("no calendars found for this account")
	}

	writable := writableCalendars(calendars)
	if len(writable) == 0 {
		return "", "", errNoWritableCalendar
	}

	// Track the most recent transient lookup error so we can surface it
	// if NO calendar matches. A single calendar erroring shouldn't kill
	// the whole search — the next calendar might hold the event. Log
	// each transient failure so a consistently-broken calendar is
	// debuggable even when the search ultimately succeeds elsewhere.
	var lastLookupErr error
	for _, c := range writable {
		eventID, err := s.findEventByICalUID(ctx, grantID, c.ID, icalUID)
		if err == nil {
			return c.ID, eventID, nil
		}
		if !errors.Is(err, errEventNotImported) {
			slog.Warn("RSVP per-calendar lookup failed",
				"calendarID", c.ID,
				"icalUID", icalUID,
				"err", err,
			)
			lastLookupErr = err
		}
	}

	if lastLookupErr != nil {
		return "", "", fmt.Errorf("failed to look up event: %w", lastLookupErr)
	}
	return "", "", errEventNotImported
}

// writableCalendars returns the calendars callers should search for an
// invite, primary first, in calendar-list order. Encapsulates the
// "primary preferred, fall back to first writable" rule.
func writableCalendars(calendars []domain.Calendar) []domain.Calendar {
	out := make([]domain.Calendar, 0, len(calendars))
	for _, c := range calendars {
		if c.IsPrimary && !c.ReadOnly {
			out = append(out, c)
		}
	}
	for _, c := range calendars {
		if !c.IsPrimary && !c.ReadOnly {
			out = append(out, c)
		}
	}
	return out
}

// findEventByICalUID resolves an iCalendar UID to a Nylas event ID by
// querying the events endpoint with `ical_uid=<uid>`. The Nylas v3
// listing supports this filter directly, so we avoid scanning the full
// calendar.
//
// Returns errEventNotImported when no event matches — typical cause is
// that the calendar auto-importer hasn't ingested the invite yet.
func (s *Server) findEventByICalUID(ctx context.Context, grantID, calendarID, icalUID string) (string, error) {
	resp, err := s.nylasClient.GetEventsWithCursor(ctx, grantID, calendarID, &domain.EventQueryParams{
		ICalUID: icalUID,
		Limit:   1,
	})
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Data) == 0 {
		return "", errEventNotImported
	}
	return resp.Data[0].ID, nil
}
