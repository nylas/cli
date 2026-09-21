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
tokens are cleared even when the server cannot be reached.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := createLoginServiceFn()
			if err != nil {
				return wrapOAuthError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			if err := svc.Logout(ctx); err != nil {
				return wrapOAuthError(err)
			}

			_, _ = common.Green.Fprintln(cmd.OutOrStdout(), "✓ Logged out")
			return nil
		},
	}
}
