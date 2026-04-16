package air

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/nylas/cli/internal/air/cache"
)

type cacheRuntimeManager interface {
	GetDB(email string) (*sql.DB, error)
	Close() error
	ClearCache(email string) error
	ClearAllCaches() error
	ListCachedAccounts() ([]string, error)
	GetStats(email string) (*cache.CacheStats, error)
	DBPath(email string) string
}

func newCacheRuntimeManager(cfg cache.Config, encCfg cache.EncryptionConfig) (cacheRuntimeManager, error) {
	if encCfg.Enabled {
		return cache.NewEncryptedManager(cfg, encCfg)
	}
	return cache.NewManager(cfg)
}

func migrateCacheEncryption(basePath string, enabled bool) error {
	cfg := cache.DefaultConfig()
	cfg.BasePath = basePath

	plainMgr, err := cache.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("create cache manager for migration: %w", err)
	}
	defer func() { _ = plainMgr.Close() }()

	encryptedMgr, err := cache.NewEncryptedManager(cfg, cache.EncryptionConfig{Enabled: true})
	if err != nil {
		return fmt.Errorf("create encrypted cache manager for migration: %w", err)
	}
	defer func() { _ = encryptedMgr.Close() }()

	accounts, err := plainMgr.ListCachedAccounts()
	if err != nil {
		return fmt.Errorf("list cached accounts for migration: %w", err)
	}

	for _, email := range accounts {
		dbPath := plainMgr.DBPath(email)
		isEncrypted, err := cache.IsEncrypted(dbPath)
		if err != nil {
			return fmt.Errorf("detect cache encryption for %s: %w", email, err)
		}

		switch {
		case enabled && !isEncrypted:
			if err := encryptedMgr.MigrateToEncrypted(email); err != nil {
				return fmt.Errorf("migrate cache to encrypted for %s: %w", email, err)
			}
		case !enabled && isEncrypted:
			if err := encryptedMgr.MigrateToUnencrypted(email); err != nil {
				return fmt.Errorf("migrate cache to unencrypted for %s: %w", email, err)
			}
		}
	}

	return nil
}

func (s *Server) reconfigureCacheRuntime() error {
	if s.demoMode || s.cacheSettings == nil {
		return nil
	}

	s.stopBackgroundSync()

	cacheCfg := cache.DefaultConfig()
	if basePath := s.cacheSettings.BasePath(); basePath != "" {
		cacheCfg.BasePath = basePath
	}
	cacheCfg = s.cacheSettings.ToConfig(cacheCfg.BasePath)
	encCfg := s.cacheSettings.ToEncryptionConfig()

	var photoStore *cache.PhotoStore
	err := func() error {
		s.runtimeMu.Lock()
		defer s.runtimeMu.Unlock()

		if s.photoStore != nil {
			if err := s.photoStore.Close(); err != nil {
				return fmt.Errorf("close existing photo store: %w", err)
			}
			s.photoStore = nil
		}

		if s.cacheManager != nil {
			if err := s.cacheManager.Close(); err != nil {
				return fmt.Errorf("close existing cache manager: %w", err)
			}
			s.cacheManager = nil
		}
		s.clearOfflineQueues()

		if !s.cacheSettings.IsCacheEnabled() {
			return nil
		}

		if err := migrateCacheEncryption(cacheCfg.BasePath, encCfg.Enabled); err != nil {
			return err
		}

		cacheManager, err := newCacheRuntimeManager(cacheCfg, encCfg)
		if err != nil {
			return fmt.Errorf("initialize cache manager: %w", err)
		}
		s.cacheManager = cacheManager

		photoStore, err = openPhotoStore(cacheCfg.BasePath)
		if err != nil {
			_ = cacheManager.Close()
			s.cacheManager = nil
			return err
		}
		s.photoStore = photoStore

		if s.cacheSettings.Get().OfflineQueueEnabled {
			if err := s.initializeOfflineQueuesLocked(); err != nil {
				_ = s.photoStore.Close()
				s.photoStore = nil
				_ = cacheManager.Close()
				s.cacheManager = nil
				return err
			}
		}

		return nil
	}()
	if err != nil {
		return err
	}

	if photoStore != nil {
		go prunePhotoStore(photoStore)
	}
	s.startBackgroundSync()

	return nil
}

func openPhotoStore(basePath string) (*cache.PhotoStore, error) {
	photoDB, err := cache.OpenSharedDB(basePath, "photos.db")
	if err != nil {
		return nil, fmt.Errorf("open photo database: %w", err)
	}

	photoStore, err := cache.NewPhotoStore(photoDB, basePath, cache.DefaultPhotoTTL)
	if err != nil {
		_ = photoDB.Close()
		return nil, fmt.Errorf("initialize photo store: %w", err)
	}

	return photoStore, nil
}

func prunePhotoStore(photoStore *cache.PhotoStore) {
	if pruned, err := photoStore.Prune(); err == nil && pruned > 0 {
		fmt.Fprintf(os.Stderr, "Pruned %d expired photos from cache\n", pruned)
	}
}
