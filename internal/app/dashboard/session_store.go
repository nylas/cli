package dashboard

import (
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// loadDashboardTokens retrieves the stored dashboard access and session tokens.
// Returns ErrDashboardNotLoggedIn when no user token is present.
func loadDashboardTokens(secrets ports.SecretStore) (userToken, orgToken string, err error) {
	userToken, err = secrets.Get(ports.KeyDashboardUserToken)
	if err != nil || userToken == "" {
		return "", "", fmt.Errorf("%w", domain.ErrDashboardNotLoggedIn)
	}
	orgToken, _ = secrets.Get(ports.KeyDashboardOrgToken)
	return userToken, orgToken, nil
}
