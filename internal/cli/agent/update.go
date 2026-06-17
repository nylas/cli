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
	var appPassword string
	var name string

	cmd := &cobra.Command{
		Use:   "update [agent-id|email]",
		Short: "Update an agent account",
		Long: `Update mutable settings on a Nylas agent account.

You can look up an account by grant ID or by email address. If omitted, the CLI
resolves a local provider=nylas grant when one can be identified safely.

Examples:
  nylas agent account update --app-password "MySecureP4ssword!2024"
  nylas agent account update 123456 --app-password "MySecureP4ssword!2024"
  nylas agent account update me@yourapp.nylas.email --name "Support Bot"
  nylas agent account update me@yourapp.nylas.email --app-password "MySecureP4ssword!2024" --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier, err := getAgentIdentifier(args)
			if err != nil {
				return err
			}
			return runUpdate(identifier, name, cmd.Flags().Changed("name"), appPassword, common.IsJSON(cmd))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Set the agent account display name (1-256 characters)")
	cmd.Flags().StringVar(&appPassword, "app-password", "", "Rotate or add the IMAP/SMTP app password")

	return cmd
}

func runUpdate(identifier, name string, nameProvided bool, appPassword string, jsonOutput bool) error {
	appPassword = strings.TrimSpace(appPassword)
	if err := validateAgentAppPassword(appPassword); err != nil {
		common.PrintError(err.Error())
		return err
	}
	name = strings.TrimSpace(name)
	if nameProvided {
		if err := validateAgentName(name); err != nil {
			common.PrintError(err.Error())
			return err
		}
	}
	if appPassword == "" && !nameProvided {
		return common.NewUserError(
			"agent account update requires at least one field",
			"Use --app-password or --name",
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

		account, err := client.UpdateAgentAccount(ctx, grantID, current.Email, resolveEffectiveName(current.Name, name, nameProvided), appPassword)
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

// resolveEffectiveName preserves the account's current name unless the caller
// explicitly supplied a new one. The grant update replaces the full record, so
// an omitted name would otherwise clear the existing display name.
func resolveEffectiveName(current, provided string, nameProvided bool) string {
	if nameProvided {
		return provided
	}
	return current
}
