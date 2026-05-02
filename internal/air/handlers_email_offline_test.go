package air

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/air/cache"
	"github.com/nylas/cli/internal/domain"
)

// TestHandleListEmails_APIErrorFallsBackToCache pins the server-online,
// upstream-failed cache fallback. The handler is expected to return
// cached results with HasMore:false (so the UI doesn't try to paginate
// past the snapshot) instead of surfacing a 500. This was an untested
// branch — without coverage a future refactor of the cache fallback
// loop could silently leave the UI dead in the water on transient
// Nylas outages even though the cache is healthy.
func TestHandleListEmails_APIErrorFallsBackToCache(t *testing.T) {
	t.Parallel()

	server, client, accountEmail := newCachedTestServer(t)
	putCachedEmail(t, server, accountEmail, &cache.CachedEmail{
		ID:        "cached-1",
		FolderID:  "inbox",
		Subject:   "Cached when API is down",
		FromName:  "Cache",
		FromEmail: "cache@example.com",
		Date:      time.Now(),
		CachedAt:  time.Now(),
	})

	// Single-message cache won't satisfy the "full page" short-circuit
	// when a folder filter is applied (the threshold is len >= limit),
	// so we hit the API path. Force the API to fail with a transient
	// error — handler must serve the stale cache rather than 500.
	apiCalled := false
	client.GetMessagesWithParamsFunc = func(_ context.Context, _ string, _ *domain.MessageQueryParams) ([]domain.Message, error) {
		apiCalled = true
		return nil, errors.New("nylas 503 service unavailable")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/emails?folder=inbox", nil)
	w := httptest.NewRecorder()
	server.handleListEmails(w, req)

	if !apiCalled {
		t.Fatal("expected the API to be called once before falling back to cache")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200 (cache fallback)", w.Code, w.Body.String())
	}
	var resp EmailsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Emails) != 1 || resp.Emails[0].ID != "cached-1" {
		t.Errorf("response emails=%+v, want single cached-1", resp.Emails)
	}
	// HasMore must be false — paginating past the cache snapshot would
	// hit the same broken upstream and confuse the user.
	if resp.HasMore {
		t.Errorf("HasMore=true on cache fallback; expected false to prevent retry-paginate")
	}
}

// TestHandleUpdateEmail_OnlineTransientErrorQueuesAction pins the
// transient-error branch in handleUpdateEmail. When the API call
// returns a network-shaped error AND the offline queue is configured,
// the handler must enqueue the action, flip server state to offline,
// and return a 200 "queued" envelope — not a 500.
func TestHandleUpdateEmail_OnlineTransientErrorQueuesAction(t *testing.T) {
	t.Parallel()

	server, client, _ := newCachedTestServer(t)
	// Force a transient error type that shouldQueueEmailAction recognises.
	client.UpdateMessageFunc = func(context.Context, string, string, *domain.UpdateMessageRequest) (*domain.Message, error) {
		return nil, &transientNetErr{}
	}

	req := httptest.NewRequest(http.MethodPut, "/api/emails/email-1",
		strings.NewReader(`{"unread":false}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.handleUpdateEmail(w, req, "email-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200 (transient error must queue, not 500)", w.Code, w.Body.String())
	}
	var resp UpdateEmailResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success || !strings.Contains(resp.Message, "queued") {
		t.Errorf("response=%+v, want Success:true with 'queued' in message", resp)
	}
	// Server should now be marked offline so subsequent calls take the
	// fast offline-first path instead of round-tripping the broken API.
	if server.IsOnline() {
		t.Error("expected SetOnline(false) after transient API error, got isOnline=true")
	}
}

// TestHandleUpdateEmail_OfflineButQueueFails_FallsThroughToAPI pins
// the rare case where the server is offline AND the offline queue
// itself is broken. The handler should still attempt the live API
// call (it might succeed — IsOnline can be stale) rather than dropping
// the user's action silently.
func TestHandleUpdateEmail_OfflineButQueueFails_FallsThroughToAPI(t *testing.T) {
	t.Parallel()

	server, client, _ := newCachedTestServer(t)
	server.SetOnline(false)
	// Disable the offline queue so enqueueMessageUpdate fails. With no
	// account-email lookup possible the helper short-circuits with an
	// "offline queue unavailable" error.
	server.offlineQueues = nil
	server.cacheSettings.OfflineQueueEnabled = false

	apiCalled := false
	client.UpdateMessageFunc = func(context.Context, string, string, *domain.UpdateMessageRequest) (*domain.Message, error) {
		apiCalled = true
		return &domain.Message{ID: "email-1"}, nil
	}

	req := httptest.NewRequest(http.MethodPut, "/api/emails/email-1",
		strings.NewReader(`{"unread":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.handleUpdateEmail(w, req, "email-1")

	if !apiCalled {
		t.Error("expected handler to attempt live API call when queue is unavailable")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status=%d body=%s, want 200 (live retry succeeded)", w.Code, w.Body.String())
	}
}

// TestShouldQueueEmailAction pins the predicate that gates whether
// upstream errors enter the offline queue. Without direct coverage,
// regressions in the net.Error / context.DeadlineExceeded matching
// are invisible — the only signal would be "archives queue properly
// most of the time."
func TestShouldQueueEmailAction(t *testing.T) {
	t.Parallel()

	server, _, _ := newCachedTestServer(t)

	cases := []struct {
		name    string
		err     error
		online  bool
		enabled bool
		want    bool
	}{
		{
			name:    "offline + queue enabled → queue",
			err:     errors.New("any error"),
			online:  false,
			enabled: true,
			want:    true,
		},
		{
			name:    "online + transient net.Error → queue",
			err:     &transientNetErr{},
			online:  true,
			enabled: true,
			want:    true,
		},
		{
			name:    "online + context deadline exceeded → queue",
			err:     context.DeadlineExceeded,
			online:  true,
			enabled: true,
			want:    true,
		},
		{
			name:    "online + plain error (4xx-shaped) → don't queue",
			err:     errors.New("nylas 401 unauthorized"),
			online:  true,
			enabled: true,
			want:    false,
		},
		{
			name:    "queue disabled → never queue",
			err:     &transientNetErr{},
			online:  false,
			enabled: false,
			want:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// shouldQueueEmailAction is a method on *Server, not pure —
			// reset state between cases. Each subtest is sequential so
			// the shared server is safe.
			server.SetOnline(tc.online)
			server.cacheSettings.OfflineQueueEnabled = tc.enabled
			got := server.shouldQueueEmailAction(tc.err)
			if got != tc.want {
				t.Errorf("shouldQueueEmailAction(%v) [online=%v enabled=%v] = %v, want %v",
					tc.err, tc.online, tc.enabled, got, tc.want)
			}
		})
	}
}

// TestNormalizeDemoFolder pins the alias map for demo-mode folder
// filtering. Without coverage, "Sent Items" → "sent" silently breaks
// when a future refactor inlines the switch into demoEmailIsInFolder.
func TestNormalizeDemoFolder(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"inbox", "inbox"},
		{"INBOX", "inbox"},
		{" inbox ", "inbox"},
		{"sent", "sent"},
		{"Sent Items", "sent"}, // Microsoft display name
		{"Sent Mail", "sent"},  // Gmail display name
		{"drafts", "drafts"},
		{"Draft", "drafts"},
		{"archive", "archive"},
		{"All Mail", "all"}, // Gmail "All Mail" routes to the "show everything" target
		{"all", "all"},
		{"trash", "trash"},
		{"Deleted Items", "trash"}, // Microsoft
		{"Deleted", "trash"},
		{"spam", "spam"},
		{"Junk", "spam"},
		{"Junk Email", "spam"}, // Outlook
		{"starred", "starred"},
		{"unknown-folder", "unknown-folder"}, // pass-through, lowercased
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeDemoFolder(tc.input)
			if got != tc.want {
				t.Errorf("normalizeDemoFolder(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestDemoEmailIsInFolder_AllAliasMatchesEverything pins the special
// "all" branch — a UI-supplied "all" target should match every demo
// email regardless of its folders[].
func TestDemoEmailIsInFolder_AllAliasMatchesEverything(t *testing.T) {
	t.Parallel()

	cases := []EmailResponse{
		{Folders: []string{"inbox"}},
		{Folders: []string{"sent"}},
		{Folders: []string{"trash"}},
		{Folders: nil},
		{Folders: []string{}},
	}
	for i, e := range cases {
		if !demoEmailIsInFolder(e, "all") {
			t.Errorf("case %d: demoEmailIsInFolder(%+v, \"all\") = false, want true", i, e)
		}
	}
}

// TestFilterDemoEmails_FolderAllReturnsEverything exercises the same
// "all means everything" intent end-to-end through the call chain a UI
// request actually takes:
//
//	filterDemoEmails(emails, "all", false, false)
//	  → normalizeDemoFolder("all")        // canonicalises folder string
//	  → demoEmailIsInFolder(e, target)    // matches against e.Folders
//
// `demoEmailIsInFolder` has a dedicated `if target == "all" { return
// true }` branch (handlers_email_demo.go:219), but
// `normalizeDemoFolder` collapses "all" into "archive" via the alias
// case `"archive", "all", "all mail"`. The branch is therefore
// unreachable from the UI path, and `filterDemoEmails(emails, "all",
// ...)` returns only emails whose Folders[] contains "archive" — one
// email — instead of all 13.
//
// EXPECTED FAILURE today: assertion expects len(filtered) ==
// len(demoEmails()), got 1. After the fix (drop "all"/"all mail" from
// the archive aliases — or remove the dead branch and update
// TestNormalizeDemoFolder) this test passes.
func TestFilterDemoEmails_FolderAllReturnsEverything(t *testing.T) {
	t.Parallel()

	all := demoEmails()
	got := filterDemoEmails(all, "all", false, false)

	if len(got) != len(all) {
		t.Errorf("filterDemoEmails(_, \"all\") returned %d email(s), want %d (every demo email)",
			len(got), len(all))
	}
}

// TestHandleGetEmail_CachedBodyServedWhenAPIWouldFail pins the user-
// visible promise: "when Nylas is down but the cache holds the email,
// I can still read it." The handler's cache-first short-circuit
// returns the cached body before ever calling GetMessage, so the API
// stub here exists as a fail-loud guard — if a refactor were to
// reorder the lookups so the API call fires first AND fails, the
// stub would record it AND the response would still need to deliver
// the body (via the fallback at handlers_email.go ~241).
func TestHandleGetEmail_CachedBodyServedWhenAPIWouldFail(t *testing.T) {
	t.Parallel()

	server, client, accountEmail := newCachedTestServer(t)
	putCachedEmail(t, server, accountEmail, &cache.CachedEmail{
		ID:        "email-1",
		FolderID:  "inbox",
		Subject:   "Cached body survives outage",
		FromName:  "Cache",
		FromEmail: "cache@example.com",
		BodyHTML:  "<p>cached body</p>",
		Date:      time.Now(),
		CachedAt:  time.Now(),
	})
	// API failure stub: in the documented flow this never fires (the
	// cache hit short-circuits first), but we leave it wired so any
	// refactor that bypasses the cache-first lookup will land in a
	// 500 instead of silently masking the regression.
	client.GetMessageFunc = func(context.Context, string, string) (*domain.Message, error) {
		return nil, errors.New("nylas 503 service unavailable")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/emails/email-1", nil)
	w := httptest.NewRecorder()
	server.handleGetEmail(w, req, "email-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200 (cached body must be served)", w.Code, w.Body.String())
	}
	var resp EmailResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != "email-1" {
		t.Errorf("response id=%q, want email-1", resp.ID)
	}
	if resp.Body != "<p>cached body</p>" {
		t.Errorf("response body=%q, want full cached BodyHTML — the cached-email response must include Body, not just metadata", resp.Body)
	}
}

// TestHandleGetEmail_APIErrorWhenNotCached_Returns500 covers the
// other end of the same branch: cache empty, API fails. The user
// sees a generic 500 (no upstream-error leakage) rather than a stuck
// loading state. Pins both the user-visible code AND the privacy
// contract that we don't echo Nylas's raw error string back.
func TestHandleGetEmail_APIErrorWhenNotCached_Returns500(t *testing.T) {
	t.Parallel()

	server, client, _ := newCachedTestServer(t)
	client.GetMessageFunc = func(context.Context, string, string) (*domain.Message, error) {
		// Include identifying noise in the upstream error so we can
		// assert the handler doesn't echo it back.
		return nil, errors.New("nylas 503: grant_id=secret-grant-12345 endpoint=/messages/email-1")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/emails/uncached-id", nil)
	w := httptest.NewRecorder()
	server.handleGetEmail(w, req, "uncached-id")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s, want 500", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "secret-grant-12345") || strings.Contains(body, "/messages/") {
		t.Errorf("response leaked upstream error details: %s", body)
	}
}

// TestHandleDeleteEmail_OfflineButQueueFails_FallsThroughToAPI pins
// that an offline server with a broken queue still attempts the live
// API call. Without this fallthrough a misconfigured cache silently
// swallows every delete the user issues — invisible data loss.
func TestHandleDeleteEmail_OfflineButQueueFails_FallsThroughToAPI(t *testing.T) {
	t.Parallel()

	server, client, _ := newCachedTestServer(t)
	server.SetOnline(false)
	// Disable queueing so enqueueMessageDelete fails closed.
	server.cacheSettings.OfflineQueueEnabled = false

	apiCalled := false
	client.DeleteMessageFunc = func(context.Context, string, string) error {
		apiCalled = true
		return nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/emails/email-1", nil)
	w := httptest.NewRecorder()
	server.handleDeleteEmail(w, req, "email-1")

	if !apiCalled {
		t.Error("expected handler to attempt live DeleteMessage when queue is unavailable")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status=%d body=%s, want 200 (live retry succeeded)", w.Code, w.Body.String())
	}
}

// TestHandleDeleteEmail_OnlineTransientErrorQueuesAction pins the
// online-transient-error path for delete: API errors with a
// queue-eligible error AND the queue is healthy → enqueue and return
// 200. Mirrors the update-side test so a future refactor can't
// accidentally diverge the two flows.
func TestHandleDeleteEmail_OnlineTransientErrorQueuesAction(t *testing.T) {
	t.Parallel()

	server, client, _ := newCachedTestServer(t)
	client.DeleteMessageFunc = func(context.Context, string, string) error {
		return &transientNetErr{}
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/emails/email-1", nil)
	w := httptest.NewRecorder()
	server.handleDeleteEmail(w, req, "email-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200 (transient error must queue)", w.Code, w.Body.String())
	}
	var resp UpdateEmailResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success || !strings.Contains(resp.Message, "queued") {
		t.Errorf("response=%+v, want Success:true with 'queued' in message", resp)
	}
	if server.IsOnline() {
		t.Error("expected SetOnline(false) after transient API error, got isOnline=true")
	}
}

// TestHandleDeleteEmail_OnlineTransientErrorQueueFails_Returns500
// pins the worst-case for delete: API errors with a transient AND the
// queue write itself fails. The user must see a 500 rather than a
// silently-dropped delete. Delete is irreversible — silent loss of
// the user's intent here is the most damaging branch in the package.
func TestHandleDeleteEmail_OnlineTransientErrorQueueFails_Returns500(t *testing.T) {
	t.Parallel()

	server, client, _ := newCachedTestServer(t)
	client.DeleteMessageFunc = func(context.Context, string, string) error {
		return &transientNetErr{}
	}
	// Disable the queue so enqueueMessageDelete fails inside the
	// shouldQueueEmailAction branch.
	server.cacheSettings.OfflineQueueEnabled = false

	req := httptest.NewRequest(http.MethodDelete, "/api/emails/email-1", nil)
	w := httptest.NewRecorder()
	server.handleDeleteEmail(w, req, "email-1")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s, want 500 (queue-write-fails branch)", w.Code, w.Body.String())
	}
	var resp UpdateEmailResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Success {
		t.Errorf("response Success=true on double-failure, expected false")
	}
	if resp.Error == "" {
		t.Error("response Error should describe the failure, got empty")
	}
}

// transientNetErr implements net.Error with Timeout()=true so the
// handler classifies it as "queue this and try again later" rather
// than a permanent 4xx.
type transientNetErr struct{}

func (transientNetErr) Error() string   { return "simulated transient network error" }
func (transientNetErr) Timeout() bool   { return true }
func (transientNetErr) Temporary() bool { return true }

// Compile-time assertion: transientNetErr satisfies net.Error.
var _ net.Error = transientNetErr{}
