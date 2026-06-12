package workspace

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List workspaces",
		Long: `List all workspaces.

Examples:
  nylas workspace list
  nylas workspace list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				workspaces, err := client.ListWorkspaces(ctx)
				if err != nil {
					return struct{}{}, common.WrapListError("workspaces", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, common.PrintJSON(workspaces)
				}

				if len(workspaces) == 0 {
					common.PrintEmptyStateWithHint("workspaces", "Create one with: nylas workspace create --name \"Workspace Name\"")
					return struct{}{}, nil
				}

				_, _ = common.BoldWhite.Printf("Workspaces (%d)\n\n", len(workspaces))
				for i, ws := range workspaces {
					printWorkspaceSummary(ws, i)
				}
				fmt.Println()
				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <workspace-id>",
		Short: "Show workspace details",
		Long: `Show details for a single workspace.

Examples:
  nylas workspace get <workspace-id>
  nylas workspace get <workspace-id> --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				workspace, err := client.GetWorkspace(ctx, args[0])
				if err != nil {
					return struct{}{}, common.WrapGetError("workspace", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, common.PrintJSON(workspace)
				}

				printWorkspaceDetails(*workspace)
				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}

func newCreateCmd() *cobra.Command {
	var (
		name      string
		wsDomain  string
		autoGroup bool
		policyID  string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a workspace",
		Long: `Create a new workspace.

Examples:
  nylas workspace create --name "My Workspace" --domain example.com
  nylas workspace create --name "My Workspace" --domain example.com --policy-id <policy-id>
  nylas workspace create --name "My Workspace" --domain example.com --auto-group`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				req := &domain.CreateWorkspaceRequest{
					Name:     name,
					Domain:   wsDomain,
					PolicyID: policyID,
				}
				if cmd.Flags().Changed("auto-group") {
					req.AutoGroup = &autoGroup
				}

				workspace, err := client.CreateWorkspace(ctx, req)
				if err != nil {
					return struct{}{}, common.WrapCreateError("workspace", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, common.PrintJSON(workspace)
				}

				common.PrintSuccess("Created workspace: %s (%s)", workspace.Name, workspace.ID)
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Workspace name (required)")
	cmd.Flags().StringVar(&wsDomain, "domain", "", "Workspace domain (required)")
	cmd.Flags().BoolVar(&autoGroup, "auto-group", false, "Enable auto-grouping")
	cmd.Flags().StringVar(&policyID, "policy-id", "", "Policy ID to attach")

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("domain")

	return cmd
}

func newUpdateCmd() *cobra.Command {
	var (
		policyID string
		rulesIDs []string
	)

	cmd := &cobra.Command{
		Use:   "update <workspace-id>",
		Short: "Update a workspace",
		Long: `Update a workspace's policy or rules.

Examples:
  nylas workspace update <workspace-id> --policy-id <policy-id>
  nylas workspace update <workspace-id> --rules-ids rule1,rule2`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("policy-id") && !cmd.Flags().Changed("rules-ids") {
				return common.NewUserError(
					"workspace update requires at least one field",
					"Use --policy-id or --rules-ids to specify what to update",
				)
			}

			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				req := &domain.UpdateWorkspaceRequest{}

				if cmd.Flags().Changed("policy-id") {
					req.PolicyID = &policyID
				}
				if cmd.Flags().Changed("rules-ids") {
					req.RulesIDs = &rulesIDs
				}

				workspace, err := client.UpdateWorkspace(ctx, args[0], req)
				if err != nil {
					return struct{}{}, common.WrapUpdateError("workspace", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, common.PrintJSON(workspace)
				}

				common.PrintSuccess("Updated workspace: %s (%s)", workspace.Name, workspace.ID)
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&policyID, "policy-id", "", "Policy ID to attach")
	cmd.Flags().StringSliceVar(&rulesIDs, "rules-ids", nil, "Rule IDs to attach (comma-separated)")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <workspace-id>",
		Short: "Delete a workspace",
		Long: `Delete a workspace permanently.

Examples:
  nylas workspace delete <workspace-id>
  nylas workspace delete <workspace-id> --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return common.NewUserError("deletion requires confirmation", "Re-run with --yes to delete the workspace")
			}

			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				if err := client.DeleteWorkspace(ctx, args[0]); err != nil {
					return struct{}{}, common.WrapDeleteError("workspace", err)
				}

				common.PrintSuccess("Deleted workspace: %s", args[0])
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func printWorkspaceSummary(ws domain.Workspace, index int) {
	name := ws.Name
	if name == "" {
		name = ws.ID
	}
	marker := ""
	if ws.Default {
		marker = "  " + common.Dim.Sprint("(default)")
	}
	fmt.Printf("%d. %s  %s%s\n", index+1, common.Cyan.Sprint(name), common.Dim.Sprint(ws.ID), marker)
	if ws.PolicyID != "" {
		_, _ = common.Dim.Printf("   Policy ID: %s\n", ws.PolicyID)
	}
	if len(ws.RulesIDs) > 0 {
		_, _ = common.Dim.Printf("   Rules: %s\n", strings.Join(ws.RulesIDs, ", "))
	}
}

func printWorkspaceDetails(ws domain.Workspace) {
	fmt.Println(strings.Repeat("─", 60))
	_, _ = common.BoldWhite.Printf("Workspace: %s\n", ws.Name)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("ID:            %s\n", ws.ID)
	fmt.Printf("Application:   %s\n", ws.ApplicationID)
	if ws.Name != "" {
		fmt.Printf("Name:          %s\n", ws.Name)
	}
	if ws.Domain != nil && *ws.Domain != "" {
		fmt.Printf("Domain:        %s\n", *ws.Domain)
	}
	fmt.Printf("Auto Group:    %t\n", ws.AutoGroup)
	fmt.Printf("Default:       %t\n", ws.Default)
	if ws.PolicyID != "" {
		fmt.Printf("Policy ID:     %s\n", ws.PolicyID)
	}
	if len(ws.RulesIDs) > 0 {
		fmt.Printf("Rules:         %s\n", strings.Join(ws.RulesIDs, ", "))
	}
	if !ws.CreatedAt.IsZero() {
		fmt.Printf("Created:       %s (%s)\n", ws.CreatedAt.Format(common.DisplayDateTime), common.FormatTimeAgo(ws.CreatedAt.Time))
	}
	if !ws.UpdatedAt.IsZero() {
		fmt.Printf("Updated:       %s (%s)\n", ws.UpdatedAt.Format(common.DisplayDateTime), common.FormatTimeAgo(ws.UpdatedAt.Time))
	}
	fmt.Println()
}
