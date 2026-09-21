package oauth

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

func newStatusCmd() *cobra.Command {
	var remote bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the current OAuth session",
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

func presentAbsent(present bool) string {
	if present {
		return "present"
	}
	return "absent"
}
