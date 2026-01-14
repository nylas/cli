package demo

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newDemoEmailDraftsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drafts",
		Short: "Manage sample email drafts",
		Long:  "Demo draft commands showing sample drafts.",
	}

	cmd.AddCommand(newDemoEmailDraftsListCmd())
	cmd.AddCommand(newDemoEmailDraftsCreateCmd())
	cmd.AddCommand(newDemoEmailDraftsDeleteCmd())
	cmd.AddCommand(newDemoEmailDraftsSendCmd())

	return cmd
}

func newDemoEmailDraftsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List sample drafts",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			drafts, _ := client.GetDrafts(ctx, "demo-grant", 10)

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📝 Demo Mode - Sample Drafts"))
			fmt.Println()
			fmt.Printf("Found %d drafts:\n\n", len(drafts))

			for _, d := range drafts {
				to := ""
				if len(d.To) > 0 {
					to = d.To[0].Email
				}
				fmt.Printf("  📝 %s\n", common.BoldWhite.Sprint(d.Subject))
				fmt.Printf("     To: %s\n", to)
				_, _ = common.Dim.Printf("     ID: %s\n", d.ID)
				fmt.Println()
			}

			fmt.Println(common.Dim.Sprint("To manage your real drafts: nylas auth login"))

			return nil
		},
	}
}

func newDemoEmailDraftsCreateCmd() *cobra.Command {
	var to, subject, body string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a draft (simulated)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println()
			fmt.Println(common.Dim.Sprint("📝 Demo Mode - Create Draft (Simulated)"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			fmt.Printf("To:      %s\n", to)
			fmt.Printf("Subject: %s\n", subject)
			fmt.Println(strings.Repeat("─", 50))
			if body != "" {
				fmt.Println(body)
				fmt.Println(strings.Repeat("─", 50))
			}
			fmt.Println()
			_, _ = common.Green.Println("✓ Draft would be created with ID: draft-demo-new")
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To create real drafts: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().StringVar(&to, "to", "recipient@example.com", "Recipient email")
	cmd.Flags().StringVar(&subject, "subject", "Draft Subject", "Email subject")
	cmd.Flags().StringVar(&body, "body", "", "Email body")

	return cmd
}

func newDemoEmailDraftsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [draft-id]",
		Short: "Delete a draft (simulated)",
		RunE: func(cmd *cobra.Command, args []string) error {
			draftID := "draft-001"
			if len(args) > 0 {
				draftID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📝 Demo Mode - Delete Draft (Simulated)"))
			fmt.Println()
			_, _ = common.Green.Printf("✓ Draft '%s' would be deleted\n", draftID)
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To manage real drafts: nylas auth login"))

			return nil
		},
	}
}

func newDemoEmailDraftsSendCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "send [draft-id]",
		Short: "Send a draft (simulated)",
		RunE: func(cmd *cobra.Command, args []string) error {
			draftID := "draft-001"
			if len(args) > 0 {
				draftID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📝 Demo Mode - Send Draft (Simulated)"))
			fmt.Println()
			_, _ = common.Green.Printf("✓ Draft '%s' would be sent\n", draftID)
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To send real drafts: nylas auth login"))

			return nil
		},
	}
}

// newDemoEmailAttachmentsCmd manages sample attachments.
func newDemoEmailAttachmentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attachments",
		Short: "Manage sample email attachments",
		Long:  "Demo attachment commands.",
	}

	cmd.AddCommand(newDemoEmailAttachmentsListCmd())
	cmd.AddCommand(newDemoEmailAttachmentsDownloadCmd())

	return cmd
}

func newDemoEmailAttachmentsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [message-id]",
		Short: "List attachments for a message",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			messageID := "msg-001"
			if len(args) > 0 {
				messageID = args[0]
			}

			attachments, _ := client.ListAttachments(ctx, "demo-grant", messageID)

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📎 Demo Mode - Sample Attachments"))
			fmt.Printf("Message: %s\n\n", messageID)

			for _, a := range attachments {
				fmt.Printf("  📎 %s\n", common.BoldWhite.Sprint(a.Filename))
				fmt.Printf("     Type: %s\n", a.ContentType)
				fmt.Printf("     Size: %s\n", formatDemoBytes(a.Size))
				_, _ = common.Dim.Printf("     ID: %s\n", a.ID)
				fmt.Println()
			}

			fmt.Println(common.Dim.Sprint("To view real attachments: nylas auth login"))

			return nil
		},
	}
}

func newDemoEmailAttachmentsDownloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "download [attachment-id]",
		Short: "Download an attachment (simulated)",
		RunE: func(cmd *cobra.Command, args []string) error {
			attachmentID := "attach-001"
			if len(args) > 0 {
				attachmentID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📎 Demo Mode - Download Attachment (Simulated)"))
			fmt.Println()
			_, _ = common.Green.Printf("✓ Attachment '%s' would be downloaded to current directory\n", attachmentID)
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To download real attachments: nylas auth login"))

			return nil
		},
	}
}

func formatDemoBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// newDemoEmailScheduledCmd manages scheduled messages.
