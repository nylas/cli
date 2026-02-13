// files.go provides CLI commands for managing Slack files.

package slack

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// newFilesCmd creates the files command group.
func newFilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "files",
		Aliases: []string{"file", "f"},
		Short:   "Manage Slack files and attachments",
		Long: `Commands for listing, viewing, and downloading files shared in Slack.

Examples:
  # List files in a channel
  nylas slack files list --channel general

  # List files by channel ID
  nylas slack files list --channel-id C1234567890

  # Show file details
  nylas slack files show F1234567890

  # Download a file
  nylas slack files download F1234567890

  # Download to specific path
  nylas slack files download F1234567890 -o ~/Downloads/photo.png`,
	}

	cmd.AddCommand(newFilesListCmd())
	cmd.AddCommand(newFilesShowCmd())
	cmd.AddCommand(newFilesDownloadCmd())

	return cmd
}

// newFilesListCmd creates the files list command.
func newFilesListCmd() *cobra.Command {
	var (
		channelID   string
		channelName string
		userID      string
		fileTypes   string
		limit       int
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List files in a channel or workspace",
		Long: `List files shared in a Slack channel or across the workspace.

Examples:
  # List all files you have access to
  nylas slack files list

  # List files in a specific channel
  nylas slack files list --channel general

  # List only images
  nylas slack files list --channel general --types images

  # List files uploaded by a specific user
  nylas slack files list --user U1234567890`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getSlackClientOrError()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			// Resolve channel name to ID if provided
			resolvedChannelID := channelID
			if channelName != "" && channelID == "" {
				resolvedChannelID, err = resolveChannelName(ctx, client, channelName)
				if err != nil {
					return common.NewUserError(fmt.Sprintf("channel not found: %s", channelName), "Use --channel-id with the channel ID instead")
				}
			}

			// Parse file types
			var types []string
			if fileTypes != "" {
				types = strings.Split(fileTypes, ",")
				for i := range types {
					types[i] = strings.TrimSpace(types[i])
				}
			}

			params := &domain.SlackFileQueryParams{
				ChannelID: resolvedChannelID,
				UserID:    userID,
				Types:     types,
				Limit:     limit,
			}

			resp, err := client.ListFiles(ctx, params)
			if err != nil {
				return common.WrapListError("files", err)
			}

			if len(resp.Files) == 0 {
				common.PrintEmptyState("files")
				return nil
			}

			// Handle structured output (JSON/YAML/quiet)
			if common.IsStructuredOutput(cmd) {
				out := common.GetOutputWriter(cmd)
				return out.Write(resp.Files)
			}

			fmt.Printf("Found %d file(s):\n\n", len(resp.Files))
			fmt.Println(strings.Repeat("─", 70))

			for i, f := range resp.Files {
				// File name and type
				_, _ = common.Bold.Printf("%d. %s", i+1, f.Name)
				if f.Title != "" && f.Title != f.Name {
					_, _ = common.Dim.Printf(" (%s)", f.Title)
				}
				fmt.Println()

				// Details
				fmt.Printf("   ID:   %s\n", common.Cyan.Sprint(f.ID))
				fmt.Printf("   Type: %s", f.MimeType)
				if f.FileType != "" {
					fmt.Printf(" (.%s)", f.FileType)
				}
				fmt.Println()
				fmt.Printf("   Size: %s\n", common.FormatSize(f.Size))

				// Image dimensions if available
				if f.ImageWidth > 0 && f.ImageHeight > 0 {
					fmt.Printf("   Dimensions: %dx%d\n", f.ImageWidth, f.ImageHeight)
				}

				// Created time
				if f.Created > 0 {
					created := time.Unix(f.Created, 0)
					_, _ = common.Dim.Printf("   Uploaded: %s\n", created.Format(common.DisplayDateTime))
				}

				if i < len(resp.Files)-1 {
					fmt.Println()
				}
			}

			fmt.Println(strings.Repeat("─", 70))
			fmt.Printf("\nUse 'nylas slack files download <file-id>' to download a file.\n")

			return nil
		},
	}

	cmd.Flags().StringVarP(&channelName, "channel", "c", "", "Channel name (without #)")
	cmd.Flags().StringVar(&channelID, "channel-id", "", "Channel ID")
	cmd.Flags().StringVarP(&userID, "user", "u", "", "Filter by user ID")
	cmd.Flags().StringVar(&fileTypes, "types", "", "Filter by file types (comma-separated: images,pdfs,docs)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum number of files to return")

	return cmd
}

// newFilesShowCmd creates the files show command.
func newFilesShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <file-id>",
		Short: "Show file details",
		Long:  "Display detailed metadata for a specific file.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fileID := args[0]

			client, err := getSlackClientOrError()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			file, err := client.GetFileInfo(ctx, fileID)
			if err != nil {
				return common.WrapGetError("file info", err)
			}

			// Handle structured output (JSON/YAML/quiet)
			if common.IsStructuredOutput(cmd) {
				out := common.GetOutputWriter(cmd)
				return out.Write(file)
			}

			fmt.Println(strings.Repeat("─", 60))
			_, _ = common.Bold.Printf("Name:      %s\n", file.Name)
			if file.Title != "" && file.Title != file.Name {
				fmt.Printf("Title:     %s\n", file.Title)
			}
			fmt.Printf("ID:        %s\n", common.Cyan.Sprint(file.ID))
			fmt.Printf("Type:      %s", file.MimeType)
			if file.FileType != "" {
				fmt.Printf(" (.%s)", file.FileType)
			}
			fmt.Println()
			fmt.Printf("Size:      %s (%d bytes)\n", common.FormatSize(file.Size), file.Size)

			if file.ImageWidth > 0 && file.ImageHeight > 0 {
				fmt.Printf("Dimensions: %dx%d pixels\n", file.ImageWidth, file.ImageHeight)
			}

			if file.Created > 0 {
				created := time.Unix(file.Created, 0)
				fmt.Printf("Uploaded:  %s\n", created.Format(common.DisplayDateTime))
			}

			if file.UserID != "" {
				_, _ = common.Dim.Printf("Uploader:  %s\n", file.UserID)
			}

			if file.Permalink != "" {
				_, _ = common.Dim.Printf("Link:      %s\n", file.Permalink)
			}

			fmt.Println(strings.Repeat("─", 60))

			return nil
		},
	}

	return cmd
}

// newFilesDownloadCmd creates the files download command.
func newFilesDownloadCmd() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "download <file-id>",
		Short: "Download a file",
		Long: `Download a file from Slack to your local machine.

Examples:
  # Download using original filename
  nylas slack files download F1234567890

  # Download to specific path
  nylas slack files download F1234567890 -o ~/Downloads/photo.png

  # Download to a directory (uses original filename)
  nylas slack files download F1234567890 -o ~/Downloads/`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fileID := args[0]

			client, err := getSlackClientOrError()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			// Get file metadata first to get filename and download URL
			file, err := client.GetFileInfo(ctx, fileID)
			if err != nil {
				return common.WrapGetError("file info", err)
			}

			if file.DownloadURL == "" {
				return common.NewUserError(
					"file has no download URL",
					"This file may be external or inaccessible",
				)
			}

			// Sanitize filename to prevent path traversal attacks
			// filepath.Base strips directory components like "../" or "../../"
			safeFilename := filepath.Base(file.Name)
			if safeFilename == "" || safeFilename == "." || safeFilename == ".." {
				safeFilename = "file"
				if file.FileType != "" {
					safeFilename += "." + file.FileType
				}
			}

			// Determine output path
			if outputPath == "" {
				outputPath = safeFilename
			}

			// Clean the output path to resolve . and ..
			outputPath = filepath.Clean(outputPath)

			// If outputPath is a directory, append sanitized filename
			if info, statErr := os.Stat(outputPath); statErr == nil && info.IsDir() {
				outputPath = filepath.Join(outputPath, safeFilename)
			}

			// Security: Ensure output path is within current directory or temp directory
			absPath, err := filepath.Abs(outputPath)
			if err != nil {
				return common.WrapGetError("output path", err)
			}
			cwd, err := os.Getwd()
			if err != nil {
				return common.WrapGetError("current directory", err)
			}
			tempDir := os.TempDir()
			if !strings.HasPrefix(absPath, cwd) && !strings.HasPrefix(absPath, tempDir) {
				return common.NewInputError(fmt.Sprintf("output path must be within current directory or temp directory (got: %s)", absPath))
			}

			// Validate the final path is not a directory
			if info, statErr := os.Stat(outputPath); statErr == nil && info.IsDir() {
				return common.NewInputError(fmt.Sprintf("output path is a directory: %s", outputPath))
			}

			// Download the file
			reader, err := client.DownloadFile(ctx, file.DownloadURL)
			if err != nil {
				return common.WrapDownloadError("file", err)
			}
			defer func() { _ = reader.Close() }()

			// Create output file
			outFile, err := os.Create(outputPath)
			if err != nil {
				return common.WrapCreateError("output file", err)
			}
			defer func() { _ = outFile.Close() }()

			// Copy content
			written, err := io.Copy(outFile, reader)
			if err != nil {
				return common.WrapWriteError("file", err)
			}

			_, _ = common.Green.Printf("✓ Downloaded %s (%s) to %s\n", file.Name, common.FormatSize(written), outputPath)

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: original filename)")

	return cmd
}
