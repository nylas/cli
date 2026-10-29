package dashboard

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	dashboardapp "github.com/nylas/cli/internal/app/dashboard"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out of the Nylas Dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			authSvc, _, err := createAuthService()
			if errors.Is(err, domain.ErrDashboardDPoPKeyInvalid) {
				return logoutWithInvalidKey()
			}
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			err = common.RunWithSpinner("Logging out...", func() error {
				return authSvc.Logout(ctx)
			})
			if err != nil {
				return wrapDashboardError(err)
			}

			_, _ = common.Green.Println("✓ Logged out")
			return nil
		},
	}
}

// logoutWithInvalidKey is the way out of a stored DPoP key that is not a key.
// Every other dashboard command fails on it, and the session bound to it
// cannot be revoked without it, so both are removed locally.
func logoutWithInvalidKey() error {
	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return wrapDashboardError(err)
	}
	ctx, cancel := common.CreateContext()
	defer cancel()
	if err := dashboardapp.ForgetSession(ctx, secretStore, common.DashboardSessionLock(secretStore)); err != nil {
		return wrapDashboardError(err)
	}
	_, _ = common.Yellow.Println("The stored DPoP key was not valid: the session was removed locally but not revoked on the server.")
	_, _ = common.Green.Println("✓ Logged out")
	return nil
}
