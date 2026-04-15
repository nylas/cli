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

	cacheCfg := cache.DefaultConfig()
	if basePath := s.cacheSettings.BasePath(); basePath != "" {
		cacheCfg.BasePath = basePath
	}
	cacheCfg = s.cacheSettings.ToConfig(cacheCfg.BasePath)
	encCfg := s.cacheSettings.ToEncryptionConfig()

	if s.cacheManager != nil {
		if err := s.cacheManager.Close(); err != nil {
			return fmt.Errorf("close existing cache manager: %w", err)
		}
		s.cacheManager = nil
	}

	if !s.cacheSettings.IsCacheEnabled() {
		s.clearOfflineQueues()
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

	if err := s.ensurePhotoStore(cacheCfg.BasePath); err != nil {
		return err
	}

	if s.cacheSettings.Get().OfflineQueueEnabled {
		if err := s.initializeOfflineQueues(); err != nil {
			return err
		}
	} else {
		s.clearOfflineQueues()
	}

	return nil
}

func (s *Server) ensurePhotoStore(basePath string) error {
	if s.photoStore != nil {
		return nil
	}

	photoDB, err := cache.OpenSharedDB(basePath, "photos.db")
	if err != nil {
		return fmt.Errorf("open photo database: %w", err)
	}

	photoStore, err := cache.NewPhotoStore(photoDB, basePath, cache.DefaultPhotoTTL)
	if err != nil {
		_ = photoDB.Close()
		return fmt.Errorf("initialize photo store: %w", err)
	}
	s.photoStore = photoStore

	// Prune expired photos asynchronously after startup.
	go func() {
		if pruned, err := photoStore.Prune(); err == nil && pruned > 0 {
			fmt.Fprintf(os.Stderr, "Pruned %d expired photos from cache\n", pruned)
		}
	}()

	return nil
}
