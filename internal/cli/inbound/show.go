package inbound

import (
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "show <inbox-id>",
		Short: "Show details of an inbound inbox",
		Long: `Show detailed information about a specific inbound inbox.

Examples:
  # Show inbox details
  nylas inbound show abc123

  # Show as JSON
  nylas inbound show abc123 --json

  # Use environment variable for inbox ID
  export NYLAS_INBOUND_GRANT_ID=abc123
  nylas inbound show`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(args, jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func runShow(args []string, jsonOutput bool) error {
	inboxID, err := getInboxID(args)
	if err != nil {
		printError("%v", err)
		return err
	}

	client, err := getClient()
	if err != nil {
		printError("%v", err)
		return err
	}

	ctx, cancel := common.CreateContext()
	defer cancel()

	inbox, err := client.GetInboundInbox(ctx, inboxID)
	if err != nil {
		printError("Failed to get inbox: %v", err)
		return err
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(inbox, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	printInboxDetails(*inbox)

	return nil
}
