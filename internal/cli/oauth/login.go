package oauth

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/app/oauthlogin"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// loginPresetMCP is `--for mcp`: a session the hosted MCP server accepts.
const loginPresetMCP = "mcp"

func newLoginCmd() *cobra.Command {
	var (
		scopes []string
		preset string
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to the Nylas authorization server via the browser",
		Long: `Run an OAuth 2.1 authorization code flow with PKCE.

The CLI is a static public client (no client secret; PKCE proves the
exchange). It opens a browser for consent, receives the redirect on
http://127.0.0.1:<port>/callback, and stores the tokens in the system keyring.

--for mcp logs in for the hosted Nylas MCP server used by
'nylas mcp serve --auth oauth': it requests the data scopes the MCP tools use
(email.read, email.send, calendar.read, calendar.write, contacts.read,
notetaker.read, grants.read) plus offline_access, and names the MCP server of
the configured region (https://mcp.us.nylas.com or https://mcp.eu.nylas.com)
as the RFC 8707 resource, so the token is issued for that server only. Scopes
the server does not offer are left out and reported.

Set NYLAS_OAUTH_CLIENT_ID to use a different client id against a local or
dev authorization server.`,
		Example: `  # Log in with the default scopes (openid, email, offline_access)
  nylas oauth login

  # Log in for 'nylas mcp serve --auth oauth'
  nylas oauth login --for mcp

  # Log in against a local authorization server
  NYLAS_DASHBOARD_ACCOUNT_URL=http://localhost:3001 nylas oauth login`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, err := loginOptions(preset, scopes, configuredRegion())
			if err != nil {
				return err
			}

			svc, err := createLoginServiceFn()
			if err != nil {
				return wrapOAuthError(err)
			}

			// The user has to read a consent screen, so this needs the long
			// interactive timeout rather than the API one.
			ctx, cancel := common.CreateLongContext()
			defer cancel()

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Opening your browser to complete sign-in...")

			result, err := svc.Login(ctx, opts)
			if err != nil {
				return wrapOAuthError(err)
			}

			out := cmd.OutOrStdout()
			_, _ = common.Green.Fprintln(out, "✓ Logged in")
			_, _ = fmt.Fprintf(out, "  Issuer:     %s\n", result.Issuer)
			_, _ = fmt.Fprintf(out, "  Client ID:  %s\n", result.ClientID)
			if result.Resource != "" {
				_, _ = fmt.Fprintf(out, "  Resource:   %s\n", result.Resource)
			}
			if result.Scope != "" {
				_, _ = fmt.Fprintf(out, "  Scopes:     %s\n", result.Scope)
			}
			if !result.ExpiresAt.IsZero() {
				_, _ = fmt.Fprintf(out, "  Expires:    %s\n", result.ExpiresAt.Local().Format("2006-01-02 15:04:05 MST"))
			}
			if len(result.DroppedScopes) > 0 {
				_, _ = common.Yellow.Fprintf(out,
					"  Not offered by this server, so not requested: %s\n", strings.Join(result.DroppedScopes, " "))
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
	cmd.Flags().StringVar(&preset, "for", "",
		"log in for a Nylas resource server: mcp (scopes and resource for 'nylas mcp serve --auth oauth')")

	return cmd
}

// loginOptions turns the flags into a login request. A preset decides both
// the scopes and the resource, so it cannot be combined with --scope: a
// hand-picked scope list under the MCP resource is a token the MCP tools
// would refuse halfway through a conversation.
func loginOptions(preset string, scopes []string, region string) (oauthlogin.LoginOptions, error) {
	switch preset {
	case "":
		return oauthlogin.LoginOptions{Scopes: scopes}, nil
	case loginPresetMCP:
		if len(scopes) > 0 {
			return oauthlogin.LoginOptions{}, common.NewUserError(
				"--for and --scope cannot be combined",
				"--for mcp chooses the scopes the MCP tools need",
			)
		}
		resource, err := domain.MCPResourceForRegion(region)
		if err != nil {
			return oauthlogin.LoginOptions{}, err
		}
		return oauthlogin.LoginOptions{
			Scopes:                domain.MCPOAuthScopes(),
			Resource:              resource,
			DropUnsupportedScopes: true,
		}, nil
	default:
		return oauthlogin.LoginOptions{}, common.NewUserError(
			fmt.Sprintf("unknown --for value %q", preset),
			"supported: mcp",
		)
	}
}
