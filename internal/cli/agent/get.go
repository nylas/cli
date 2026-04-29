package agent

import (
	"context"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [agent-id|email]",
		Short: "Show an agent account",
		Long: `Show a Nylas agent account.

You can look up an account by grant ID or by email address. If omitted, the CLI
resolves a local provider=nylas grant when one can be identified safely.

Examples:
  nylas agent account get
  nylas agent account get 123456
  nylas agent account get me@yourapp.nylas.email
  nylas agent account get me@yourapp.nylas.email --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier, err := getAgentIdentifier(args)
			if err != nil {
				return err
			}
			return runGet(identifier, common.IsJSON(cmd))
		},
	}

	return cmd
}

func runGet(identifier string, jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		grantID, err := resolveAgentID(ctx, client, identifier)
		if err != nil {
			return struct{}{}, common.WrapGetError("agent account", err)
		}

		account, err := client.GetAgentAccount(ctx, grantID)
		if err != nil {
			return struct{}{}, common.WrapGetError("agent account", err)
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(account)
		}

		printAgentDetails(*account)
		return struct{}{}, nil
	})

	return err
}
