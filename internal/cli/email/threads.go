package email

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newThreadsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "threads",
		Short: "Manage email threads/conversations",
		Long:  "List, view, mark, and delete email threads (conversations).",
	}

	cmd.AddCommand(newThreadsListCmd())
	cmd.AddCommand(newThreadsShowCmd())
	cmd.AddCommand(newThreadsMarkCmd())
	cmd.AddCommand(newThreadsDeleteCmd())
	cmd.AddCommand(newThreadsSearchCmd())

	return cmd
}

func newThreadsListCmd() *cobra.Command {
	var limit int
	var unread bool
	var starred bool
	var subject string
	var showID bool

	cmd := &cobra.Command{
		Use:   "list [grant-id]",
		Short: "List email threads",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				params := &domain.ThreadQueryParams{
					Limit: limit,
				}

				if cmd.Flags().Changed("unread") {
					params.Unread = &unread
				}
				if cmd.Flags().Changed("starred") {
					params.Starred = &starred
				}
				if subject != "" {
					params.Subject = subject
				}

				threads, err := client.GetThreads(ctx, grantID, params)
				if err != nil {
					return struct{}{}, common.WrapGetError("threads", err)
				}

				// JSON output (including empty array)
				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(threads)
				}

				if len(threads) == 0 {
					common.PrintEmptyState("threads")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d threads:\n\n", len(threads))

				for _, t := range threads {
					DisplayThreadListItem(t, showID)
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of threads to fetch")
	cmd.Flags().BoolVarP(&unread, "unread", "u", false, "Only show unread threads")
	cmd.Flags().BoolVarP(&starred, "starred", "s", false, "Only show starred threads")
	cmd.Flags().StringVar(&subject, "subject", "", "Filter by subject")
	cmd.Flags().BoolVar(&showID, "id", false, "Show thread IDs")

	return cmd
}

func newThreadsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <thread-id> [grant-id]",
		Short: "Show thread details",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			threadID := args[0]
			remainingArgs := args[1:]

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				thread, err := client.GetThread(ctx, grantID, threadID)
				if err != nil {
					return struct{}{}, common.WrapGetError("thread", err)
				}

				// Print thread details
				fmt.Println("════════════════════════════════════════════════════════════")
				_, _ = common.BoldWhite.Printf("Thread: %s\n", thread.Subject)
				fmt.Println("════════════════════════════════════════════════════════════")

				fmt.Printf("Participants: %s\n", common.FormatParticipants(thread.Participants))
				fmt.Printf("Messages:     %d\n", len(thread.MessageIDs))
				if len(thread.DraftIDs) > 0 {
					fmt.Printf("Drafts:       %d\n", len(thread.DraftIDs))
				}

				status := []string{}
				if thread.Unread {
					status = append(status, common.Cyan.Sprint("unread"))
				}
				if thread.Starred {
					status = append(status, common.Yellow.Sprint("starred"))
				}
				if thread.HasAttachments {
					status = append(status, "has attachments")
				}
				if len(status) > 0 {
					fmt.Printf("Status:       %s\n", common.FormatParticipants(nil))
				}

				fmt.Printf("\nFirst message: %s\n", thread.EarliestMessageDate.Format(common.DisplayDateTime))
				fmt.Printf("Latest:        %s\n", thread.LatestMessageRecvDate.Format(common.DisplayDateTime))

				fmt.Println("\nSnippet:")
				fmt.Println(thread.Snippet)

				fmt.Println("\nMessage IDs:")
				for i, msgID := range thread.MessageIDs {
					fmt.Printf("  %d. %s\n", i+1, msgID)
				}

				return struct{}{}, nil
			})
			return err
		},
	}
}

func newThreadsMarkCmd() *cobra.Command {
	var markRead, markUnread, markStar, markUnstar bool

	cmd := &cobra.Command{
		Use:   "mark <thread-id> [grant-id]",
		Short: "Mark thread as read/unread or starred/unstarred",
		Long:  "Update thread status: mark as read, unread, starred, or unstarcommon.Red.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			threadID := args[0]
			remainingArgs := args[1:]

			// Check flags
			flagCount := 0
			if markRead {
				flagCount++
			}
			if markUnread {
				flagCount++
			}
			if markStar {
				flagCount++
			}
			if markUnstar {
				flagCount++
			}

			if flagCount == 0 {
				return common.NewInputError("specify at least one of --read, --unread, --star, or --unstar")
			}

			if markRead && markUnread {
				return common.NewMutuallyExclusiveError("read", "unread")
			}
			if markStar && markUnstar {
				return common.NewMutuallyExclusiveError("star", "unstar")
			}

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				req := &domain.UpdateMessageRequest{}

				if markRead {
					unread := false
					req.Unread = &unread
				} else if markUnread {
					unread := true
					req.Unread = &unread
				}

				if markStar {
					starred := true
					req.Starred = &starred
				} else if markUnstar {
					starred := false
					req.Starred = &starred
				}

				thread, err := client.UpdateThread(ctx, grantID, threadID, req)
				if err != nil {
					return struct{}{}, common.WrapUpdateError("thread", err)
				}

				status := []string{}
				if markRead {
					status = append(status, "marked read")
				}
				if markUnread {
					status = append(status, "marked unread")
				}
				if markStar {
					status = append(status, "starred")
				}
				if markUnstar {
					status = append(status, "unstarred")
				}

				printSuccess("Thread %s: %s (subject: %s)", threadID[:12]+"...", fmt.Sprintf("%v", status), thread.Subject)
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVar(&markRead, "read", false, "Mark thread as read")
	cmd.Flags().BoolVar(&markUnread, "unread", false, "Mark thread as unread")
	cmd.Flags().BoolVar(&markStar, "star", false, "Star the thread")
	cmd.Flags().BoolVar(&markUnstar, "unstar", false, "Unstar the thread")

	return cmd
}

func newThreadsDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <thread-id> [grant-id]",
		Short: "Delete a thread",
		Long:  "Delete an email thread and all its messages.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			threadID := args[0]
			remainingArgs := args[1:]

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				// Get thread info for confirmation
				if !force {
					thread, err := client.GetThread(ctx, grantID, threadID)
					if err != nil {
						return struct{}{}, common.WrapGetError("thread", err)
					}

					fmt.Println("Delete this thread?")
					fmt.Printf("  Subject:      %s\n", thread.Subject)
					fmt.Printf("  Messages:     %d\n", len(thread.MessageIDs))
					fmt.Printf("  Participants: %s\n", common.FormatParticipants(thread.Participants))
					fmt.Print("\n[y/N]: ")

					var confirm string
					_, _ = fmt.Scanln(&confirm) // Ignore error - empty string treated as "no"
					if confirm != "y" && confirm != "Y" && confirm != "yes" {
						fmt.Println("Cancelled.")
						return struct{}{}, nil
					}
				}

				err := client.DeleteThread(ctx, grantID, threadID)
				if err != nil {
					return struct{}{}, common.WrapDeleteError("thread", err)
				}

				printSuccess("Thread deleted")
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation")

	return cmd
}

func newThreadsSearchCmd() *cobra.Command {
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
		showID        bool
	)

	cmd := &cobra.Command{
		Use:   "search [grant-id]",
		Short: "Search threads",
		Long: `Search for email threads using filters.

Note: Thread search supports filtering by specific fields. Use --subject
for text matching on the subject line.

Examples:
  # Search by subject
  nylas email threads search --subject "project update"

  # Search with multiple filters
  nylas email threads search --subject "meeting" --from "boss@company.com" --unread

  # Search by sender
  nylas email threads search --from "support@example.com"

  # Search with date filters
  nylas email threads search --subject "invoice" --after 2024-01-01 --before 2024-12-31`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				params := &domain.ThreadQueryParams{
					Limit: limit,
				}

				if from != "" {
					params.From = from
				}
				if to != "" {
					params.To = to
				}
				if subject != "" {
					params.Subject = subject
				}
				if inFolder != "" {
					params.In = []string{inFolder}
				}
				if cmd.Flags().Changed("unread") {
					params.Unread = &unread
				}
				if cmd.Flags().Changed("starred") {
					params.Starred = &starred
				}
				if cmd.Flags().Changed("has-attachment") {
					params.HasAttachment = &hasAttachment
				}

				// Parse date filters
				if after != "" {
					t, err := parseDate(after)
					if err != nil {
						return struct{}{}, common.WrapDateParseError("after", err)
					}
					params.LatestMsgAfter = t.Unix()
				}
				if before != "" {
					t, err := parseDate(before)
					if err != nil {
						return struct{}{}, common.WrapDateParseError("before", err)
					}
					params.LatestMsgBefore = t.Unix()
				}

				threads, err := client.GetThreads(ctx, grantID, params)
				if err != nil {
					return struct{}{}, common.WrapFetchError("threads", err)
				}

				if len(threads) == 0 {
					common.PrintEmptyStateWithHint("threads", "try different search terms")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d threads:\n\n", len(threads))

				for _, t := range threads {
					DisplayThreadListItem(t, showID)
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of results")
	cmd.Flags().StringVar(&from, "from", "", "Filter by sender")
	cmd.Flags().StringVar(&to, "to", "", "Filter by recipient")
	cmd.Flags().StringVar(&subject, "subject", "", "Filter by subject")
	cmd.Flags().StringVar(&after, "after", "", "Threads with messages after date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&before, "before", "", "Threads with messages before date (YYYY-MM-DD)")
	cmd.Flags().BoolVar(&hasAttachment, "has-attachment", false, "Only threads with attachments")
	cmd.Flags().BoolVar(&unread, "unread", false, "Only unread threads")
	cmd.Flags().BoolVar(&starred, "starred", false, "Only starred threads")
	cmd.Flags().StringVar(&inFolder, "in", "", "Filter by folder (e.g., INBOX, SENT)")
	cmd.Flags().BoolVar(&showID, "id", false, "Show thread IDs")

	return cmd
}
