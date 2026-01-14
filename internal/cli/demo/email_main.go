package demo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// newDemoEmailCmd creates the demo email command with subcommands.

func newDemoEmailCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "email",
		Short: "Explore email features with sample data",
		Long:  "Demo email commands showing sample messages, threads, and simulated operations.",
	}

	// Core commands
	cmd.AddCommand(newDemoEmailListCmd())
	cmd.AddCommand(newDemoEmailReadCmd())
	cmd.AddCommand(newDemoEmailSendCmd())
	cmd.AddCommand(newDemoEmailSearchCmd())
	cmd.AddCommand(newDemoEmailMarkCmd())
	cmd.AddCommand(newDemoEmailDeleteCmd())

	// Subcommand groups
	cmd.AddCommand(newDemoEmailFoldersCmd())
	cmd.AddCommand(newDemoEmailThreadsCmd())
	cmd.AddCommand(newDemoEmailDraftsCmd())
	cmd.AddCommand(newDemoEmailAttachmentsCmd())

	return cmd
}

// newDemoEmailListCmd lists sample emails.
func newDemoEmailListCmd() *cobra.Command {
	var limit int
	var showID bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sample emails",
		Long:  "Display a list of realistic sample emails to explore the CLI output format.",
		Example: `  # List sample emails
  nylas demo email list

  # List with IDs shown
  nylas demo email list --id

  # Limit to 5 emails
  nylas demo email list --limit 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			messages, err := client.GetMessages(ctx, "demo-grant", limit)
			if err != nil {
				return common.WrapListError("messages", err)
			}

			if limit > 0 && limit < len(messages) {
				messages = messages[:limit]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📧 Demo Mode - Sample Emails"))
			fmt.Println(common.Dim.Sprint("These are sample emails for demonstration purposes."))
			fmt.Println()
			fmt.Printf("Found %d messages:\n\n", len(messages))

			for i, msg := range messages {
				printDemoMessageSummary(msg, i+1, showID)
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("To connect your real email: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of messages to show")
	cmd.Flags().BoolVar(&showID, "id", false, "Show message IDs")

	return cmd
}

// newDemoEmailReadCmd reads a sample email.
func newDemoEmailReadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read [message-id]",
		Short: "Read a sample email",
		Long:  "Display a sample email to see the full message format.",
		Example: `  # Read first sample email
  nylas demo email read

  # Read specific message
  nylas demo email read msg-001`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			messageID := "msg-001"
			if len(args) > 0 {
				messageID = args[0]
			}

			msg, err := client.GetMessage(ctx, "demo-grant", messageID)
			if err != nil {
				return common.WrapGetError("message", err)
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📧 Demo Mode - Sample Email"))
			fmt.Println()
			printDemoMessage(*msg)

			fmt.Println(common.Dim.Sprint("To connect your real email: nylas auth login"))

			return nil
		},
	}

	return cmd
}

// newDemoEmailSendCmd simulates sending an email.
func newDemoEmailSendCmd() *cobra.Command {
	var to string
	var subject string
	var body string

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Simulate sending an email",
		Long: `Simulate sending an email to see how the send command works.

No actual email is sent - this is just a demonstration of the command flow.`,
		Example: `  # Simulate sending an email
  nylas demo email send --to user@example.com --subject "Hello" --body "Test message"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if to == "" {
				to = "recipient@example.com"
			}
			if subject == "" {
				subject = "Demo Email Subject"
			}
			if body == "" {
				body = "This is a demo email body.\n\nNo actual email was sent."
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📧 Demo Mode - Simulated Send"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			_, _ = common.BoldWhite.Printf("To:      %s\n", to)
			_, _ = common.BoldWhite.Printf("Subject: %s\n", subject)
			fmt.Println(strings.Repeat("─", 50))
			fmt.Println(body)
			fmt.Println(strings.Repeat("─", 50))
			fmt.Println()
			_, _ = common.Green.Println("✓ Email would be sent (demo mode - no actual email sent)")
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To send real emails, connect your account: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "Recipient email address")
	cmd.Flags().StringVar(&subject, "subject", "", "Email subject")
	cmd.Flags().StringVar(&body, "body", "", "Email body")

	return cmd
}

// printDemoMessageSummary prints a single-line message summary.
func printDemoMessageSummary(msg domain.Message, index int, showID bool) {
	status := " "
	if msg.Unread {
		status = common.Cyan.Sprint("●")
	}

	star := " "
	if msg.Starred {
		star = common.Yellow.Sprint("★")
	}

	from := formatDemoContacts(msg.From)
	if len(from) > 20 {
		from = from[:17] + "..."
	}

	subject := msg.Subject
	if len(subject) > 40 {
		subject = subject[:37] + "..."
	}

	dateStr := formatDemoTimeAgo(msg.Date)
	if len(dateStr) > 12 {
		dateStr = msg.Date.Format("Jan 2")
	}

	if showID {
		fmt.Printf("%s %s %-20s %-40s %s\n", status, star, from, subject, common.Dim.Sprint(dateStr))
		_, _ = common.Dim.Printf("      ID: %s\n", msg.ID)
	} else {
		fmt.Printf("%s %s %-20s %-40s %s\n", status, star, from, subject, common.Dim.Sprint(dateStr))
	}
}

// printDemoMessage prints a full message.
func printDemoMessage(msg domain.Message) {
	status := ""
	if msg.Unread {
		status += common.Cyan.Sprint("●") + " "
	}
	if msg.Starred {
		status += common.Yellow.Sprint("★") + " "
	}

	fmt.Println(strings.Repeat("─", 60))
	_, _ = common.BoldWhite.Printf("Subject: %s\n", msg.Subject)
	fmt.Printf("From:    %s\n", formatDemoContacts(msg.From))
	if len(msg.To) > 0 {
		fmt.Printf("To:      %s\n", formatDemoContacts(msg.To))
	}
	fmt.Printf("Date:    %s (%s)\n", msg.Date.Format("Jan 2, 2006 3:04 PM"), formatDemoTimeAgo(msg.Date))
	if status != "" {
		fmt.Printf("Status:  %s\n", status)
	}
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println(msg.Body)
	fmt.Println()
}

// formatDemoContacts formats multiple contacts for display.
func formatDemoContacts(contacts []domain.EmailParticipant) string {
	names := make([]string, len(contacts))
	for i, c := range contacts {
		if c.Name != "" {
			names[i] = c.Name
		} else {
			names[i] = c.Email
		}
	}
	return strings.Join(names, ", ")
}

// formatDemoTimeAgo formats a time as a relative string.
func formatDemoTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 48*time.Hour {
		return "yesterday"
	}
	days := int(diff.Hours() / 24)
	return fmt.Sprintf("%d days ago", days)
}

// newDemoEmailSearchCmd searches sample emails.
