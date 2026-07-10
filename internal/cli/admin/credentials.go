package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

// resolveConnectorID returns the explicit connector provider when given
// (rejecting deprecated ones), otherwise auto-detects it when the application
// has exactly one connector. Connector credentials are keyed by provider (e.g.
// "google") in the /v3/connectors/{provider}/creds path, so a provider is always
// required. The resolution policy lives in domain.ResolveConnectorProvider so the
// CLI and RPC surfaces cannot diverge; this wrapper only supplies the connector
// list and maps errors to CLI user errors.
func resolveConnectorID(ctx context.Context, client ports.NylasClient, explicit string) (string, error) {
	connectors, listErr := client.ListConnectors(ctx)

	provider, err := domain.ResolveConnectorProvider(connectors, explicit)
	if err == nil {
		return provider, nil
	}

	// Discovery failed: surface the real listing error rather than a misleading
	// "no connectors found" (empty input) or "unknown connector" (an explicit
	// legacy ID we simply couldn't map because the list was unavailable).
	if listErr != nil && (explicit == "" || errors.Is(err, domain.ErrUnknownConnector)) {
		return "", fmt.Errorf("resolve connector: %w", listErr)
	}

	var multi *domain.MultipleConnectorsError
	switch {
	case errors.As(err, &multi):
		return "", common.NewUserError(
			fmt.Sprintf("multiple connectors found (%s); specify which one", strings.Join(multi.Providers, ", ")),
			"Pass the provider with --connector (e.g. --connector google)",
		)
	case errors.Is(err, domain.ErrDeprecatedConnector):
		return "", common.NewUserError(
			fmt.Sprintf("connector provider %q is no longer supported", explicit),
			"Choose a supported connector (e.g. --connector google)",
		)
	case errors.Is(err, domain.ErrUnknownConnector):
		return "", common.NewUserError(
			fmt.Sprintf("unknown connector provider %q", explicit),
			"Use a supported provider (google, microsoft, imap, icloud, yahoo, ews, virtual-calendar, zoom, nylas)",
		)
	default: // domain.ErrNoConnectors
		return "", common.NewUserError(
			"no connectors found",
			"Create a connector first, or pass the provider with --connector (e.g. --connector google)",
		)
	}
}

func newCredentialsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "credentials",
		Aliases: []string{"credential", "cred"},
		Short:   "Manage connector credentials",
		Long: `Manage authentication credentials for connectors (OAuth, service accounts, etc.).

API reference: https://developer.nylas.com/docs/reference/api/connector-credentials/`,
	}

	cmd.AddCommand(newCredentialListCmd())
	cmd.AddCommand(newCredentialShowCmd())
	cmd.AddCommand(newCredentialCreateCmd())
	cmd.AddCommand(newCredentialUpdateCmd())
	cmd.AddCommand(newCredentialDeleteCmd())

	return cmd
}

func newCredentialListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [connector]",
		Aliases: []string{"ls"},
		Short:   "List credentials for a connector",
		Long:    "List all authentication credentials for a connector. The connector\nprovider is auto-detected when the application has exactly one connector.",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			explicit := ""
			if len(args) > 0 {
				explicit = args[0]
			}
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				connectorID, err := resolveConnectorID(ctx, client, explicit)
				if err != nil {
					return struct{}{}, err
				}
				credentials, err := client.ListCredentials(ctx, connectorID)
				if err != nil {
					return struct{}{}, common.WrapListError("credentials", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(credentials)
				}

				if len(credentials) == 0 {
					common.PrintEmptyState("credentials")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d credential(s):\n\n", len(credentials))

				table := common.NewTable("NAME", "ID", "CREATED AT")
				for _, cred := range credentials {
					table.AddRow(common.Cyan.Sprint(cred.Name), cred.ID, formatUnixTime(cred.CreatedAt))
				}
				table.Render()

				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}

func newCredentialShowCmd() *cobra.Command {
	var connector string
	cmd := &cobra.Command{
		Use:   "show <credential-id>",
		Short: "Show credential details",
		Long:  "Show detailed information about a specific credential. The connector\nprovider is auto-detected when the application has exactly one connector.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			credentialID := args[0]
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				connectorID, err := resolveConnectorID(ctx, client, connector)
				if err != nil {
					return struct{}{}, err
				}
				credential, err := client.GetCredential(ctx, connectorID, credentialID)
				if err != nil {
					return struct{}{}, common.WrapGetError("credential", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(credential)
				}

				_, _ = common.Bold.Printf("Credential: %s\n", credential.Name)
				fmt.Printf("  ID: %s\n", common.Cyan.Sprint(credential.ID))
				fmt.Printf("  Created At: %s\n", formatUnixTime(credential.CreatedAt))
				fmt.Printf("  Updated At: %s\n", formatUnixTime(credential.UpdatedAt))

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&connector, "connector", "", "Connector provider (e.g. google); auto-detected if only one connector exists")

	return cmd
}

func newCredentialCreateCmd() *cobra.Command {
	var (
		connectorID    string
		name           string
		credentialType string
		clientID       string
		clientSecret   string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a credential",
		Long: `Create a connector credential from a provider OAuth application.

--client-id and --client-secret are the PROVIDER's OAuth app credentials (e.g.
your own Google Cloud project or Azure app) — NOT your Nylas application's. Nylas
uses them to broker authentication through that provider app (for example, to
authenticate enterprise customers who own their provider application, or to run
your own app alongside the Nylas Shared GCP App).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				connectorProvider, rerr := resolveConnectorID(ctx, client, connectorID)
				if rerr != nil {
					return struct{}{}, rerr
				}

				req := &domain.CreateCredentialRequest{
					Name:           name,
					CredentialType: credentialType,
				}

				// Build credential data based on type
				if clientID != "" || clientSecret != "" {
					req.CredentialData = map[string]any{
						"client_id":     clientID,
						"client_secret": clientSecret,
					}
				}

				credential, err := client.CreateCredential(ctx, connectorProvider, req)
				if err != nil {
					return struct{}{}, common.WrapCreateError("credential", err)
				}

				_, _ = common.Green.Printf("Created credential: %s\n", credential.Name)
				fmt.Printf("  ID: %s\n", common.Cyan.Sprint(credential.ID))

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&connectorID, "connector", "", "Connector provider (e.g. google); auto-detected if only one connector exists")
	cmd.Flags().StringVar(&connectorID, "connector-id", "", "")
	_ = cmd.Flags().MarkDeprecated("connector-id", "use --connector")
	// Both bind the same target; forbid passing both so one doesn't silently win.
	cmd.MarkFlagsMutuallyExclusive("connector", "connector-id")
	cmd.Flags().StringVar(&name, "name", "", "Credential name (required)")
	// Valid v3 credential_type values are connector, serviceaccount, and
	// adminconsent. This command builds credential_data only from
	// --client-id/--client-secret, which is the `connector` override flow;
	// serviceaccount/adminconsent need credential_data this command doesn't
	// collect yet, so only `connector` is advertised.
	//
	// --client-id/--client-secret are the PROVIDER's OAuth app credentials (e.g.
	// a Google Cloud / Azure app), NOT your Nylas application's — they populate
	// credential_data so Nylas can broker auth through that provider app.
	cmd.Flags().StringVar(&credentialType, "type", "", "Credential type: connector (required)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "Provider OAuth app client ID (e.g. your Google Cloud / Azure app), required")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "Provider OAuth app client secret, required")

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("type")
	// The v3 create request requires credential_data; this command builds it from
	// the client ID/secret, so both are required for the connector flow.
	_ = cmd.MarkFlagRequired("client-id")
	_ = cmd.MarkFlagRequired("client-secret")

	return cmd
}

func newCredentialUpdateCmd() *cobra.Command {
	var (
		name      string
		connector string
	)

	cmd := &cobra.Command{
		Use:   "update <credential-id>",
		Short: "Update a credential",
		Long:  "Update an existing credential. The connector provider is auto-detected\nwhen the application has exactly one connector.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			credentialID := args[0]
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				connectorID, err := resolveConnectorID(ctx, client, connector)
				if err != nil {
					return struct{}{}, err
				}

				req := &domain.UpdateCredentialRequest{}

				if name != "" {
					req.Name = &name
				}

				credential, err := client.UpdateCredential(ctx, connectorID, credentialID, req)
				if err != nil {
					return struct{}{}, common.WrapUpdateError("credential", err)
				}

				common.PrintUpdateSuccess("credential", credential.Name)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Credential name")
	cmd.Flags().StringVar(&connector, "connector", "", "Connector provider (e.g. google); auto-detected if only one connector exists")

	return cmd
}

func newCredentialDeleteCmd() *cobra.Command {
	var (
		yes       bool
		connector string
	)

	cmd := &cobra.Command{
		Use:   "delete <credential-id>",
		Short: "Delete a credential",
		Long:  "Delete a credential permanently. The connector provider is auto-detected\nwhen the application has exactly one connector.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			credentialID := args[0]
			if !yes {
				if !common.Confirm(fmt.Sprintf("Are you sure you want to delete credential %s?", credentialID), false) {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				connectorID, err := resolveConnectorID(ctx, client, connector)
				if err != nil {
					return struct{}{}, err
				}

				if err := client.DeleteCredential(ctx, connectorID, credentialID); err != nil {
					return struct{}{}, common.WrapDeleteError("credential", err)
				}

				_, _ = common.Green.Printf("Deleted credential: %s\n", credentialID)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&connector, "connector", "", "Connector provider (e.g. google); auto-detected if only one connector exists")

	return cmd
}

// formatUnixTime formats a UnixTime pointer to a human-readable string
func formatUnixTime(t *domain.UnixTime) string {
	if t == nil || t.IsZero() {
		return "-"
	}
	return t.Format(common.DisplayDateTime)
}
