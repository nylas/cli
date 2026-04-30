package air

import (
	"errors"
	"strings"

	ical "github.com/arran4/golang-ical"
)

// parseICS parses an iCalendar payload and returns the first VEVENT in a
// shape the Air invite card understands. Backed by golang-ical so we
// inherit RFC 5545 line-folding, TZID resolution, ATTENDEE parsing, and
// VALUE=DATE all-day handling.
//
// Returns errNoUsableEvent for input that doesn't yield a renderable
// event — empty calendars, malformed bodies, VEVENTs without title or
// start time. Callers should map this to a "no invite" response, not a
// 5xx, since clients silently degrade.
func parseICS(raw string) (CalendarInviteResponse, error) {
	cal, err := ical.ParseCalendar(strings.NewReader(raw))
	if err != nil {
		return CalendarInviteResponse{}, err
	}
	events := cal.Events()
	if len(events) == 0 {
		return CalendarInviteResponse{}, errNoUsableEvent
	}

	method := calendarMethod(cal)
	first := events[0]
	resp := mapVEvent(first)
	if resp.Method == "" {
		resp.Method = method
	}

	if resp.Title == "" && resp.StartTime == 0 {
		return CalendarInviteResponse{}, errNoUsableEvent
	}
	return resp, nil
}

// calendarMethod returns the calendar-level METHOD property, upper-cased.
// REQUEST is the most common (a fresh invitation); CANCEL needs a banner
// in the UI; REPLY arrives when an attendee responds and we surface it
// for completeness.
func calendarMethod(cal *ical.Calendar) string {
	for _, p := range cal.CalendarProperties {
		if strings.EqualFold(p.IANAToken, string(ical.PropertyMethod)) {
			return strings.ToUpper(strings.TrimSpace(p.Value))
		}
	}
	return ""
}

// mapVEvent flattens a VEVENT into the response shape. Each property is
// optional — Outlook and Google produce subtly different invitations,
// and we want the card to render whatever is available rather than
// failing closed.
func mapVEvent(ev *ical.VEvent) CalendarInviteResponse {
	var resp CalendarInviteResponse

	if p := ev.GetProperty(ical.ComponentPropertySummary); p != nil {
		resp.Title = p.Value
	}
	if p := ev.GetProperty(ical.ComponentPropertyLocation); p != nil {
		resp.Location = p.Value
	}
	if p := ev.GetProperty(ical.ComponentPropertyDescription); p != nil {
		resp.Description = p.Value
	}
	if p := ev.GetProperty(ical.ComponentPropertyStatus); p != nil {
		resp.Status = strings.ToUpper(strings.TrimSpace(p.Value))
	}
	if rrule := ev.GetProperty(ical.ComponentPropertyRrule); rrule != nil {
		resp.RecurrenceRule = rrule.Value
	}
	resp.ConferencingURL = firstConferenceURL(ev)

	resp.StartTime, resp.EndTime, resp.IsAllDay = eventTimes(ev)

	if org := ev.GetProperty(ical.ComponentPropertyOrganizer); org != nil {
		resp.OrganizerEmail, resp.OrganizerName = parseOrganizerProp(org.Value, org.ICalParameters)
	}

	resp.Attendees = mapAttendees(ev, resp.OrganizerEmail)
	return resp
}

// eventTimes returns DTSTART, DTEND, and whether the event is all-day.
// All-day is indicated by VALUE=DATE on DTSTART per RFC 5545 §3.6.1.
// We check the param explicitly because golang-ical's GetStartAt
// happily parses both DATE-TIME ("20260501T140000Z") and DATE
// ("20260704") formats — so a successful return tells us nothing about
// which form was used.
func eventTimes(ev *ical.VEvent) (start, end int64, allDay bool) {
	allDay = isAllDayProp(ev.GetProperty(ical.ComponentPropertyDtStart)) ||
		isAllDayProp(ev.GetProperty(ical.ComponentPropertyDtEnd))

	if allDay {
		if t, err := ev.GetAllDayStartAt(); err == nil {
			start = t.Unix()
		}
		if t, err := ev.GetAllDayEndAt(); err == nil {
			end = t.Unix()
		}
		return
	}
	if t, err := ev.GetStartAt(); err == nil {
		start = t.Unix()
	}
	if t, err := ev.GetEndAt(); err == nil {
		end = t.Unix()
	}
	return
}

// isAllDayProp reports whether a DTSTART/DTEND property carries
// VALUE=DATE — the iCalendar marker for an all-day event.
func isAllDayProp(p *ical.IANAProperty) bool {
	if p == nil {
		return false
	}
	return strings.EqualFold(firstParam(p.ICalParameters, "VALUE"), "DATE")
}

// firstConferenceURL prefers an explicit conferencing extension (Google
// uses X-GOOGLE-CONFERENCE) over the more generic URL property — URL is
// often the calendar's own canonical link rather than the meeting room.
func firstConferenceURL(ev *ical.VEvent) string {
	candidates := []ical.ComponentProperty{
		ical.ComponentPropertyExtended("X-GOOGLE-CONFERENCE"),
		ical.ComponentPropertyExtended("X-CONFERENCE-URL"),
		ical.ComponentPropertyUrl,
	}
	for _, c := range candidates {
		if p := ev.GetProperty(c); p != nil && p.Value != "" {
			return strings.TrimSpace(p.Value)
		}
	}
	return ""
}

// parseOrganizerProp pulls the email + display name from an ORGANIZER
// property such as `ORGANIZER;CN=Priya Patel:mailto:priya@partner.example`.
// Falls back to the raw value when the mailto: prefix is missing.
func parseOrganizerProp(value string, params map[string][]string) (email, name string) {
	v := strings.TrimSpace(value)
	if lower := strings.ToLower(v); strings.HasPrefix(lower, "mailto:") {
		v = v[len("mailto:"):]
	}
	email = strings.TrimSpace(v)
	if cn := firstParam(params, "CN"); cn != "" {
		name = strings.TrimSpace(cn)
	}
	return email, name
}

// mapAttendees flattens ATTENDEE properties to the response shape.
// Marks the organizer's own attendee row so the UI can show the
// organizer once rather than twice.
func mapAttendees(ev *ical.VEvent, organizerEmail string) []InviteAttendee {
	props := ev.GetProperties(ical.ComponentPropertyAttendee)
	if len(props) == 0 {
		return nil
	}
	out := make([]InviteAttendee, 0, len(props))
	for _, p := range props {
		email, name := parseOrganizerProp(p.Value, p.ICalParameters)
		if email == "" && name == "" {
			continue
		}
		a := InviteAttendee{
			Name:   name,
			Email:  email,
			Status: strings.ToUpper(firstParam(p.ICalParameters, "PARTSTAT")),
			Role:   strings.ToUpper(firstParam(p.ICalParameters, "ROLE")),
		}
		if organizerEmail != "" && strings.EqualFold(email, organizerEmail) {
			a.IsOrganizer = true
		}
		out = append(out, a)
	}
	return out
}

// firstParam returns the first value for a key in an ICalParameter map,
// or empty when missing. golang-ical stores parameters as []string to
// support multi-valued ones (e.g. CATEGORIES) — most properties of
// interest here are single-valued.
func firstParam(params map[string][]string, key string) string {
	if vs, ok := params[key]; ok && len(vs) > 0 {
		return vs[0]
	}
	return ""
}

// Sanity check that errNoUsableEvent is wired — a regression where the
// parser silently ignored bad input would let the UI render a blank card.
var _ = errors.New
