package auth

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

func newProvidersCmd() *cobra.Command {
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "providers",
		Short: "List available authentication providers",
		Long: `List all available authentication providers (connectors).

Providers represent the different email/calendar services that Nylas can connect to:
- Google (Gmail, Google Workspace)
- Microsoft (Outlook, Office 365)
- iCloud
- Yahoo
- IMAP (Custom email servers)

This command shows connectors configured for your Nylas application.`,
		Example: `  # List all providers
  nylas auth providers

  # Output as JSON
  nylas auth providers --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := common.CreateContext()
			defer cancel()

			client, err := common.GetNylasClient()
			if err != nil {
				return err
			}

			connectors, err := client.ListConnectors(ctx)
			if err != nil {
				return common.WrapFetchError("providers", err)
			}

			if outputJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(connectors)
			}

			// Display as table
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Available Authentication Providers:")
			_, _ = fmt.Fprintln(cmd.OutOrStdout())

			if len(connectors) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No providers configured.")
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nTo add a provider, use: nylas admin connectors create")
				return nil
			}

			for _, connector := range connectors {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", connector.Provider)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    Name:       %s\n", connector.Name)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    ID:         %s\n", connector.ID)
				if len(connector.Scopes) > 0 {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    Scopes:     %d configured\n", len(connector.Scopes))
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}
