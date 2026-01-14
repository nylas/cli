package admin

import (
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
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
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "list <connector-id>",
		Aliases: []string{"ls"},
		Short:   "List credentials for a connector",
		Long:    "List all authentication credentials for a specific connector.",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			credentials, err := client.ListCredentials(ctx, args[0])
			if err != nil {
				return common.WrapListError("credentials", err)
			}

			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(credentials)
			}

			if len(credentials) == 0 {
				common.PrintEmptyState("credentials")
				return nil
			}

			fmt.Printf("Found %d credential(s):\n\n", len(credentials))

			table := common.NewTable("NAME", "ID", "TYPE", "CREATED AT")
			for _, cred := range credentials {
				table.AddRow(common.Cyan.Sprint(cred.Name), cred.ID, common.Green.Sprint(cred.CredentialType), formatUnixTime(cred.CreatedAt))
			}
			table.Render()

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func newCredentialShowCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "show <credential-id>",
		Short: "Show credential details",
		Long:  "Show detailed information about a specific credential.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			credential, err := client.GetCredential(ctx, args[0])
			if err != nil {
				return common.WrapGetError("credential", err)
			}

			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(credential)
			}

			_, _ = common.Bold.Printf("Credential: %s\n", credential.Name)
			fmt.Printf("  ID: %s\n", common.Cyan.Sprint(credential.ID))
			fmt.Printf("  Connector ID: %s\n", credential.ConnectorID)
			fmt.Printf("  Type: %s\n", common.Green.Sprint(credential.CredentialType))
			fmt.Printf("  Created At: %s\n", formatUnixTime(credential.CreatedAt))
			fmt.Printf("  Updated At: %s\n", formatUnixTime(credential.UpdatedAt))

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func newCredentialCreateCmd() *cobra.Command {
	var (
		connectorID    string
		name           string
		credentialType string
		clientID       string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a credential",
		Long:  "Create a new authentication credential for a connector.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			req := &domain.CreateCredentialRequest{
				Name:           name,
				CredentialType: credentialType,
			}

			// Build credential data based on type
			if clientID != "" {
				req.CredentialData = map[string]any{
					"client_id": clientID,
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			credential, err := client.CreateCredential(ctx, connectorID, req)
			if err != nil {
				return common.WrapCreateError("credential", err)
			}

			_, _ = common.Green.Printf("Created credential: %s\n", credential.Name)
			fmt.Printf("  ID: %s\n", common.Cyan.Sprint(credential.ID))
			fmt.Printf("  Type: %s\n", credential.CredentialType)

			return nil
		},
	}

	cmd.Flags().StringVar(&connectorID, "connector-id", "", "Connector ID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Credential name (required)")
	cmd.Flags().StringVar(&credentialType, "type", "", "Credential type (oauth, service_account, connector) (required)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth client ID")

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
			client, err := getClient()
			if err != nil {
				return err
			}

			req := &domain.UpdateCredentialRequest{}

			if name != "" {
				req.Name = &name
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			credential, err := client.UpdateCredential(ctx, args[0], req)
			if err != nil {
				return common.WrapUpdateError("credential", err)
			}

			_, _ = common.Green.Printf("Updated credential: %s\n", credential.Name)

			return nil
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

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			if err := client.DeleteCredential(ctx, args[0]); err != nil {
				return common.WrapDeleteError("credential", err)
			}

			_, _ = common.Green.Printf("Deleted credential: %s\n", args[0])

			return nil
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
