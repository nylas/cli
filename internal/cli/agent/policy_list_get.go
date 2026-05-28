package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newPolicyListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List policies",
		Long: `List all policies from /v3/policies.

Shows which agent workspace has each policy attached.

Examples:
  nylas agent policy list
  nylas agent policy list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolicyList(common.IsJSON(cmd))
		},
	}

	return cmd
}

func runPolicyList(jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		policies, err := client.ListPolicies(ctx)
		if err != nil {
			return struct{}{}, common.WrapListError("policies", err)
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(policies)
		}

		if len(policies) == 0 {
			common.PrintEmptyStateWithHint("policies", "Create one with: nylas agent policy create --name \"Policy Name\"")
			return struct{}{}, nil
		}

		workspaceRefs := buildWorkspacePolicyRefs(ctx, client)

		_, _ = common.BoldWhite.Printf("Policies (%d)\n\n", len(policies))
		for i, policy := range policies {
			printPolicySummary(policy, i, workspaceRefs[policy.ID])
		}
		fmt.Println()
		return struct{}{}, nil
	})

	return err
}

func buildWorkspacePolicyRefs(ctx context.Context, client ports.NylasClient) map[string][]policyAgentAccountRef {
	accounts, err := client.ListAgentAccounts(ctx)
	if err != nil {
		return nil
	}

	refs := make(map[string][]policyAgentAccountRef)
	for _, account := range accounts {
		workspaceID := strings.TrimSpace(account.WorkspaceID)
		if workspaceID == "" {
			continue
		}
		workspace, err := client.GetWorkspace(ctx, workspaceID)
		if err != nil || workspace == nil {
			continue
		}
		policyID := strings.TrimSpace(workspace.PolicyID)
		if policyID == "" {
			continue
		}
		refs[policyID] = append(refs[policyID], policyAgentAccountRef{
			GrantID:     account.ID,
			Email:       account.Email,
			WorkspaceID: workspaceID,
		})
	}
	return refs
}

func newPolicyGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <policy-id>",
		Short: "Show a policy",
		Long: `Show details for a single policy.

Examples:
  nylas agent policy get <policy-id>
  nylas agent policy get <policy-id> --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolicyGet(args[0], common.IsJSON(cmd))
		},
	}

	return cmd
}

func newPolicyReadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read <policy-id>",
		Short: "Read a policy",
		Long: `Read details for a single policy.

Examples:
  nylas agent policy read <policy-id>
  nylas agent policy read <policy-id> --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolicyGet(args[0], common.IsJSON(cmd))
		},
	}

	return cmd
}

func runPolicyGet(policyID string, jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		scope, err := loadAgentPolicyScope(ctx, client)
		if err != nil {
			return struct{}{}, err
		}

		resolved, err := resolvePolicyForAgentOps(scope, policyID)
		if err != nil {
			return struct{}{}, err
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(resolved.Policy)
		}

		printPolicyDetails(*resolved.Policy)
		return struct{}{}, nil
	})

	return err
}
