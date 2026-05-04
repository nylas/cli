package air

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

// TestHandleEmailInvite_RealAttachmentDownloadFailure_LogsFailure pins
// the silent-failure gap in tryParseAttachmentInvite
// (handlers_email_invite.go:188-191). When DownloadAttachment fails on
// a real (non-synthetic) attachment ID — e.g., a transient Nylas 5xx,
// disk-full while streaming, decryption error — the function returns
// (CalendarInviteResponse{}, false) with no log. The handler then
// falls through to raw_mime, which is the right behavior for the user
// (the invite card might still render via the inline path) but means
// support cannot diagnose "RSVP card never appears for X.ics" without
// knowing which leg of the parser failed.
//
// EXPECTED FAILURE today: handlers_email_invite.go:189-191 returns
// silently. After the fix an slog Debug or Warn entry should record
// the attachment ID and the underlying download error so a wedged
// attachments endpoint shows up in production logs.
func TestHandleEmailInvite_RealAttachmentDownloadFailure_LogsFailure(t *testing.T) {
	// No t.Parallel — captureSlog mutates process-global slog default.
	server, client, _ := newCachedTestServer(t)

	const realAttID = "real-att-canary-DOWNLOAD-XYZ"
	const downloadErrSentinel = "nylas-503-download-canary-7777"
	client.GetMessageFunc = func(_ context.Context, _, msgID string) (*domain.Message, error) {
		return &domain.Message{
			ID:      msgID,
			Subject: "Calendar invite",
			Attachments: []domain.Attachment{
				{ID: realAttID, Filename: "invite.ics", ContentType: "text/calendar"},
			},
		}, nil
	}
	client.DownloadAttachmentFunc = func(context.Context, string, string, string) (io.ReadCloser, error) {
		return nil, errors.New(downloadErrSentinel)
	}
	// raw_mime fallback exists but does not contain a calendar part —
	// confirms the handler completes (200 has_invite=false) while still
	// expecting the download leg to have left a log breadcrumb.
	client.GetMessageWithFieldsFunc = func(_ context.Context, _ string, msgID, _ string) (*domain.Message, error) {
		return &domain.Message{ID: msgID, RawMIME: ""}, nil
	}

	logs := captureSlog(t)

	r := httptest.NewRequest(http.MethodGet, "/api/emails/email-1/invite", http.NoBody)
	w := httptest.NewRecorder()
	server.handleEmailInvite(w, r, "email-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s — silent fall-through must still produce a 200",
			w.Code, w.Body.String())
	}

	got := logs.String()
	assert.Contains(t, got, realAttID,
		"slog must record the failed attachment ID for diagnosability; got %q", got)
	assert.Contains(t, got, downloadErrSentinel,
		"slog must record the underlying download error so transient Nylas "+
			"failures are diagnosable; got %q", got)
}

// TestHandleEmailInvite_OversizedAttachment_LogsFailure pins the silent
// drop at handlers_email_invite.go:194-197 — when the streamed body
// exceeds maxICSBytes (or io.ReadAll returns an error), the function
// silently returns false. An attacker-controlled multi-MB ICS would
// land here with no record at all. Also covers the legitimate case
// where Nylas (rarely) ships a misencoded attachment that streams much
// larger than reported.
//
// EXPECTED FAILURE today: silent return. After the fix an slog Warn
// should record the attID + filename so oversize-DOS attempts and
// runaway attachments are visible.
func TestHandleEmailInvite_OversizedAttachment_LogsFailure(t *testing.T) {
	// No t.Parallel — captureSlog mutates process-global slog default.
	server, client, _ := newCachedTestServer(t)

	const fatAttID = "fat-att-canary-OVERSIZE-XYZ"
	client.GetMessageFunc = func(_ context.Context, _, msgID string) (*domain.Message, error) {
		return &domain.Message{
			ID: msgID,
			Attachments: []domain.Attachment{
				{ID: fatAttID, Filename: "huge-invite.ics", ContentType: "text/calendar", Size: 1},
			},
		}, nil
	}
	// 1MB+1 of 'A' — exceeds maxICSBytes (1<<20 in handlers_email_invite.go),
	// so the io.LimitReader read returns len(raw) == maxICSBytes+1, which
	// trips the oversized-payload swallow at line ~194-197.
	huge := strings.Repeat("A", (1<<20)+1)
	client.DownloadAttachmentFunc = func(context.Context, string, string, string) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(huge)), nil
	}
	client.GetMessageWithFieldsFunc = func(_ context.Context, _ string, msgID, _ string) (*domain.Message, error) {
		return &domain.Message{ID: msgID, RawMIME: ""}, nil
	}

	logs := captureSlog(t)

	r := httptest.NewRequest(http.MethodGet, "/api/emails/email-1/invite", http.NoBody)
	w := httptest.NewRecorder()
	server.handleEmailInvite(w, r, "email-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s — silent fall-through must still produce a 200",
			w.Code, w.Body.String())
	}

	got := logs.String()
	assert.Contains(t, got, fatAttID,
		"slog must record the oversize attachment ID — currently silent at "+
			"handlers_email_invite.go:194-197; got %q", got)
}

// TestHandleEmailInvite_MalformedICS_LogsFailure pins the third silent
// drop in tryParseAttachmentInvite (handlers_email_invite.go:199-202):
// parseICS errors are swallowed entirely. A malformed calendar payload
// should be visible to support, even if the user-facing path falls
// through to raw_mime.
//
// EXPECTED FAILURE today: parseICS error returned without log. After
// the fix an slog Debug entry should fire with the attachment ID and
// the parse error so misencoded ICS payloads are diagnosable.
func TestHandleEmailInvite_MalformedICS_LogsFailure(t *testing.T) {
	// No t.Parallel — captureSlog mutates process-global slog default.
	server, client, _ := newCachedTestServer(t)

	const badICSAttID = "bad-ics-att-canary-PARSE-XYZ"
	client.GetMessageFunc = func(_ context.Context, _, msgID string) (*domain.Message, error) {
		return &domain.Message{
			ID: msgID,
			Attachments: []domain.Attachment{
				{ID: badICSAttID, Filename: "broken.ics", ContentType: "text/calendar"},
			},
		}, nil
	}
	client.DownloadAttachmentFunc = func(context.Context, string, string, string) (io.ReadCloser, error) {
		// Not a valid VCALENDAR — parseICS will error.
		return io.NopCloser(strings.NewReader("THIS IS NOT VALID ICS PAYLOAD")), nil
	}
	client.GetMessageWithFieldsFunc = func(_ context.Context, _ string, msgID, _ string) (*domain.Message, error) {
		return &domain.Message{ID: msgID, RawMIME: ""}, nil
	}

	logs := captureSlog(t)

	r := httptest.NewRequest(http.MethodGet, "/api/emails/email-1/invite", http.NoBody)
	w := httptest.NewRecorder()
	server.handleEmailInvite(w, r, "email-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s — silent fall-through must still produce a 200",
			w.Code, w.Body.String())
	}

	got := logs.String()
	assert.Contains(t, got, badICSAttID,
		"slog must record the malformed-ICS attachment ID — currently silent at "+
			"handlers_email_invite.go:199-202; got %q", got)
}
