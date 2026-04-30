package air

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/nylas/cli/internal/air/cache"
	"github.com/nylas/cli/internal/domain"
)

// startBackgroundSync starts background sync for the active account.
func (s *Server) startBackgroundSync() {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()

	if s.syncRunning || s.demoMode || s.grantStore == nil || s.cacheSettings == nil || !s.cacheSettings.IsCacheEnabled() || !s.hasCacheRuntime() {
		return
	}

	grant, err := s.resolveDefaultGrantInfo()
	if err != nil {
		return
	}

	stopCh := make(chan struct{})
	s.syncWg.Go(func() {
		s.syncAccountLoop(stopCh, grant.Email, grant.ID)
	})
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
	defer recoverSyncPanic(email)

	interval := s.cacheSettings.GetSyncInterval()
	if interval < time.Minute {
		interval = 5 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial sync
	s.runSyncIteration(email, grantID)

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			s.runSyncIteration(email, grantID)
		}
	}
}

// runSyncIteration calls syncAccount with panic isolation so a single
// failure does not kill the loop or the process.
func (s *Server) runSyncIteration(email, grantID string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Air sync iteration panic for %s: %v\n%s\n", email, r, debug.Stack())
		}
	}()
	s.syncAccount(email, grantID)
}

// recoverSyncPanic logs panics that escape the sync loop. The deferred
// syncWg.Done() still fires because it's registered first.
func recoverSyncPanic(email string) {
	if r := recover(); r != nil {
		fmt.Fprintf(os.Stderr, "Air sync loop panic for %s: %v\n%s\n", email, r, debug.Stack())
	}
}

// syncAccount syncs a single account's data from the API.
func (s *Server) syncAccount(email, grantID string) {
	if s.nylasClient == nil || !s.IsOnline() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Folders first so syncEmails can fan out per system folder. Without
	// this, the unfiltered top-N email fetch is dominated by Inbox on busy
	// accounts and Sent/Drafts/Archive arrive in cache only by accident —
	// the sidebar then shows correct counts only after the user clicks each
	// folder, and offline mode can't render them at all.
	folders := s.syncFolders(ctx, email, grantID)

	s.syncEmails(ctx, email, grantID, folders)

	s.syncEvents(ctx, email, grantID)

	s.syncContacts(ctx, email, grantID)
}

// syncEmails hydrates the email cache from the API. When a folder list is
// available, fetches the top-N messages for each primary system folder
// (Inbox/Sent/Drafts/Archive/Trash/Spam) so each gets representative
// coverage. Falls back to a single unfiltered top-N fetch when the folder
// API returned nothing — keeps prior behavior on folder-API outages.
func (s *Server) syncEmails(ctx context.Context, email, grantID string, folders []domain.Folder) {
	if s.nylasClient == nil || !s.hasCacheRuntime() {
		return
	}

	const perFolderLimit = 50

	targets := primarySystemFolderIDs(folders)
	var messages []domain.Message

	if len(targets) == 0 {
		msgs, err := s.nylasClient.GetMessages(ctx, grantID, 100)
		if err != nil {
			s.SetOnline(false)
			return
		}
		s.SetOnline(true)
		messages = msgs
	} else {
		// Mark online optimistically — we'll flip it back to offline if
		// every folder fetch fails. A single folder failing (e.g. label
		// rename in flight, transient rate-limit) shouldn't kill the whole
		// sync iteration.
		s.SetOnline(true)
		successes := 0
		for _, fid := range targets {
			if ctx.Err() != nil {
				break
			}
			params := &domain.MessageQueryParams{Limit: perFolderLimit, In: []string{fid}}
			msgs, err := s.nylasClient.GetMessagesWithParams(ctx, grantID, params)
			if err != nil {
				continue
			}
			successes++
			messages = append(messages, msgs...)
		}
		if successes == 0 {
			s.SetOnline(false)
			return
		}
	}

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

// syncFolders syncs folders from the API to cache and returns the fetched
// list so callers can drive folder-aware sync paths (notably per-folder
// email hydration). Returns nil on failure — callers must tolerate that.
func (s *Server) syncFolders(ctx context.Context, email, grantID string) []domain.Folder {
	if s.nylasClient == nil || !s.hasCacheRuntime() {
		return nil
	}

	folders, err := s.nylasClient.GetFolders(ctx, grantID)
	if err != nil {
		return nil
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
	return folders
}

// primarySystemFolderIDs picks the folder IDs we hydrate during background
// sync, ordered by user-visit priority. Order matters: under a context
// deadline a partial pass still gets the most-used folders covered first.
// Custom labels/folders are skipped — the sidebar's eager refresh and
// on-click load handle those.
func primarySystemFolderIDs(folders []domain.Folder) []string {
	if len(folders) == 0 {
		return nil
	}
	priority := map[string]int{
		"inbox":   0,
		"sent":    1,
		"drafts":  2,
		"archive": 3,
		"trash":   4,
		"spam":    5,
	}
	type entry struct {
		id  string
		ord int
	}
	out := make([]entry, 0, len(folders))
	seen := make(map[string]struct{}, len(folders))
	for i := range folders {
		f := &folders[i]
		ord, ok := priority[strings.ToLower(strings.TrimSpace(f.SystemFolder))]
		if !ok {
			continue
		}
		if _, dup := seen[f.ID]; dup {
			continue
		}
		seen[f.ID] = struct{}{}
		out = append(out, entry{id: f.ID, ord: ord})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ord < out[j].ord })
	ids := make([]string, len(out))
	for i, e := range out {
		ids[i] = e.id
	}
	return ids
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
