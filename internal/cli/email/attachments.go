package email

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

// newAttachmentsCmd creates the attachments command group.
func newAttachmentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attachments",
		Short: "Manage email attachments",
		Long:  "Commands to list, view, and download email attachments.",
	}

	cmd.AddCommand(newAttachmentsListCmd())
	cmd.AddCommand(newAttachmentsShowCmd())
	cmd.AddCommand(newAttachmentsDownloadCmd())

	return cmd
}

// newAttachmentsListCmd creates the attachments list command.
func newAttachmentsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <message-id> [grant-id]",
		Short: "List attachments in a message",
		Long:  "List all attachments in a specific email message.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			messageID := args[0]
			remainingArgs := args[1:]

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				attachments, err := client.ListAttachments(ctx, grantID, messageID)
				if err != nil {
					return struct{}{}, common.WrapListError("attachments", err)
				}

				// JSON output (including empty array)
				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(attachments)
				}

				if len(attachments) == 0 {
					common.PrintEmptyState("attachments")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d attachment(s):\n\n", len(attachments))
				fmt.Println(strings.Repeat("─", 70))

				for i, a := range attachments {
					fmt.Printf("%d. %s\n", i+1, common.BoldWhite.Sprint(a.Filename))
					fmt.Printf("   ID:   %s\n", a.ID)
					fmt.Printf("   Type: %s\n", a.ContentType)
					fmt.Printf("   Size: %s\n", common.FormatSize(a.Size))
					if a.IsInline {
						fmt.Printf("   Inline: yes\n")
					}
					if i < len(attachments)-1 {
						fmt.Println()
					}
				}

				fmt.Println(strings.Repeat("─", 70))
				fmt.Printf("\nUse 'nylas email attachments download <attachment-id> <message-id>' to download.\n")

				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}

// newAttachmentsShowCmd creates the attachments show command.
func newAttachmentsShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <attachment-id> <message-id> [grant-id]",
		Short: "Show attachment metadata",
		Long:  "Display detailed metadata for a specific attachment.",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			attachmentID := args[0]
			messageID := args[1]
			remainingArgs := args[2:]

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				attachment, err := client.GetAttachment(ctx, grantID, messageID, attachmentID)
				if err != nil {
					return struct{}{}, common.WrapGetError("attachment", err)
				}

				fmt.Println(strings.Repeat("─", 60))
				_, _ = common.BoldWhite.Printf("Filename:     %s\n", attachment.Filename)
				fmt.Printf("ID:           %s\n", attachment.ID)
				fmt.Printf("Content Type: %s\n", attachment.ContentType)
				fmt.Printf("Size:         %s (%d bytes)\n", common.FormatSize(attachment.Size), attachment.Size)
				if attachment.ContentID != "" {
					fmt.Printf("Content ID:   %s\n", attachment.ContentID)
				}
				fmt.Printf("Inline:       %v\n", attachment.IsInline)
				fmt.Println(strings.Repeat("─", 60))

				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}

// newAttachmentsDownloadCmd creates the attachments download command.
func newAttachmentsDownloadCmd() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "download <attachment-id> <message-id> [grant-id]",
		Short: "Download an attachment",
		Long:  "Download an attachment to a local file.",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			attachmentID := args[0]
			messageID := args[1]
			remainingArgs := args[2:]

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				// Get attachment metadata first to get filename
				attachment, err := client.GetAttachment(ctx, grantID, messageID, attachmentID)
				if err != nil {
					return struct{}{}, common.WrapGetError("attachment metadata", err)
				}

				// Sanitize filename to prevent path traversal attacks
				// filepath.Base strips directory components like "../" or "../../"
				safeFilename := filepath.Base(attachment.Filename)
				if safeFilename == "" || safeFilename == "." || safeFilename == ".." {
					safeFilename = "attachment"
				}

				// Determine output path
				finalOutputPath := outputPath
				if finalOutputPath == "" {
					finalOutputPath = safeFilename
				}

				// Clean the output path to resolve . and ..
				finalOutputPath = filepath.Clean(finalOutputPath)

				// If outputPath is a directory, append sanitized filename
				if info, err := os.Stat(finalOutputPath); err == nil && info.IsDir() {
					finalOutputPath = filepath.Join(finalOutputPath, safeFilename)
				}

				// Validate the final path is not a directory
				if info, err := os.Stat(finalOutputPath); err == nil && info.IsDir() {
					return struct{}{}, common.NewInputError(fmt.Sprintf("output path is a directory: %s", finalOutputPath))
				}

				// Download the attachment
				reader, err := client.DownloadAttachment(ctx, grantID, messageID, attachmentID)
				if err != nil {
					return struct{}{}, common.WrapDownloadError("attachment", err)
				}
				defer func() { _ = reader.Close() }()

				// Create output file
				file, err := os.Create(finalOutputPath)
				if err != nil {
					return struct{}{}, common.WrapCreateError("output file", err)
				}
				defer func() { _ = file.Close() }()

				// Copy content
				written, err := io.Copy(file, reader)
				if err != nil {
					return struct{}{}, common.WrapWriteError("file", err)
				}

				printSuccess("Downloaded %s (%s) to %s", attachment.Filename, common.FormatSize(written), finalOutputPath)
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: original filename)")

	return cmd
}
