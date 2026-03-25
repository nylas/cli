// Package setup provides the first-time user experience wizard for the Nylas CLI.
package setup

import (
	"os"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/ports"
)

// SetupStatus describes what is already configured in the CLI.
type SetupStatus struct {
	HasDashboardAuth bool
	HasAPIKey        bool
	HasActiveApp     bool
	HasGrants        bool
	ActiveAppID      string
	ActiveAppRegion  string
}

// IsFirstRun returns true when the CLI has never been configured.
// A user is "first run" when there is no API key (keyring or env) and
// no dashboard session token.
func IsFirstRun() bool {
	// Check environment variable first (cheapest check).
	if os.Getenv("NYLAS_API_KEY") != "" {
		return false
	}

	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		// Can't access secrets — treat as first run so the welcome message shows.
		return true
	}

	if hasKey(secretStore, ports.KeyAPIKey) {
		return false
	}
	if hasKey(secretStore, ports.KeyDashboardUserToken) {
		return false
	}

	return true
}

// GetSetupStatus returns a detailed view of the current setup state.
func GetSetupStatus() SetupStatus {
	status := SetupStatus{}

	// Check env-based API key.
	if os.Getenv("NYLAS_API_KEY") != "" {
		status.HasAPIKey = true
	}

	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return status
	}

	status.HasAPIKey = status.HasAPIKey || hasKey(secretStore, ports.KeyAPIKey)
	status.HasDashboardAuth = hasKey(secretStore, ports.KeyDashboardUserToken)

	appID, _ := secretStore.Get(ports.KeyDashboardAppID)
	appRegion, _ := secretStore.Get(ports.KeyDashboardAppRegion)
	if appID != "" && appRegion != "" {
		status.HasActiveApp = true
		status.ActiveAppID = appID
		status.ActiveAppRegion = appRegion
	}

	grantStore := keyring.NewGrantStore(secretStore)
	grants, err := grantStore.ListGrants()
	if err == nil && len(grants) > 0 {
		status.HasGrants = true
	}

	return status
}

// hasKey returns true if a non-empty value exists for the given key.
func hasKey(store ports.SecretStore, key string) bool {
	val, err := store.Get(key)
	return err == nil && val != ""
}
