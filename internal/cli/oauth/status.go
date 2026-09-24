package oauth

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func newStatusCmd() *cobra.Command {
	var remote bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the current OAuth session",
		Long: `Show the stored OAuth session without printing any token.

The access token's claims (audience, grants, scopes, expiry) are DECODED
locally and NOT VERIFIED: the CLI does not check the token's signature. They
describe what the stored token says about itself; only the resource server's
answer is authoritative. Use --verify to ask the authorization server.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := createLoginServiceFn()
			if err != nil {
				return wrapOAuthError(err)
			}

			session, err := svc.Status()
			if err != nil {
				return wrapOAuthError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			out := cmd.OutOrStdout()
			expired := session.Tokens.IsExpired(time.Now())
			if expired {
				_, _ = common.Yellow.Fprintln(out, "● Logged in (access token expired)")
			} else {
				_, _ = common.Green.Fprintln(out, "✓ Logged in")
			}

			_, _ = fmt.Fprintf(out, "  Issuer:     %s\n", session.Issuer)
			_, _ = fmt.Fprintf(out, "  Client ID:  %s\n", session.ClientID)
			if session.Tokens.Scope != "" {
				_, _ = fmt.Fprintf(out, "  Scopes:     %s\n", session.Tokens.Scope)
			}
			if !session.Tokens.ExpiresAt.IsZero() {
				_, _ = fmt.Fprintf(out, "  Expires:    %s\n", session.Tokens.ExpiresAt.Local().Format("2006-01-02 15:04:05 MST"))
			}
			_, _ = fmt.Fprintf(out, "  Refresh:    %s\n", presentAbsent(session.Tokens.RefreshToken != ""))
			_, _ = fmt.Fprintf(out, "  ID token:   %s\n", presentAbsent(session.Tokens.IDToken != ""))
			printTokenClaims(out, session.Tokens.AccessToken)

			if !remote {
				return nil
			}

			// --verify proves the token against the live server rather than
			// trusting what is on disk.
			info, err := svc.UserInfo(ctx)
			if err != nil {
				return wrapOAuthError(err)
			}
			_, _ = fmt.Fprintf(out, "  Subject:    %s\n", info.Subject)
			if info.Email != "" {
				_, _ = fmt.Fprintf(out, "  Email:      %s (verified: %t)\n", info.Email, info.EmailVerified)
			}
			if info.Org != "" {
				_, _ = fmt.Fprintf(out, "  Org:        %s\n", info.Org)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&remote, "verify", false,
		"call the userinfo endpoint to confirm the token is accepted")

	return cmd
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
