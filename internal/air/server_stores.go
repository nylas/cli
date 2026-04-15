package air

import (
	"fmt"
	"net/http"

	"github.com/nylas/cli/internal/air/cache"
)

// requireDefaultGrant gets the default grant ID, writing an error response if not available.
// Returns the grant ID and true if successful, or empty string and false if error written.
// Callers should return immediately when ok is false.
func (s *Server) requireDefaultGrant(w http.ResponseWriter) (grantID string, ok bool) {
	grantID, err := s.grantStore.GetDefaultGrant()
	if err != nil || grantID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "No default account. Please select an account first.",
		})
		return "", false
	}
	return grantID, true
}

// getEmailStore returns the email store for the given email account.
func (s *Server) getEmailStore(email string) (*cache.EmailStore, error) {
	if s.cacheManager == nil {
		return nil, fmt.Errorf("cache not initialized")
	}
	db, err := s.cacheManager.GetDB(email)
	if err != nil {
		return nil, err
	}
	return cache.NewEmailStore(db), nil
}

// getEventStore returns the event store for the given email account.
func (s *Server) getEventStore(email string) (*cache.EventStore, error) {
	if s.cacheManager == nil {
		return nil, fmt.Errorf("cache not initialized")
	}
	db, err := s.cacheManager.GetDB(email)
	if err != nil {
		return nil, err
	}
	return cache.NewEventStore(db), nil
}

// getContactStore returns the contact store for the given email account.
func (s *Server) getContactStore(email string) (*cache.ContactStore, error) {
	if s.cacheManager == nil {
		return nil, fmt.Errorf("cache not initialized")
	}
	db, err := s.cacheManager.GetDB(email)
	if err != nil {
		return nil, err
	}
	return cache.NewContactStore(db), nil
}

// getFolderStore returns the folder store for the given email account.
func (s *Server) getFolderStore(email string) (*cache.FolderStore, error) {
	if s.cacheManager == nil {
		return nil, fmt.Errorf("cache not initialized")
	}
	db, err := s.cacheManager.GetDB(email)
	if err != nil {
		return nil, err
	}
	return cache.NewFolderStore(db), nil
}

// getSyncStore returns the sync store for the given email account.
func (s *Server) getSyncStore(email string) (*cache.SyncStore, error) {
	if s.cacheManager == nil {
		return nil, fmt.Errorf("cache not initialized")
	}
	db, err := s.cacheManager.GetDB(email)
	if err != nil {
		return nil, err
	}
	return cache.NewSyncStore(db), nil
}

// getOfflineQueue returns the offline queue for the given email account.
func (s *Server) getOfflineQueue(email string) (*cache.OfflineQueue, error) {
	if s.cacheManager == nil {
		return nil, fmt.Errorf("cache not initialized")
	}

	s.offlineQueuesMu.RLock()
	queue, exists := s.offlineQueues[email]
	s.offlineQueuesMu.RUnlock()
	if exists && queue != nil {
		return queue, nil
	}

	db, err := s.cacheManager.GetDB(email)
	if err != nil {
		return nil, err
	}

	queue, err = cache.NewOfflineQueue(db)
	if err != nil {
		return nil, err
	}

	s.offlineQueuesMu.Lock()
	defer s.offlineQueuesMu.Unlock()

	if existing := s.offlineQueues[email]; existing != nil {
		return existing, nil
	}

	s.offlineQueues[email] = queue
	return queue, nil
}

func (s *Server) initializeOfflineQueues() error {
	if s.cacheManager == nil {
		return nil
	}

	accounts, err := s.cacheManager.ListCachedAccounts()
	if err != nil {
		return err
	}

	s.clearOfflineQueues()
	for _, email := range accounts {
		if _, err := s.getOfflineQueue(email); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) clearOfflineQueues() {
	s.offlineQueuesMu.Lock()
	s.offlineQueues = make(map[string]*cache.OfflineQueue)
	s.offlineQueuesMu.Unlock()
}

func (s *Server) offlineQueuesSnapshot() map[string]*cache.OfflineQueue {
	s.offlineQueuesMu.RLock()
	defer s.offlineQueuesMu.RUnlock()

	snapshot := make(map[string]*cache.OfflineQueue, len(s.offlineQueues))
	for email, queue := range s.offlineQueues {
		snapshot[email] = queue
	}
	return snapshot
}
