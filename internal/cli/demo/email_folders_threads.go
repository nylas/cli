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

func newDemoEmailFoldersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "folders",
		Short: "Manage sample email folders",
		Long:  "Demo folder commands showing sample folders.",
	}

	cmd.AddCommand(newDemoEmailFoldersListCmd())
	cmd.AddCommand(newDemoEmailFoldersCreateCmd())
	cmd.AddCommand(newDemoEmailFoldersDeleteCmd())

	return cmd
}

func newDemoEmailFoldersListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List sample folders",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			folders, _ := client.GetFolders(ctx, "demo-grant")

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📁 Demo Mode - Sample Folders"))
			fmt.Println()

			for _, f := range folders {
				system := ""
				if f.SystemFolder != "" {
					system = common.Dim.Sprintf(" (%s)", f.SystemFolder)
				}
				fmt.Printf("  📁 %-15s %s%s\n", f.Name, common.Dim.Sprintf("%d items", f.TotalCount), system)
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("To manage your real folders: nylas auth login"))

			return nil
		},
	}
}

func newDemoEmailFoldersCreateCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a folder (simulated)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				name = "New Folder"
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📁 Demo Mode - Create Folder (Simulated)"))
			fmt.Println()
			_, _ = common.Green.Printf("✓ Folder '%s' would be created\n", name)
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To create real folders: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Folder name")

	return cmd
}

func newDemoEmailFoldersDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [folder-id]",
		Short: "Delete a folder (simulated)",
		RunE: func(cmd *cobra.Command, args []string) error {
			folderID := "work"
			if len(args) > 0 {
				folderID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📁 Demo Mode - Delete Folder (Simulated)"))
			fmt.Println()
			_, _ = common.Green.Printf("✓ Folder '%s' would be deleted\n", folderID)
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To manage real folders: nylas auth login"))

			return nil
		},
	}
}

// newDemoEmailThreadsCmd manages sample threads.
func newDemoEmailThreadsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "threads",
		Short: "Manage sample email threads",
		Long:  "Demo thread commands showing sample conversations.",
	}

	cmd.AddCommand(newDemoEmailThreadsListCmd())
	cmd.AddCommand(newDemoEmailThreadsReadCmd())

	return cmd
}

func newDemoEmailThreadsListCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sample threads",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			threads, _ := client.GetThreads(ctx, "demo-grant", nil)

			if limit > 0 && limit < len(threads) {
				threads = threads[:limit]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📧 Demo Mode - Sample Threads"))
			fmt.Println()
			fmt.Printf("Found %d threads:\n\n", len(threads))

			for _, t := range threads {
				status := " "
				if t.Unread {
					status = common.Cyan.Sprint("●")
				}
				star := " "
				if t.Starred {
					star = common.Yellow.Sprint("★")
				}

				subject := t.Subject
				if len(subject) > 40 {
					subject = subject[:37] + "..."
				}

				fmt.Printf("%s %s %-40s %s\n", status, star, subject, common.Dim.Sprintf("%d messages", len(t.MessageIDs)))
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("To view your real threads: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of threads to show")

	return cmd
}

func newDemoEmailThreadsReadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "read [thread-id]",
		Short: "Read a sample thread",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			threadID := "thread-001"
			if len(args) > 0 {
				threadID = args[0]
			}

			thread, _ := client.GetThread(ctx, "demo-grant", threadID)

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📧 Demo Mode - Sample Thread"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 60))
			_, _ = common.BoldWhite.Printf("Subject: %s\n", thread.Subject)
			fmt.Printf("Messages: %d\n", len(thread.MessageIDs))
			fmt.Printf("Participants: %s\n", formatDemoParticipants(thread.Participants))
			fmt.Println(strings.Repeat("─", 60))
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To view your real threads: nylas auth login"))

			return nil
		},
	}
}

func formatDemoParticipants(participants []domain.EmailParticipant) string {
	names := make([]string, len(participants))
	for i, p := range participants {
		if p.Name != "" {
			names[i] = p.Name
		} else {
			names[i] = p.Email
		}
	}
	return strings.Join(names, ", ")
}

// newDemoEmailDraftsCmd manages sample drafts.
