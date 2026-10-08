package common

import (
	"os/user"
	"path/filepath"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/filelock"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/ports"
)

// Lock files that serialise session writes across every CLI process on the
// machine. When both are needed, take DashboardSessionLockFile first: a
// dashboard renewal holds it while it asks the OAuth session for a token,
// which may take OAuthSessionLockFile to refresh.
const (
	OAuthSessionLockFile     = "oauth-session.lock"
	DashboardSessionLockFile = "dashboard-session.lock"
)

// OAuthSessionLock guards the `nylas oauth login` session.
func OAuthSessionLock(secrets ports.SecretStore) *filelock.Lock {
	return filelock.New(SessionLockPath(secrets, OAuthSessionLockFile))
}

// DashboardSessionLock guards the dashboard session keys.
func DashboardSessionLock(secrets ports.SecretStore) *filelock.Lock {
	return filelock.New(SessionLockPath(secrets, DashboardSessionLockFile))
}

// SessionLockPath puts a lock with the secrets it protects, so every process
// sharing a session also shares its lock.
//
// The encrypted file store lives in the config directory, so its lock does
// too. The system keyring is one store per user whatever XDG_CONFIG_HOME says,
// and an editor or MCP host can start `nylas mcp serve` with a different
// XDG_CONFIG_HOME than the user's shell. A lock that followed it would let two
// processes write the same session at once. So for the keyring the lock sits
// under the account's home directory from the user database, not from the
// environment.
func SessionLockPath(secrets ports.SecretStore, name string) string {
	if _, ok := secrets.(*keyring.EncryptedFileStore); ok {
		return filepath.Join(config.DefaultConfigDir(), name)
	}
	if account, err := user.Current(); err == nil && account.HomeDir != "" {
		return filepath.Join(account.HomeDir, ".config", "nylas", name)
	}
	return filepath.Join(config.DefaultConfigDir(), name)
}
