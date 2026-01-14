package admin

import (
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
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
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List connectors",
		Long:    "List all email provider connectors.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			connectors, err := client.ListConnectors(ctx)
			if err != nil {
				return common.WrapListError("connectors", err)
			}

			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(connectors)
			}

			if len(connectors) == 0 {
				common.PrintEmptyState("connectors")
				return nil
			}

			fmt.Printf("Found %d connector(s):\n\n", len(connectors))

			table := common.NewTable("NAME", "ID", "PROVIDER", "SCOPES")
			for _, conn := range connectors {
				scopeCount := fmt.Sprintf("%d", len(conn.Scopes))
				table.AddRow(common.Cyan.Sprint(conn.Name), conn.ID, common.Green.Sprint(conn.Provider), scopeCount)
			}
			table.Render()

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func newConnectorShowCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "show <connector-id>",
		Short: "Show connector details",
		Long:  "Show detailed information about a specific connector.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			connector, err := client.GetConnector(ctx, args[0])
			if err != nil {
				return common.WrapGetError("connector", err)
			}

			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(connector)
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

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func newConnectorCreateCmd() *cobra.Command {
	var (
		name     string
		provider string
		clientID string
		scopes   []string
		imapHost string
		imapPort int
		smtpHost string
		smtpPort int
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a connector",
		Long:  "Create a new email provider connector.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			req := &domain.CreateConnectorRequest{
				Name:     name,
				Provider: provider,
			}

			if clientID != "" || imapHost != "" {
				req.Settings = &domain.ConnectorSettings{
					ClientID: clientID,
					IMAPHost: imapHost,
					IMAPPort: imapPort,
					SMTPHost: smtpHost,
					SMTPPort: smtpPort,
				}
			}

			if len(scopes) > 0 {
				req.Scopes = scopes
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			connector, err := client.CreateConnector(ctx, req)
			if err != nil {
				return common.WrapCreateError("connector", err)
			}

			// #nosec G104 -- color output errors are non-critical, best-effort display
			_, _ = common.Green.Printf("✓ Created connector: %s\n", connector.Name)
			fmt.Printf("  ID: %s\n", common.Cyan.Sprint(connector.ID))
			fmt.Printf("  Provider: %s\n", connector.Provider)

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Connector name (required)")
	cmd.Flags().StringVar(&provider, "provider", "", "Provider (google, microsoft, imap, etc.) (required)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth client ID")
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
			client, err := getClient()
			if err != nil {
				return err
			}

			req := &domain.UpdateConnectorRequest{}

			if name != "" {
				req.Name = &name
			}

			if len(scopes) > 0 {
				req.Scopes = scopes
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			connector, err := client.UpdateConnector(ctx, args[0], req)
			if err != nil {
				return common.WrapUpdateError("connector", err)
			}

			_, _ = common.Green.Printf("✓ Updated connector: %s\n", connector.Name)

			return nil
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

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			if err := client.DeleteConnector(ctx, args[0]); err != nil {
				return common.WrapDeleteError("connector", err)
			}

			_, _ = common.Green.Printf("✓ Deleted connector: %s\n", args[0])

			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
