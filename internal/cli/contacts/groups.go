package contacts

import (
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newGroupsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "groups",
		Aliases: []string{"group"},
		Short:   "Manage contact groups",
		Long:    "List, create, update, and delete contact groups.",
	}

	cmd.AddCommand(newGroupsListCmd())
	cmd.AddCommand(newGroupsShowCmd())
	cmd.AddCommand(newGroupsCreateCmd())
	cmd.AddCommand(newGroupsUpdateCmd())
	cmd.AddCommand(newGroupsDeleteCmd())

	return cmd
}

func newGroupsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list [grant-id]",
		Aliases: []string{"ls"},
		Short:   "List contact groups",
		Long:    "List all contact groups for the specified grant or default account.",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			grantID, err := getGrantID(args)
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			groups, err := client.GetContactGroups(ctx, grantID)
			if err != nil {
				return common.WrapListError("contact groups", err)
			}

			if len(groups) == 0 {
				common.PrintEmptyState("contact groups")
				return nil
			}

			fmt.Printf("Found %d contact group(s):\n\n", len(groups))

			table := common.NewTable("NAME", "ID", "PATH")
			for _, group := range groups {
				table.AddRow(
					common.Cyan.Sprint(group.Name),
					common.Dim.Sprint(group.ID),
					group.Path,
				)
			}
			table.Render()

			return nil
		},
	}
}

func newGroupsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <group-id> [grant-id]",
		Short: "Show contact group details",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupID := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

			var grantID string
			if len(args) > 1 {
				grantID = args[1]
			} else {
				grantID, err = getGrantID(nil)
				if err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			group, err := client.GetContactGroup(ctx, grantID, groupID)
			if err != nil {
				return common.WrapGetError("contact group", err)
			}

			fmt.Println("════════════════════════════════════════════════════════════")
			_, _ = common.BoldWhite.Printf("Contact Group: %s\n", group.Name)
			fmt.Println("════════════════════════════════════════════════════════════")

			fmt.Printf("ID:   %s\n", common.Dim.Sprint(group.ID))
			fmt.Printf("Name: %s\n", group.Name)
			if group.Path != "" {
				fmt.Printf("Path: %s\n", group.Path)
			}

			return nil
		},
	}
}

func newGroupsCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name> [grant-id]",
		Short: "Create a new contact group",
		Long: `Create a new contact group.

Examples:
  nylas contacts groups create "VIP Clients"
  nylas contacts groups create "Team Members" <grant-id>`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

			var grantID string
			if len(args) > 1 {
				grantID = args[1]
			} else {
				grantID, err = getGrantID(nil)
				if err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			req := &domain.CreateContactGroupRequest{
				Name: name,
			}

			group, err := client.CreateContactGroup(ctx, grantID, req)
			if err != nil {
				return common.WrapCreateError("contact group", err)
			}

			fmt.Printf("%s Created contact group '%s' (ID: %s)\n", common.Green.Sprint("✓"), group.Name, group.ID)

			return nil
		},
	}
}

func newGroupsUpdateCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "update <group-id> [grant-id]",
		Short: "Update a contact group",
		Long: `Update an existing contact group.

Examples:
  nylas contacts groups update <group-id> --name "New Name"`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupID := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

			var grantID string
			if len(args) > 1 {
				grantID = args[1]
			} else {
				grantID, err = getGrantID(nil)
				if err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			req := &domain.UpdateContactGroupRequest{}

			if cmd.Flags().Changed("name") {
				req.Name = &name
			}

			group, err := client.UpdateContactGroup(ctx, grantID, groupID, req)
			if err != nil {
				return common.WrapUpdateError("contact group", err)
			}

			fmt.Printf("%s Updated contact group '%s'\n", common.Green.Sprint("✓"), group.Name)

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "New group name")

	return cmd
}

func newGroupsDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <group-id> [grant-id]",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete a contact group",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupID := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

			var grantID string
			if len(args) > 1 {
				grantID = args[1]
			} else {
				grantID, err = getGrantID(nil)
				if err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			if !force {
				fmt.Printf("Are you sure you want to delete contact group %s? [y/N] ", groupID)
				var confirm string
				_, _ = fmt.Scanln(&confirm) // Ignore error - empty string treated as "no"
				if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			err = client.DeleteContactGroup(ctx, grantID, groupID)
			if err != nil {
				return common.WrapDeleteError("contact group", err)
			}

			fmt.Printf("%s Contact group deleted\n", common.Green.Sprint("✓"))

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation")

	return cmd
}
