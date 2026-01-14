package email

import (
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
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
				return common.WrapGetError("threads", err)
			}

			if len(threads) == 0 {
				common.PrintEmptyState("threads")
				return nil
			}

			fmt.Printf("Found %d threads:\n\n", len(threads))

			for _, t := range threads {
				printThreadRow(t, showID)
			}

			return nil
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

			thread, err := client.GetThread(ctx, grantID, threadID)
			if err != nil {
				return common.WrapGetError("thread", err)
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

			return nil
		},
	}
}
