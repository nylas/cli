package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

type statusResult struct {
	ConnectorConfigured bool                  `json:"connector_configured"`
	ConnectorID         string                `json:"connector_id,omitempty"`
	AccountCount        int                   `json:"account_count"`
	Accounts            []domain.AgentAccount `json:"accounts,omitempty"`
}

func newStatusCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show agent connector status",
		Long: `Show the state of the nylas connector and managed agent accounts.

Examples:
  nylas agent status
  nylas agent status --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func runStatus(jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		connectors, err := client.ListConnectors(ctx)
		if err != nil {
			return struct{}{}, common.WrapGetError("connectors", err)
		}

		connector := findNylasConnector(connectors)

		accounts, err := client.ListAgentAccounts(ctx)
		if err != nil {
			return struct{}{}, common.WrapListError("agent accounts", err)
		}

		result := statusResult{
			ConnectorConfigured: connector != nil,
			AccountCount:        len(accounts),
			Accounts:            accounts,
		}
		if connector != nil {
			result.ConnectorID = connector.ID
		}

		if jsonOutput {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return struct{}{}, nil
		}

		_, _ = common.BoldWhite.Println("Agent Status")
		if result.ConnectorConfigured {
			if result.ConnectorID != "" {
				fmt.Printf("  Connector: %s\n", common.Green.Sprintf("ready (%s)", result.ConnectorID))
			} else {
				fmt.Printf("  Connector: %s\n", common.Green.Sprint("ready"))
			}
		} else {
			fmt.Printf("  Connector: %s\n", common.Red.Sprint("missing"))
		}
		fmt.Printf("  Accounts:  %s\n", common.Cyan.Sprintf("%d", result.AccountCount))

		if len(accounts) > 0 {
			fmt.Println()
			for i, account := range accounts {
				printAgentSummary(account, i)
			}
		}

		return struct{}{}, nil
	})

	return err
}

func findNylasConnector(connectors []domain.Connector) *domain.Connector {
	for i := range connectors {
		if connectors[i].Provider == string(domain.ProviderNylas) {
			return &connectors[i]
		}
	}
	return nil
}
