package air

import (
	"context"
	"errors"
	"io"
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

// handleEmailInvite returns parsed iCalendar invite data for an email.
// Resolution order:
//  1. attachments[] entry with text/calendar or .ics filename — Microsoft,
//     custom senders typically arrive this way.
//  2. inline text/calendar part inside raw_mime (Gmail's invitation
//     shape — the ICS rides as a multipart/alternative leaf).
//
// Returns has_invite=false when neither path yields a calendar payload.
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

	msg, err := s.nylasClient.GetMessage(ctx, grantID, emailID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch email: "+err.Error())
		return
	}

	att := findCalendarAttachment(msg.Attachments)
	if att != nil {
		if parsed, ok := s.tryParseAttachmentInvite(ctx, grantID, emailID, att); ok {
			writeJSON(w, http.StatusOK, parsed)
			return
		}
		// Download or parse failed. Don't surface a 5xx — Nylas
		// frequently returns synthetic attachment IDs (v0:base64(...):...)
		// that look like real attachments but cannot be downloaded.
		// Falling through to the raw_mime walker recovers the calendar
		// payload directly from the MIME tree.
	}

	// Fetch raw MIME and look for a text/calendar part — both Gmail's
	// inline-multipart shape AND Nylas's "synthetic attachment that
	// can't be downloaded" case land here.
	full, err := s.nylasClient.GetMessageWithFields(ctx, grantID, emailID, "raw_mime")
	if err != nil || full == nil || full.RawMIME == "" {
		writeJSON(w, http.StatusOK, CalendarInviteResponse{HasInvite: false})
		return
	}
	parts := findInlineCalendarParts(full.RawMIME)
	if len(parts) == 0 {
		writeJSON(w, http.StatusOK, CalendarInviteResponse{HasInvite: false})
		return
	}

	parsed, err := parseICS(parts[0].Body)
	if err != nil {
		writeJSON(w, http.StatusOK, CalendarInviteResponse{HasInvite: false})
		return
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
	writeJSON(w, http.StatusOK, parsed)
}

// tryParseAttachmentInvite attempts the legacy path: download an ICS
// attachment via the Nylas attachments endpoint and parse it. Returns
// ok=false on any failure so the caller can fall back to raw_mime.
// Errors are intentionally swallowed: we don't want 5xxs for transient
// download problems when the calendar payload is recoverable from MIME.
func (s *Server) tryParseAttachmentInvite(ctx context.Context, grantID, emailID string, att *domain.Attachment) (CalendarInviteResponse, bool) {
	body, err := s.nylasClient.DownloadAttachment(ctx, grantID, emailID, att.ID)
	if err != nil {
		return CalendarInviteResponse{}, false
	}
	defer func() { _ = body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(body, maxICSBytes+1))
	if err != nil || len(raw) > maxICSBytes {
		return CalendarInviteResponse{}, false
	}

	parsed, err := parseICS(string(raw))
	if err != nil {
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
