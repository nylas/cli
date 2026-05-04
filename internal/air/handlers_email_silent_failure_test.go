package air

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nylas/cli/internal/air/cache"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

// captureSlog redirects slog.Default to a buffer for the duration of t.
// Tests using it must not call t.Parallel() — slog.Default is process-global.
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	prev := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

// TestHandleDeleteEmail_OfflineQueueFails_LogsFailure exposes the
// observability divergence between the near-twin handlers
// handleUpdateEmail and handleDeleteEmail. Both follow the same
// shape:
//
//	if !s.IsOnline() {
//	    if err := enqueue(...); err == nil { return queued-200 }
//	    // err != nil: queue is broken, falling through to API
//	}
//	... live API call ...
//
// handleUpdateEmail logs the offline-enqueue failure at
// handlers_email.go:305 ("offline enqueue failed, attempting live API
// call"). handleDeleteEmail does not — the err return from
// enqueueMessageDelete on line 372 is silently dropped (no else
// clause). A queue health regression that affects both code paths is
// debuggable for update and invisible for delete.
//
// EXPECTED FAILURE today: the slog buffer for the delete path is
// empty (no warning emitted). After the fix it contains a warning
// referencing the emailID. The fail-first message is the bug receipt.
func TestHandleDeleteEmail_OfflineQueueFails_LogsFailure(t *testing.T) {
	// No t.Parallel — captureSlog mutates the process-global default.
	server, client, _ := newCachedTestServer(t)
	server.SetOnline(false)
	// Disable the offline queue so enqueueMessageDelete returns
	// "offline queue unavailable" (handlers_email_offline.go:51).
	server.cacheSettings.OfflineQueueEnabled = false

	apiCalled := false
	client.DeleteMessageFunc = func(context.Context, string, string) error {
		apiCalled = true
		return nil
	}

	const sentinelEmailID = "deletefail-canary-offline-XYZ"
	logs := captureSlog(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/emails/"+sentinelEmailID, nil)
	w := httptest.NewRecorder()
	server.handleDeleteEmail(w, req, sentinelEmailID)

	if !apiCalled {
		t.Fatal("expected fall-through to live DeleteMessage when queue fails")
	}
	got := logs.String()
	assert.Contains(t, got, sentinelEmailID,
		"slog should record the offline-queue failure for emailID %q (parity with handleUpdateEmail line 305); got %q",
		sentinelEmailID, got)
}

// TestHandleDeleteEmail_QueueWriteAfterTransientError_LogsQueueFailure
// pins the second observability gap. When the live API errors with a
// transient/queue-eligible error AND the offline queue write itself
// fails, handleUpdateEmail logs both error contexts together at
// handlers_email.go:330-335:
//
//	slog.Error("queue enqueue after transient API error failed",
//	    "emailID", emailID, "apiErr", err, "queueErr", queueErr)
//
// handleDeleteEmail's mirror branch (handlers_email.go:384-393) has no
// such log — only the catch-all "Failed to delete email" log fires
// with `err = apiErr`, dropping the `queueErr` context entirely. The
// double-failure mode is exactly when a queue health alert matters
// most (the user is also about to see a 500), and it's exactly when
// the divergence makes it invisible.
//
// EXPECTED FAILURE today: the slog buffer captures "Failed to delete
// email" but does not record the queueErr substring "offline queue
// unavailable". After the fix the queueErr is co-logged with apiErr.
func TestHandleDeleteEmail_QueueWriteAfterTransientError_LogsQueueFailure(t *testing.T) {
	// No t.Parallel — captureSlog mutates the process-global default.
	server, client, _ := newCachedTestServer(t)
	client.DeleteMessageFunc = func(context.Context, string, string) error {
		return &transientNetErr{}
	}
	// To reach the inner queueErr branch we need shouldQueueEmailAction
	// to say YES (queue is configured AND error looks transient) but
	// the actual enqueue to fail. Achieved by replacing the grant
	// store's grant with one whose Email is empty: withAuthGrant still
	// resolves the default grant (ID-based), but
	// `getAccountEmail(grantID)` returns "" because the grant's Email
	// is blank, which trips enqueueMessageDelete's first guard:
	// `if accountEmail == "" { return errors.New("offline queue unavailable") }`.
	grantStore := server.grantStore.(*testGrantStore)
	grantStore.grants = []domain.GrantInfo{{
		ID:       "grant-123",
		Email:    "", // ← the load-bearing empty
		Provider: domain.ProviderGoogle,
	}}
	grantStore.defaultGrant = "grant-123"

	const sentinelEmailID = "deletefail-canary-double-XYZ"
	logs := captureSlog(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/emails/"+sentinelEmailID, nil)
	w := httptest.NewRecorder()
	server.handleDeleteEmail(w, req, sentinelEmailID)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s, want 500 (double-failure path)",
			w.Code, w.Body.String())
	}
	got := logs.String()
	// Sentinel from the queue-error message in handlers_email_offline.go:51.
	const queueErrSentinel = "offline queue unavailable"
	assert.Contains(t, got, queueErrSentinel,
		"slog should record the queueErr context (parity with handleUpdateEmail lines 330-335); got %q",
		got)
	// Also assert the test's emailID appears so the log is correlatable
	// to the failing request — same correlation handleUpdateEmail offers.
	assert.Contains(t, got, sentinelEmailID,
		"slog should record emailID for correlation; got %q", got)
}

// TestHandleGetEmail_CacheFillFailure_LogsFailure pins the parity gap
// between handleListEmails (handlers_email.go:144 logs cache-fill
// failures) and handleGetEmail (line 261-264 silently does
// `_ = s.withEmailStore(...)`). The user-facing behavior is correct —
// cache write failures must not break the response — but the silent
// drop means a wedged single-message cache drifts further from server
// state on every request, with no log to debug from.
//
// We force the write to fail by dropping the underlying SQLite emails
// table after lazy-init: the lookup call returns "no such table" (no
// cache hit, falls through), the live GetMessage succeeds, the
// post-fetch Put then fails for the same reason. Today no log fires.
// After the fix, slog should record the failure with the sentinel
// emailID for support diagnosability.
//
// EXPECTED FAILURE today: the slog buffer for the cache-fill path is
// silent. After the fix it contains a warning referencing the emailID
// (parity with handleListEmails).
func TestHandleGetEmail_CacheFillFailure_LogsFailure(t *testing.T) {
	// No t.Parallel — captureSlog mutates the process-global default.
	server, client, accountEmail := newCachedTestServer(t)

	const sentinelEmailID = "cachefill-canary-XYZ"
	client.GetMessageFunc = func(_ context.Context, _, msgID string) (*domain.Message, error) {
		return &domain.Message{
			ID:      msgID,
			Subject: "Test message",
			From:    []domain.EmailParticipant{{Email: "x@example.com"}},
		}, nil
	}

	// Lazy-init the emails table by acquiring the store once, then
	// drop it so the post-GetMessage Put fails with "no such table:
	// emails". The handler-side `_ = s.withEmailStore(...)` swallows
	// that error today.
	if err := server.withEmailStore(accountEmail, func(_ *cache.EmailStore) error { return nil }); err != nil {
		t.Fatalf("lazy-init emails store: %v", err)
	}
	db, err := server.cacheManager.GetDB(accountEmail)
	if err != nil {
		t.Fatalf("get db: %v", err)
	}
	if _, err := db.Exec("DROP TABLE IF EXISTS emails"); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	logs := captureSlog(t)

	req := httptest.NewRequest(http.MethodGet, "/api/emails/"+sentinelEmailID, nil)
	w := httptest.NewRecorder()
	server.handleGetEmail(w, req, sentinelEmailID)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200 — user-facing response must not break on cache failure",
			w.Code, w.Body.String())
	}

	got := logs.String()
	assert.Contains(t, got, sentinelEmailID,
		"slog should record the emailID for cache-fill failures (parity with handleListEmails:144); got %q",
		got)
}
