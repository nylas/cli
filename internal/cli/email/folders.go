package email

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newFoldersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "folders",
		Short: "Manage email folders/labels",
		Long:  "List, create, update, and delete email folders or labels.",
	}

	cmd.AddCommand(newFoldersListCmd())
	cmd.AddCommand(newFoldersShowCmd())
	cmd.AddCommand(newFoldersCreateCmd())
	cmd.AddCommand(newFoldersRenameCmd())
	cmd.AddCommand(newFoldersDeleteCmd())

	return cmd
}

func newFoldersListCmd() *cobra.Command {
	var showID bool

	cmd := &cobra.Command{
		Use:   "list [grant-id]",
		Short: "List all folders",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				folders, err := client.GetFolders(ctx, grantID)
				if err != nil {
					return struct{}{}, common.WrapGetError("folders", err)
				}

				// JSON output (including empty array)
				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(folders)
				}

				if len(folders) == 0 {
					common.PrintEmptyState("folders")
					return struct{}{}, nil
				}

				fmt.Println("Folders:")
				fmt.Println()

				if showID {
					fmt.Printf("%-36s %-30s %-12s %8s %8s\n", "ID", "NAME", "TYPE", "TOTAL", "UNREAD")
					fmt.Println("------------------------------------------------------------------------------------------------------")
				} else {
					fmt.Printf("%-30s %-12s %8s %8s\n", "NAME", "TYPE", "TOTAL", "UNREAD")
					fmt.Println("------------------------------------------------------------")
				}

				for _, f := range folders {
					folderType := f.SystemFolder
					if folderType == "" {
						folderType = "custom"
					}

					name := f.Name
					if len(name) > 28 {
						name = name[:25] + "..."
					}

					unreadPadded := fmt.Sprintf("%8d", f.UnreadCount)
					if f.UnreadCount > 0 {
						unreadPadded = common.Cyan.Sprint(unreadPadded)
					}

					if showID {
						fmt.Printf("%-36s %-30s %-12s %8d %s\n",
							common.Dim.Sprint(f.ID), name, folderType, f.TotalCount, unreadPadded)
					} else {
						fmt.Printf("%-30s %-12s %8d %s\n",
							name, folderType, f.TotalCount, unreadPadded)
					}
				}

				fmt.Println()
				if !showID {
					_, _ = common.Dim.Printf("Use --id to see folder IDs for --folder flag\n")
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVar(&showID, "id", false, "Show folder IDs")

	return cmd
}

func newFoldersShowCmd() *cobra.Command {
	client, _ := common.GetNylasClient()

	return common.NewShowCommand(common.ShowCommandConfig{
		Use:          "show <folder-id> [grant-id]",
		Short:        "Show folder details",
		ResourceName: "folder",
		GetFunc: func(ctx context.Context, grantID, resourceID string) (interface{}, error) {
			return client.GetFolder(ctx, grantID, resourceID)
		},
		DisplayFunc: func(resource interface{}) error {
			folder := resource.(*domain.Folder)

			fmt.Println("════════════════════════════════════════════════════════════")
			_, _ = common.BoldWhite.Printf("Folder: %s\n", folder.Name)
			fmt.Println("════════════════════════════════════════════════════════════")

			fmt.Printf("ID:           %s\n", folder.ID)
			fmt.Printf("Name:         %s\n", folder.Name)

			if folder.SystemFolder != "" {
				fmt.Printf("System Type:  %s\n", folder.SystemFolder)
			}
			if folder.ParentID != "" {
				fmt.Printf("Parent ID:    %s\n", folder.ParentID)
			}

			fmt.Printf("Total Count:  %d\n", folder.TotalCount)
			if folder.UnreadCount > 0 {
				_, _ = common.Cyan.Printf("Unread Count: %d\n", folder.UnreadCount)
			} else {
				fmt.Printf("Unread Count: %d\n", folder.UnreadCount)
			}

			if folder.BackgroundColor != "" {
				fmt.Printf("Background:   %s\n", folder.BackgroundColor)
			}
			if folder.TextColor != "" {
				fmt.Printf("Text Color:   %s\n", folder.TextColor)
			}

			return nil
		},
		GetClient: common.GetNylasClient,
	})
}

func newFoldersCreateCmd() *cobra.Command {
	var parentID string
	var bgColor string
	var textColor string

	cmd := &cobra.Command{
		Use:   "create <name> [grant-id]",
		Short: "Create a new folder",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			remainingArgs := args[1:]

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				req := &domain.CreateFolderRequest{
					Name:            name,
					ParentID:        parentID,
					BackgroundColor: bgColor,
					TextColor:       textColor,
				}

				folder, err := client.CreateFolder(ctx, grantID, req)
				if err != nil {
					return struct{}{}, common.WrapCreateError("folder", err)
				}

				printSuccess("Created folder '%s' (ID: %s)", folder.Name, folder.ID)
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent folder ID")
	cmd.Flags().StringVar(&bgColor, "bg-color", "", "Background color (hex)")
	cmd.Flags().StringVar(&textColor, "text-color", "", "Text color (hex)")

	return cmd
}

func newFoldersRenameCmd() *cobra.Command {
	var bgColor string
	var textColor string
	var parentID string

	cmd := &cobra.Command{
		Use:   "rename <folder-id> <new-name> [grant-id]",
		Short: "Rename a folder",
		Long:  "Rename a folder and optionally update its colors.",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			folderID := args[0]
			newName := args[1]
			remainingArgs := args[2:]

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				req := &domain.UpdateFolderRequest{
					Name: newName,
				}

				if cmd.Flags().Changed("bg-color") {
					req.BackgroundColor = bgColor
				}
				if cmd.Flags().Changed("text-color") {
					req.TextColor = textColor
				}
				if cmd.Flags().Changed("parent") {
					req.ParentID = parentID
				}

				folder, err := client.UpdateFolder(ctx, grantID, folderID, req)
				if err != nil {
					return struct{}{}, common.WrapUpdateError("folder", err)
				}

				printSuccess("Folder renamed to '%s'", folder.Name)
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&bgColor, "bg-color", "", "Background color (hex)")
	cmd.Flags().StringVar(&textColor, "text-color", "", "Text color (hex)")
	cmd.Flags().StringVar(&parentID, "parent", "", "Parent folder ID")

	return cmd
}

func newFoldersDeleteCmd() *cobra.Command {
	return common.NewDeleteCommand(common.DeleteCommandConfig{
		Use:          "delete <folder-id> [grant-id]",
		Short:        "Delete a folder",
		ResourceName: "folder",
		DeleteFunc: func(ctx context.Context, grantID, resourceID string) error {
			client, err := common.GetNylasClient()
			if err != nil {
				return err
			}
			return client.DeleteFolder(ctx, grantID, resourceID)
		},
		GetClient: common.GetNylasClient,
	})
}
