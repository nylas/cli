package demo

import (
	"fmt"
	"strings"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newDemoContactsGroupsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "groups",
		Short: "Manage contact groups",
		Long:  "Demo commands for managing contact groups.",
	}

	cmd.AddCommand(newDemoGroupsListCmd())
	cmd.AddCommand(newDemoGroupsShowCmd())
	cmd.AddCommand(newDemoGroupsCreateCmd())
	cmd.AddCommand(newDemoGroupsDeleteCmd())
	cmd.AddCommand(newDemoGroupsAddMemberCmd())
	cmd.AddCommand(newDemoGroupsRemoveMemberCmd())

	return cmd
}

func newDemoGroupsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List contact groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println()
			fmt.Println(common.Dim.Sprint("👥 Demo Mode - Contact Groups"))
			fmt.Println()

			groups := []struct {
				name  string
				count int
			}{
				{"Work Colleagues", 25},
				{"Friends & Family", 42},
				{"Clients", 18},
				{"Vendors", 12},
				{"Newsletter Subscribers", 156},
			}

			for _, g := range groups {
				fmt.Printf("  %s %s %s\n", common.Cyan.Sprint("●"), common.BoldWhite.Sprint(g.name), common.Dim.Sprintf("(%d)", g.count))
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("To manage your real contact groups: nylas auth login"))

			return nil
		},
	}
}

func newDemoGroupsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [group-id]",
		Short: "Show group details",
		RunE: func(cmd *cobra.Command, args []string) error {
			groupName := "Work Colleagues"
			if len(args) > 0 {
				groupName = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("👥 Demo Mode - Contact Group Details"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			_, _ = common.BoldWhite.Printf("Group: %s\n", groupName)
			fmt.Printf("  Members:     25\n")
			fmt.Printf("  Created:     Jan 15, 2024\n")
			fmt.Printf("  Description: Team members and work contacts\n")
			fmt.Println()
			fmt.Println("Members:")
			fmt.Printf("  • John Smith (john@example.com)\n")
			fmt.Printf("  • Jane Doe (jane@example.com)\n")
			fmt.Printf("  • Bob Wilson (bob@example.com)\n")
			_, _ = common.Dim.Printf("  ... and 22 more\n")
			fmt.Println(strings.Repeat("─", 50))

			return nil
		},
	}
}

func newDemoGroupsCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create [name]",
		Short: "Create a contact group",
		RunE: func(cmd *cobra.Command, args []string) error {
			groupName := "New Group"
			if len(args) > 0 {
				groupName = args[0]
			}

			fmt.Println()
			_, _ = common.Green.Printf("✓ Group '%s' would be created (demo mode)\n", groupName)
			_, _ = common.Dim.Printf("  Group ID: group-demo-%d\n", time.Now().Unix())

			return nil
		},
	}
}

func newDemoGroupsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [group-id]",
		Short: "Delete a contact group",
		RunE: func(cmd *cobra.Command, args []string) error {
			groupID := "group-demo-123"
			if len(args) > 0 {
				groupID = args[0]
			}

			fmt.Println()
			_, _ = common.Green.Printf("✓ Group %s would be deleted (demo mode)\n", groupID)

			return nil
		},
	}
}

func newDemoGroupsAddMemberCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-member [group-id] [contact-id]",
		Short: "Add a contact to a group",
		RunE: func(cmd *cobra.Command, args []string) error {
			groupID := "group-demo-123"
			contactID := "contact-demo-456"
			if len(args) > 0 {
				groupID = args[0]
			}
			if len(args) > 1 {
				contactID = args[1]
			}

			fmt.Println()
			_, _ = common.Green.Printf("✓ Contact %s would be added to group %s (demo mode)\n", contactID, groupID)

			return nil
		},
	}
}

func newDemoGroupsRemoveMemberCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-member [group-id] [contact-id]",
		Short: "Remove a contact from a group",
		RunE: func(cmd *cobra.Command, args []string) error {
			groupID := "group-demo-123"
			contactID := "contact-demo-456"
			if len(args) > 0 {
				groupID = args[0]
			}
			if len(args) > 1 {
				contactID = args[1]
			}

			fmt.Println()
			_, _ = common.Green.Printf("✓ Contact %s would be removed from group %s (demo mode)\n", contactID, groupID)

			return nil
		},
	}
}

// ============================================================================
// PHOTO COMMAND
// ============================================================================

// newDemoContactsPhotoCmd creates the photo subcommand.
