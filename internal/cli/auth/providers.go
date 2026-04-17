package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
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
			connectors = common.FilterVisibleConnectors(connectors)

			if outputJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(connectors)
			}

			renderProviders(cmd.OutOrStdout(), connectors)
			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}

func renderProviders(w io.Writer, connectors []domain.Connector) {
	_, _ = fmt.Fprintln(w, "Available Authentication Providers:")
	_, _ = fmt.Fprintln(w)

	if len(connectors) == 0 {
		_, _ = fmt.Fprintln(w, "No providers configured.")
		_, _ = fmt.Fprintln(w, "\nTo add a provider, use: nylas admin connectors create")
		return
	}

	for _, connector := range connectors {
		title := connector.Name
		if title == "" {
			title = providerDisplayName(connector.Provider)
		}

		_, _ = fmt.Fprintf(w, "  %s\n", title)
		_, _ = fmt.Fprintf(w, "    Provider:   %s\n", connector.Provider)
		if connector.Name != "" && connector.Name != title {
			_, _ = fmt.Fprintf(w, "    Name:       %s\n", connector.Name)
		}
		if connector.ID != "" {
			_, _ = fmt.Fprintf(w, "    ID:         %s\n", connector.ID)
		}
		if len(connector.Scopes) > 0 {
			_, _ = fmt.Fprintf(w, "    Scopes:     %d configured\n", len(connector.Scopes))
		}
		_, _ = fmt.Fprintln(w)
	}
}

func providerDisplayName(provider string) string {
	switch provider {
	case "google":
		return "Google"
	case "microsoft":
		return "Microsoft"
	case "imap":
		return "IMAP"
	case "icloud":
		return "iCloud"
	case "ews":
		return "EWS"
	case "virtual-calendar":
		return "Virtual Calendar"
	default:
		return titleProviderName(provider)
	}
}

func titleProviderName(provider string) string {
	normalized := strings.ReplaceAll(strings.ReplaceAll(provider, "-", " "), "_", " ")
	parts := strings.Fields(normalized)
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
