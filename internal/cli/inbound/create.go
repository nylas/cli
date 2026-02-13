package inbound

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "create <email>",
		Short: "Create a new inbound inbox",
		Long: `Create a new inbound inbox with a managed email address.

You can provide either a full email address or just the local part (prefix).
Wildcards (*) are supported for catch-all patterns.

Examples:
  # Create with full email address
  nylas inbound create support@yourapp.nylas.email

  # Create with just the prefix (domain added by API)
  nylas inbound create support

  # Create a wildcard catch-all inbox
  nylas inbound create "e2e-*@yourapp.nylas.email"

  # Create and output as JSON
  nylas inbound create tickets --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(args[0], jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func runCreate(email string, jsonOutput bool) error {
	email = strings.TrimSpace(email)
	if email == "" {
		printError("Email address cannot be empty")
		return common.NewInputError("email address cannot be empty")
	}

	if strings.Contains(email, " ") {
		printError("Email address should not contain spaces")
		return common.NewInputError("invalid email address - should not contain spaces")
	}

	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		inbox, err := client.CreateInboundInbox(ctx, email)
		if err != nil {
			return struct{}{}, common.WrapCreateError("inbound inbox", err)
		}

		// Save the new grant to local store so it appears in `nylas auth list`
		saveGrantLocally(inbox.ID, inbox.Email)

		if jsonOutput {
			data, _ := json.MarshalIndent(inbox, "", "  ")
			fmt.Println(string(data))
			return struct{}{}, nil
		}

		printSuccess("Inbound inbox created successfully!")
		fmt.Println()
		printInboxDetails(*inbox)

		fmt.Println()
		_, _ = common.Dim.Println("Next steps:")
		_, _ = common.Dim.Printf("  1. Set up a webhook: nylas webhooks create --url <your-url> --triggers message.created\n")
		_, _ = common.Dim.Printf("  2. View messages: nylas inbound messages %s\n", inbox.ID)
		_, _ = common.Dim.Printf("  3. Monitor in real-time: nylas inbound monitor %s\n", inbox.ID)

		return struct{}{}, nil
	})

	return err
}
