package dashboard

import (
	"errors"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// loadDashboardTokens retrieves the stored dashboard access and session tokens.
// Returns ErrDashboardNotLoggedIn when no user token is present, and
// ErrDashboardServerMismatch when the session was issued for other servers
// than server (see checkSessionServer).
//
// It reads without the dashboard session lock, so a login for another server
// can replace the session mid-read. The user token and server are re-read
// last; if either changed, the read straddled a write and is retried, so one
// server's token is never paired with another server's name.
func loadDashboardTokens(secrets ports.SecretStore, server string) (userToken, orgToken string, err error) {
	for range 3 {
		userToken, orgToken, err = readDashboardTokens(secrets, server)
		if err != nil {
			return "", "", err
		}
		again, againErr := secrets.Get(ports.KeyDashboardUserToken)
		stored, storedErr := secrets.Get(ports.KeyDashboardSessionServer)
		if againErr == nil && again == userToken &&
			(server == "" || (storedErr == nil && stored == server)) {
			return userToken, orgToken, nil
		}
	}
	return "", "", fmt.Errorf("%w: the stored session kept changing while it was read", domain.ErrDashboardNotLoggedIn)
}

func readDashboardTokens(secrets ports.SecretStore, server string) (userToken, orgToken string, err error) {
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
	if err := checkSessionServer(secrets, server); err != nil {
		return "", "", err
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

// checkSessionServer refuses a stored session issued for other servers than
// server, so its tokens are never sent to one that did not issue them. An
// empty server skips the check.
//
// A session stored before servers were recorded is adopted by the first
// server that reads it: that is where every earlier command already sent it,
// and refusing it would sign out everyone who upgrades. The exception is one
// exchanged from OAuth, whose issuing account server is known.
func checkSessionServer(secrets ports.SecretStore, server string) error {
	if server == "" {
		return nil
	}
	stored, err := secrets.Get(ports.KeyDashboardSessionServer)
	switch {
	case errors.Is(err, domain.ErrSecretNotFound) || (err == nil && stored == ""):
		if err := checkLegacyOAuthSessionServer(secrets, server); err != nil {
			return err
		}
		if err := secrets.Set(ports.KeyDashboardSessionServer, server); err != nil {
			return fmt.Errorf("failed to record the dashboard session server: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("failed to read the dashboard session server: %w", err)
	case stored != server:
		return fmt.Errorf(
			"%w: it was issued for %s, but the CLI is now configured for %s; log in again, or restore the NYLAS_DASHBOARD_* settings it was issued under",
			domain.ErrDashboardServerMismatch, stored, server,
		)
	}
	return nil
}

// checkLegacyOAuthSessionServer refuses an unrecorded session exchanged from
// OAuth when the OAuth session it came from belongs to another account server.
// That is the one case where the issuer of an old session is known.
func checkLegacyOAuthSessionServer(secrets ports.SecretStore, server string) error {
	if !isOAuthSession(secrets) {
		return nil
	}
	issuedBy, err := secrets.Get(ports.KeyOAuthServerURL)
	if err != nil || issuedBy == "" {
		return nil
	}
	configured := domain.DashboardSessionAccountURL(server)
	if configured == "" || domain.NormalizeOAuthIssuer(issuedBy) == configured {
		return nil
	}
	return fmt.Errorf(
		"%w: it was issued by %s, but the CLI is now configured for %s; log in again, or restore the NYLAS_DASHBOARD_* settings it was issued under",
		domain.ErrDashboardServerMismatch, domain.NormalizeOAuthIssuer(issuedBy), configured,
	)
}
