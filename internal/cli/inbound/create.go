package inbound

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "create <email-prefix>",
		Short: "Create a new inbound inbox",
		Long: `Create a new inbound inbox with a managed email address.

The email prefix you provide will be combined with your application's
Nylas domain to create the full email address (e.g., support@yourapp.nylas.email).

Examples:
  # Create a support inbox
  nylas inbound create support
  # Creates: support@yourapp.nylas.email

  # Create a leads inbox
  nylas inbound create leads
  # Creates: leads@yourapp.nylas.email

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

func runCreate(emailPrefix string, jsonOutput bool) error {
	// Validate email prefix
	emailPrefix = strings.TrimSpace(emailPrefix)
	if emailPrefix == "" {
		printError("Email prefix cannot be empty")
		return common.NewInputError("email prefix cannot be empty")
	}

	// Basic validation - no @ symbol, no spaces
	if strings.Contains(emailPrefix, "@") || strings.Contains(emailPrefix, " ") {
		printError("Email prefix should not contain '@' or spaces. Just provide the local part (e.g., 'support')")
		return common.NewInputError("invalid email prefix - should not contain '@' or spaces")
	}

	client, err := getClient()
	if err != nil {
		printError("%v", err)
		return err
	}

	ctx, cancel := common.CreateContext()
	defer cancel()

	inbox, err := client.CreateInboundInbox(ctx, emailPrefix)
	if err != nil {
		printError("Failed to create inbox: %v", err)
		return err
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(inbox, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	printSuccess("Inbound inbox created successfully!")
	fmt.Println()
	printInboxDetails(*inbox)

	fmt.Println()
	_, _ = common.Dim.Println("Next steps:")
	_, _ = common.Dim.Printf("  1. Set up a webhook: nylas webhooks create --url <your-url> --triggers message.created\n")
	_, _ = common.Dim.Printf("  2. View messages: nylas inbound messages %s\n", inbox.ID)
	_, _ = common.Dim.Printf("  3. Monitor in real-time: nylas inbound monitor %s\n", inbox.ID)

	return nil
}
