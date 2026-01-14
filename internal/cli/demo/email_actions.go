package demo

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newDemoEmailSearchCmd() *cobra.Command {
	var query string

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search sample emails",
		Long:  "Search through sample emails to see how search works.",
		Example: `  # Search for emails
  nylas demo email search --query "meeting"
  nylas demo email search -q "project"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			messages, _ := client.GetMessages(ctx, "demo-grant", 10)

			// Filter by query if provided
			if query != "" {
				var filtered []domain.Message
				for _, msg := range messages {
					if strings.Contains(strings.ToLower(msg.Subject), strings.ToLower(query)) ||
						strings.Contains(strings.ToLower(msg.Body), strings.ToLower(query)) {
						filtered = append(filtered, msg)
					}
				}
				messages = filtered
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("🔍 Demo Mode - Email Search"))
			if query != "" {
				fmt.Printf("Searching for: %s\n", common.BoldWhite.Sprint(query))
			}
			fmt.Println()
			fmt.Printf("Found %d messages:\n\n", len(messages))

			for i, msg := range messages {
				printDemoMessageSummary(msg, i+1, false)
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("To search your real emails: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().StringVarP(&query, "query", "q", "", "Search query")

	return cmd
}

// newDemoEmailMarkCmd marks sample emails.
func newDemoEmailMarkCmd() *cobra.Command {
	var read, unread, starred, unstarred bool

	cmd := &cobra.Command{
		Use:   "mark [message-id]",
		Short: "Mark sample emails (simulated)",
		Long:  "Simulate marking emails as read/unread/starcommon.Red.",
		Example: `  # Mark as read
  nylas demo email mark msg-001 --read

  # Mark as starred
  nylas demo email mark msg-001 --starred`,
		RunE: func(cmd *cobra.Command, args []string) error {
			messageID := "msg-001"
			if len(args) > 0 {
				messageID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📧 Demo Mode - Mark Email (Simulated)"))
			fmt.Println()

			if read {
				_, _ = common.Green.Printf("✓ Message %s would be marked as read\n", messageID)
			}
			if unread {
				_, _ = common.Green.Printf("✓ Message %s would be marked as unread\n", messageID)
			}
			if starred {
				_, _ = common.Green.Printf("✓ Message %s would be starred\n", messageID)
			}
			if unstarred {
				_, _ = common.Green.Printf("✓ Message %s would be unstarred\n", messageID)
			}

			if !read && !unread && !starred && !unstarred {
				fmt.Println("No action specified. Use --read, --unread, --starred, or --unstarred")
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("To manage your real emails: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().BoolVar(&read, "read", false, "Mark as read")
	cmd.Flags().BoolVar(&unread, "unread", false, "Mark as unread")
	cmd.Flags().BoolVar(&starred, "starred", false, "Mark as starred")
	cmd.Flags().BoolVar(&unstarred, "unstarred", false, "Remove star")

	return cmd
}

// newDemoEmailDeleteCmd deletes sample emails (simulated).
func newDemoEmailDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [message-id]",
		Short: "Delete sample email (simulated)",
		Long:  "Simulate deleting an email.",
		Example: `  # Delete an email
  nylas demo email delete msg-001`,
		RunE: func(cmd *cobra.Command, args []string) error {
			messageID := "msg-001"
			if len(args) > 0 {
				messageID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📧 Demo Mode - Delete Email (Simulated)"))
			fmt.Println()
			_, _ = common.Green.Printf("✓ Message %s would be deleted\n", messageID)
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To manage your real emails: nylas auth login"))

			return nil
		},
	}

	return cmd
}

// newDemoEmailFoldersCmd manages sample folders.
