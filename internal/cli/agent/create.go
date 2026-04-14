package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var jsonOutput bool
	var appPassword string

	cmd := &cobra.Command{
		Use:   "create <email>",
		Short: "Create a new agent account",
		Long: `Create a new Nylas agent account.

This command always creates a provider=nylas grant. If the nylas connector
does not exist yet, it will be created automatically first.

Examples:
  nylas agent create me@yourapp.nylas.email
  nylas agent create support@yourapp.nylas.email --json
  nylas agent create debug@yourapp.nylas.email --app-password 'ValidAgentPass123ABC!'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(args[0], appPassword, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&appPassword, "app-password", "", "Optional IMAP/SMTP app password for mail-client access")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func runCreate(email, appPassword string, jsonOutput bool) error {
	email = strings.TrimSpace(email)
	if email == "" {
		printError("Email address cannot be empty")
		return common.NewInputError("email address cannot be empty")
	}
	if strings.Contains(email, " ") {
		printError("Email address should not contain spaces")
		return common.NewInputError("invalid email address - should not contain spaces")
	}
	if err := validateAgentAppPassword(appPassword); err != nil {
		printError(err.Error())
		return err
	}

	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		connector, err := ensureNylasConnector(ctx, client)
		if err != nil {
			return struct{}{}, common.WrapCreateError("nylas connector", err)
		}

		account, err := client.CreateAgentAccount(ctx, email, appPassword)
		if err != nil {
			return struct{}{}, common.WrapCreateError("agent account", err)
		}

		saveGrantLocally(account.ID, account.Email)

		if jsonOutput {
			data, _ := json.MarshalIndent(account, "", "  ")
			fmt.Println(string(data))
			return struct{}{}, nil
		}

		printSuccess("Agent account created successfully!")
		fmt.Println()
		printAgentDetails(*account)
		if connector != nil {
			_, _ = common.Dim.Printf("Connector: %s\n", formatConnectorSummary(*connector))
		}

		fmt.Println()
		_, _ = common.Dim.Println("Next steps:")
		_, _ = common.Dim.Printf("  1. List agent accounts: nylas agent list\n")
		_, _ = common.Dim.Printf("  2. Check connector status: nylas agent status\n")
		_, _ = common.Dim.Printf("  3. Delete this account: nylas agent delete %s\n", account.ID)

		return struct{}{}, nil
	})

	return err
}

func validateAgentAppPassword(appPassword string) error {
	if appPassword == "" {
		return nil
	}
	if len(appPassword) < 18 || len(appPassword) > 40 {
		return common.NewInputError("app password must be between 18 and 40 characters")
	}

	var hasUpper, hasLower, hasDigit bool
	for _, r := range appPassword {
		if r < 33 || r > 126 {
			return common.NewInputError("app password must use printable ASCII characters only and cannot contain spaces")
		}
		switch {
		case 'A' <= r && r <= 'Z':
			hasUpper = true
		case 'a' <= r && r <= 'z':
			hasLower = true
		case '0' <= r && r <= '9':
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return common.NewInputError("app password must include at least one uppercase letter, one lowercase letter, and one digit")
	}

	return nil
}

func formatConnectorSummary(connector domain.Connector) string {
	if connector.ID == "" {
		return connector.Provider
	}
	return fmt.Sprintf("%s (%s)", connector.Provider, connector.ID)
}
