package common

import (
	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/domain"
)

// FormatGrantStatus renders the common valid/invalid grant states consistently.
func FormatGrantStatus(status string) string {
	switch status {
	case "valid":
		return Green.Sprint("active")
	case "invalid":
		return Red.Sprint("invalid")
	default:
		return Yellow.Sprint(status)
	}
}

// SaveGrantLocally stores a grant in the local grant store so it can be reused by CLI flows.
func SaveGrantLocally(grantID, email string, provider domain.Provider) {
	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return
	}

	grantStore := keyring.NewGrantStore(secretStore)
	_ = grantStore.SaveGrant(domain.GrantInfo{
		ID:       grantID,
		Email:    email,
		Provider: provider,
	})
}

// RemoveGrantLocally removes a grant from the local grant store.
func RemoveGrantLocally(grantID string) {
	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return
	}

	grantStore := keyring.NewGrantStore(secretStore)
	_ = grantStore.DeleteGrant(grantID)
}
