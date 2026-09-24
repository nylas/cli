package oauth

import (
	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Revoke the OAuth session and clear stored tokens",
		Long: `Revoke the refresh token and remove the stored session.

Revoking the refresh token takes the whole token family with it. Local
tokens are cleared even when the server cannot be reached. A dashboard
session that came from 'nylas oauth login' is ended too; one from
'nylas dashboard login' is left alone.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := createLoginServiceFn()
			if err != nil {
				return wrapOAuthError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			dashboardErr := clearDashboardSessionFn(ctx)
			if err := svc.Logout(ctx); err != nil {
				return wrapOAuthError(err)
			}
			if dashboardErr != nil {
				_, _ = common.Yellow.Fprintf(cmd.OutOrStdout(),
					"  Could not end the dashboard session: %v\n", dashboardErr)
			}

			_, _ = common.Green.Fprintln(cmd.OutOrStdout(), "✓ Logged out")
			return nil
		},
	}
}
