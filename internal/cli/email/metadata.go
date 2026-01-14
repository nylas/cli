package email

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newMetadataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metadata",
		Short: "Manage message metadata",
		Long: `Manage message metadata for filtering and organization.

Nylas supports metadata on messages for custom organization and filtering.
Only five indexed keys (key1-key5) can be used for filtering in API queries.
You can store up to 50 custom key-value pairs per message, but only the
indexed keys (key1-key5) support filtering via the --metadata flag.

Metadata is set when sending or creating drafts. You cannot update metadata
on existing messages through the API.`,
	}

	cmd.AddCommand(newMetadataShowCmd())
	cmd.AddCommand(newMetadataInfoCmd())

	return cmd
}

func newMetadataShowCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "show <message-id> [grant-id]",
		Short: "Show metadata for a message",
		Long: `Display all metadata key-value pairs for a specific message.

This shows all metadata stored on the message, including both indexed
(key1-key5) and custom keys.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			messageID := args[0]

			var grantID string
			if len(args) > 1 {
				grantID = args[1]
			} else {
				grantID, err = common.GetGrantID(nil)
				if err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			message, err := client.GetMessage(ctx, grantID, messageID)
			if err != nil {
				return common.WrapGetError("message", err)
			}

			if asJSON {
				data, err := json.MarshalIndent(message.Metadata, "", "  ")
				if err != nil {
					return common.WrapMarshalError("metadata", err)
				}
				fmt.Println(string(data))
				return nil
			}

			if len(message.Metadata) == 0 {
				common.PrintEmptyState("metadata")
				return nil
			}

			fmt.Printf("Metadata for message %s:\n\n", messageID)
			for key, value := range message.Metadata {
				indexed := ""
				if strings.HasPrefix(key, "key") && len(key) == 4 {
					if key >= "key1" && key <= "key5" {
						indexed = " (indexed - searchable)"
					}
				}
				fmt.Printf("  %s: %s%s\n", key, value, indexed)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")

	return cmd
}

func newMetadataInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show metadata usage information",
		Long: `Display information about how to use metadata with Nylas messages.

This command provides a comprehensive guide on:
- How to set metadata when sending messages
- Which keys are indexed and searchable
- How to filter messages by metadata
- Limitations and best practices`,
		RunE: func(cmd *cobra.Command, args []string) error {
			printMetadataInfo()
			return nil
		},
	}

	return cmd
}

func printMetadataInfo() {
	info := `
Metadata Usage Guide
====================

Nylas supports custom metadata on messages for organization and filtering.

INDEXED KEYS (Searchable)
--------------------------
Only five keys are indexed and can be used for filtering:
  • key1
  • key2
  • key3
  • key4
  • key5

These keys support filtering in list queries using the --metadata flag.

SETTING METADATA
----------------
Metadata can only be set when:
  1. Sending a new message
  2. Creating a draft

You CANNOT update metadata on existing messages.

Example - Send with metadata:
  nylas email send \\
    --to user@example.com \\
    --subject "Project Update" \\
    --body "Status report" \\
    --metadata key1=project-alpha \\
    --metadata key2=status-update

FILTERING BY METADATA
---------------------
Use the --metadata flag to filter messages by indexed keys:

  nylas email list --metadata key1:project-alpha
  nylas email list --metadata key2:urgent

Format: --metadata key:value

LIMITATIONS
-----------
  1. Only key1-key5 can be used for filtering
  2. You can store up to 50 custom key-value pairs per message
  3. Cannot combine metadata filters with some provider-specific filters
  4. Metadata cannot be updated on existing messages
  5. Only works with indexed keys (key1-key5) for filtering

VIEWING METADATA
----------------
To see all metadata on a message:
  nylas email metadata show <message-id>
  nylas email metadata show <message-id> --json

STORAGE vs FILTERING
--------------------
  Storage:   Up to 50 custom key-value pairs
  Filtering: Only key1-key5 are indexed

Example: You can store { "project": "alpha", "key1": "alpha" }
But only filtering by key1 will work in API queries.

BEST PRACTICES
--------------
  1. Use key1-key5 for values you need to filter by
  2. Use descriptive custom keys for non-searchable metadata
  3. Plan your metadata schema before implementation
  4. Document which indexed keys map to which concepts
  5. Consider your filtering needs when choosing keys

For more information, see:
https://developer.nylas.com/docs/dev-guide/metadata/
`
	fmt.Println(info)
}
