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

func newConnectorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "connectors",
		Aliases: []string{"connector", "conn"},
		Short:   "Manage email provider connectors",
		Long:    "Manage email provider connectors (Google, Microsoft, IMAP, etc.).",
	}

	cmd.AddCommand(newConnectorListCmd())
	cmd.AddCommand(newConnectorShowCmd())
	cmd.AddCommand(newConnectorCreateCmd())
	cmd.AddCommand(newConnectorUpdateCmd())
	cmd.AddCommand(newConnectorDeleteCmd())

	return cmd
}

func newConnectorListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List connectors",
		Long:    "List all email provider connectors.",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				connectors, err := client.ListConnectors(ctx)
				if err != nil {
					return struct{}{}, common.WrapListError("connectors", err)
				}
				connectors = common.FilterVisibleConnectors(connectors)

				if common.IsJSON(cmd) {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(connectors)
				}

				if len(connectors) == 0 {
					common.PrintEmptyState("connectors")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d connector(s):\n\n", len(connectors))

				table := common.NewTable("NAME", "ID", "PROVIDER", "SCOPES")
				for _, conn := range connectors {
					scopeCount := fmt.Sprintf("%d", len(conn.Scopes))
					table.AddRow(common.Cyan.Sprint(conn.Name), conn.ID, common.Green.Sprint(conn.Provider), scopeCount)
				}
				table.Render()

				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}

func newConnectorShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <connector-id>",
		Short: "Show connector details",
		Long:  "Show detailed information about a specific connector.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			connectorID := args[0]
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				connector, err := client.GetConnector(ctx, connectorID)
				if err != nil {
					return struct{}{}, common.WrapGetError("connector", err)
				}
				if common.IsDeprecatedConnectorProvider(connector.Provider) {
					return struct{}{}, common.NewUserError("connector not found", "The inbox connector is no longer supported")
				}

				if common.IsJSON(cmd) {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(connector)
				}

				// #nosec G104 -- color output errors are non-critical, best-effort display
				_, _ = common.Bold.Printf("Connector: %s\n", connector.Name)
				fmt.Printf("  ID: %s\n", common.Cyan.Sprint(connector.ID))
				fmt.Printf("  Provider: %s\n", common.Green.Sprint(connector.Provider))

				if len(connector.Scopes) > 0 {
					fmt.Printf("\nScopes (%d):\n", len(connector.Scopes))
					for i, scope := range connector.Scopes {
						fmt.Printf("  %d. %s\n", i+1, scope)
					}
				}

				if connector.Settings != nil {
					fmt.Printf("\nSettings:\n")
					if connector.Settings.ClientID != "" {
						fmt.Printf("  Client ID: %s\n", connector.Settings.ClientID)
					}
					if connector.Settings.IMAPHost != "" {
						fmt.Printf("  IMAP Host: %s\n", connector.Settings.IMAPHost)
						fmt.Printf("  IMAP Port: %d\n", connector.Settings.IMAPPort)
					}
					if connector.Settings.SMTPHost != "" {
						fmt.Printf("  SMTP Host: %s\n", connector.Settings.SMTPHost)
						fmt.Printf("  SMTP Port: %d\n", connector.Settings.SMTPPort)
					}
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}

func newConnectorCreateCmd() *cobra.Command {
	var (
		name         string
		provider     string
		clientID     string
		clientSecret string
		scopes       []string
		imapHost     string
		imapPort     int
		smtpHost     string
		smtpPort     int
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a connector",
		Long:  "Create a new email provider connector.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := common.ValidateSupportedConnectorProvider(provider); err != nil {
				return err
			}

			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				req := &domain.CreateConnectorRequest{
					Name:     name,
					Provider: provider,
				}

				if clientID != "" || clientSecret != "" || imapHost != "" {
					req.Settings = &domain.ConnectorSettings{
						ClientID:     clientID,
						ClientSecret: clientSecret,
						IMAPHost:     imapHost,
						IMAPPort:     imapPort,
						SMTPHost:     smtpHost,
						SMTPPort:     smtpPort,
					}
				}

				if len(scopes) > 0 {
					req.Scopes = scopes
				}

				connector, err := client.CreateConnector(ctx, req)
				if err != nil {
					return struct{}{}, common.WrapCreateError("connector", err)
				}

				// #nosec G104 -- color output errors are non-critical, best-effort display
				_, _ = common.Green.Printf("✓ Created connector: %s\n", connector.Name)
				fmt.Printf("  ID: %s\n", common.Cyan.Sprint(connector.ID))
				fmt.Printf("  Provider: %s\n", connector.Provider)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Connector name (required)")
	cmd.Flags().StringVar(&provider, "provider", "", "Provider (google, microsoft, imap, etc.) (required)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth client ID")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth client secret")
	cmd.Flags().StringSliceVar(&scopes, "scopes", []string{}, "OAuth scopes (comma-separated)")
	cmd.Flags().StringVar(&imapHost, "imap-host", "", "IMAP host (for IMAP provider)")
	cmd.Flags().IntVar(&imapPort, "imap-port", 993, "IMAP port")
	cmd.Flags().StringVar(&smtpHost, "smtp-host", "", "SMTP host (for IMAP provider)")
	cmd.Flags().IntVar(&smtpPort, "smtp-port", 587, "SMTP port")

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("provider")

	return cmd
}

func newConnectorUpdateCmd() *cobra.Command {
	var (
		name   string
		scopes []string
	)

	cmd := &cobra.Command{
		Use:   "update <connector-id>",
		Short: "Update a connector",
		Long:  "Update an existing connector.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			connectorID := args[0]
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				req := &domain.UpdateConnectorRequest{}

				if name != "" {
					req.Name = &name
				}

				if len(scopes) > 0 {
					req.Scopes = scopes
				}

				connector, err := client.UpdateConnector(ctx, connectorID, req)
				if err != nil {
					return struct{}{}, common.WrapUpdateError("connector", err)
				}

				common.PrintUpdateSuccess("connector", connector.Name)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Connector name")
	cmd.Flags().StringSliceVar(&scopes, "scopes", []string{}, "OAuth scopes (comma-separated)")

	return cmd
}

func newConnectorDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <connector-id>",
		Short: "Delete a connector",
		Long:  "Delete a connector permanently.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Printf("Are you sure you want to delete connector %s? (y/N): ", args[0])
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			connectorID := args[0]
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				if err := client.DeleteConnector(ctx, connectorID); err != nil {
					return struct{}{}, common.WrapDeleteError("connector", err)
				}

				_, _ = common.Green.Printf("✓ Deleted connector: %s\n", connectorID)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
