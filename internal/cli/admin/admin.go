// Package admin provides admin-related CLI commands.
package admin

import (
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

var client ports.NylasClient

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
	cmd.AddCommand(newConnectorsCmd())
	cmd.AddCommand(newCredentialsCmd())
	cmd.AddCommand(newGrantsCmd())

	return cmd
}

func getClient() (ports.NylasClient, error) {
	if client != nil {
		return client, nil
	}

	// Use common client initialization which supports both keyring and env vars
	return common.GetNylasClient()
}
