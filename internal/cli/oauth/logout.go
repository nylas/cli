package oauth

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

func newLogoutCmd() *cobra.Command {
	var (
		credential string
		all        bool
	)

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Revoke the OAuth session and clear stored tokens",
		Long: `Revoke the refresh token and remove the stored session.

Revoking the refresh token takes the whole token family with it. Local
tokens are cleared even when the server cannot be reached. A dashboard
session that came from 'nylas oauth login' is ended too; one from
'nylas dashboard login' is left alone.

Without flags this logs out of the default session and leaves MCP-only
credentials ('nylas oauth login --mcp-only') in place. --credential logs out
of one of those instead, and --all logs out of the default session and every
MCP-only credential.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if credential != "" && all {
				return common.NewUserError("--credential and --all cannot be combined", "")
			}
			svc, err := createLoginServiceFn()
			if err != nil {
				return wrapOAuthError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			if credential != "" {
				if err := svc.LogoutMCPCredential(ctx, credential); err != nil {
					return wrapOAuthError(err)
				}
				_, _ = common.Green.Fprintf(cmd.OutOrStdout(), "✓ Logged out of MCP credential %s\n", credential)
				return nil
			}

			dashboardErr, logoutErr := clearDashboardSessionFn(ctx, func() error {
				return svc.Logout(ctx)
			})
			if all {
				records, err := svc.ListMCPCredentials()
				logoutErr = errors.Join(logoutErr, err)
				for _, record := range records {
					logoutErr = errors.Join(logoutErr, svc.LogoutMCPCredential(ctx, record.ID))
				}
			}
			if logoutErr != nil {
				return wrapOAuthError(logoutErr)
			}
			if structuredOutput(cmd) {
				report := logoutReport{LoggedOut: true}
				if dashboardErr != nil {
					report.DashboardError = dashboardErr.Error()
				}
				return common.GetOutputWriter(cmd).Write(report)
			}
			if dashboardErr != nil {
				_, _ = common.Yellow.Fprintf(cmd.OutOrStdout(),
					"  Could not end the dashboard session: %v\n", dashboardErr)
			}

			_, _ = common.Green.Fprintln(cmd.OutOrStdout(), "✓ Logged out")
			return nil
		},
	}

	cmd.Flags().StringVar(&credential, "credential", "",
		"log out of only this MCP-only credential (an id from 'nylas oauth status')")
	cmd.Flags().BoolVar(&all, "all", false,
		"log out of the default session and every MCP-only credential")
	return cmd
}
