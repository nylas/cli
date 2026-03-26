// Package dashboard provides the CLI commands for Nylas Dashboard
// account authentication and application management.
package dashboard

import (
	"github.com/spf13/cobra"
)

// NewDashboardCmd creates the dashboard command group.
func NewDashboardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Nylas Dashboard account and application management",
		Long: `Authenticate with the Nylas Dashboard and manage applications.

Commands:
  register   Create a new Nylas Dashboard account (SSO only)
  login      Log in to your Nylas Dashboard account
  sso        Authenticate via SSO (Google, Microsoft, GitHub)
  logout     Log out of the Nylas Dashboard
  status     Show current dashboard authentication status
  refresh    Refresh dashboard session tokens
  apps       Manage Nylas applications
  orgs       Manage organizations (list, switch)`,
	}

	cmd.AddCommand(newRegisterCmd())
	cmd.AddCommand(newLoginCmd())
	cmd.AddCommand(newSSOCmd())
	cmd.AddCommand(newLogoutCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newRefreshCmd())
	cmd.AddCommand(newAppsCmd())
	cmd.AddCommand(newOrgsCmd())

	return cmd
}
