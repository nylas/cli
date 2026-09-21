// Package oauth provides CLI commands for logging in to the Nylas OAuth 2.1
// authorization server.
//
// Distinct from `nylas auth`, which connects an end user's mailbox or
// calendar as a provider grant, and from `nylas dashboard login`, which opens
// a dashboard management session. This authenticates the person operating the
// CLI and yields an OIDC identity plus an access token.
package oauth

import (
	"github.com/spf13/cobra"
)

// NewOAuthCmd creates the oauth command group.
func NewOAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oauth",
		Short: "Log in to the Nylas authorization server (OAuth 2.1 / OIDC)",
		Long: `Authenticate with the Nylas OAuth 2.1 authorization server.

Opens a browser to complete an authorization code flow with PKCE and
stores the resulting tokens in the system keyring.

Commands:
  login    Log in via the browser
  status   Show the current OAuth session
  token    Print a valid access token, refreshing it if needed
  logout   Revoke the session and clear stored tokens

The authorization server is hosted by dashboard-account; point the CLI at
a local one with NYLAS_DASHBOARD_ACCOUNT_URL.`,
	}

	cmd.AddCommand(newLoginCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newTokenCmd())
	cmd.AddCommand(newLogoutCmd())

	return cmd
}
