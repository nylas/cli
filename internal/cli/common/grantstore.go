package common

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/grantcache"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

const (
	legacyGrantsKey       = "grants"
	legacyDefaultGrantKey = "default_grant"
)

// NewDefaultGrantStore returns the local grant metadata store. It does not
// create an API client or require API credentials; grant listing remains a
// live API concern.
func NewDefaultGrantStore() (ports.GrantStore, error) {
	path, err := DefaultGrantCachePath()
	if err != nil {
		return nil, err
	}
	cacheExists := grantCacheExists(path)
	store := grantcache.New(path)
	migrateLegacyGrantStore(store, cacheExists)
	return store, nil
}

// DefaultGrantCachePath returns the cache path used for non-secret grant
// metadata and local default-grant preference.
func DefaultGrantCachePath() (string, error) {
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "nylas", "grants.json"), nil
}

func migrateLegacyGrantStore(store ports.GrantStore, cacheExists bool) {
	if secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir()); err == nil {
		migrateLegacyFromSecretStore(store, secretStore)
	}
	if fileStore, err := keyring.NewEncryptedFileStore(config.DefaultConfigDir()); err == nil {
		migrateLegacyFromSecretStore(store, fileStore)
	}
	if !cacheExists {
		migrateDefaultFromConfig(store)
	}
}

func grantCacheExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func migrateLegacyFromSecretStore(store ports.GrantStore, secrets ports.SecretStore) {
	legacyGrants, hasLegacyGrants := readLegacyGrants(secrets)
	legacyDefault, hasLegacyDefault := readLegacySecret(secrets, legacyDefaultGrantKey)

	if hasLegacyGrants {
		current, err := store.ListGrants()
		if err == nil && len(current) == 0 && len(legacyGrants) > 0 {
			if err := store.ReplaceGrants(legacyGrants); err != nil {
				return
			}
		}
	}

	if hasLegacyDefault {
		if _, err := store.GetDefaultGrant(); errors.Is(err, domain.ErrNoDefaultGrant) {
			if err := store.SetDefaultGrant(legacyDefault); err != nil {
				return
			}
		}
	}

	if hasLegacyGrants {
		_ = secrets.Delete(legacyGrantsKey)
	}
	if hasLegacyDefault {
		_ = secrets.Delete(legacyDefaultGrantKey)
	}
}

func migrateDefaultFromConfig(store ports.GrantStore) {
	if _, err := store.GetDefaultGrant(); !errors.Is(err, domain.ErrNoDefaultGrant) {
		return
	}
	cfg, err := config.NewDefaultFileStore().Load()
	if err != nil || cfg == nil || cfg.DefaultGrant == "" {
		return
	}
	_ = store.SetDefaultGrant(cfg.DefaultGrant)
}

func readLegacyGrants(secrets ports.SecretStore) ([]domain.GrantInfo, bool) {
	data, ok := readLegacySecret(secrets, legacyGrantsKey)
	if !ok {
		return nil, false
	}
	var grants []domain.GrantInfo
	if err := json.Unmarshal([]byte(data), &grants); err != nil {
		return nil, false
	}
	return grants, true
}

func readLegacySecret(secrets ports.SecretStore, key string) (string, bool) {
	value, err := secrets.Get(key)
	if err != nil || value == "" {
		return "", false
	}
	return value, true
}
