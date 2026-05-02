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

// errEventNotImported: the invite UID hasn't been ingested by the Nylas
// calendar importer yet — surface as 404 with a "try again" hint.
var errEventNotImported = errors.New("invite has not been imported into your calendar yet")

// errNoWritableCalendar: grant has only read-only calendars (e.g. a
// "Holidays" subscription) — surface as 422, retry won't help.
var errNoWritableCalendar = errors.New("no writable calendar available")

// handleEmailRSVP forwards the user's RSVP choice to Nylas.
//
// Security invariant: never trust a client-supplied event ID — always
// re-resolve via the email's VEVENT UID. Otherwise a forged frontend
// could RSVP to arbitrary events on the user's behalf.
func (s *Server) handleEmailRSVP(w http.ResponseWriter, r *http.Request, emailID string) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var body rsvpRequest
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&body); err != nil {
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
	body.Comment = strings.TrimSpace(body.Comment)
	if len(body.Comment) > rsvpCommentMaxBytes {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("comment must be %d bytes or fewer", rsvpCommentMaxBytes))
		return
	}

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
		// Both initial GetMessage failure and the raw_mime fallback land
		// here — both are transient. 502 lets the frontend offer a retry.
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
		// Some Microsoft senders ship invites without a UID.
		writeError(w, http.StatusUnprocessableEntity, "Invite has no UID — open the event in your calendar to RSVP")
		return
	}

	// Search ALL writable calendars: invites often land in non-primary ones
	// (work + personal under one Google account, shared team calendars).
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

// findInviteEventAcrossCalendars searches every writable calendar
// (primary first) for an event matching icalUID. Returns
// errEventNotImported when no calendar contains the UID. A single
// calendar erroring doesn't kill the search — the next might hold it.
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

// writableCalendars returns writable calendars, primary first.
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

// findEventByICalUID resolves a UID via Nylas v3's `ical_uid` filter.
// Returns errEventNotImported when no event matches.
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
