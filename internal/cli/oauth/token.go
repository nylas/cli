package oauth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

func newTokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Print a valid access token, refreshing it if needed",
		Long: `Print the current access token.

If the stored token has expired it is refreshed first, and the rotated
refresh token replaces the old one in the keyring.`,
		Example: `  # Call an OAuth-protected endpoint
  curl -H "Authorization: Bearer $(nylas oauth token)" https://example/resource`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := createLoginServiceFn()
			if err != nil {
				return wrapOAuthError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			token, err := svc.AccessToken(ctx)
			if err != nil {
				return wrapOAuthError(err)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), token)
			return nil
		},
	}
}
