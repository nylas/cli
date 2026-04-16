package air

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	nylasmock "github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/air/cache"
	"github.com/nylas/cli/internal/domain"
)

type testGrantStore struct {
	grants       []domain.GrantInfo
	defaultGrant string
}

type failingCacheRuntimeManager struct {
	closeErr error
}

func (m *failingCacheRuntimeManager) GetDB(string) (*sql.DB, error)              { return nil, nil }
func (m *failingCacheRuntimeManager) Close() error                               { return m.closeErr }
func (m *failingCacheRuntimeManager) ClearCache(string) error                    { return nil }
func (m *failingCacheRuntimeManager) ClearAllCaches() error                      { return nil }
func (m *failingCacheRuntimeManager) ListCachedAccounts() ([]string, error)      { return nil, nil }
func (m *failingCacheRuntimeManager) GetStats(string) (*cache.CacheStats, error) { return nil, nil }
func (m *failingCacheRuntimeManager) DBPath(string) string                       { return "" }

func (s *testGrantStore) SaveGrant(info domain.GrantInfo) error {
	s.grants = append(s.grants, info)
	return nil
}

func (s *testGrantStore) GetGrant(grantID string) (*domain.GrantInfo, error) {
	for i := range s.grants {
		if s.grants[i].ID == grantID {
			return &s.grants[i], nil
		}
	}
	return nil, domain.ErrGrantNotFound
}

func (s *testGrantStore) GetGrantByEmail(email string) (*domain.GrantInfo, error) {
	for i := range s.grants {
		if s.grants[i].Email == email {
			return &s.grants[i], nil
		}
	}
	return nil, domain.ErrGrantNotFound
}

func (s *testGrantStore) ListGrants() ([]domain.GrantInfo, error) { return s.grants, nil }
func (s *testGrantStore) DeleteGrant(grantID string) error        { return nil }
func (s *testGrantStore) SetDefaultGrant(grantID string) error {
	s.defaultGrant = grantID
	return nil
}
func (s *testGrantStore) GetDefaultGrant() (string, error) { return s.defaultGrant, nil }
func (s *testGrantStore) ClearGrants() error {
	s.grants = nil
	s.defaultGrant = ""
	return nil
}

func newCachedTestServer(t *testing.T) (*Server, *nylasmock.MockClient, string) {
	t.Helper()

	tmpDir := t.TempDir()
	manager, err := cache.NewManager(cache.Config{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("new cache manager: %v", err)
	}

	settings := cache.DefaultSettings()
	settings.Enabled = true
	settings.OfflineQueueEnabled = true

	email := "user@example.com"
	grantID := "grant-123"
	client := nylasmock.NewMockClient()

	server := &Server{
		cacheManager:  manager,
		cacheSettings: settings,
		grantStore: &testGrantStore{
			grants: []domain.GrantInfo{{
				ID:       grantID,
				Email:    email,
				Provider: domain.ProviderGoogle,
			}},
			defaultGrant: grantID,
		},
		nylasClient:   client,
		offlineQueues: make(map[string]*cache.OfflineQueue),
		syncStopCh:    make(chan struct{}),
		isOnline:      true,
	}

	t.Cleanup(func() {
		_ = manager.Close()
	})

	return server, client, email
}

func putCachedEmail(t *testing.T, server *Server, accountEmail string, email *cache.CachedEmail) {
	t.Helper()

	store, err := server.getEmailStore(accountEmail)
	if err != nil {
		t.Fatalf("get email store: %v", err)
	}
	if err := store.Put(email); err != nil {
		t.Fatalf("put cached email: %v", err)
	}
}

func TestHandleListEmails_UsesCacheFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		query  string
		wantID string
	}{
		{name: "from filter", query: "/api/emails?from=alice@example.com", wantID: "email-alice"},
		{name: "search query", query: "/api/emails?search=Quarterly", wantID: "email-alice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server, client, accountEmail := newCachedTestServer(t)
			client.GetMessagesWithParamsFunc = func(_ context.Context, _ string, _ *domain.MessageQueryParams) ([]domain.Message, error) {
				t.Fatal("expected cache hit without API request")
				return nil, nil
			}

			putCachedEmail(t, server, accountEmail, &cache.CachedEmail{
				ID:        "email-alice",
				FolderID:  "inbox",
				Subject:   "Quarterly planning",
				Snippet:   "Q2 planning notes",
				FromName:  "Alice",
				FromEmail: "alice@example.com",
				Date:      time.Now(),
				Unread:    true,
				CachedAt:  time.Now(),
			})
			putCachedEmail(t, server, accountEmail, &cache.CachedEmail{
				ID:        "email-bob",
				FolderID:  "inbox",
				Subject:   "Budget review",
				Snippet:   "FYI",
				FromName:  "Bob",
				FromEmail: "bob@example.com",
				Date:      time.Now().Add(-time.Minute),
				CachedAt:  time.Now(),
			})

			req := httptest.NewRequest(http.MethodGet, tt.query, nil)
			w := httptest.NewRecorder()

			server.handleListEmails(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", w.Code)
			}

			var resp EmailsResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if len(resp.Emails) != 1 {
				t.Fatalf("expected 1 email, got %d", len(resp.Emails))
			}
			if resp.Emails[0].ID != tt.wantID {
				t.Fatalf("expected %s, got %s", tt.wantID, resp.Emails[0].ID)
			}
		})
	}
}

func TestHandleUpdateEmail_UpdatesCacheOnSuccess(t *testing.T) {
	t.Parallel()

	server, client, accountEmail := newCachedTestServer(t)
	putCachedEmail(t, server, accountEmail, &cache.CachedEmail{
		ID:        "email-1",
		FolderID:  "inbox",
		Subject:   "Hello",
		FromEmail: "sender@example.com",
		Date:      time.Now(),
		Unread:    true,
		Starred:   false,
		CachedAt:  time.Now(),
	})

	reqBody := bytes.NewBufferString(`{"unread":false,"starred":true,"folders":["archive"]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/emails/email-1", reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleUpdateEmail(w, req, "email-1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if !client.UpdateMessageCalled {
		t.Fatal("expected UpdateMessage to be called")
	}

	store, err := server.getEmailStore(accountEmail)
	if err != nil {
		t.Fatalf("get email store: %v", err)
	}
	cached, err := store.Get("email-1")
	if err != nil {
		t.Fatalf("get cached email: %v", err)
	}

	if cached.Unread {
		t.Fatal("expected cached email to be marked read")
	}
	if !cached.Starred {
		t.Fatal("expected cached email to be starred")
	}
	if cached.FolderID != "archive" {
		t.Fatalf("expected folder archive, got %s", cached.FolderID)
	}
}

func TestHandleDeleteEmail_QueuesOfflineAndRemovesCachedEmail(t *testing.T) {
	t.Parallel()

	server, client, accountEmail := newCachedTestServer(t)
	server.SetOnline(false)
	putCachedEmail(t, server, accountEmail, &cache.CachedEmail{
		ID:        "email-1",
		FolderID:  "inbox",
		Subject:   "Hello",
		FromEmail: "sender@example.com",
		Date:      time.Now(),
		CachedAt:  time.Now(),
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/emails/email-1", nil)
	w := httptest.NewRecorder()

	server.handleDeleteEmail(w, req, "email-1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if client.DeleteMessageCalled {
		t.Fatal("did not expect DeleteMessage API call while offline")
	}

	queue, err := server.getOfflineQueue(accountEmail)
	if err != nil {
		t.Fatalf("get offline queue: %v", err)
	}
	count, err := queue.Count()
	if err != nil {
		t.Fatalf("count offline queue: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 queued action, got %d", count)
	}

	store, err := server.getEmailStore(accountEmail)
	if err != nil {
		t.Fatalf("get email store: %v", err)
	}
	if _, err := store.Get("email-1"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected cached email to be removed, got err=%v", err)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/cache/status", nil)
	statusRes := httptest.NewRecorder()
	server.handleCacheStatus(statusRes, statusReq)

	var status CacheStatusResponse
	if err := json.NewDecoder(statusRes.Body).Decode(&status); err != nil {
		t.Fatalf("decode cache status: %v", err)
	}
	if status.PendingActions != 1 {
		t.Fatalf("expected 1 pending action, got %d", status.PendingActions)
	}
}

func TestHandleDeleteEmail_RemovesCachedEmailOnSuccess(t *testing.T) {
	t.Parallel()

	server, client, accountEmail := newCachedTestServer(t)
	putCachedEmail(t, server, accountEmail, &cache.CachedEmail{
		ID:        "email-1",
		FolderID:  "inbox",
		Subject:   "Hello",
		FromEmail: "sender@example.com",
		Date:      time.Now(),
		CachedAt:  time.Now(),
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/emails/email-1", nil)
	w := httptest.NewRecorder()

	server.handleDeleteEmail(w, req, "email-1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if !client.DeleteMessageCalled {
		t.Fatal("expected DeleteMessage API call")
	}

	store, err := server.getEmailStore(accountEmail)
	if err != nil {
		t.Fatalf("get email store: %v", err)
	}
	if _, err := store.Get("email-1"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected cached email to be removed, got err=%v", err)
	}
}

func TestInitCacheRuntime_UsesEncryptedManagerWhenEnabled(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	settings, err := cache.LoadSettings(tmpDir)
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if err := settings.Update(func(s *cache.Settings) {
		s.Enabled = true
		s.EncryptionEnabled = true
	}); err != nil {
		t.Fatalf("update settings: %v", err)
	}

	server := &Server{
		cacheSettings: settings,
		offlineQueues: make(map[string]*cache.OfflineQueue),
		syncStopCh:    make(chan struct{}),
		isOnline:      true,
	}

	server.initCacheRuntime()

	if server.cacheManager == nil {
		t.Fatal("expected cache manager to be initialized")
	}
	if _, ok := server.cacheManager.(*cache.EncryptedManager); !ok {
		t.Fatalf("expected encrypted cache manager, got %T", server.cacheManager)
	}
}

func TestReconfigureCacheRuntime_WaitsForInFlightCacheAccess(t *testing.T) {
	t.Parallel()

	server, _, accountEmail := newCachedTestServer(t)
	server.nylasClient = nil
	server.SetOnline(false)
	putCachedEmail(t, server, accountEmail, &cache.CachedEmail{
		ID:        "email-1",
		FolderID:  "inbox",
		Subject:   "Hello",
		FromEmail: "sender@example.com",
		Date:      time.Now(),
		CachedAt:  time.Now(),
	})

	entered := make(chan struct{})
	release := make(chan struct{})
	accessDone := make(chan error, 1)
	go func() {
		accessDone <- server.withEmailStore(accountEmail, func(store *cache.EmailStore) error {
			close(entered)
			<-release
			_, err := store.Get("email-1")
			return err
		})
	}()

	<-entered

	reconfigureDone := make(chan error, 1)
	go func() {
		reconfigureDone <- server.reconfigureCacheRuntime()
	}()

	select {
	case err := <-reconfigureDone:
		t.Fatalf("reconfigure returned before in-flight access completed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)

	if err := <-accessDone; err != nil {
		t.Fatalf("in-flight cache access failed: %v", err)
	}
	if err := <-reconfigureDone; err != nil {
		t.Fatalf("reconfigure cache runtime: %v", err)
	}

	t.Cleanup(func() {
		_ = server.Stop()
	})
}

func TestReconfigureCacheRuntime_UnlocksRuntimeMutexOnCloseError(t *testing.T) {
	t.Parallel()

	settings := cache.DefaultSettings()
	settings.Enabled = true

	server := &Server{
		cacheManager:  &failingCacheRuntimeManager{closeErr: errors.New("close failed")},
		cacheSettings: settings,
		offlineQueues: make(map[string]*cache.OfflineQueue),
		syncStopCh:    make(chan struct{}),
		isOnline:      true,
	}

	if err := server.reconfigureCacheRuntime(); err == nil {
		t.Fatal("expected close failure from reconfigure")
	}

	done := make(chan struct{})
	go func() {
		_ = server.hasCacheRuntime()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("runtime mutex remained locked after reconfigure error")
	}
}

func TestSyncEmails_DoesNotHoldRuntimeLockAcrossFetch(t *testing.T) {
	server, client, accountEmail := newCachedTestServer(t)
	server.SetOnline(true)

	fetchStarted := make(chan struct{})
	releaseFetch := make(chan struct{})
	client.GetMessagesFunc = func(_ context.Context, _ string, _ int) ([]domain.Message, error) {
		close(fetchStarted)
		<-releaseFetch
		return []domain.Message{}, nil
	}

	syncDone := make(chan struct{})
	go func() {
		server.syncEmails(context.Background(), accountEmail, "grant-123")
		close(syncDone)
	}()

	<-fetchStarted

	lockAcquired := make(chan struct{})
	go func() {
		server.runtimeMu.Lock()
		close(lockAcquired)
		server.runtimeMu.Unlock()
	}()

	select {
	case <-lockAcquired:
	case <-time.After(2 * time.Second):
		t.Fatal("runtime lock remained blocked while remote fetch was in progress")
	}

	close(releaseFetch)

	select {
	case <-syncDone:
	case <-time.After(5 * time.Second):
		t.Fatal("syncEmails did not finish after fetch was released")
	}

	t.Cleanup(func() {
		_ = server.Stop()
	})
}

func TestProcessOfflineQueues_UsesQueuedGrantID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		enqueue        func(t *testing.T, server *Server, accountEmail string)
		assertReplayed func(t *testing.T, client *nylasmock.MockClient)
	}{
		{
			name: "delete action",
			enqueue: func(t *testing.T, server *Server, accountEmail string) {
				t.Helper()
				if err := server.enqueueMessageDelete("grant-123", accountEmail, "email-1"); err != nil {
					t.Fatalf("enqueue delete: %v", err)
				}
			},
			assertReplayed: func(t *testing.T, client *nylasmock.MockClient) {
				t.Helper()
				if !client.DeleteMessageCalled {
					t.Fatal("expected DeleteMessage to be replayed")
				}
				if client.LastMessageID != "email-1" {
					t.Fatalf("expected delete to target email-1, got %s", client.LastMessageID)
				}
			},
		},
		{
			name: "update message action",
			enqueue: func(t *testing.T, server *Server, accountEmail string) {
				t.Helper()
				unread := false
				starred := true
				if err := server.enqueueMessageUpdate("grant-123", accountEmail, "email-2", &domain.UpdateMessageRequest{
					Unread:  &unread,
					Starred: &starred,
					Folders: []string{"archive"},
				}); err != nil {
					t.Fatalf("enqueue update: %v", err)
				}
			},
			assertReplayed: func(t *testing.T, client *nylasmock.MockClient) {
				t.Helper()
				if !client.UpdateMessageCalled {
					t.Fatal("expected UpdateMessage to be replayed")
				}
				if client.LastMessageID != "email-2" {
					t.Fatalf("expected update to target email-2, got %s", client.LastMessageID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server, client, accountEmail := newCachedTestServer(t)
			server.SetOnline(false)

			tt.enqueue(t, server, accountEmail)

			grantStore := server.grantStore.(*testGrantStore)
			grantStore.grants = []domain.GrantInfo{
				{
					ID:       "grant-other",
					Email:    accountEmail,
					Provider: domain.ProviderGoogle,
				},
				{
					ID:       "grant-123",
					Email:    accountEmail,
					Provider: domain.ProviderGoogle,
				},
			}

			server.SetOnline(true)

			tt.assertReplayed(t, client)
			if client.LastGrantID != "grant-123" {
				t.Fatalf("expected replay to use queued grant grant-123, got %s", client.LastGrantID)
			}

			queue, err := server.getOfflineQueue(accountEmail)
			if err != nil {
				t.Fatalf("get offline queue: %v", err)
			}
			count, err := queue.Count()
			if err != nil {
				t.Fatalf("queue count: %v", err)
			}
			if count != 0 {
				t.Fatalf("expected queue to be drained, got %d pending action(s)", count)
			}
		})
	}
}
