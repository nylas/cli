package air

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// TestHandleEmailInvite_RawMIMEFallback drives the Gmail-style invitation
// path: attachments[] is empty, so the handler must fetch raw_mime and
// pull the calendar payload from the inline body part. Pins the fix for
// "Air email viewer fails to render Gmail invitations" — Nylas doesn't
// surface inline parts as attachments, and the previous handler simply
// reported has_invite=false in that case.
func TestHandleEmailInvite_RawMIMEFallback(t *testing.T) {
	t.Parallel()

	server, client, _ := newCachedTestServer(t)

	client.GetMessageFunc = func(_ context.Context, _ string, messageID string) (*domain.Message, error) {
		return &domain.Message{
			ID:          messageID,
			Subject:     "Event Invitation: Meeting",
			Attachments: nil, // The whole point: no calendar attachment surfaced.
		}, nil
	}
	client.GetMessageWithFieldsFunc = func(_ context.Context, _ string, messageID, fields string) (*domain.Message, error) {
		if fields != "raw_mime" {
			t.Fatalf("handler asked for fields=%q, want raw_mime", fields)
		}
		return &domain.Message{
			ID:      messageID,
			Subject: "Event Invitation: Meeting",
			RawMIME: gmailStyleInviteMIME,
		}, nil
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/emails/email-1/invite", http.NoBody)
	server.handleEmailInvite(w, r, "email-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var got CalendarInviteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !got.HasInvite {
		t.Errorf("HasInvite=false, want true (raw_mime fallback should detect inline calendar)")
	}
	if got.Title != "Meeting" {
		t.Errorf("Title=%q, want Meeting", got.Title)
	}
	if got.Method != "REQUEST" {
		t.Errorf("Method=%q, want REQUEST", got.Method)
	}
	if !strings.HasPrefix(got.AttachmentID, inlineCalendarPrefix) {
		t.Errorf("AttachmentID=%q, want prefix %q (inline calendar marker)", got.AttachmentID, inlineCalendarPrefix)
	}
	if got.Filename == "" {
		t.Errorf("Filename should be populated for inline calendar parts (defaulted to invite.ics)")
	}
}

// TestHandleEmailInvite_NoCalendarAtAll exercises the silent-degrade
// path: neither attachments[] nor raw_mime contains a calendar part.
// The handler must return has_invite=false rather than 5xx so the
// frontend can keep the regular email render.
func TestHandleEmailInvite_NoCalendarAtAll(t *testing.T) {
	t.Parallel()

	server, client, _ := newCachedTestServer(t)
	client.GetMessageFunc = func(_ context.Context, _ string, messageID string) (*domain.Message, error) {
		return &domain.Message{ID: messageID, Subject: "Just a regular email"}, nil
	}
	client.GetMessageWithFieldsFunc = func(_ context.Context, _ string, messageID, _ string) (*domain.Message, error) {
		return &domain.Message{
			ID:      messageID,
			RawMIME: "From: a@example.com\r\nSubject: Hi\r\nContent-Type: text/plain\r\n\r\nNo calendar here.\r\n",
		}, nil
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/emails/email-2/invite", http.NoBody)
	server.handleEmailInvite(w, r, "email-2")

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got CalendarInviteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.HasInvite {
		t.Errorf("expected HasInvite=false for non-invite email, got %+v", got)
	}
}

// TestHandleEmailInvite_SyntheticAttachmentFallsBackToRawMIME pins the
// regression where Nylas's single-message API returns a synthetic
// attachment entry (id like "v0:base64(name):base64(ct):size") that
// looks downloadable but actually 404s on the attachments endpoint.
// The pre-fix handler returned 5xx; the fix falls through to the
// raw_mime walker so the user still gets the invite card.
func TestHandleEmailInvite_SyntheticAttachmentFallsBackToRawMIME(t *testing.T) {
	t.Parallel()

	server, client, _ := newCachedTestServer(t)

	// Synthetic attachment ID in the v0:... format Nylas v3 uses for
	// inline parts surfaced via the single-message endpoint.
	syntheticID := "v0:aW52aXRlLmljcw==:dGV4dC9jYWxlbmRhcjsgY2hhcnNldD11dGYtOA==:443"
	client.GetMessageFunc = func(_ context.Context, _ string, messageID string) (*domain.Message, error) {
		return &domain.Message{
			ID:      messageID,
			Subject: "Event Invitation: Meeting",
			Attachments: []domain.Attachment{
				{ID: syntheticID, Filename: "invite.ics", ContentType: "text/calendar; charset=utf-8", Size: 443},
			},
		}, nil
	}
	client.DownloadAttachmentFunc = func(_ context.Context, _, _, attID string) (io.ReadCloser, error) {
		if attID != syntheticID {
			t.Fatalf("expected download for synthetic id, got %q", attID)
		}
		return nil, errors.New("nylas API error: attachment not found") // mirrors the real prod failure
	}
	client.GetMessageWithFieldsFunc = func(_ context.Context, _ string, messageID, _ string) (*domain.Message, error) {
		return &domain.Message{ID: messageID, RawMIME: gmailStyleInviteMIME}, nil
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/emails/email-99/invite", http.NoBody)
	server.handleEmailInvite(w, r, "email-99")

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got CalendarInviteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.HasInvite {
		t.Errorf("HasInvite=false; want true (raw_mime fallback should recover from undownloadable attachment)")
	}
	// Keep the synthetic ID so the frontend's name-based attachment-row
	// dedup still matches the existing entry.
	if got.AttachmentID != syntheticID {
		t.Errorf("AttachmentID=%q, want synthetic id %q (preserved through fallback)", got.AttachmentID, syntheticID)
	}
	if got.Filename != "invite.ics" {
		t.Errorf("Filename=%q", got.Filename)
	}
}

// TestHandleEmailInvite_RealAttachmentTakesPriority pins the existing
// happy path: when attachments[] already has an ICS, the handler uses
// it directly without paying for a raw_mime round-trip. Microsoft and
// custom senders typically arrive this way.
func TestHandleEmailInvite_RealAttachmentTakesPriority(t *testing.T) {
	t.Parallel()

	server, client, _ := newCachedTestServer(t)
	client.GetMessageFunc = func(_ context.Context, _ string, messageID string) (*domain.Message, error) {
		return &domain.Message{
			ID:      messageID,
			Subject: "Meeting Invitation",
			Attachments: []domain.Attachment{
				{ID: "att-1", Filename: "invite.ics", ContentType: "text/calendar"},
			},
		}, nil
	}
	client.DownloadAttachmentFunc = func(_ context.Context, _, _, _ string) (io.ReadCloser, error) {
		body := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//T//EN\r\nMETHOD:REQUEST\r\n" +
			"BEGIN:VEVENT\r\nUID:real-1\r\nSUMMARY:Attached invite\r\n" +
			"DTSTART:20260501T140000Z\r\nDTEND:20260501T150000Z\r\n" +
			"END:VEVENT\r\nEND:VCALENDAR\r\n"
		return io.NopCloser(strings.NewReader(body)), nil
	}
	rawMIMEHit := false
	client.GetMessageWithFieldsFunc = func(_ context.Context, _ string, _, _ string) (*domain.Message, error) {
		rawMIMEHit = true
		return nil, errors.New("should not be called when attachments[] has the calendar")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/emails/email-3/invite", http.NoBody)
	server.handleEmailInvite(w, r, "email-3")

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if rawMIMEHit {
		t.Errorf("raw_mime fallback was called even though attachments[] had the calendar")
	}
	var got CalendarInviteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.AttachmentID != "att-1" {
		t.Errorf("AttachmentID=%q, want att-1 (real attachment)", got.AttachmentID)
	}
	if got.Title != "Attached invite" {
		t.Errorf("Title=%q", got.Title)
	}
}

// Sanity-check the inline-calendar attachment-ID helpers — these gate
// the frontend's "is this a synthetic part?" check, so a typo on the
// prefix would break inline rendering.
func TestInlineCalendarAttachmentID(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		want       string
		wantPrefix bool
	}{
		{"with content-id", "abc@example.com", inlineCalendarPrefix + "abc@example.com", true},
		{"empty content-id", "", inlineCalendarPrefix + "default", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := inlineCalendarAttachmentID(tc.input)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
			if isInlineCalendarAttachmentID(got) != tc.wantPrefix {
				t.Errorf("isInlineCalendarAttachmentID(%q) wrong", got)
			}
		})
	}

	if isInlineCalendarAttachmentID("regular-attachment-id") {
		t.Errorf("real attachment ID should not match inline prefix")
	}
}

// TestParseICS_AttendeesAndMethod pins the new fields surfaced by the
// golang-ical-backed parser. Without these the UI can't show "3 going,
// 1 declined" or the cancellation banner.
func TestParseICS_AttendeesAndMethod(t *testing.T) {
	body := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//T//EN\r\n" +
		"METHOD:REQUEST\r\n" +
		"BEGIN:VEVENT\r\nUID:e1\r\nSUMMARY:Standup\r\n" +
		"DTSTART:20260501T140000Z\r\nDTEND:20260501T143000Z\r\n" +
		"ORGANIZER;CN=Priya Patel:mailto:priya@partner.example\r\n" +
		"ATTENDEE;CN=Alice;PARTSTAT=ACCEPTED;ROLE=REQ-PARTICIPANT:mailto:alice@example.com\r\n" +
		"ATTENDEE;CN=Bob;PARTSTAT=DECLINED;ROLE=REQ-PARTICIPANT:mailto:bob@example.com\r\n" +
		"ATTENDEE;CN=Carol;PARTSTAT=TENTATIVE:mailto:carol@example.com\r\n" +
		"ATTENDEE;CN=Priya Patel;PARTSTAT=ACCEPTED:mailto:priya@partner.example\r\n" +
		"END:VEVENT\r\nEND:VCALENDAR\r\n"

	resp, err := parseICS(body)
	if err != nil {
		t.Fatalf("parseICS: %v", err)
	}
	if resp.Method != "REQUEST" {
		t.Errorf("Method=%q, want REQUEST", resp.Method)
	}
	if len(resp.Attendees) != 4 {
		t.Fatalf("Attendees count=%d, want 4", len(resp.Attendees))
	}

	// Build a map by email for stable lookups regardless of ordering.
	byEmail := map[string]InviteAttendee{}
	for _, a := range resp.Attendees {
		byEmail[a.Email] = a
	}

	if a, ok := byEmail["alice@example.com"]; !ok || a.Status != "ACCEPTED" {
		t.Errorf("Alice attendee wrong: %+v", a)
	}
	if a, ok := byEmail["bob@example.com"]; !ok || a.Status != "DECLINED" {
		t.Errorf("Bob attendee wrong: %+v", a)
	}
	if a, ok := byEmail["priya@partner.example"]; !ok || !a.IsOrganizer {
		t.Errorf("Priya should be flagged as organizer: %+v", a)
	}
}

// TestParseICS_CancelMethod pins that METHOD=CANCEL is preserved so the
// UI can render the cancellation banner instead of the RSVP buttons.
func TestParseICS_CancelMethod(t *testing.T) {
	body := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//T//EN\r\nMETHOD:CANCEL\r\n" +
		"BEGIN:VEVENT\r\nUID:e1\r\nSUMMARY:Cancelled meeting\r\n" +
		"DTSTART:20260501T140000Z\r\nDTEND:20260501T150000Z\r\n" +
		"STATUS:CANCELLED\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

	resp, err := parseICS(body)
	if err != nil {
		t.Fatalf("parseICS: %v", err)
	}
	if resp.Method != "CANCEL" {
		t.Errorf("Method=%q, want CANCEL", resp.Method)
	}
	if resp.Status != "CANCELLED" {
		t.Errorf("Status=%q, want CANCELLED", resp.Status)
	}
}
