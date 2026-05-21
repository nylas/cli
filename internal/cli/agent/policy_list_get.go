package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newPolicyListCmd() *cobra.Command {
	var allPolicies bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List policies for the default agent workspace",
		Long: `List policies for the current default agent workspace.

By default, this command resolves the current default grant and shows the
policy attached to that provider=nylas account's workspace. Use --all to list
every policy referenced by a provider=nylas workspace.

Examples:
  nylas agent policy list
  nylas agent policy list --all
  nylas agent policy list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolicyList(common.IsJSON(cmd), allPolicies)
		},
	}

	cmd.Flags().BoolVar(&allPolicies, "all", false, "List all policies referenced by provider=nylas workspaces")

	return cmd
}

func runPolicyList(jsonOutput, allPolicies bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		if allPolicies {
			scope, err := loadAgentPolicyScope(ctx, client)
			if err != nil {
				return struct{}{}, err
			}
			policies := scope.AgentPolicies

			if jsonOutput {
				return struct{}{}, common.PrintJSON(policies)
			}

			if len(policies) == 0 {
				common.PrintEmptyStateWithHint("policies attached to nylas agent workspaces", "Create a provider=nylas account with --policy-id to see it here")
				return struct{}{}, nil
			}

			_, _ = common.BoldWhite.Printf("Policies (%d)\n\n", len(policies))
			for i, policy := range policies {
				printPolicySummary(policy, i, scope.PolicyRefsByID[policy.ID])
			}
			fmt.Println()
			return struct{}{}, nil
		}

		grantID, err := common.GetGrantID(nil)
		if err != nil {
			return struct{}{}, common.WrapGetError("default grant", err)
		}

		account, err := client.GetAgentAccount(ctx, grantID)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidGrant) {
				return struct{}{}, common.NewUserError(
					"default grant is not a nylas agent account",
					"Use 'nylas auth switch <grant-id>' to select a provider=nylas account, or run 'nylas agent policy list --all'",
				)
			}
			return struct{}{}, common.WrapGetError("default agent account", err)
		}

		policy, accountRef, err := resolveAgentAccountWorkspacePolicy(ctx, client, *account)
		if err != nil {
			return struct{}{}, err
		}
		if policy == nil {
			if jsonOutput {
				fmt.Println("[]")
				return struct{}{}, nil
			}
			common.PrintEmptyStateWithHint(
				"policy on the default agent workspace",
				"Use 'nylas agent policy list --all' to inspect all workspace-attached policies",
			)
			return struct{}{}, nil
		}

		policies := []domain.Policy{*policy}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(policies)
		}

		_, _ = common.BoldWhite.Printf("Policies (%d)\n\n", len(policies))
		printPolicySummary(*policy, 0, []policyAgentAccountRef{accountRef})
		fmt.Println()
		return struct{}{}, nil
	})

	return err
}

func resolveAgentAccountWorkspacePolicy(ctx context.Context, client interface {
	GetWorkspace(context.Context, string) (*domain.Workspace, error)
	GetPolicy(context.Context, string) (*domain.Policy, error)
}, account domain.AgentAccount) (*domain.Policy, policyAgentAccountRef, error) {
	accountRef := policyAgentAccountRef{
		GrantID:     account.ID,
		Email:       account.Email,
		WorkspaceID: strings.TrimSpace(account.WorkspaceID),
	}

	policyID := strings.TrimSpace(account.Settings.PolicyID)
	if accountRef.WorkspaceID != "" {
		workspace, err := client.GetWorkspace(ctx, accountRef.WorkspaceID)
		if err != nil {
			return nil, accountRef, common.WrapGetError("workspace", err)
		}
		if workspace == nil {
			return nil, accountRef, common.NewUserError("workspace not found", "The API returned an empty workspace response")
		}
		policyID = strings.TrimSpace(workspace.PolicyID)
	}
	if policyID == "" {
		return nil, accountRef, nil
	}

	policy, err := client.GetPolicy(ctx, policyID)
	if err != nil {
		return nil, accountRef, common.WrapGetError("policy", err)
	}
	return policy, accountRef, nil
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
