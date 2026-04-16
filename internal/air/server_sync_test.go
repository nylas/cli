//go:build !integration

package air

import (
	"sync"
	"testing"
	"time"

	"github.com/nylas/cli/internal/air/cache"
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
	server.syncEmails(t.Context(), "test@example.com", "grant-123")
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

// Note: startBackgroundSync requires a valid grantStore and will panic if nil.
// This is intentional - the server should always be properly initialized.
// A test for empty grants list would require a mock grant store.

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

	// Add to wait group before starting
	server.syncWg.Add(1)

	// Start the loop in a goroutine
	done := make(chan struct{})
	go func() {
		server.syncAccountLoop(stopCh, "test@example.com", "grant-123")
		close(done)
	}()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Signal stop
	close(stopCh)

	// Wait for it to finish with timeout
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("syncAccountLoop did not stop in time")
	}

	// Verify wait group is decremented
	server.syncWg.Wait()
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
