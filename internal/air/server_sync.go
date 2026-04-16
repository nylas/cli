package air

import (
	"context"
	"database/sql"
	"time"

	"github.com/nylas/cli/internal/air/cache"
	"github.com/nylas/cli/internal/domain"
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
	if s.nylasClient == nil || !s.hasCacheRuntime() {
		return
	}

	messages, err := s.nylasClient.GetMessages(ctx, grantID, 100)
	if err != nil {
		s.SetOnline(false)
		return
	}
	s.SetOnline(true)

	_ = s.withAccountDB(email, func(db *sql.DB) error {
		store := cache.NewEmailStore(db)
		syncStore := cache.NewSyncStore(db)

		for i := range messages {
			cached := domainMessageToCached(&messages[i])
			_ = store.Put(cached)
		}

		state, _ := syncStore.Get("emails")
		if state == nil {
			state = &cache.SyncState{Resource: "emails"}
		}
		state.LastSync = time.Now()
		_ = syncStore.Set(state)
		return nil
	})
}

// syncFolders syncs folders from the API to cache.
func (s *Server) syncFolders(ctx context.Context, email, grantID string) {
	if s.nylasClient == nil || !s.hasCacheRuntime() {
		return
	}

	folders, err := s.nylasClient.GetFolders(ctx, grantID)
	if err != nil {
		return
	}

	_ = s.withFolderStore(email, func(store *cache.FolderStore) error {
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
	if s.nylasClient == nil || !s.hasCacheRuntime() {
		return
	}

	calendars, err := s.nylasClient.GetCalendars(ctx, grantID)
	if err != nil {
		return
	}

	type calendarEvents struct {
		calendarID string
		events     []domain.Event
	}
	eventGroups := make([]calendarEvents, 0, len(calendars))

	for i := range calendars {
		cal := &calendars[i]
		events, err := s.nylasClient.GetEvents(ctx, grantID, cal.ID, nil)
		if err != nil {
			continue
		}
		eventGroups = append(eventGroups, calendarEvents{
			calendarID: cal.ID,
			events:     events,
		})
	}

	_ = s.withEventStore(email, func(store *cache.EventStore) error {
		for _, group := range eventGroups {
			for i := range group.events {
				cached := domainEventToCached(&group.events[i], group.calendarID)
				_ = store.Put(cached)
			}
		}
		return nil
	})
}

// syncContacts syncs contacts from the API to cache.
func (s *Server) syncContacts(ctx context.Context, email, grantID string) {
	if s.nylasClient == nil || !s.hasCacheRuntime() {
		return
	}

	contacts, err := s.nylasClient.GetContacts(ctx, grantID, nil)
	if err != nil {
		return
	}

	_ = s.withContactStore(email, func(store *cache.ContactStore) error {
		for i := range contacts {
			cached := domainContactToCached(&contacts[i])
			_ = store.Put(cached)
		}
		return nil
	})
}
