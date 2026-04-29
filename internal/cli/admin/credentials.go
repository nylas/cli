package admin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newCredentialsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "credentials",
		Aliases: []string{"credential", "cred"},
		Short:   "Manage connector credentials",
		Long:    "Manage authentication credentials for connectors (OAuth, service accounts, etc.).",
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
		Use:     "list <connector-id>",
		Aliases: []string{"ls"},
		Short:   "List credentials for a connector",
		Long:    "List all authentication credentials for a specific connector.",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			connectorID := args[0]
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
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

				table := common.NewTable("NAME", "ID", "TYPE", "CREATED AT")
				for _, cred := range credentials {
					table.AddRow(common.Cyan.Sprint(cred.Name), cred.ID, common.Green.Sprint(cred.CredentialType), formatUnixTime(cred.CreatedAt))
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
	cmd := &cobra.Command{
		Use:   "show <credential-id>",
		Short: "Show credential details",
		Long:  "Show detailed information about a specific credential.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			credentialID := args[0]
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				credential, err := client.GetCredential(ctx, credentialID)
				if err != nil {
					return struct{}{}, common.WrapGetError("credential", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(credential)
				}

				_, _ = common.Bold.Printf("Credential: %s\n", credential.Name)
				fmt.Printf("  ID: %s\n", common.Cyan.Sprint(credential.ID))
				fmt.Printf("  Connector ID: %s\n", credential.ConnectorID)
				fmt.Printf("  Type: %s\n", common.Green.Sprint(credential.CredentialType))
				fmt.Printf("  Created At: %s\n", formatUnixTime(credential.CreatedAt))
				fmt.Printf("  Updated At: %s\n", formatUnixTime(credential.UpdatedAt))

				return struct{}{}, nil
			})
			return err
		},
	}

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
		Long:  "Create a new authentication credential for a connector.",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
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

				credential, err := client.CreateCredential(ctx, connectorID, req)
				if err != nil {
					return struct{}{}, common.WrapCreateError("credential", err)
				}

				_, _ = common.Green.Printf("Created credential: %s\n", credential.Name)
				fmt.Printf("  ID: %s\n", common.Cyan.Sprint(credential.ID))
				fmt.Printf("  Type: %s\n", credential.CredentialType)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&connectorID, "connector-id", "", "Connector ID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Credential name (required)")
	cmd.Flags().StringVar(&credentialType, "type", "", "Credential type (oauth, service_account, connector) (required)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth client ID")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth client secret")

	_ = cmd.MarkFlagRequired("connector-id")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}

func newCredentialUpdateCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "update <credential-id>",
		Short: "Update a credential",
		Long:  "Update an existing credential.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			credentialID := args[0]
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				req := &domain.UpdateCredentialRequest{}

				if name != "" {
					req.Name = &name
				}

				credential, err := client.UpdateCredential(ctx, credentialID, req)
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

	return cmd
}

func newCredentialDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <credential-id>",
		Short: "Delete a credential",
		Long:  "Delete a credential permanently.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Printf("Are you sure you want to delete credential %s? (y/N): ", args[0])
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			credentialID := args[0]
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				if err := client.DeleteCredential(ctx, credentialID); err != nil {
					return struct{}{}, common.WrapDeleteError("credential", err)
				}

				_, _ = common.Green.Printf("Deleted credential: %s\n", credentialID)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

// formatUnixTime formats a UnixTime pointer to a human-readable string
func formatUnixTime(t *domain.UnixTime) string {
	if t == nil || t.IsZero() {
		return "-"
	}
	return t.Format(common.DisplayDateTime)
}
