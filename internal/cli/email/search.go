package email

import (
	"context"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
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
		jsonOutput    bool
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
			remainingArgs := args[1:]

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				// Auto-paginate when limit exceeds API maximum
				needsPagination := limit > common.MaxAPILimit
				apiLimit := limit
				if needsPagination {
					apiLimit = common.MaxAPILimit
				}

				params := &domain.MessageQueryParams{
					Limit: apiLimit,
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
						return struct{}{}, common.WrapDateParseError("after", err)
					}
					params.ReceivedAfter = t.Unix()
				}
				if before != "" {
					t, err := parseDate(before)
					if err != nil {
						return struct{}{}, common.WrapDateParseError("before", err)
					}
					params.ReceivedBefore = t.Unix()
				}

				var messages []domain.Message
				var err error

				if needsPagination {
					fetcher := func(ctx context.Context, cursor string) (common.PageResult[domain.Message], error) {
						params.PageToken = cursor
						resp, fetchErr := client.GetMessagesWithCursor(ctx, grantID, params)
						if fetchErr != nil {
							return common.PageResult[domain.Message]{}, fetchErr
						}
						return common.PageResult[domain.Message]{
							Data:       resp.Data,
							NextCursor: resp.Pagination.NextCursor,
						}, nil
					}

					config := common.DefaultPaginationConfig()
					config.PageSize = apiLimit
					config.MaxItems = limit

					messages, err = common.FetchAllPages(ctx, config, fetcher)
				} else {
					messages, err = client.GetMessagesWithParams(ctx, grantID, params)
				}
				if err != nil {
					return struct{}{}, common.WrapSearchError("messages", err)
				}

				if jsonOutput {
					return struct{}{}, common.PrintJSON(messages)
				}

				if len(messages) == 0 {
					common.PrintEmptyStateWithHint("messages", "try different search terms")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d messages:\n\n", len(messages))
				for i, msg := range messages {
					printMessageSummary(msg, i+1)
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of results (auto-paginates if >200)")
	cmd.Flags().StringVar(&from, "from", "", "Filter by sender")
	cmd.Flags().StringVar(&to, "to", "", "Filter by recipient")
	cmd.Flags().StringVar(&subject, "subject", "", "Filter by subject")
	cmd.Flags().StringVar(&after, "after", "", "Messages after date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&before, "before", "", "Messages before date (YYYY-MM-DD)")
	cmd.Flags().BoolVar(&hasAttachment, "has-attachment", false, "Only messages with attachments")
	cmd.Flags().BoolVar(&unread, "unread", false, "Only unread messages")
	cmd.Flags().BoolVar(&starred, "starred", false, "Only starred messages")
	cmd.Flags().StringVar(&inFolder, "in", "", "Filter by folder (e.g., INBOX, SENT)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

// parseDate parses a date string in YYYY-MM-DD format using local timezone.
func parseDate(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, time.Local)
}
