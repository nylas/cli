package air

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// maxICSBytes caps a single iCalendar payload. 1 MB is far above any
// real invitation and keeps memory predictable when an attacker stitches
// a large fake attachment.
const maxICSBytes = 1 << 20

// CalendarInviteResponse is the JSON returned by /api/emails/{id}/invite.
// It is intentionally a small subset of iCalendar VEVENT — just what the
// Air UI needs to render a Gmail-style "you have an invite" card.
type CalendarInviteResponse struct {
	HasInvite       bool             `json:"has_invite"`
	AttachmentID    string           `json:"attachment_id,omitempty"`
	Filename        string           `json:"filename,omitempty"`
	ICalUID         string           `json:"ical_uid,omitempty"` // VEVENT UID; lets the RSVP endpoint resolve a Nylas event ID
	Title           string           `json:"title,omitempty"`
	Location        string           `json:"location,omitempty"`
	Description     string           `json:"description,omitempty"`
	StartTime       int64            `json:"start_time,omitempty"`
	EndTime         int64            `json:"end_time,omitempty"`
	IsAllDay        bool             `json:"is_all_day,omitempty"`
	OrganizerName   string           `json:"organizer_name,omitempty"`
	OrganizerEmail  string           `json:"organizer_email,omitempty"`
	ConferencingURL string           `json:"conferencing_url,omitempty"`
	Status          string           `json:"status,omitempty"` // CONFIRMED / TENTATIVE / CANCELLED
	Method          string           `json:"method,omitempty"` // REQUEST / CANCEL / REPLY
	Attendees       []InviteAttendee `json:"attendees,omitempty"`
	RecurrenceRule  string           `json:"recurrence_rule,omitempty"` // raw RRULE for callers that want to summarise
}

// InviteAttendee is a participant on the VEVENT — surfaced so the Air
// invite card can render the same "Alice (accepted), Bob (declined)"
// list that Gmail shows. Distinct from the demo-data Attendee in
// data.go which is a UI-only avatar.
type InviteAttendee struct {
	Name        string `json:"name,omitempty"`
	Email       string `json:"email,omitempty"`
	Status      string `json:"status,omitempty"` // ACCEPTED / DECLINED / TENTATIVE / NEEDS-ACTION
	Role        string `json:"role,omitempty"`
	IsOrganizer bool   `json:"is_organizer,omitempty"`
}

// errInviteFetchFailed flags a transient upstream failure on the second
// raw_mime fetch (the fallback after attachment download fails). Callers
// distinguish it from "the email simply has no invite" so they can choose
// between silent degrade (preview card) and 502 (RSVP — the user is
// actively trying to do something and deserves an actionable error).
var errInviteFetchFailed = errors.New("invite: failed to fetch raw_mime")

// handleEmailInvite returns parsed iCalendar invite data for an email.
// Returns has_invite=false when neither attachments[] nor raw_mime yields
// a calendar payload — the frontend silently degrades to the regular
// email render in that case. Transient raw_mime fetch failures are also
// silently degraded here so a flaky upstream doesn't break the inbox
// view; the RSVP endpoint surfaces the same condition as 502 because the
// user is actively trying to act on the invite.
func (s *Server) handleEmailInvite(w http.ResponseWriter, r *http.Request, emailID string) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	if s.demoMode {
		writeJSON(w, http.StatusOK, demoInviteFor(emailID))
		return
	}

	grantID := s.withAuthGrant(w, nil)
	if grantID == "" {
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	resp, err := s.resolveEmailInvite(ctx, grantID, emailID)
	if err != nil {
		// Preserve the legacy "preview card silently disappears" UX even
		// when raw_mime can't be fetched. Only the initial GetMessage
		// failure (a hard error) reaches here.
		if errors.Is(err, errInviteFetchFailed) {
			writeJSON(w, http.StatusOK, CalendarInviteResponse{HasInvite: false})
			return
		}
		writeUpstreamError(w, http.StatusInternalServerError,
			"Failed to fetch email — please try again", err,
			"emailID", emailID)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// resolveEmailInvite parses a message's iCalendar invite, if any.
// Resolution order:
//  1. attachments[] with text/calendar or .ics (Microsoft/custom senders).
//  2. inline text/calendar in raw_mime (Gmail's multipart/alternative shape).
//
// Returns HasInvite=false on no-invite. Non-nil error only on initial
// GetMessage failure — attachment-download and raw_mime errors are swallowed.
func (s *Server) resolveEmailInvite(ctx context.Context, grantID, emailID string) (CalendarInviteResponse, error) {
	msg, err := s.nylasClient.GetMessage(ctx, grantID, emailID)
	if err != nil {
		return CalendarInviteResponse{}, err
	}

	att := findCalendarAttachment(msg.Attachments)
	if att != nil {
		if parsed, ok := s.tryParseAttachmentInvite(ctx, grantID, emailID, att); ok {
			return parsed, nil
		}
		// Download or parse failed. Don't surface as an error — Nylas
		// frequently returns synthetic attachment IDs (v0:base64(...):...)
		// that look like real attachments but cannot be downloaded.
		// Falling through to the raw_mime walker recovers the calendar
		// payload directly from the MIME tree.
	}

	// Fetch raw MIME and look for a text/calendar part — both Gmail's
	// inline-multipart shape AND Nylas's "synthetic attachment that
	// can't be downloaded" case land here. A network error on this call
	// is *transient* and distinguishable from "email has no invite", so
	// we surface it as errInviteFetchFailed: callers in actionable paths
	// (RSVP) can return 502, while the preview path silently degrades.
	full, err := s.nylasClient.GetMessageWithFields(ctx, grantID, emailID, "raw_mime")
	if err != nil {
		// Double-wrap so callers can errors.Is against errInviteFetchFailed
		// (the sentinel) AND unwrap the underlying transport error for
		// logging. Go 1.20+ supports multiple %w in a single Errorf.
		return CalendarInviteResponse{}, fmt.Errorf("%w: %w", errInviteFetchFailed, err)
	}
	if full == nil || full.RawMIME == "" {
		return CalendarInviteResponse{HasInvite: false}, nil
	}
	parts := findInlineCalendarParts(full.RawMIME)
	if len(parts) == 0 {
		return CalendarInviteResponse{HasInvite: false}, nil
	}

	parsed, err := parseICS(parts[0].Body)
	if err != nil {
		return CalendarInviteResponse{HasInvite: false}, nil
	}
	parsed.HasInvite = true
	// If we had a Nylas attachment entry (even an undownloadable
	// synthetic one), keep its ID so the frontend's existing
	// attachment-row check can match by name; otherwise mint a stable
	// inline marker so the row still renders.
	if att != nil {
		parsed.AttachmentID = att.ID
		parsed.Filename = att.Filename
	} else {
		parsed.AttachmentID = inlineCalendarAttachmentID(parts[0].ContentID)
		if parts[0].Filename != "" {
			parsed.Filename = parts[0].Filename
		} else {
			parsed.Filename = "invite.ics"
		}
	}
	if parsed.Method == "" && parts[0].Method != "" {
		parsed.Method = parts[0].Method
	}
	return parsed, nil
}

// tryParseAttachmentInvite downloads and parses an ICS attachment.
// Returns ok=false on any failure so the caller falls back to raw_mime;
// each failure logs at slog.Debug for diagnosability.
func (s *Server) tryParseAttachmentInvite(ctx context.Context, grantID, emailID string, att *domain.Attachment) (CalendarInviteResponse, bool) {
	body, err := s.nylasClient.DownloadAttachment(ctx, grantID, emailID, att.ID)
	if err != nil {
		slog.Debug("invite attachment download failed",
			"attachment_id", att.ID, "filename", att.Filename, "err", err)
		return CalendarInviteResponse{}, false
	}
	defer func() { _ = body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(body, maxICSBytes+1))
	if err != nil || len(raw) > maxICSBytes {
		slog.Debug("invite attachment read failed or oversized",
			"attachment_id", att.ID, "filename", att.Filename,
			"size", len(raw), "max", maxICSBytes, "err", err)
		return CalendarInviteResponse{}, false
	}

	parsed, err := parseICS(string(raw))
	if err != nil {
		slog.Debug("invite attachment ICS parse failed",
			"attachment_id", att.ID, "filename", att.Filename, "err", err)
		return CalendarInviteResponse{}, false
	}
	parsed.HasInvite = true
	parsed.AttachmentID = att.ID
	parsed.Filename = att.Filename
	return parsed, true
}

// inlineCalendarPrefix prefixes synthetic attachment IDs that point at
// a text/calendar MIME part rather than a real Nylas attachment. The
// download endpoint recognises this prefix and serves the part directly
// from raw_mime instead of forwarding to Nylas.
const inlineCalendarPrefix = "inline-calendar:"

// inlineCalendarAttachmentID builds a stable synthetic attachment ID for
// a calendar MIME part. Falls back to a fixed marker when the source
// part has no Content-ID — Gmail invitations often omit it.
func inlineCalendarAttachmentID(contentID string) string {
	if contentID == "" {
		return inlineCalendarPrefix + "default"
	}
	return inlineCalendarPrefix + contentID
}

// isInlineCalendarAttachmentID reports whether an attachment ID came
// from a synthesized calendar part rather than a real Nylas attachment.
func isInlineCalendarAttachmentID(id string) bool {
	return strings.HasPrefix(id, inlineCalendarPrefix)
}

// findCalendarAttachment locates the first attachment that looks like an
// iCalendar invite — either by content type or by filename suffix.
func findCalendarAttachment(atts []domain.Attachment) *domain.Attachment {
	for i := range atts {
		if isCalendarAttachment(atts[i].ContentType, atts[i].Filename) {
			return &atts[i]
		}
	}
	return nil
}

// isCalendarAttachment is the shared "looks like an invite" predicate so
// frontend and tests can reuse the same rule.
func isCalendarAttachment(contentType, filename string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	fn := strings.ToLower(strings.TrimSpace(filename))
	return strings.HasPrefix(ct, "text/calendar") ||
		strings.HasPrefix(ct, "application/ics") ||
		strings.HasSuffix(fn, ".ics")
}

// errNoUsableEvent is returned by parseICS when the iCalendar payload is
// either malformed, has no VEVENT, or whose first VEVENT carries no
// fields the UI can render. Callers translate this into a "no invite"
// response rather than a 5xx — clients should silently degrade.
var errNoUsableEvent = errors.New("ics: no usable event")

// demoInviteFor returns canned parsed-event data so the calendar-invite
// card has something to render in demo mode without round-tripping
// through the parser.
func demoInviteFor(emailID string) CalendarInviteResponse {
	if emailID != "demo-email-invite-001" {
		return CalendarInviteResponse{HasInvite: false}
	}

	start := demoInviteStart()
	end := start.Add(time.Hour)

	return CalendarInviteResponse{
		HasInvite:       true,
		AttachmentID:    "att-invite-001",
		Filename:        "invite.ics",
		ICalUID:         "demo-invite-001@nylas.example",
		Title:           "Quarterly Sync",
		Location:        "Conference Room A / Online",
		Description:     "Quarterly review with the partner team.",
		StartTime:       start.Unix(),
		EndTime:         end.Unix(),
		IsAllDay:        false,
		OrganizerName:   "Priya Patel",
		OrganizerEmail:  "priya@partner.example",
		ConferencingURL: "https://meet.example.com/q-sync",
		Status:          "CONFIRMED",
		Method:          "REQUEST",
		Attendees: []InviteAttendee{
			{Name: "Priya Patel", Email: "priya@partner.example", Status: "ACCEPTED", Role: "CHAIR", IsOrganizer: true},
			{Name: "Alex Reed", Email: "alex@example.com", Status: "ACCEPTED", Role: "REQ-PARTICIPANT"},
			{Name: "Jamie Chen", Email: "jamie@example.com", Status: "TENTATIVE", Role: "REQ-PARTICIPANT"},
		},
	}
}

// demoInviteStart returns a stable demo start time — 24h from now,
// truncated to the hour. Extracted so tests can stub via build tag if
// the rounding behaviour ever needs pinning.
func demoInviteStart() time.Time {
	return time.Now().Add(24 * time.Hour).Truncate(time.Hour)
}
