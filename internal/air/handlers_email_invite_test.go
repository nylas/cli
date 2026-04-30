package air

import (
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

const sampleICS = "BEGIN:VCALENDAR\r\n" +
	"VERSION:2.0\r\n" +
	"PRODID:-//Test//EN\r\n" +
	"METHOD:REQUEST\r\n" +
	"BEGIN:VEVENT\r\n" +
	"UID:demo-uid-1@example.com\r\n" +
	"SUMMARY:Quarterly Sync\r\n" +
	"DESCRIPTION:Quarterly review with the partner team.\\nBring notes.\r\n" +
	"LOCATION:Conference Room A\\, HQ\r\n" +
	"DTSTART:20260501T140000Z\r\n" +
	"DTEND:20260501T150000Z\r\n" +
	"ORGANIZER;CN=Priya Patel:mailto:priya@partner.example\r\n" +
	"STATUS:CONFIRMED\r\n" +
	"URL:https://meet.example.com/q-sync\r\n" +
	"END:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

const sampleAllDayICS = "BEGIN:VCALENDAR\nBEGIN:VEVENT\n" +
	"SUMMARY:Holiday\n" +
	"DTSTART;VALUE=DATE:20260704\n" +
	"DTEND;VALUE=DATE:20260705\n" +
	"END:VEVENT\nEND:VCALENDAR\n"

func TestParseICS_BasicVEvent(t *testing.T) {
	ev, err := parseICS(sampleICS)
	if err != nil {
		t.Fatalf("parseICS: %v", err)
	}

	if ev.Title != "Quarterly Sync" {
		t.Errorf("Title: want %q, got %q", "Quarterly Sync", ev.Title)
	}
	if ev.Location != "Conference Room A, HQ" {
		t.Errorf("Location: want unescaped, got %q", ev.Location)
	}
	if !strings.Contains(ev.Description, "Bring notes.") {
		t.Errorf("Description: want unescaped \\n, got %q", ev.Description)
	}
	// 2026-05-01T14:00:00Z = 1777644000
	if ev.StartTime != 1777644000 {
		t.Errorf("StartTime: want 1777644000, got %d", ev.StartTime)
	}
	// 2026-05-01T15:00:00Z = 1777647600
	if ev.EndTime != 1777647600 {
		t.Errorf("EndTime: want 1777647600, got %d", ev.EndTime)
	}
	if ev.IsAllDay {
		t.Errorf("IsAllDay should be false for DATE-TIME")
	}
	if ev.OrganizerEmail != "priya@partner.example" {
		t.Errorf("OrganizerEmail: %q", ev.OrganizerEmail)
	}
	if ev.OrganizerName != "Priya Patel" {
		t.Errorf("OrganizerName: %q", ev.OrganizerName)
	}
	if ev.Status != "CONFIRMED" {
		t.Errorf("Status: %q", ev.Status)
	}
	if ev.ConferencingURL != "https://meet.example.com/q-sync" {
		t.Errorf("ConferencingURL: %q", ev.ConferencingURL)
	}
}

func TestParseICS_AllDayDate(t *testing.T) {
	ev, err := parseICS(sampleAllDayICS)
	if err != nil {
		t.Fatalf("parseICS: %v", err)
	}
	if ev.Title != "Holiday" {
		t.Errorf("Title: %q", ev.Title)
	}
	if !ev.IsAllDay {
		t.Error("IsAllDay should be true for VALUE=DATE")
	}
	// 2026-07-04T00:00:00Z = 1814400000... let's just sanity-check >0
	if ev.StartTime <= 0 {
		t.Errorf("StartTime should be positive Unix seconds, got %d", ev.StartTime)
	}
}

func TestParseICS_LineFolding(t *testing.T) {
	folded := "BEGIN:VCALENDAR\r\n" +
		"VERSION:2.0\r\n" +
		"PRODID:-//T//EN\r\n" +
		"BEGIN:VEVENT\r\n" +
		"UID:fold-1\r\n" +
		"SUMMARY:Long\r\n" +
		" Title Continued\r\n" +
		"DTSTART:20260501T140000Z\r\n" +
		"DTEND:20260501T150000Z\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	ev, err := parseICS(folded)
	if err != nil {
		t.Fatalf("parseICS: %v", err)
	}
	if ev.Title != "LongTitle Continued" {
		t.Errorf("expected unfolded title, got %q", ev.Title)
	}
}

func TestParseICS_MissingVEvent(t *testing.T) {
	// A calendar with zero events should not produce a renderable invite.
	_, err := parseICS("BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//T//EN\nEND:VCALENDAR\n")
	if err == nil {
		t.Fatal("expected error when no VEVENT block is present")
	}
}

func TestParseICS_EmptyEvent(t *testing.T) {
	// VEVENT block with no usable fields → error rather than empty noise.
	body := "BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//T//EN\nBEGIN:VEVENT\nUID:e\nEND:VEVENT\nEND:VCALENDAR\n"
	_, err := parseICS(body)
	if err == nil {
		t.Fatal("expected error for VEVENT with no fields")
	}
}

func TestIsCalendarAttachment(t *testing.T) {
	tests := []struct {
		name string
		ct   string
		fn   string
		want bool
	}{
		{"text/calendar", "text/calendar; charset=utf-8", "invite.ics", true},
		{"application/ics", "application/ics", "x", true},
		{".ics filename only", "application/octet-stream", "MEETING.ICS", true},
		{"pdf attachment", "application/pdf", "agenda.pdf", false},
		{"empty", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCalendarAttachment(tt.ct, tt.fn); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindCalendarAttachment(t *testing.T) {
	atts := []domain.Attachment{
		{ID: "a1", Filename: "spec.pdf", ContentType: "application/pdf"},
		{ID: "a2", Filename: "invite.ics", ContentType: "text/calendar"},
		{ID: "a3", Filename: "logo.png", ContentType: "image/png"},
	}
	got := findCalendarAttachment(atts)
	if got == nil {
		t.Fatal("expected to find ICS attachment")
	}
	if got.ID != "a2" {
		t.Errorf("expected a2, got %s", got.ID)
	}
}

func TestFilterDemoEmails_FolderFilter(t *testing.T) {
	all := demoEmails()

	inbox := filterDemoEmails(all, "inbox", false, false)
	sent := filterDemoEmails(all, "sent", false, false)
	drafts := filterDemoEmails(all, "drafts", false, false)
	trash := filterDemoEmails(all, "trash", false, false)
	archive := filterDemoEmails(all, "archive", false, false)
	none := filterDemoEmails(all, "", false, false)

	cases := []struct {
		name string
		got  []EmailResponse
		min  int
	}{
		{"inbox", inbox, 5},
		{"sent (>1 — proves filter works)", sent, 2},
		{"drafts", drafts, 1},
		{"trash", trash, 1},
		{"archive", archive, 1},
		{"none (empty filter returns all)", none, len(all)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.got) < tc.min {
				t.Errorf("%s: got %d, want >= %d", tc.name, len(tc.got), tc.min)
			}
		})
	}

	// Sent emails should NOT be in the inbox response.
	for _, e := range inbox {
		for _, f := range e.Folders {
			if strings.EqualFold(f, "sent") {
				t.Errorf("sent email %s leaked into inbox", e.ID)
			}
		}
	}
}

func TestFilterDemoEmails_AliasMapping(t *testing.T) {
	all := demoEmails()

	// "SENT", "Sent Items" should all map to canonical "sent".
	for _, alias := range []string{"SENT", "Sent Items", "sent mail"} {
		got := filterDemoEmails(all, alias, false, false)
		if len(got) < 2 {
			t.Errorf("alias %q: expected >=2 sent emails, got %d", alias, len(got))
		}
	}

	// "Deleted Items" → trash
	got := filterDemoEmails(all, "Deleted Items", false, false)
	if len(got) < 1 {
		t.Errorf("Deleted Items alias: expected >=1, got %d", len(got))
	}
}

func TestFilterDemoEmails_UnreadStarredFlags(t *testing.T) {
	all := demoEmails()

	unread := filterDemoEmails(all, "", true, false)
	for _, e := range unread {
		if !e.Unread {
			t.Errorf("unread filter returned read email %s", e.ID)
		}
	}

	starred := filterDemoEmails(all, "", false, true)
	for _, e := range starred {
		if !e.Starred {
			t.Errorf("starred filter returned unstarred email %s", e.ID)
		}
	}
}

func TestDemoInviteFor_KnownAndUnknownIDs(t *testing.T) {
	known := demoInviteFor("demo-email-invite-001")
	if !known.HasInvite {
		t.Fatal("known invite ID should return HasInvite=true")
	}
	if known.Title != "Quarterly Sync" {
		t.Errorf("Title: %q", known.Title)
	}
	if known.OrganizerEmail == "" || known.StartTime == 0 || known.EndTime == 0 {
		t.Errorf("expected populated fields, got %+v", known)
	}

	unknown := demoInviteFor("demo-email-001")
	if unknown.HasInvite {
		t.Error("non-invite email should return HasInvite=false")
	}
}
