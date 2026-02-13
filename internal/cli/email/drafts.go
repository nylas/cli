package email

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newDraftsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drafts",
		Short: "Manage email drafts",
		Long:  "List, create, edit, and send draft emails.",
	}

	cmd.AddCommand(newDraftsListCmd())
	cmd.AddCommand(newDraftsCreateCmd())
	cmd.AddCommand(newDraftsShowCmd())
	cmd.AddCommand(newDraftsSendCmd())
	cmd.AddCommand(newDraftsDeleteCmd())

	return cmd
}

func newDraftsListCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list [grant-id]",
		Short: "List drafts",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				drafts, err := client.GetDrafts(ctx, grantID, limit)
				if err != nil {
					return struct{}{}, common.WrapGetError("drafts", err)
				}

				// JSON output (including empty array)
				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(drafts)
				}

				if len(drafts) == 0 {
					common.PrintEmptyState("drafts")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d drafts:\n\n", len(drafts))
				fmt.Printf("%-15s %-25s %-35s %s\n", "ID", "TO", "SUBJECT", "UPDATED")
				fmt.Println("--------------------------------------------------------------------------------")

				for _, d := range drafts {
					toStr := ""
					if len(d.To) > 0 {
						toStr = common.FormatParticipants(d.To)
					}
					if len(toStr) > 23 {
						toStr = toStr[:20] + "..."
					}

					subj := d.Subject
					if subj == "" {
						subj = "(no subject)"
					}
					if len(subj) > 33 {
						subj = subj[:30] + "..."
					}

					// Show first 12 chars of ID
					idShort := d.ID
					if len(idShort) > 12 {
						idShort = idShort[:12] + "..."
					}

					dateStr := common.FormatTimeAgo(d.UpdatedAt)

					fmt.Printf("%-15s %-25s %-35s %s\n", idShort, toStr, subj, common.Dim.Sprint(dateStr))
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of drafts to fetch")

	return cmd
}

func newDraftsCreateCmd() *cobra.Command {
	var to []string
	var cc []string
	var subject string
	var body string
	var replyTo string
	var attachFiles []string

	cmd := &cobra.Command{
		Use:   "create [grant-id]",
		Short: "Create a new draft",
		Long:  "Create a new draft email with optional attachments.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Interactive mode if nothing provided (runs before WithClient)
			if len(to) == 0 && subject == "" && body == "" && len(attachFiles) == 0 {
				reader := bufio.NewReader(os.Stdin)

				fmt.Print("To (comma-separated, optional): ")
				input, _ := reader.ReadString('\n')
				to = parseEmails(strings.TrimSpace(input))

				fmt.Print("Subject: ")
				subject, _ = reader.ReadString('\n')
				subject = strings.TrimSpace(subject)

				fmt.Println("Body (end with a line containing only '.'):")
				var bodyLines []string
				for {
					line, _ := reader.ReadString('\n')
					line = strings.TrimSuffix(line, "\n")
					if line == "." {
						break
					}
					bodyLines = append(bodyLines, line)
				}
				body = strings.Join(bodyLines, "\n")
			}

			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				// Parse and validate recipients
				toContacts, err := parseContacts(to)
				if err != nil {
					return struct{}{}, common.WrapRecipientError("to", err)
				}

				req := &domain.CreateDraftRequest{
					Subject:      subject,
					Body:         body,
					To:           toContacts,
					ReplyToMsgID: replyTo,
				}

				if len(cc) > 0 {
					ccContacts, err := parseContacts(cc)
					if err != nil {
						return struct{}{}, common.WrapRecipientError("cc", err)
					}
					req.Cc = ccContacts
				}

				// Load attachments from files
				if len(attachFiles) > 0 {
					attachments, err := loadAttachmentsFromFiles(attachFiles)
					if err != nil {
						return struct{}{}, common.WrapLoadError("attachments", err)
					}
					req.Attachments = attachments
					fmt.Printf("Attaching %d file(s)...\n", len(attachments))
				}

				draft, err := client.CreateDraft(ctx, grantID, req)
				if err != nil {
					return struct{}{}, common.WrapCreateError("draft", err)
				}

				printSuccess("Draft created! ID: %s", draft.ID)
				if len(draft.Attachments) > 0 {
					fmt.Printf("  Attachments: %d\n", len(draft.Attachments))
				}
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringSliceVarP(&to, "to", "t", nil, "Recipient email addresses")
	cmd.Flags().StringSliceVar(&cc, "cc", nil, "CC email addresses")
	cmd.Flags().StringVarP(&subject, "subject", "s", "", "Email subject")
	cmd.Flags().StringVarP(&body, "body", "b", "", "Email body")
	cmd.Flags().StringVar(&replyTo, "reply-to", "", "Message ID to reply to")
	cmd.Flags().StringSliceVarP(&attachFiles, "attach", "a", nil, "File paths to attach")

	return cmd
}

// loadAttachmentsFromFiles reads files and creates attachment objects.
func loadAttachmentsFromFiles(filePaths []string) ([]domain.Attachment, error) {
	attachments := make([]domain.Attachment, 0, len(filePaths))

	for _, path := range filePaths {
		// Clean the path to resolve . and .. and validate it
		cleanPath := filepath.Clean(path)

		// Ensure the path exists and is a regular file
		info, err := os.Stat(cleanPath)
		if err != nil {
			return nil, fmt.Errorf("cannot access file %s: %w", path, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("path is a directory, not a file: %s", path)
		}

		file, err := os.Open(cleanPath)
		if err != nil {
			return nil, fmt.Errorf("cannot open file %s: %w", path, err)
		}

		content, err := io.ReadAll(file)
		_ = file.Close()
		if err != nil {
			return nil, fmt.Errorf("cannot read file %s: %w", path, err)
		}

		filename := filepath.Base(path)
		contentType := detectContentType(filename, content)

		attachments = append(attachments, domain.Attachment{
			Filename:    filename,
			ContentType: contentType,
			Content:     content,
			Size:        int64(len(content)),
		})
	}

	return attachments, nil
}

// detectContentType tries to determine the MIME type from filename extension or content.
func detectContentType(filename string, content []byte) string {
	// Try extension first
	ext := filepath.Ext(filename)
	if ext != "" {
		mimeType := mime.TypeByExtension(ext)
		if mimeType != "" {
			return mimeType
		}
	}

	// Fall back to content sniffing (basic)
	// http.DetectContentType only looks at first 512 bytes
	if len(content) > 0 {
		// Check for common file signatures
		if len(content) >= 4 {
			switch {
			case content[0] == 0x25 && content[1] == 0x50 && content[2] == 0x44 && content[3] == 0x46:
				return "application/pdf"
			case content[0] == 0x50 && content[1] == 0x4B && content[2] == 0x03 && content[3] == 0x04:
				// ZIP-based formats (docx, xlsx, pptx, etc.)
				if strings.HasSuffix(strings.ToLower(filename), ".docx") {
					return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
				} else if strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
					return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
				} else if strings.HasSuffix(strings.ToLower(filename), ".pptx") {
					return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
				}
				return "application/zip"
			case content[0] == 0x89 && content[1] == 0x50 && content[2] == 0x4E && content[3] == 0x47:
				return "image/png"
			case content[0] == 0xFF && content[1] == 0xD8 && content[2] == 0xFF:
				return "image/jpeg"
			case content[0] == 0x47 && content[1] == 0x49 && content[2] == 0x46:
				return "image/gif"
			}
		}
	}

	return "application/octet-stream"
}

func newDraftsShowCmd() *cobra.Command {
	client, _ := common.GetNylasClient()

	return common.NewShowCommand(common.ShowCommandConfig{
		Use:          "show <draft-id> [grant-id]",
		Short:        "Show draft details",
		ResourceName: "draft",
		GetFunc: func(ctx context.Context, grantID, resourceID string) (interface{}, error) {
			return client.GetDraft(ctx, grantID, resourceID)
		},
		DisplayFunc: func(resource interface{}) error {
			draft := resource.(*domain.Draft)

			fmt.Println("════════════════════════════════════════════════════════════")
			_, _ = common.BoldWhite.Printf("Draft: %s\n", draft.Subject)
			fmt.Println("════════════════════════════════════════════════════════════")

			fmt.Printf("ID:      %s\n", draft.ID)
			if len(draft.To) > 0 {
				fmt.Printf("To:      %s\n", common.FormatParticipants(draft.To))
			}
			if len(draft.Cc) > 0 {
				fmt.Printf("Cc:      %s\n", common.FormatParticipants(draft.Cc))
			}
			fmt.Printf("Updated: %s\n", draft.UpdatedAt.Format(common.DisplayDateTime))

			// Show attachments if any
			if len(draft.Attachments) > 0 {
				fmt.Printf("\nAttachments (%d):\n", len(draft.Attachments))
				for i, a := range draft.Attachments {
					fmt.Printf("  %d. %s (%s, %s)\n", i+1, a.Filename, a.ContentType, common.FormatSize(a.Size))
				}
			}

			if draft.Body != "" {
				fmt.Println("\nBody:")
				fmt.Println("────────────────────────────────────────────────────────────")
				fmt.Println(common.StripHTML(draft.Body))
			}

			return nil
		},
		GetClient: common.GetNylasClient,
	})
}

func newDraftsSendCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "send <draft-id> [grant-id]",
		Short: "Send a draft",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			draftID := args[0]
			remainingArgs := args[1:]

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				// Get draft info first
				draft, err := client.GetDraft(ctx, grantID, draftID)
				if err != nil {
					return struct{}{}, common.WrapGetError("draft", err)
				}

				// Confirmation
				if !force {
					fmt.Println("Send this draft?")
					fmt.Printf("  To:      %s\n", common.FormatParticipants(draft.To))
					fmt.Printf("  Subject: %s\n", draft.Subject)
					fmt.Print("\n[y/N]: ")

					var confirm string
					_, _ = fmt.Scanln(&confirm) // Ignore error - empty string treated as "no"
					if confirm != "y" && confirm != "Y" && confirm != "yes" {
						fmt.Println("Cancelled.")
						return struct{}{}, nil
					}
				}

				msg, err := client.SendDraft(ctx, grantID, draftID)
				if err != nil {
					return struct{}{}, common.WrapSendError("draft", err)
				}

				printSuccess("Draft sent! Message ID: %s", msg.ID)
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation")

	return cmd
}

func newDraftsDeleteCmd() *cobra.Command {
	return common.NewDeleteCommand(common.DeleteCommandConfig{
		Use:          "delete <draft-id> [grant-id]",
		Short:        "Delete a draft",
		ResourceName: "draft",
		DeleteFunc: func(ctx context.Context, grantID, resourceID string) error {
			client, err := common.GetNylasClient()
			if err != nil {
				return err
			}
			return client.DeleteDraft(ctx, grantID, resourceID)
		},
		GetClient: common.GetNylasClient,
	})
}
