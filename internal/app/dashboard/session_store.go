package dashboard

import (
	"errors"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// loadDashboardTokens retrieves the stored dashboard access and session tokens.
// Returns ErrDashboardNotLoggedIn when no user token is present.
func loadDashboardTokens(secrets ports.SecretStore) (userToken, orgToken string, err error) {
	userToken, err = secrets.Get(ports.KeyDashboardUserToken)
	if err != nil {
		if errors.Is(err, domain.ErrSecretNotFound) {
			return "", "", fmt.Errorf("%w", domain.ErrDashboardNotLoggedIn)
		}
		return "", "", fmt.Errorf("failed to load dashboard user token: %w", err)
	}
	if userToken == "" {
		return "", "", fmt.Errorf("%w", domain.ErrDashboardNotLoggedIn)
	}

	orgToken, err = secrets.Get(ports.KeyDashboardOrgToken)
	if err != nil {
		if errors.Is(err, domain.ErrSecretNotFound) {
			return userToken, "", nil
		}
		return "", "", fmt.Errorf("failed to load dashboard organization token: %w", err)
	}
	return userToken, orgToken, nil
}
