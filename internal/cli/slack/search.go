// search.go provides message search functionality for Slack workspaces.

package slack

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

// newSearchCmd creates the search command for searching messages.
func newSearchCmd() *cobra.Command {
	var (
		query  string
		limit  int
		showID bool
	)

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search messages",
		Long: `Search for messages in your Slack workspace.

Uses Slack's search syntax. Examples:
  - "from:@alice" - messages from a user
  - "in:#general" - messages in a channel
  - "has:link" - messages with links
  - "before:2024-01-01" - messages before a date

Examples:
  # Search for messages
  nylas slack search --query "project update"

  # Search with Slack modifiers
  nylas slack search --query "from:@alice in:#general"

  # Limit results
  nylas slack search --query "important" --limit 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if query == "" {
				return common.NewUserError("search query is required", "Use --query")
			}

			client, err := getSlackClientOrError()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			messages, err := client.SearchMessages(ctx, query, limit)
			if err != nil {
				return common.WrapSearchError("messages", err)
			}

			if len(messages) == 0 {
				fmt.Printf("No messages found for: %s\n", query)
				return nil
			}

			// Handle structured output (JSON/YAML/quiet)
			if common.IsStructuredOutput(cmd) {
				out := common.GetOutputWriter(cmd)
				return out.Write(messages)
			}

			_, _ = common.Cyan.Printf("Found %d messages:\n\n", len(messages))

			for _, msg := range messages {
				printMessage(msg, showID, false)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "Search query")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum number of results")
	cmd.Flags().BoolVar(&showID, "id", false, "Show message IDs")

	_ = cmd.MarkFlagRequired("query")

	return cmd
}
