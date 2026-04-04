// Package admin provides admin-related CLI commands.
package admin

import (
	"github.com/spf13/cobra"
)

// NewAdminCmd creates the admin command group.
func NewAdminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Administration commands (requires API key)",
		Long: `Administration commands for managing applications, connectors, credentials, and grants.

These commands require API key authentication and are used for managing
the Nylas platform at an organizational level.`,
	}

	cmd.AddCommand(newApplicationsCmd())
	cmd.AddCommand(newCallbackURIsCmd())
	cmd.AddCommand(newConnectorsCmd())
	cmd.AddCommand(newCredentialsCmd())
	cmd.AddCommand(newGrantsCmd())

	return cmd
}
