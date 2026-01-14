package auth

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
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

			client, err := getProvidersClient()
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

func getProvidersClient() (ports.NylasClient, error) {
	configStore := config.NewDefaultFileStore()
	cfg, err := configStore.Load()
	if err != nil {
		cfg = &domain.Config{Region: "us"}
	}

	// Check environment variables first (highest priority)
	apiKey := os.Getenv("NYLAS_API_KEY")
	clientID := os.Getenv("NYLAS_CLIENT_ID")
	clientSecret := os.Getenv("NYLAS_CLIENT_SECRET")

	// If API key not in env, try keyring/file store
	if apiKey == "" {
		secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
		if err == nil {
			apiKey, _ = secretStore.Get(ports.KeyAPIKey)
			if clientID == "" {
				clientID, _ = secretStore.Get(ports.KeyClientID)
			}
			if clientSecret == "" {
				clientSecret, _ = secretStore.Get(ports.KeyClientSecret)
			}
		}
	}

	if apiKey == "" {
		return nil, fmt.Errorf("API key not configured. Set NYLAS_API_KEY environment variable or run 'nylas auth config'")
	}

	c := nylas.NewHTTPClient()
	c.SetRegion(cfg.Region)
	c.SetCredentials(clientID, clientSecret, apiKey)

	return c, nil
}
