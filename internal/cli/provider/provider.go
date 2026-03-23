// Package provider implements CLI commands for managing provider integrations.
package provider

import "github.com/spf13/cobra"

// NewProviderCmd creates the 'provider' command group.
func NewProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider",
		Short: "Manage provider integrations",
		Long:  "Set up and manage provider integrations (Google, Microsoft, etc.)",
	}

	cmd.AddCommand(newSetupCmd())
	cmd.AddCommand(newStatusCmd())

	return cmd
}
