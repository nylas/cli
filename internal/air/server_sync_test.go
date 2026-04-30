//go:build !integration

package air

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nylas/cli/internal/air/cache"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestSyncAccount_NilClient(t *testing.T) {
	t.Parallel()

	// Server with nil nylasClient should return early from syncAccount
	server := &Server{
		nylasClient: nil,
		isOnline:    true,
		onlineMu:    sync.RWMutex{},
	}

	// This should not panic and should return early
	server.syncAccount("test@example.com", "grant-123")

	// No error expected, just verify it doesn't panic
}

func TestSyncAccount_Offline(t *testing.T) {
	t.Parallel()

	// Server that is offline should return early from syncAccount
	server := &Server{
		nylasClient: nil, // Would be set in real scenario
		isOnline:    false,
		onlineMu:    sync.RWMutex{},
	}

	// This should return early because IsOnline() returns false
	server.syncAccount("test@example.com", "grant-123")

	// No error expected, just verify it doesn't panic
}

func TestSyncEmails_NoCacheManager(t *testing.T) {
	t.Parallel()

	// Server with nil cacheManager should return early
	server := &Server{
		cacheManager: nil,
		isOnline:     true,
		onlineMu:     sync.RWMutex{},
	}

	// This should not panic
	server.syncEmails(t.Context(), "test@example.com", "grant-123", nil)
}

func TestSyncFolders_NoCacheManager(t *testing.T) {
	t.Parallel()

	// Server with nil cacheManager should return early
	server := &Server{
		cacheManager: nil,
		isOnline:     true,
		onlineMu:     sync.RWMutex{},
	}

	// This should not panic
	server.syncFolders(t.Context(), "test@example.com", "grant-123")
}

func TestSyncEvents_NoCacheManager(t *testing.T) {
	t.Parallel()

	// Server with nil cacheManager should return early
	server := &Server{
		cacheManager: nil,
		isOnline:     true,
		onlineMu:     sync.RWMutex{},
	}

	// This should not panic
	server.syncEvents(t.Context(), "test@example.com", "grant-123")
}

// Pins the per-system-folder hydration: when syncFolders returns folders
// with system types (inbox/sent/drafts/etc.), syncEmails must call
// GetMessagesWithParams once per primary folder, in priority order, and
// must not fall back to the unfiltered GetMessages path. Together with the
// frontend eager refresh this is what gives non-Inbox folders correct cache
// coverage on first paint and offline.
func TestSyncEmails_PerFolderHydration(t *testing.T) {
	t.Parallel()

	server, client, accountEmail := newCachedTestServer(t)

	folders := []domain.Folder{
		{ID: "TRASH", Name: "Trash", SystemFolder: "trash"},
		{ID: "INBOX", Name: "Inbox", SystemFolder: "inbox"},
		{ID: "LABEL_42", Name: "Custom", SystemFolder: ""},
		{ID: "SENT", Name: "Sent", SystemFolder: "sent"},
		{ID: "DRAFTS", Name: "Drafts", SystemFolder: "drafts"},
	}

	var mu sync.Mutex
	calls := []string{}
	client.GetMessagesWithParamsFunc = func(_ context.Context, _ string, p *domain.MessageQueryParams) ([]domain.Message, error) {
		mu.Lock()
		if len(p.In) > 0 {
			calls = append(calls, p.In[0])
		}
		mu.Unlock()
		// Return one message per folder so we can also assert the cache
		// got hydrated.
		return []domain.Message{{
			ID:      "msg-" + p.In[0],
			Subject: "Hi from " + p.In[0],
			Folders: []string{p.In[0]},
			Date:    time.Now(),
		}}, nil
	}
	client.GetMessagesFunc = func(context.Context, string, int) ([]domain.Message, error) {
		t.Fatal("unfiltered GetMessages must not be called when system folders are available")
		return nil, nil
	}

	server.syncEmails(t.Context(), accountEmail, "grant-123", folders)

	mu.Lock()
	got := append([]string(nil), calls...)
	mu.Unlock()

	// Priority order: inbox, sent, drafts, archive, trash, spam. Custom
	// label must be skipped. We only declared four primary folders.
	want := []string{"INBOX", "SENT", "DRAFTS", "TRASH"}
	assert.Equal(t, want, got)

	// The cache should now hold one message per primary folder.
	store, err := server.getEmailStore(accountEmail)
	if err != nil {
		t.Fatalf("get email store: %v", err)
	}
	for _, fid := range want {
		got, err := store.Get("msg-" + fid)
		if err != nil || got == nil {
			t.Fatalf("expected message msg-%s in cache, got err=%v", fid, err)
		}
	}
}

// Pins the fallback: when no folders are available (folder-API outage),
// syncEmails must still hydrate the cache using the unfiltered top-N fetch
// — losing folder coverage but keeping forward progress.
func TestSyncEmails_FallsBackWhenFoldersEmpty(t *testing.T) {
	t.Parallel()

	server, client, accountEmail := newCachedTestServer(t)

	called := false
	client.GetMessagesFunc = func(context.Context, string, int) ([]domain.Message, error) {
		called = true
		return []domain.Message{{ID: "msg-fallback", Date: time.Now()}}, nil
	}
	client.GetMessagesWithParamsFunc = func(context.Context, string, *domain.MessageQueryParams) ([]domain.Message, error) {
		t.Fatal("per-folder fetch must not run when folder list is empty")
		return nil, nil
	}

	server.syncEmails(t.Context(), accountEmail, "grant-123", nil)

	if !called {
		t.Fatal("expected fallback GetMessages call")
	}
	store, err := server.getEmailStore(accountEmail)
	if err != nil {
		t.Fatalf("get email store: %v", err)
	}
	got, err := store.Get("msg-fallback")
	if err != nil || got == nil {
		t.Fatalf("expected fallback message in cache, err=%v", err)
	}
}

// Pins primarySystemFolderIDs ordering and dedup directly so the priority
// contract is regression-protected even if the calling code changes.
func TestPrimarySystemFolderIDs_OrdersAndFilters(t *testing.T) {
	t.Parallel()

	folders := []domain.Folder{
		{ID: "SPAM", SystemFolder: "spam"},
		{ID: "LABEL_FOO", SystemFolder: ""},
		{ID: "INBOX", SystemFolder: "INBOX"}, // upper-case to verify normalization
		{ID: "INBOX", SystemFolder: "inbox"}, // duplicate, skip
		{ID: "ARCHIVE", SystemFolder: "archive"},
		{ID: "SENT", SystemFolder: " sent "}, // padded, verify trim
	}

	got := primarySystemFolderIDs(folders)
	assert.Equal(t, []string{"INBOX", "SENT", "ARCHIVE", "SPAM"}, got)
}

func TestPrimarySystemFolderIDs_EmptyInput(t *testing.T) {
	t.Parallel()
	if got := primarySystemFolderIDs(nil); got != nil {
		t.Fatalf("expected nil for nil input, got %v", got)
	}
	if got := primarySystemFolderIDs([]domain.Folder{}); got != nil {
		t.Fatalf("expected nil for empty input, got %v", got)
	}
}

func TestSyncContacts_NoCacheManager(t *testing.T) {
	t.Parallel()

	// Server with nil cacheManager should return early
	server := &Server{
		cacheManager: nil,
		isOnline:     true,
		onlineMu:     sync.RWMutex{},
	}

	// This should not panic
	server.syncContacts(t.Context(), "test@example.com", "grant-123")
}

func TestStartBackgroundSyncOnlyStartsDefaultGrant(t *testing.T) {
	t.Parallel()

	server, client, _ := newCachedTestServer(t)
	grants := []domain.GrantInfo{
		{ID: "grant-other-1", Email: "other-1@example.com", Provider: domain.ProviderGoogle},
		{ID: "grant-default", Email: "default@example.com", Provider: domain.ProviderGoogle},
		{ID: "grant-other-2", Email: "other-2@example.com", Provider: domain.ProviderNylas},
	}
	server.grantStore = &testGrantStore{
		grants:       grants,
		defaultGrant: "grant-default",
	}
	server.SetOnline(true)

	var mu sync.Mutex
	calledGrantIDs := []string{}
	client.GetMessagesFunc = func(_ context.Context, grantID string, _ int) ([]domain.Message, error) {
		mu.Lock()
		calledGrantIDs = append(calledGrantIDs, grantID)
		mu.Unlock()
		return []domain.Message{}, nil
	}
	client.GetFoldersFunc = func(_ context.Context, _ string) ([]domain.Folder, error) {
		return []domain.Folder{}, nil
	}
	client.GetCalendarsFunc = func(_ context.Context, _ string) ([]domain.Calendar, error) {
		return []domain.Calendar{}, nil
	}
	client.GetContactsFunc = func(_ context.Context, _ string, _ *domain.ContactQueryParams) ([]domain.Contact, error) {
		return []domain.Contact{}, nil
	}

	server.startBackgroundSync()
	t.Cleanup(server.stopBackgroundSync)

	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(calledGrantIDs) >= 1
	}, time.Second, 10*time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	got := append([]string(nil), calledGrantIDs...)
	mu.Unlock()

	assert.Equal(t, []string{"grant-default"}, got)
}

func TestInitializeOfflineQueuesLockedOnlyOpensDefaultGrant(t *testing.T) {
	t.Parallel()

	server, _, _ := newCachedTestServer(t)
	grants := []domain.GrantInfo{
		{ID: "grant-other-1", Email: "other-1@example.com", Provider: domain.ProviderGoogle},
		{ID: "grant-default", Email: "default@example.com", Provider: domain.ProviderGoogle},
		{ID: "grant-other-2", Email: "other-2@example.com", Provider: domain.ProviderNylas},
	}
	server.grantStore = &testGrantStore{
		grants:       grants,
		defaultGrant: "grant-default",
	}
	server.offlineQueues = make(map[string]*cache.OfflineQueue)

	for _, grant := range grants {
		if _, err := server.cacheManager.GetDB(grant.Email); err != nil {
			t.Fatalf("create cached db for %s: %v", grant.Email, err)
		}
	}

	if err := server.initializeOfflineQueuesLocked(); err != nil {
		t.Fatalf("initialize offline queues: %v", err)
	}

	if len(server.offlineQueues) != 1 {
		t.Fatalf("expected exactly one offline queue, got %d", len(server.offlineQueues))
	}
	if server.offlineQueues["default@example.com"] == nil {
		t.Fatal("expected default account offline queue to be initialized")
	}
	if server.offlineQueues["other-1@example.com"] != nil || server.offlineQueues["other-2@example.com"] != nil {
		t.Fatal("did not expect non-default account offline queues to be initialized")
	}
}

func TestSyncAccountLoop_StopsOnChannel(t *testing.T) {
	t.Parallel()

	// Create a server with a stop channel
	stopCh := make(chan struct{})
	server := &Server{
		nylasClient:   nil,
		isOnline:      false,
		onlineMu:      sync.RWMutex{},
		syncWg:        sync.WaitGroup{},
		syncStopCh:    stopCh,
		cacheSettings: cache.DefaultSettings(),
	}

	// Spawn via WaitGroup.Go to mirror production. WaitGroup.Go handles
	// Add/Done for us, so we no longer call Add manually here.
	server.syncWg.Go(func() {
		server.syncAccountLoop(stopCh, "test@example.com", "grant-123")
	})

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Signal stop
	close(stopCh)

	// Wait for it to finish with timeout via a sentinel goroutine.
	done := make(chan struct{})
	go func() {
		server.syncWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("syncAccountLoop did not stop in time")
	}
}

func TestSyncAccountLoop_MinInterval(t *testing.T) {
	t.Parallel()

	// Test that sync interval has a minimum from DefaultSettings
	settings := cache.DefaultSettings()

	// Verify the default interval is reasonable
	interval := settings.GetSyncInterval()

	// Default should be at least 1 minute (the minimum enforced in syncAccountLoop)
	assert.GreaterOrEqual(t, int64(interval), int64(time.Minute))
}

func TestServer_SyncWaitGroup_Usage(t *testing.T) {
	t.Parallel()

	// Verify that syncWg can be used correctly
	server := &Server{
		syncWg: sync.WaitGroup{},
	}

	// Add and Done should work without panic
	server.syncWg.Add(1)
	server.syncWg.Done()

	// Wait should not block
	done := make(chan struct{})
	go func() {
		server.syncWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Error("Wait() blocked unexpectedly")
	}
}

func TestServer_SyncLifecycle_NoWorkers(t *testing.T) {
	t.Parallel()

	server := &Server{}
	server.stopBackgroundSync()
	server.restartBackgroundSync()
}

func TestRestartBackgroundSync_ReplacesStopChannel(t *testing.T) {
	t.Parallel()

	server, _, _ := newCachedTestServer(t)
	server.nylasClient = nil
	server.SetOnline(false)
	server.startBackgroundSync()
	t.Cleanup(func() {
		server.stopBackgroundSync()
	})

	if !server.syncRunning {
		t.Fatal("expected sync workers to be running")
	}

	initialStopCh := server.syncStopCh
	server.restartBackgroundSync()

	if !server.syncRunning {
		t.Fatal("expected sync workers to remain running after restart")
	}
	if server.syncStopCh == nil {
		t.Fatal("expected restarted sync workers to have a stop channel")
	}
	if server.syncStopCh == initialStopCh {
		t.Fatal("expected restart to replace the stop channel")
	}
	select {
	case <-initialStopCh:
	default:
		t.Fatal("expected restart to stop the previous workers")
	}
}
