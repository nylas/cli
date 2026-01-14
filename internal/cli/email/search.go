package email

import (
	"fmt"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var (
		limit         int
		from          string
		to            string
		subject       string
		after         string
		before        string
		hasAttachment bool
		unread        bool
		starred       bool
		inFolder      string
	)

	cmd := &cobra.Command{
		Use:   "search <query> [grant-id]",
		Short: "Search emails",
		Long: `Search for emails matching a query string or filters.

Examples:
  # Search by subject
  nylas email search "project update"

  # Search with filters
  nylas email search "meeting" --from "boss@company.com" --unread

  # Search by sender (use * for any subject)
  nylas email search "*" --from "support@example.com"

  # Search in a specific folder
  nylas email search "invoice" --in INBOX

  # Search with date filters
  nylas email search "invoice" --after 2024-01-01 --before 2024-12-31

  # Search for messages with attachments
  nylas email search "*" --has-attachment --from "hr@company.com"`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

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

			params := &domain.MessageQueryParams{
				Limit: limit,
			}

			// Use query as subject search unless it's a wildcard
			// If --subject flag is also provided, it takes precedence
			if subject != "" {
				params.Subject = subject
			} else if query != "*" && query != "" {
				params.Subject = query
			}

			if from != "" {
				params.From = from
			}
			if to != "" {
				params.To = to
			}
			if inFolder != "" {
				params.In = []string{inFolder}
			}
			if cmd.Flags().Changed("has-attachment") {
				params.HasAttachment = &hasAttachment
			}
			if cmd.Flags().Changed("unread") {
				params.Unread = &unread
			}
			if cmd.Flags().Changed("starred") {
				params.Starred = &starred
			}

			// Parse date filters
			if after != "" {
				t, err := parseDate(after)
				if err != nil {
					return common.WrapDateParseError("after", err)
				}
				params.ReceivedAfter = t.Unix()
			}
			if before != "" {
				t, err := parseDate(before)
				if err != nil {
					return common.WrapDateParseError("before", err)
				}
				params.ReceivedBefore = t.Unix()
			}

			messages, err := client.GetMessagesWithParams(ctx, grantID, params)
			if err != nil {
				return common.WrapSearchError("messages", err)
			}

			if len(messages) == 0 {
				common.PrintEmptyStateWithHint("messages", "try different search terms")
				return nil
			}

			fmt.Printf("Found %d messages:\n\n", len(messages))
			for i, msg := range messages {
				printMessageSummary(msg, i+1)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of results")
	cmd.Flags().StringVar(&from, "from", "", "Filter by sender")
	cmd.Flags().StringVar(&to, "to", "", "Filter by recipient")
	cmd.Flags().StringVar(&subject, "subject", "", "Filter by subject")
	cmd.Flags().StringVar(&after, "after", "", "Messages after date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&before, "before", "", "Messages before date (YYYY-MM-DD)")
	cmd.Flags().BoolVar(&hasAttachment, "has-attachment", false, "Only messages with attachments")
	cmd.Flags().BoolVar(&unread, "unread", false, "Only unread messages")
	cmd.Flags().BoolVar(&starred, "starred", false, "Only starred messages")
	cmd.Flags().StringVar(&inFolder, "in", "", "Filter by folder (e.g., INBOX, SENT)")

	return cmd
}

// parseDate parses a date string in YYYY-MM-DD format using local timezone.
func parseDate(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, time.Local)
}
