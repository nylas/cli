package oauth

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/app/oauthlogin"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func newStatusCmd() *cobra.Command {
	var (
		remote     bool
		credential string
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the stored OAuth sessions",
		Long: `Show the stored OAuth sessions without printing any token.

The default session is the one 'nylas oauth login' stores and the dashboard
commands use. MCP-only credentials from 'nylas oauth login --mcp-only' are
listed after it, each with the id 'nylas mcp serve --auth oauth --credential'
takes, the MCP server it is for, its organization and the IAM principal it is
limited to, if any. They never give dashboard access.

The access token's claims (audience, principal, grants, scopes, expiry) are
DECODED locally and NOT VERIFIED: the CLI does not check the token's
signature. They describe what the stored token says about itself; only the
resource server's answer is authoritative. Use --verify to ask the
authorization server.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := createLoginServiceFn()
			if err != nil {
				return wrapOAuthError(err)
			}
			out := cmd.OutOrStdout()

			if credential != "" {
				bound, err := svc.ForMCPCredential(credential)
				if err == nil {
					err = showSession(cmd, bound, remote)
				}
				return wrapOAuthError(err)
			}

			records, err := svc.ListMCPCredentials()
			if err != nil {
				return wrapOAuthError(err)
			}
			if err := showSession(cmd, svc, remote); err != nil {
				if !errors.Is(err, domain.ErrOAuthNotLoggedIn) || len(records) == 0 {
					return wrapOAuthError(err)
				}
				_, _ = fmt.Fprintln(out, "No default session (the dashboard commands are not signed in by OAuth)")
			}

			for _, record := range records {
				_, _ = fmt.Fprintf(out, "\nMCP credential %s\n", record.ID)
				bound, err := svc.ForMCPCredential(record.ID)
				if err == nil {
					err = showSession(cmd, bound, false)
				}
				if err != nil {
					_, _ = common.Yellow.Fprintf(out, "  unavailable: %v\n", err)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&remote, "verify", false,
		"call the userinfo endpoint to confirm the token is accepted")
	cmd.Flags().StringVar(&credential, "credential", "",
		"show only this MCP-only credential (an id from 'nylas oauth status')")

	return cmd
}

// showSession prints one stored session, and with remote proves its token
// against the live server rather than trusting what is on disk.
func showSession(cmd *cobra.Command, svc loginService, remote bool) error {
	session, err := svc.Status()
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	if session.Tokens.IsExpired(time.Now()) {
		_, _ = common.Yellow.Fprintln(out, "● Logged in (access token expired)")
	} else {
		_, _ = common.Green.Fprintln(out, "✓ Logged in")
	}

	_, _ = fmt.Fprintf(out, "  Access:     %s\n", accessMode(session))
	_, _ = fmt.Fprintf(out, "  Issuer:     %s\n", session.Issuer)
	_, _ = fmt.Fprintf(out, "  Client ID:  %s\n", session.ClientID)
	if session.Tokens.Scope != "" {
		_, _ = fmt.Fprintf(out, "  Scopes:     %s\n", session.Tokens.Scope)
	}
	if !session.Tokens.ExpiresAt.IsZero() {
		_, _ = fmt.Fprintf(out, "  Expires:    %s\n", session.Tokens.ExpiresAt.Local().Format("2006-01-02 15:04:05 MST"))
	}
	_, _ = fmt.Fprintf(out, "  Refresh:    %s\n", presentAbsent(session.Tokens.RefreshToken != ""))
	printTokenClaims(out, session.Tokens.AccessToken)

	if !remote {
		return nil
	}
	ctx, cancel := common.CreateContext()
	defer cancel()
	info, err := svc.UserInfo(ctx)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "  Subject:    %s\n", info.Subject)
	if info.Email != "" {
		_, _ = fmt.Fprintf(out, "  Email:      %s (verified: %t)\n", info.Email, info.EmailVerified)
	}
	if info.Org != "" {
		_, _ = fmt.Fprintf(out, "  Org:        %s\n", info.Org)
	}
	return nil
}

// accessMode says what a stored session can be used for, from its granted
// scopes and the principal its token names.
func accessMode(session *oauthlogin.Session) string {
	if claims, err := domain.DecodeOAuthAccessToken(session.Tokens.AccessToken); err == nil && claims.Principal != nil {
		return "MCP only, limited to an IAM principal (no dashboard access)"
	}
	if slices.Contains(strings.Fields(session.Tokens.Scope), domain.OAuthScopeDashboardSession) {
		return "dashboard commands (your full role) and the MCP server"
	}
	if session.Resource != "" {
		return "MCP only (no dashboard access)"
	}
	return "identity only (no dashboard access)"
}

// printTokenClaims shows what the access token says about itself. It never
// prints the token, and a token that does not decode is reported, not fatal:
// the session is still usable, the server is what reads the token.
func printTokenClaims(out io.Writer, accessToken string) {
	_, _ = fmt.Fprintln(out, "  Token claims (decoded locally, signature NOT verified):")
	claims, err := domain.DecodeOAuthAccessToken(accessToken)
	if err != nil {
		_, _ = fmt.Fprintf(out, "    unavailable: %v\n", err)
		return
	}

	audience := "(none)"
	if len(claims.Audience) > 0 {
		audience = strings.Join(claims.Audience, ", ")
	}
	_, _ = fmt.Fprintf(out, "    Audience: %s\n", audience)
	if claims.Org != "" {
		_, _ = fmt.Fprintf(out, "    Org:      %s\n", claims.Org)
	}
	_, _ = fmt.Fprintf(out, "    Principal: %s\n", describePrincipal(claims.Principal))

	scopes := "(none)"
	if claims.Scope != "" {
		scopes = claims.Scope
	}
	_, _ = fmt.Fprintf(out, "    Scopes:   %s\n", scopes)

	if claims.ExpiresAt.IsZero() {
		_, _ = fmt.Fprintln(out, "    Expires:  (no exp claim)")
	} else {
		_, _ = fmt.Fprintf(out, "    Expires:  %s\n", claims.ExpiresAt.Local().Format("2006-01-02 15:04:05 MST"))
	}

	if len(claims.Grants) == 0 {
		_, _ = fmt.Fprintln(out, "    Grants:   (none)")
		return
	}
	_, _ = fmt.Fprintf(out, "    Grants:   %d\n", len(claims.Grants))
	for _, grant := range claims.Grants {
		_, _ = fmt.Fprintf(out, "      - %s (application %s)\n", grant.ID, grant.ApplicationID)
	}
}

func presentAbsent(present bool) string {
	if present {
		return "present"
	}
	return "absent"
}
