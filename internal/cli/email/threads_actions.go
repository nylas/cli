package email

import (
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newThreadsMarkCmd() *cobra.Command {
	var markRead, markUnread, markStar, markUnstar bool

	cmd := &cobra.Command{
		Use:   "mark <thread-id> [grant-id]",
		Short: "Mark thread as read/unread or starred/unstarred",
		Long:  "Update thread status: mark as read, unread, starred, or unstarcommon.Red.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			threadID := args[0]

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
				return common.WrapUpdateError("thread", err)
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
			return nil
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

			// Get thread info for confirmation
			if !force {
				thread, err := client.GetThread(ctx, grantID, threadID)
				if err != nil {
					return common.WrapGetError("thread", err)
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
					return nil
				}
			}

			err = client.DeleteThread(ctx, grantID, threadID)
			if err != nil {
				return common.WrapDeleteError("thread", err)
			}

			printSuccess("Thread deleted")
			return nil
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
			client, err := getClient()
			if err != nil {
				return err
			}

			grantID, err := common.GetGrantID(args)
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

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
					return common.WrapDateParseError("after", err)
				}
				params.LatestMsgAfter = t.Unix()
			}
			if before != "" {
				t, err := parseDate(before)
				if err != nil {
					return common.WrapDateParseError("before", err)
				}
				params.LatestMsgBefore = t.Unix()
			}

			threads, err := client.GetThreads(ctx, grantID, params)
			if err != nil {
				return common.WrapFetchError("threads", err)
			}

			if len(threads) == 0 {
				common.PrintEmptyStateWithHint("threads", "try different search terms")
				return nil
			}

			fmt.Printf("Found %d threads:\n\n", len(threads))

			for _, t := range threads {
				printThreadRow(t, showID)
			}

			return nil
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

// printThreadRow prints a single thread in the list format.
func printThreadRow(t domain.Thread, showID bool) {
	status := " "
	if t.Unread {
		status = common.Cyan.Sprint("●")
	}

	star := " "
	if t.Starred {
		star = common.Yellow.Sprint("★")
	}

	attach := " "
	if t.HasAttachments {
		attach = "📎"
	}

	// Format participants
	participants := common.FormatParticipants(t.Participants)
	if len(participants) > 25 {
		participants = participants[:22] + "..."
	}

	subj := t.Subject
	if len(subj) > 35 {
		subj = subj[:32] + "..."
	}

	msgCount := fmt.Sprintf("(%d)", len(t.MessageIDs))
	dateStr := common.FormatTimeAgo(t.LatestMessageRecvDate)

	fmt.Printf("%s %s %s %-25s %-35s %-5s %s\n",
		status, star, attach, participants, subj, msgCount, common.Dim.Sprint(dateStr))

	if showID {
		_, _ = common.Dim.Printf("      ID: %s\n", t.ID)
	}
}
