package air

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// CalendarInviteResponse is the JSON returned by /api/emails/{id}/invite.
// It is intentionally a small subset of iCalendar VEVENT — just what the
// Air UI needs to render a Gmail-style "you have an invite" card.
type CalendarInviteResponse struct {
	HasInvite       bool   `json:"has_invite"`
	AttachmentID    string `json:"attachment_id,omitempty"`
	Filename        string `json:"filename,omitempty"`
	Title           string `json:"title,omitempty"`
	Location        string `json:"location,omitempty"`
	Description     string `json:"description,omitempty"`
	StartTime       int64  `json:"start_time,omitempty"`
	EndTime         int64  `json:"end_time,omitempty"`
	IsAllDay        bool   `json:"is_all_day,omitempty"`
	OrganizerName   string `json:"organizer_name,omitempty"`
	OrganizerEmail  string `json:"organizer_email,omitempty"`
	ConferencingURL string `json:"conferencing_url,omitempty"`
	Status          string `json:"status,omitempty"` // CONFIRMED / TENTATIVE / CANCELLED
}

// handleEmailInvite returns parsed iCalendar invite data for an email.
// It detects the first text/calendar (or .ics) attachment, downloads its
// content, and parses out the fields the Air UI shows.
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
	if att == nil {
		writeJSON(w, http.StatusOK, CalendarInviteResponse{HasInvite: false})
		return
	}

	body, err := s.nylasClient.DownloadAttachment(ctx, grantID, emailID, att.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to download invite: "+err.Error())
		return
	}
	defer func() { _ = body.Close() }()

	// Cap the read so a hostile/oversized attachment cannot DoS us.
	const maxICSBytes = 1 << 20 // 1 MB
	raw, err := io.ReadAll(io.LimitReader(body, maxICSBytes+1))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read invite: "+err.Error())
		return
	}
	if len(raw) > maxICSBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "Invite attachment exceeds 1 MB limit")
		return
	}

	parsed, err := parseICS(string(raw))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Failed to parse invite: "+err.Error())
		return
	}

	parsed.HasInvite = true
	parsed.AttachmentID = att.ID
	parsed.Filename = att.Filename
	writeJSON(w, http.StatusOK, parsed)
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

// parseICS is a tiny iCalendar parser. It handles:
//   - line unfolding (RFC 5545 §3.1: lines starting with space/tab are
//     continuations of the previous line)
//   - CRLF and LF line endings
//   - basic VEVENT properties: SUMMARY, LOCATION, DESCRIPTION, DTSTART,
//     DTEND, ORGANIZER, STATUS, X-CONFERENCE-URL / URL
//   - DATE-only DTSTART/DTEND (all-day events)
//   - DATE-TIME values in UTC (suffix Z) or local time
//
// It does NOT try to be a full RFC 5545 parser — it covers the
// invitation-card use case and rejects garbage early.
func parseICS(raw string) (CalendarInviteResponse, error) {
	if !strings.Contains(raw, "BEGIN:VEVENT") {
		return CalendarInviteResponse{}, errors.New("no VEVENT block found")
	}

	// Normalise line endings then unfold continuations.
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	rawLines := strings.Split(raw, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, ln := range rawLines {
		if len(ln) > 0 && (ln[0] == ' ' || ln[0] == '\t') && len(lines) > 0 {
			lines[len(lines)-1] += ln[1:]
			continue
		}
		lines = append(lines, ln)
	}

	var (
		inEvent bool
		ev      CalendarInviteResponse
	)
	for _, ln := range lines {
		switch {
		case ln == "BEGIN:VEVENT":
			inEvent = true
			continue
		case ln == "END:VEVENT":
			// Stop at the first VEVENT — invites typically have one.
			if ev.Title == "" && ev.StartTime == 0 {
				return CalendarInviteResponse{}, errors.New("VEVENT had no usable fields")
			}
			return ev, nil
		case !inEvent:
			continue
		}

		name, params, value, ok := splitICSLine(ln)
		if !ok {
			continue
		}

		switch name {
		case "SUMMARY":
			ev.Title = unescapeICSText(value)
		case "LOCATION":
			ev.Location = unescapeICSText(value)
		case "DESCRIPTION":
			ev.Description = unescapeICSText(value)
		case "DTSTART":
			ts, allDay := parseICSDate(value, params)
			ev.StartTime = ts
			if allDay {
				ev.IsAllDay = true
			}
		case "DTEND":
			ts, allDay := parseICSDate(value, params)
			ev.EndTime = ts
			if allDay {
				ev.IsAllDay = true
			}
		case "ORGANIZER":
			ev.OrganizerEmail, ev.OrganizerName = parseOrganizer(params, value)
		case "STATUS":
			ev.Status = strings.ToUpper(strings.TrimSpace(value))
		case "URL", "X-CONFERENCE-URL", "X-GOOGLE-CONFERENCE":
			if ev.ConferencingURL == "" {
				ev.ConferencingURL = strings.TrimSpace(value)
			}
		}
	}

	if ev.Title == "" && ev.StartTime == 0 {
		return CalendarInviteResponse{}, errors.New("VEVENT had no usable fields")
	}
	return ev, nil
}

// splitICSLine splits "NAME[;PARAM=VAL...]:VALUE" into its three parts.
// Returns ok=false on malformed input. Param parsing is permissive.
func splitICSLine(line string) (name string, params map[string]string, value string, ok bool) {
	colon := strings.IndexByte(line, ':')
	if colon < 1 {
		return "", nil, "", false
	}
	left := line[:colon]
	value = line[colon+1:]

	parts := strings.Split(left, ";")
	name = strings.ToUpper(strings.TrimSpace(parts[0]))
	if name == "" {
		return "", nil, "", false
	}
	if len(parts) > 1 {
		params = make(map[string]string, len(parts)-1)
		for _, p := range parts[1:] {
			eq := strings.IndexByte(p, '=')
			if eq <= 0 {
				continue
			}
			params[strings.ToUpper(p[:eq])] = strings.Trim(p[eq+1:], `"`)
		}
	}
	return name, params, value, true
}

// parseICSDate accepts VALUE=DATE (all-day) and DATE-TIME forms. Returns
// 0 when the value cannot be parsed. The boolean is true for all-day.
func parseICSDate(value string, params map[string]string) (int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	if v, ok := params["VALUE"]; ok && strings.EqualFold(v, "DATE") {
		if t, err := time.Parse("20060102", value); err == nil {
			return t.UTC().Unix(), true
		}
		return 0, true
	}

	// DATE-TIME with explicit UTC marker.
	if strings.HasSuffix(value, "Z") {
		if t, err := time.Parse("20060102T150405Z", value); err == nil {
			return t.Unix(), false
		}
	}

	// Local DATE-TIME (no zone). Treat as UTC for stability — the UI
	// displays in the viewer's locale anyway.
	if t, err := time.Parse("20060102T150405", value); err == nil {
		return t.UTC().Unix(), false
	}

	// As a last resort, try date-only.
	if t, err := time.Parse("20060102", value); err == nil {
		return t.UTC().Unix(), true
	}
	return 0, false
}

// unescapeICSText reverses RFC 5545 text escaping (\\, \,, \;, \n).
func unescapeICSText(v string) string {
	v = strings.ReplaceAll(v, `\\`, "\x00")
	v = strings.ReplaceAll(v, `\,`, ",")
	v = strings.ReplaceAll(v, `\;`, ";")
	v = strings.ReplaceAll(v, `\n`, "\n")
	v = strings.ReplaceAll(v, `\N`, "\n")
	return strings.ReplaceAll(v, "\x00", `\`)
}

// parseOrganizer extracts the email + display name from an ORGANIZER
// line such as `ORGANIZER;CN=Priya Patel:mailto:priya@partner.example`.
func parseOrganizer(params map[string]string, value string) (email, name string) {
	v := strings.TrimSpace(value)
	v = strings.TrimPrefix(strings.ToLower(v), "mailto:")
	// Re-grab the original-cased email if mailto was a prefix only.
	if lower := strings.ToLower(value); strings.HasPrefix(lower, "mailto:") {
		v = value[len("mailto:"):]
	}
	email = strings.TrimSpace(v)
	if cn, ok := params["CN"]; ok {
		name = strings.TrimSpace(cn)
	}
	return email, name
}

// demoInviteFor returns canned parsed-event data so the calendar-invite
// card has something to render in demo mode without round-tripping
// through the parser.
func demoInviteFor(emailID string) CalendarInviteResponse {
	if emailID != "demo-email-invite-001" {
		return CalendarInviteResponse{HasInvite: false}
	}

	// Fixed event in the future so the formatted time is consistent.
	start := time.Now().Add(24 * time.Hour).Truncate(time.Hour)
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
	}
}
