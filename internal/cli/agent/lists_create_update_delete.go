package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newAgentListCreateCmd() *cobra.Command {
	var (
		name        string
		listType    string
		description string
		items       []string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a list",
		Long: `Create a new list, optionally seeding it with items.

The type (domain, tld, or address) is immutable after creation and determines
which rule fields the list can match in in_list conditions.

Examples:
  nylas agent list create --name "Blocked domains" --type domain
  nylas agent list create --name "VIPs" --type address --item ceo@example.com --item cfo@example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := buildListCreatePayload(name, listType, description)
			if err != nil {
				return err
			}
			return runAgentListCreate(payload, items, common.IsJSON(cmd))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "List name")
	cmd.Flags().StringVar(&listType, "type", "", "List type: domain, tld, or address (immutable)")
	cmd.Flags().StringVar(&description, "description", "", "List description")
	cmd.Flags().StringArrayVar(&items, "item", nil, "Item to add after creation (repeatable)")

	return cmd
}

func runAgentListCreate(payload map[string]any, items []string, jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		list, err := client.CreateList(ctx, payload)
		if err != nil {
			return struct{}{}, common.WrapCreateError("list", err)
		}

		if len(items) > 0 {
			updated, err := client.AddListItems(ctx, list.ID, items)
			if err != nil {
				return struct{}{}, fmt.Errorf("list %s created but adding items failed: %w", list.ID, err)
			}
			list = updated
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(list)
		}

		common.PrintSuccess("List created successfully!")
		fmt.Println()
		printAgentListSummary(*list)
		return struct{}{}, nil
	})

	return err
}

func newAgentListUpdateCmd() *cobra.Command {
	var (
		name        string
		description string
	)

	cmd := &cobra.Command{
		Use:   "update <list-id>",
		Short: "Update a list's name or description",
		Long: `Update a list's metadata. The type cannot be changed.

Examples:
  nylas agent list update <list-id> --name "New name"
  nylas agent list update <list-id> --description "Updated description"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload := map[string]any{}
			if cmd.Flags().Changed("name") {
				if strings.TrimSpace(name) == "" {
					return common.NewUserError("list name cannot be empty", "Pass --name \"List Name\"")
				}
				payload["name"] = strings.TrimSpace(name)
			}
			if cmd.Flags().Changed("description") {
				payload["description"] = strings.TrimSpace(description)
			}
			if len(payload) == 0 {
				return common.NewUserError("nothing to update", "Pass --name and/or --description")
			}
			return runAgentListUpdate(args[0], payload, common.IsJSON(cmd))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New list name")
	cmd.Flags().StringVar(&description, "description", "", "New list description")

	return cmd
}

func runAgentListUpdate(listID string, payload map[string]any, jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		list, err := client.UpdateList(ctx, listID, payload)
		if err != nil {
			return struct{}{}, common.WrapUpdateError("list", err)
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(list)
		}

		common.PrintSuccess("List updated successfully!")
		fmt.Println()
		printAgentListSummary(*list)
		return struct{}{}, nil
	})

	return err
}

func newAgentListDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <list-id>",
		Short: "Delete a list",
		Long: `Delete a list.

Rules referencing the list via in_list conditions will no longer match it.

Examples:
  nylas agent list delete <list-id> --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return common.NewUserError("deletion requires confirmation", "Re-run with --yes to delete the list")
			}
			return runAgentListDelete(args[0])
		},
	}

	common.AddYesFlag(cmd, &yes)

	return cmd
}

func runAgentListDelete(listID string) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		if err := client.DeleteList(ctx, listID); err != nil {
			return struct{}{}, common.WrapDeleteError("list", err)
		}
		common.PrintSuccess("List deleted successfully!")
		return struct{}{}, nil
	})

	return err
}

func newAgentListAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <list-id> <item> [item...]",
		Short: "Add items to a list",
		Long: `Add items to a list (up to 1000 per request).

Values are lowercased, trimmed, and validated against the list's type by the
API; duplicates are silently ignored.

Examples:
  nylas agent list add <list-id> spam.com junk.net`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentListModifyItems(args[0], args[1:], true, common.IsJSON(cmd))
		},
	}
}

func newAgentListRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <list-id> <item> [item...]",
		Short: "Remove items from a list",
		Long: `Remove items from a list.

Examples:
  nylas agent list remove <list-id> spam.com`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentListModifyItems(args[0], args[1:], false, common.IsJSON(cmd))
		},
	}
}

func runAgentListModifyItems(listID string, items []string, add bool, jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		var (
			list *domain.AgentList
			err  error
		)
		if add {
			list, err = client.AddListItems(ctx, listID, items)
		} else {
			list, err = client.RemoveListItems(ctx, listID, items)
		}
		if err != nil {
			return struct{}{}, common.WrapUpdateError("list items", err)
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(list)
		}

		if add {
			common.PrintSuccess("Added %d item(s)", len(items))
		} else {
			common.PrintSuccess("Removed %d item(s)", len(items))
		}
		fmt.Printf("  List %s now has %d item(s)\n", list.ID, list.ItemsCount)
		return struct{}{}, nil
	})

	return err
}
