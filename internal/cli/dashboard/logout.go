package dashboard

import (
	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out of the Nylas Dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			authSvc, _, err := createAuthService()
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
