package agent

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newAgentListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"lists"},
		Short:   "Manage agent lists",
		Long: `Manage lists used by agent rule in_list conditions.

Lists are backed by the /v3/lists API. Each list holds normalized values of a
single immutable type (domain, tld, or address); rule conditions reference
lists by ID with the in_list operator, and a list's type determines which
rule fields it can match.

API reference: https://developer.nylas.com/docs/v3/agent-accounts/policies-rules-lists/

Examples:
  nylas agent list list
  nylas agent list create --name "Blocked domains" --type domain --item spam.com
  nylas agent list get <list-id>
  nylas agent list add <list-id> junk.net
  nylas agent list remove <list-id> junk.net
  nylas agent list delete <list-id> --yes`,
	}

	cmd.AddCommand(newAgentListListCmd())
	cmd.AddCommand(newAgentListGetCmd())
	cmd.AddCommand(newAgentListCreateCmd())
	cmd.AddCommand(newAgentListUpdateCmd())
	cmd.AddCommand(newAgentListDeleteCmd())
	cmd.AddCommand(newAgentListItemsCmd())
	cmd.AddCommand(newAgentListAddCmd())
	cmd.AddCommand(newAgentListRemoveCmd())

	return cmd
}

func newAgentListListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List lists",
		Long: `List all lists from /v3/lists.

Examples:
  nylas agent list list
  nylas agent list list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentListList(common.IsJSON(cmd))
		},
	}
}

func runAgentListList(jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		lists, err := client.ListLists(ctx)
		if err != nil {
			return struct{}{}, common.WrapListError("lists", err)
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(lists)
		}

		if len(lists) == 0 {
			common.PrintEmptyStateWithHint("lists", "Create one with: nylas agent list create --name \"List Name\" --type domain")
			return struct{}{}, nil
		}

		_, _ = common.BoldWhite.Printf("Lists (%d)\n\n", len(lists))
		for _, list := range lists {
			printAgentListSummary(list)
		}
		return struct{}{}, nil
	})

	return err
}

func newAgentListGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <list-id>",
		Short: "Show a list and its items",
		Long: `Show details and items for a single list.

Examples:
  nylas agent list get <list-id>
  nylas agent list get <list-id> --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentListGet(args[0], common.IsJSON(cmd))
		},
	}
}

func runAgentListGet(listID string, jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		list, err := client.GetList(ctx, listID)
		if err != nil {
			return struct{}{}, common.WrapGetError("list", err)
		}

		items, err := client.GetListItems(ctx, listID)
		if err != nil {
			return struct{}{}, common.WrapGetError("list items", err)
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(map[string]any{"list": list, "items": items})
		}

		printAgentListDetails(*list, items)
		return struct{}{}, nil
	})

	return err
}

func newAgentListItemsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "items <list-id>",
		Short: "Show list items",
		Long: `Show the items of a list.

Examples:
  nylas agent list items <list-id>
  nylas agent list items <list-id> --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentListItems(args[0], common.IsJSON(cmd))
		},
	}
}

func runAgentListItems(listID string, jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		items, err := client.GetListItems(ctx, listID)
		if err != nil {
			return struct{}{}, common.WrapGetError("list items", err)
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(items)
		}

		if len(items) == 0 {
			common.PrintEmptyStateWithHint("items", fmt.Sprintf("Add some with: nylas agent list add %s <value>", listID))
			return struct{}{}, nil
		}

		_, _ = common.BoldWhite.Printf("Items (%d)\n\n", len(items))
		for _, item := range items {
			fmt.Printf("  %s\n", item)
		}
		fmt.Println()
		return struct{}{}, nil
	})

	return err
}

func buildListCreatePayload(name, listType, description string) (map[string]any, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, common.NewUserError("list name is required", "Pass --name \"List Name\"")
	}
	listType = strings.ToLower(strings.TrimSpace(listType))
	if !slices.Contains(domain.AgentListTypes, listType) {
		return nil, common.NewUserError(
			fmt.Sprintf("invalid list type: %s", listType),
			fmt.Sprintf("Use one of: %s. The type is immutable after creation.", strings.Join(domain.AgentListTypes, ", ")),
		)
	}

	payload := map[string]any{"name": name, "type": listType}
	if description = strings.TrimSpace(description); description != "" {
		payload["description"] = description
	}
	return payload, nil
}

func printAgentListSummary(list domain.AgentList) {
	_, _ = common.BoldWhite.Printf("%s\n", list.Name)
	fmt.Printf("  ID:    %s\n", list.ID)
	fmt.Printf("  Type:  %s\n", list.Type)
	fmt.Printf("  Items: %d\n", list.ItemsCount)
	if list.Description != "" {
		fmt.Printf("  Description: %s\n", list.Description)
	}
	fmt.Println()
}

func printAgentListDetails(list domain.AgentList, items []string) {
	printAgentListSummary(list)
	if len(items) == 0 {
		fmt.Println("  (no items)")
		return
	}
	_, _ = common.BoldWhite.Printf("Items (%d)\n", len(items))
	for _, item := range items {
		fmt.Printf("  %s\n", item)
	}
	fmt.Println()
}
