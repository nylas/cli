package agent

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List agent accounts",
		Long: `List all Nylas agent accounts.

This command only shows grants created with provider=nylas.

Examples:
  nylas agent account list
  nylas agent account list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func runList(jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		accounts, err := client.ListAgentAccounts(ctx)
		if err != nil {
			return struct{}{}, common.WrapListError("agent accounts", err)
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(accounts)
		}

		if len(accounts) == 0 {
			common.PrintEmptyStateWithHint("agent accounts", "Create one with: nylas agent account create <email>")
			return struct{}{}, nil
		}

		_, _ = common.BoldWhite.Printf("Agent Accounts (%d)\n\n", len(accounts))
		for i, account := range accounts {
			printAgentSummary(account, i)
		}

		fmt.Println()
		_, _ = common.Dim.Println("Use 'nylas agent status' to verify connector readiness")
		return struct{}{}, nil
	})

	return err
}
