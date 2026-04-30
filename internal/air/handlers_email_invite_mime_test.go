package air

import (
	"strings"
	"testing"
)

// gmailStyleInviteMIME is a minimal multipart/alternative message with
// an inline text/calendar leaf. It mirrors what Gmail ships when the
// sender declines to attach the ICS as a separate file — the very case
// that hides the invite from Nylas's attachments[] list.
const gmailStyleInviteMIME = "From: organizer@example.com\r\n" +
	"To: user@example.com\r\n" +
	"Subject: Event Invitation: Meeting\r\n" +
	"Content-Type: multipart/alternative; boundary=\"BOUNDARY1\"\r\n" +
	"\r\n" +
	"--BOUNDARY1\r\n" +
	"Content-Type: text/plain; charset=UTF-8\r\n" +
	"\r\n" +
	"You have received a calendar invitation: Meeting\r\n" +
	"--BOUNDARY1\r\n" +
	"Content-Type: text/calendar; charset=UTF-8; method=REQUEST; name=invite.ics\r\n" +
	"Content-Disposition: inline; filename=invite.ics\r\n" +
	"Content-Transfer-Encoding: 7bit\r\n" +
	"\r\n" +
	"BEGIN:VCALENDAR\r\n" +
	"VERSION:2.0\r\n" +
	"PRODID:-//Test//EN\r\n" +
	"METHOD:REQUEST\r\n" +
	"BEGIN:VEVENT\r\n" +
	"UID:gmail-1@example.com\r\n" +
	"SUMMARY:Meeting\r\n" +
	"DTSTART:20260501T140000Z\r\n" +
	"DTEND:20260501T150000Z\r\n" +
	"ORGANIZER;CN=Priya Patel:mailto:priya@partner.example\r\n" +
	"END:VEVENT\r\n" +
	"END:VCALENDAR\r\n" +
	"--BOUNDARY1--\r\n"

// nestedInviteMIME wraps the calendar inside multipart/mixed →
// multipart/alternative → text/calendar. Outlook produces nesting like
// this when the body has both an HTML rendering and the attached ICS.
const nestedInviteMIME = "From: a@example.com\r\n" +
	"Subject: Meeting Invitation\r\n" +
	"Content-Type: multipart/mixed; boundary=\"OUTER\"\r\n" +
	"\r\n" +
	"--OUTER\r\n" +
	"Content-Type: multipart/alternative; boundary=\"INNER\"\r\n" +
	"\r\n" +
	"--INNER\r\n" +
	"Content-Type: text/html; charset=UTF-8\r\n" +
	"\r\n" +
	"<p>Calendar invite</p>\r\n" +
	"--INNER\r\n" +
	"Content-Type: text/calendar; charset=UTF-8; method=REQUEST\r\n" +
	"Content-Transfer-Encoding: 7bit\r\n" +
	"\r\n" +
	"BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//T//EN\r\nMETHOD:REQUEST\r\n" +
	"BEGIN:VEVENT\r\nUID:nested-1\r\nSUMMARY:Nested\r\n" +
	"DTSTART:20260501T140000Z\r\nDTEND:20260501T150000Z\r\n" +
	"END:VEVENT\r\nEND:VCALENDAR\r\n" +
	"--INNER--\r\n" +
	"--OUTER--\r\n"

func TestFindInlineCalendarParts_GmailShape(t *testing.T) {
	parts := findInlineCalendarParts(gmailStyleInviteMIME)
	if len(parts) != 1 {
		t.Fatalf("want 1 part, got %d", len(parts))
	}
	p := parts[0]
	if !strings.Contains(p.Body, "BEGIN:VEVENT") {
		t.Errorf("Body should be the VCALENDAR payload, got: %q", p.Body[:min(80, len(p.Body))])
	}
	if p.Filename != "invite.ics" {
		t.Errorf("Filename: want invite.ics, got %q", p.Filename)
	}
	if p.Method != "REQUEST" {
		t.Errorf("Method: want REQUEST, got %q", p.Method)
	}
}

func TestFindInlineCalendarParts_NestedAlternative(t *testing.T) {
	parts := findInlineCalendarParts(nestedInviteMIME)
	if len(parts) != 1 {
		t.Fatalf("want 1 part, got %d", len(parts))
	}
	if !strings.Contains(parts[0].Body, "SUMMARY:Nested") {
		t.Errorf("Body did not contain expected SUMMARY")
	}
}

func TestFindInlineCalendarParts_NoCalendar(t *testing.T) {
	plain := "From: a@example.com\r\n" +
		"Subject: Hi\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		"Just a regular email.\r\n"
	if got := findInlineCalendarParts(plain); len(got) != 0 {
		t.Errorf("plain mail should produce no parts, got %d", len(got))
	}
}

func TestFindInlineCalendarParts_EmptyAndOversize(t *testing.T) {
	if got := findInlineCalendarParts(""); got != nil {
		t.Errorf("empty input should return nil, got %d parts", len(got))
	}
	huge := strings.Repeat("x", maxRawMIMEBytes+1)
	if got := findInlineCalendarParts(huge); got != nil {
		t.Errorf("oversize input should be rejected, got %d parts", len(got))
	}
}

func TestFindInlineCalendarParts_QuotedPrintableBody(t *testing.T) {
	// Realistic case: Outlook frequently encodes calendar bodies with
	// quoted-printable so soft line breaks survive 998-octet MIME limits.
	mime := "Content-Type: multipart/alternative; boundary=B\r\n" +
		"\r\n" +
		"--B\r\n" +
		"Content-Type: text/calendar; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		"BEGIN:VCALENDAR=0D=0A" +
		"BEGIN:VEVENT=0D=0A" +
		"SUMMARY:Encoded=0D=0A" +
		"DTSTART:20260501T140000Z=0D=0A" +
		"END:VEVENT=0D=0A" +
		"END:VCALENDAR=0D=0A" +
		"\r\n--B--\r\n"
	parts := findInlineCalendarParts(mime)
	if len(parts) != 1 {
		t.Fatalf("want 1 part, got %d", len(parts))
	}
	if !strings.Contains(parts[0].Body, "SUMMARY:Encoded") {
		t.Errorf("quoted-printable body wasn't decoded: %q", parts[0].Body)
	}
}

// Round-trip the Gmail-style MIME through the parser to prove the full
// "raw_mime → walk → ICS → parse" pipeline produces a card-ready event.
func TestParseICS_FromGmailInlinePart(t *testing.T) {
	parts := findInlineCalendarParts(gmailStyleInviteMIME)
	if len(parts) == 0 {
		t.Fatal("expected an inline calendar part")
	}
	resp, err := parseICS(parts[0].Body)
	if err != nil {
		t.Fatalf("parseICS: %v", err)
	}
	if resp.Title != "Meeting" {
		t.Errorf("Title: %q", resp.Title)
	}
	if resp.Method != "REQUEST" {
		t.Errorf("Method: %q", resp.Method)
	}
	if resp.OrganizerEmail != "priya@partner.example" {
		t.Errorf("OrganizerEmail: %q", resp.OrganizerEmail)
	}
}
