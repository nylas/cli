package air

import (
	"context"
	"database/sql"
	"time"

	"github.com/nylas/cli/internal/air/cache"
)

// startBackgroundSync starts background sync goroutines for all accounts.
func (s *Server) startBackgroundSync() {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()

	if s.syncRunning || s.demoMode || s.grantStore == nil || s.cacheSettings == nil || !s.cacheSettings.IsCacheEnabled() || !s.hasCacheRuntime() {
		return
	}

	grants, err := s.grantStore.ListGrants()
	if err != nil || len(grants) == 0 {
		return
	}

	stopCh := make(chan struct{})
	started := 0
	for _, grant := range grants {
		if !grant.Provider.IsSupportedByAir() {
			continue
		}

		s.syncWg.Add(1)
		started++
		go s.syncAccountLoop(stopCh, grant.Email, grant.ID)
	}

	if started == 0 {
		return
	}

	s.syncStopCh = stopCh
	s.syncRunning = true
}

// stopBackgroundSync stops background sync goroutines and waits for them to exit.
func (s *Server) stopBackgroundSync() {
	s.syncMu.Lock()
	if !s.syncRunning {
		s.syncMu.Unlock()
		return
	}

	stopCh := s.syncStopCh
	s.syncStopCh = nil
	s.syncRunning = false
	s.syncMu.Unlock()

	if stopCh != nil {
		close(stopCh)
	}
	s.syncWg.Wait()
}

// restartBackgroundSync reapplies background sync settings immediately.
func (s *Server) restartBackgroundSync() {
	s.stopBackgroundSync()
	s.startBackgroundSync()
}

// syncAccountLoop runs the sync loop for a single account.
func (s *Server) syncAccountLoop(stopCh <-chan struct{}, email, grantID string) {
	defer s.syncWg.Done()

	interval := s.cacheSettings.GetSyncInterval()
	if interval < time.Minute {
		interval = 5 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial sync
	s.syncAccount(email, grantID)

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			s.syncAccount(email, grantID)
		}
	}
}

// syncAccount syncs a single account's data from the API.
func (s *Server) syncAccount(email, grantID string) {
	if s.nylasClient == nil || !s.IsOnline() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Sync emails
	s.syncEmails(ctx, email, grantID)

	// Sync folders
	s.syncFolders(ctx, email, grantID)

	// Sync events
	s.syncEvents(ctx, email, grantID)

	// Sync contacts
	s.syncContacts(ctx, email, grantID)
}

// syncEmails syncs emails from the API to cache.
func (s *Server) syncEmails(ctx context.Context, email, grantID string) {
	_ = s.withAccountDB(email, func(db *sql.DB) error {
		store := cache.NewEmailStore(db)
		syncStore := cache.NewSyncStore(db)

		// Get last sync state
		state, _ := syncStore.Get("emails")
		if state == nil {
			state = &cache.SyncState{Resource: "emails"}
		}

		// Fetch emails from API
		messages, err := s.nylasClient.GetMessages(ctx, grantID, 100)
		if err != nil {
			s.SetOnline(false)
			return nil
		}
		s.SetOnline(true)

		// Cache emails
		for i := range messages {
			cached := domainMessageToCached(&messages[i])
			_ = store.Put(cached)
		}

		// Update sync state
		state.LastSync = time.Now()
		_ = syncStore.Set(state)
		return nil
	})
}

// syncFolders syncs folders from the API to cache.
func (s *Server) syncFolders(ctx context.Context, email, grantID string) {
	_ = s.withFolderStore(email, func(store *cache.FolderStore) error {
		folders, err := s.nylasClient.GetFolders(ctx, grantID)
		if err != nil {
			return nil
		}

		for i := range folders {
			f := &folders[i]
			cached := &cache.CachedFolder{
				ID:          f.ID,
				Name:        f.Name,
				Type:        f.SystemFolder,
				TotalCount:  f.TotalCount,
				UnreadCount: f.UnreadCount,
				CachedAt:    time.Now(),
			}
			_ = store.Put(cached)
		}

		return nil
	})
}

// syncEvents syncs calendar events from the API to cache.
func (s *Server) syncEvents(ctx context.Context, email, grantID string) {
	_ = s.withEventStore(email, func(store *cache.EventStore) error {
		calendars, err := s.nylasClient.GetCalendars(ctx, grantID)
		if err != nil {
			return nil
		}

		for i := range calendars {
			cal := &calendars[i]
			events, err := s.nylasClient.GetEvents(ctx, grantID, cal.ID, nil)
			if err != nil {
				continue
			}

			for j := range events {
				cached := domainEventToCached(&events[j], cal.ID)
				_ = store.Put(cached)
			}
		}

		return nil
	})
}

// syncContacts syncs contacts from the API to cache.
func (s *Server) syncContacts(ctx context.Context, email, grantID string) {
	_ = s.withContactStore(email, func(store *cache.ContactStore) error {
		contacts, err := s.nylasClient.GetContacts(ctx, grantID, nil)
		if err != nil {
			return nil
		}

		for i := range contacts {
			cached := domainContactToCached(&contacts[i])
			_ = store.Put(cached)
		}
		return nil
	})
}
