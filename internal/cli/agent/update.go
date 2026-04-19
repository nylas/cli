package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var (
		jsonOutput  bool
		appPassword string
	)

	cmd := &cobra.Command{
		Use:   "update [agent-id|email]",
		Short: "Update an agent account",
		Long: `Update mutable settings on a Nylas agent account.

You can look up an account by grant ID or by email address. If omitted, the CLI
resolves a local provider=nylas grant when one can be identified safely.

Examples:
  nylas agent account update --app-password "MySecureP4ssword!2024"
  nylas agent account update 123456 --app-password "MySecureP4ssword!2024"
  nylas agent account update me@yourapp.nylas.email --app-password "MySecureP4ssword!2024" --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier, err := getAgentIdentifier(args)
			if err != nil {
				return err
			}
			return runUpdate(identifier, appPassword, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&appPassword, "app-password", "", "Rotate or add the IMAP/SMTP app password")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func runUpdate(identifier, appPassword string, jsonOutput bool) error {
	appPassword = strings.TrimSpace(appPassword)
	if err := validateAgentAppPassword(appPassword); err != nil {
		printError(err.Error())
		return err
	}
	if appPassword == "" {
		return common.NewUserError(
			"agent account update requires at least one field",
			"Use --app-password",
		)
	}

	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		grantID, err := resolveAgentID(ctx, client, identifier)
		if err != nil {
			return struct{}{}, common.WrapGetError("agent account", err)
		}

		current, err := client.GetAgentAccount(ctx, grantID)
		if err != nil {
			return struct{}{}, common.WrapGetError("agent account", err)
		}

		account, err := client.UpdateAgentAccount(ctx, grantID, current.Email, appPassword)
		if err != nil {
			return struct{}{}, common.WrapUpdateError("agent account", err)
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(account)
		}

		common.PrintUpdateSuccess("agent account", account.Email)
		fmt.Println()
		printAgentDetails(*account)
		return struct{}{}, nil
	})

	return err
}
