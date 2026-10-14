package oauth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/spf13/cobra"

	dashboardapp "github.com/nylas/cli/internal/app/dashboard"
	"github.com/nylas/cli/internal/app/oauthlogin"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// loginPresetMCP is `--for mcp`: a session the hosted MCP server accepts.
const loginPresetMCP = "mcp"

func newLoginCmd() *cobra.Command {
	var (
		scopes    []string
		preset    string
		region    string
		device    bool
		noBrowser bool
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to Nylas using a browser or device code",
		Long: `Run an OAuth 2.1 authorization code flow with PKCE, or device sign-in.

Use --device or --no-browser over SSH or on a headless machine. The CLI prints
an approval URL and user code, then waits while you sign in or sign up on
another device. SSH sessions select this flow automatically; local macOS and
Windows keep browser sign-in even without DISPLAY. If signup takes too long,
run the command again to get a new code. In device mode, the region also
fixes the regional token audience; signup must use that same region.

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

Every login also requests dashboard.session, which signs the 'nylas dashboard'
commands in. The consent screen shows it as "Use the Nylas Dashboard as you,
with your full role in this organization". It is never advertised by the
server and never left out, except with NYLAS_OAUTH_CLIENT_ID, which the server
does not allow to request it.

If you do not have an account yet, you can sign up on the page that opens.
--region (or the configured region) decides where the new organization is
created; without either it is created in the US. It does not change an
existing account.

Set NYLAS_OAUTH_CLIENT_ID to use a different client id against a local or
dev authorization server. Such a server names its own MCP servers and refuses
the production ones (invalid_target): set NYLAS_MCP_US_URL and/or
NYLAS_MCP_EU_URL to the origins it names. Each replaces that region's server,
so a token is only ever sent to the one server its region names.`,
		Example: `  # Log in with the default scopes (openid, email, offline_access, dashboard.session)
  nylas oauth login

  # Log in for 'nylas mcp serve --auth oauth'
  nylas oauth login --for mcp

  # Sign in over SSH without a local browser
  nylas oauth login --device

  # Sign up with a new organization in the EU
  nylas oauth login --region eu

  # Log in against a local authorization server
  NYLAS_DASHBOARD_ACCOUNT_URL=http://localhost:3001 nylas oauth login`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// A bad --region is the user's to fix. A bad configured region is
			// only a sign-up hint here, so it is dropped rather than blocking
			// the login; --for mcp still validates it for the resource.
			if region == "" {
				region = configuredRegion()
				if _, err := normalizeRegion(region); err != nil && preset == "" {
					region = ""
				}
			}
			opts, err := loginOptions(preset, scopes, region)
			if err != nil {
				return err
			}

			opts.Device = device || noBrowser || isSSHSession()
			opts.OnDeviceAuthorization = func(prompt domain.OAuthDevicePrompt) error {
				out := cmd.OutOrStdout()
				if _, err := fmt.Fprintf(out, "Open %s on a device with a browser.\nEnter code: %s\nOnly continue if you started this sign-in yourself.\n", prompt.VerificationURI, prompt.UserCode); err != nil {
					return err
				}
				if prompt.VerificationURIComplete != "" {
					_, err := fmt.Fprintln(out, prompt.VerificationURIComplete)
					return err
				}
				return nil
			}
			svc, err := createLoginServiceFn()
			if err != nil {
				return wrapOAuthError(err)
			}

			// The user has to read a consent screen, so this needs the long
			// interactive timeout rather than the API one.
			ctx, cancel := common.CreateLongContext()
			if opts.Device {
				cancel()
				signalCtx, stopSignals := signal.NotifyContext(cmd.Context(), os.Interrupt)
				defer stopSignals()
				ctx, cancel = context.WithTimeout(signalCtx, 10*time.Minute)
			}
			defer cancel()

			if opts.Device {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Using device sign-in; waiting for approval...")
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Opening your browser to complete sign-in...")
			}

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

			// The OAuth login already succeeded, so a failed exchange is reported
			// rather than returned: only the dashboard commands are affected.
			exchangeCtx, cancelExchange := afterConsentContext(ctx)
			defer cancelExchange()
			dashboardSession, err := exchangeDashboardSessionFn(exchangeCtx, svc)
			if errors.Is(err, dashboardapp.ErrDashboardLoginSessionKept) {
				_, _ = fmt.Fprintln(out, "  Dashboard:  still signed in with `nylas dashboard login` (run `nylas dashboard logout` first to use this login instead)")
				return nil
			}
			if err != nil {
				_, _ = common.Yellow.Fprintf(out,
					"  Dashboard commands are not signed in: %v\n", err)
				return nil
			}
			_, _ = fmt.Fprintf(out, "  Dashboard:  signed in to organization %s\n", dashboardSession.OrgPublicID)

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&scopes, "scope", nil,
		fmt.Sprintf("OAuth scopes to request (default %v)", domain.DefaultOAuthScopes()))
	cmd.Flags().StringVar(&preset, "for", "",
		"log in for a Nylas resource server: mcp (scopes and resource for 'nylas mcp serve --auth oauth')")
	cmd.Flags().StringVar(&region, "region", "",
		"region for an organization created if you sign up: us or eu (default: the configured region)")

	cmd.Flags().BoolVar(&device, "device", false, "sign in or sign up using a code on another device")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "use device sign-in without opening a local browser")
	return cmd
}

// loginOptions turns the flags into a login request. A preset decides both
// the scopes and the resource, so it cannot be combined with --scope: a
// hand-picked scope list under the MCP resource is a token the MCP tools
// would refuse halfway through a conversation.
func loginOptions(preset string, scopes []string, region string) (oauthlogin.LoginOptions, error) {
	switch preset {
	case "":
		signUpRegion, err := normalizeRegion(region)
		if err != nil {
			return oauthlogin.LoginOptions{}, err
		}
		return oauthlogin.LoginOptions{Scopes: scopes, Region: signUpRegion}, nil
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
			Region:                strings.ToLower(strings.TrimSpace(region)),
			DropUnsupportedScopes: true,
		}, nil
	default:
		return oauthlogin.LoginOptions{}, common.NewUserError(
			fmt.Sprintf("unknown --for value %q", preset),
			"supported: mcp",
		)
	}
}

// normalizeRegion accepts us or eu in any case, and empty for "not set".
func normalizeRegion(region string) (string, error) {
	switch r := strings.ToLower(strings.TrimSpace(region)); r {
	case "", "us", "eu":
		return r, nil
	default:
		return "", common.NewUserError(
			fmt.Sprintf("unknown region %q", region),
			"supported: us, eu",
		)
	}
}

// DISPLAY is deliberately irrelevant: local macOS/Windows normally omit it.
func isSSHSession() bool {
	return os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != "" || os.Getenv("SSH_TTY") != ""
}
