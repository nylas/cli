package email

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
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

			client, err := getClient()
			if err != nil {
				return err
			}

			grantID, err := common.GetGrantID(args[1:])
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			attachments, err := client.ListAttachments(ctx, grantID, messageID)
			if err != nil {
				return common.WrapListError("attachments", err)
			}

			if len(attachments) == 0 {
				common.PrintEmptyState("attachments")
				return nil
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

			return nil
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

			client, err := getClient()
			if err != nil {
				return err
			}

			grantID, err := common.GetGrantID(args[2:])
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			attachment, err := client.GetAttachment(ctx, grantID, messageID, attachmentID)
			if err != nil {
				return common.WrapGetError("attachment", err)
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

			return nil
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

			client, err := getClient()
			if err != nil {
				return err
			}

			grantID, err := common.GetGrantID(args[2:])
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			// Get attachment metadata first to get filename
			attachment, err := client.GetAttachment(ctx, grantID, messageID, attachmentID)
			if err != nil {
				return common.WrapGetError("attachment metadata", err)
			}

			// Sanitize filename to prevent path traversal attacks
			// filepath.Base strips directory components like "../" or "../../"
			safeFilename := filepath.Base(attachment.Filename)
			if safeFilename == "" || safeFilename == "." || safeFilename == ".." {
				safeFilename = "attachment"
			}

			// Determine output path
			if outputPath == "" {
				outputPath = safeFilename
			}

			// Clean the output path to resolve . and ..
			outputPath = filepath.Clean(outputPath)

			// If outputPath is a directory, append sanitized filename
			if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
				outputPath = filepath.Join(outputPath, safeFilename)
			}

			// Validate the final path is not a directory
			if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
				return common.NewInputError(fmt.Sprintf("output path is a directory: %s", outputPath))
			}

			// Download the attachment
			reader, err := client.DownloadAttachment(ctx, grantID, messageID, attachmentID)
			if err != nil {
				return common.WrapDownloadError("attachment", err)
			}
			defer func() { _ = reader.Close() }()

			// Create output file
			file, err := os.Create(outputPath)
			if err != nil {
				return common.WrapCreateError("output file", err)
			}
			defer func() { _ = file.Close() }()

			// Copy content
			written, err := io.Copy(file, reader)
			if err != nil {
				return common.WrapWriteError("file", err)
			}

			printSuccess("Downloaded %s (%s) to %s", attachment.Filename, common.FormatSize(written), outputPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: original filename)")

	return cmd
}
