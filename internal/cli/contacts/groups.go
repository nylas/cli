package contacts

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
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
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				groups, err := client.GetContactGroups(ctx, grantID)
				if err != nil {
					return struct{}{}, common.WrapListError("contact groups", err)
				}

				// JSON output (including empty array)
				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(groups)
				}

				if len(groups) == 0 {
					common.PrintEmptyState("contact groups")
					return struct{}{}, nil
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

				return struct{}{}, nil
			})
			return err
		},
	}
}

func newGroupsShowCmd() *cobra.Command {
	return common.NewShowCommand(common.ShowCommandConfig{
		Use:          "show <group-id> [grant-id]",
		Short:        "Show contact group details",
		ResourceName: "contact group",
		GetFunc: func(ctx context.Context, grantID, resourceID string) (interface{}, error) {
			client, err := common.GetNylasClient()
			if err != nil {
				return nil, err
			}
			return client.GetContactGroup(ctx, grantID, resourceID)
		},
		DisplayFunc: func(resource interface{}) error {
			group := resource.(*domain.ContactGroup)

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
		GetClient: common.GetNylasClient,
	})
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
			grantArgs := args[1:]

			_, err := common.WithClient(grantArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				req := &domain.CreateContactGroupRequest{
					Name: name,
				}

				group, err := client.CreateContactGroup(ctx, grantID, req)
				if err != nil {
					return struct{}{}, common.WrapCreateError("contact group", err)
				}

				fmt.Printf("%s Created contact group '%s' (ID: %s)\n", common.Green.Sprint("✓"), group.Name, group.ID)

				return struct{}{}, nil
			})
			return err
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
			setup, err := common.SetupUpdateCommand(args)
			if err != nil {
				return err
			}
			defer setup.Cancel()

			req := &domain.UpdateContactGroupRequest{}

			if cmd.Flags().Changed("name") {
				req.Name = &name
			}

			group, err := setup.Client.UpdateContactGroup(setup.Ctx, setup.GrantID, setup.ResourceID, req)
			if err != nil {
				return common.WrapUpdateError("contact group", err)
			}

			common.PrintUpdateSuccess("contact group", group.Name)

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "New group name")

	return cmd
}

func newGroupsDeleteCmd() *cobra.Command {
	return common.NewDeleteCommand(common.DeleteCommandConfig{
		Use:          "delete <group-id> [grant-id]",
		Aliases:      []string{"rm", "remove"},
		Short:        "Delete a contact group",
		ResourceName: "contact group",
		DeleteFunc: func(ctx context.Context, grantID, resourceID string) error {
			client, err := common.GetNylasClient()
			if err != nil {
				return err
			}
			return client.DeleteContactGroup(ctx, grantID, resourceID)
		},
		GetClient: common.GetNylasClient,
	})
}
