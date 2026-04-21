package air

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/nylas/cli/internal/air/cache"
	authapp "github.com/nylas/cli/internal/app/auth"
	"github.com/nylas/cli/internal/domain"
)

var errCacheNotInitialized = fmt.Errorf("cache not initialized")

func (s *Server) hasCacheRuntime() bool {
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	return s.cacheManager != nil
}

func (s *Server) hasPhotoStore() bool {
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	return s.photoStore != nil
}

func (s *Server) cacheAvailable() bool {
	return s.cacheSettings != nil && s.cacheSettings.IsCacheEnabled() && s.hasCacheRuntime()
}

func (s *Server) offlineQueueConfigured() bool {
	return s.cacheSettings != nil && s.cacheSettings.Get().OfflineQueueEnabled && s.hasCacheRuntime()
}

func (s *Server) withCacheManager(fn func(cacheRuntimeManager) error) error {
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()

	if s.cacheManager == nil {
		return errCacheNotInitialized
	}

	return fn(s.cacheManager)
}

func (s *Server) withAccountDB(email string, fn func(*sql.DB) error) error {
	return s.withCacheManager(func(manager cacheRuntimeManager) error {
		db, err := manager.GetDB(email)
		if err != nil {
			return err
		}
		return fn(db)
	})
}

func (s *Server) withEmailStore(email string, fn func(*cache.EmailStore) error) error {
	return s.withAccountDB(email, func(db *sql.DB) error {
		return fn(cache.NewEmailStore(db))
	})
}

func (s *Server) withEventStore(email string, fn func(*cache.EventStore) error) error {
	return s.withAccountDB(email, func(db *sql.DB) error {
		return fn(cache.NewEventStore(db))
	})
}

func (s *Server) withContactStore(email string, fn func(*cache.ContactStore) error) error {
	return s.withAccountDB(email, func(db *sql.DB) error {
		return fn(cache.NewContactStore(db))
	})
}

func (s *Server) withFolderStore(email string, fn func(*cache.FolderStore) error) error {
	return s.withAccountDB(email, func(db *sql.DB) error {
		return fn(cache.NewFolderStore(db))
	})
}

func (s *Server) withPhotoStore(fn func(*cache.PhotoStore) error) error {
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()

	if s.photoStore == nil {
		return fmt.Errorf("photo store not initialized")
	}

	return fn(s.photoStore)
}

func (s *Server) withOfflineQueue(email string, fn func(*cache.OfflineQueue) error) error {
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()

	if s.cacheManager == nil {
		return errCacheNotInitialized
	}

	s.offlineQueuesMu.RLock()
	queue := s.offlineQueues[email]
	s.offlineQueuesMu.RUnlock()

	if queue == nil {
		db, err := s.cacheManager.GetDB(email)
		if err != nil {
			return err
		}

		queue, err = cache.NewOfflineQueue(db)
		if err != nil {
			return err
		}

		s.offlineQueuesMu.Lock()
		if existing := s.offlineQueues[email]; existing != nil {
			queue = existing
		} else {
			s.offlineQueues[email] = queue
		}
		s.offlineQueuesMu.Unlock()
	}

	return fn(queue)
}

func (s *Server) listSupportedGrants() ([]domain.GrantInfo, error) {
	if s.grantStore == nil {
		return nil, domain.ErrGrantNotFound
	}

	grants, err := s.grantStore.ListGrants()
	if err != nil {
		return nil, err
	}

	supported := make([]domain.GrantInfo, 0, len(grants))
	for _, grant := range grants {
		if grant.Provider.IsSupportedByAir() {
			supported = append(supported, grant)
		}
	}

	return supported, nil
}

func (s *Server) setDefaultGrant(grantID string) error {
	if s.grantStore == nil {
		return domain.ErrGrantNotFound
	}

	return authapp.PersistDefaultGrant(s.configStore, s.grantStore, grantID)
}

func (s *Server) resolveDefaultGrantInfo() (domain.GrantInfo, error) {
	supported, err := s.listSupportedGrants()
	if err != nil {
		return domain.GrantInfo{}, err
	}
	if len(supported) == 0 {
		return domain.GrantInfo{}, domain.ErrGrantNotFound
	}

	defaultID, err := s.grantStore.GetDefaultGrant()
	switch {
	case err == nil:
		for _, grant := range supported {
			if grant.ID == defaultID {
				return grant, nil
			}
		}
	case !errors.Is(err, domain.ErrNoDefaultGrant):
		return domain.GrantInfo{}, err
	}

	selected := supported[0]
	return selected, nil
}

// requireDefaultGrant gets the default grant ID, writing an error response if not available.
// Returns the grant ID and true if successful, or empty string and false if error written.
// Callers should return immediately when ok is false.
func (s *Server) requireDefaultGrant(w http.ResponseWriter) (grantID string, ok bool) {
	grant, err := s.resolveDefaultGrantInfo()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "No supported Air account is configured. Please select a Google, Microsoft, or Nylas account first.",
		})
		return "", false
	}
	return grant.ID, true
}

// requireDefaultGrantInfo gets the default grant info, writing an error response if not available.
// Returns the grant info and true if successful, or an empty grant and false if error written.
func (s *Server) requireDefaultGrantInfo(w http.ResponseWriter) (grant domain.GrantInfo, ok bool) {
	info, err := s.resolveDefaultGrantInfo()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "No supported Air account is configured. Please choose another account.",
		})
		return domain.GrantInfo{}, false
	}

	return info, true
}

// getEmailStore returns the email store for the given email account.
func (s *Server) getEmailStore(email string) (*cache.EmailStore, error) {
	var store *cache.EmailStore
	err := s.withAccountDB(email, func(db *sql.DB) error {
		store = cache.NewEmailStore(db)
		return nil
	})
	return store, err
}

// getEventStore returns the event store for the given email account.
func (s *Server) getEventStore(email string) (*cache.EventStore, error) {
	var store *cache.EventStore
	err := s.withAccountDB(email, func(db *sql.DB) error {
		store = cache.NewEventStore(db)
		return nil
	})
	return store, err
}

// getContactStore returns the contact store for the given email account.
func (s *Server) getContactStore(email string) (*cache.ContactStore, error) {
	var store *cache.ContactStore
	err := s.withAccountDB(email, func(db *sql.DB) error {
		store = cache.NewContactStore(db)
		return nil
	})
	return store, err
}

// getFolderStore returns the folder store for the given email account.
func (s *Server) getFolderStore(email string) (*cache.FolderStore, error) {
	var store *cache.FolderStore
	err := s.withAccountDB(email, func(db *sql.DB) error {
		store = cache.NewFolderStore(db)
		return nil
	})
	return store, err
}

// getSyncStore returns the sync store for the given email account.
func (s *Server) getSyncStore(email string) (*cache.SyncStore, error) {
	var store *cache.SyncStore
	err := s.withAccountDB(email, func(db *sql.DB) error {
		store = cache.NewSyncStore(db)
		return nil
	})
	return store, err
}

// getOfflineQueue returns the offline queue for the given email account.
func (s *Server) getOfflineQueue(email string) (*cache.OfflineQueue, error) {
	var queue *cache.OfflineQueue
	err := s.withOfflineQueue(email, func(q *cache.OfflineQueue) error {
		queue = q
		return nil
	})
	return queue, err
}

func (s *Server) initializeOfflineQueuesLocked() error {
	if s.cacheManager == nil {
		return nil
	}

	accounts, err := s.cacheManager.ListCachedAccounts()
	if err != nil {
		return err
	}

	s.offlineQueuesMu.Lock()
	defer s.offlineQueuesMu.Unlock()

	s.offlineQueues = make(map[string]*cache.OfflineQueue, len(accounts))
	for _, email := range accounts {
		db, err := s.cacheManager.GetDB(email)
		if err != nil {
			return err
		}
		queue, err := cache.NewOfflineQueue(db)
		if err != nil {
			return err
		}
		s.offlineQueues[email] = queue
	}

	return nil
}

func (s *Server) clearOfflineQueues() {
	s.offlineQueuesMu.Lock()
	s.offlineQueues = make(map[string]*cache.OfflineQueue)
	s.offlineQueuesMu.Unlock()
}

func (s *Server) offlineQueueEmails() []string {
	s.offlineQueuesMu.RLock()
	defer s.offlineQueuesMu.RUnlock()

	emails := make([]string, 0, len(s.offlineQueues))
	for email := range s.offlineQueues {
		emails = append(emails, email)
	}
	return emails
}
