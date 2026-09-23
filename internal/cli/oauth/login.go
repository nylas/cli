package oauth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func newLoginCmd() *cobra.Command {
	var scopes []string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to the Nylas authorization server via the browser",
		Long: `Run an OAuth 2.1 authorization code flow with PKCE.

The CLI is a static public client (no client secret; PKCE proves the
exchange). It opens a browser for consent, receives the redirect on
http://127.0.0.1:<port>/callback, and stores the tokens in the system keyring.

Set NYLAS_OAUTH_CLIENT_ID to use a different client id against a local or
dev authorization server.`,
		Example: `  # Log in with the default scopes (openid, email, offline_access)
  nylas oauth login

  # Log in against a local authorization server
  NYLAS_DASHBOARD_ACCOUNT_URL=http://localhost:3001 nylas oauth login`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := createLoginServiceFn()
			if err != nil {
				return wrapOAuthError(err)
			}

			// The user has to read a consent screen, so this needs the long
			// interactive timeout rather than the API one.
			ctx, cancel := common.CreateLongContext()
			defer cancel()

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Opening your browser to complete sign-in...")

			result, err := svc.Login(ctx, scopes)
			if err != nil {
				return wrapOAuthError(err)
			}

			out := cmd.OutOrStdout()
			_, _ = common.Green.Fprintln(out, "✓ Logged in")
			_, _ = fmt.Fprintf(out, "  Issuer:     %s\n", result.Issuer)
			_, _ = fmt.Fprintf(out, "  Client ID:  %s\n", result.ClientID)
			if result.Scope != "" {
				_, _ = fmt.Fprintf(out, "  Scopes:     %s\n", result.Scope)
			}
			if !result.ExpiresAt.IsZero() {
				_, _ = fmt.Fprintf(out, "  Expires:    %s\n", result.ExpiresAt.Local().Format("2006-01-02 15:04:05 MST"))
			}
			if !result.HasRefresh {
				_, _ = common.Yellow.Fprintln(out,
					"  No refresh token issued — request the offline_access scope to stay signed in.")
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&scopes, "scope", nil,
		fmt.Sprintf("OAuth scopes to request (default %v)", domain.DefaultOAuthScopes()))

	return cmd
}
