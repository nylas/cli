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
	var (
		jsonOutput  bool
		allPolicies bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List policies for the default agent account",
		Long: `List policies for the current default agent account.

By default, this command resolves the current default grant and shows the
single policy attached to that provider=nylas account. Use --all to list every
policy referenced by a provider=nylas account.

Examples:
  nylas agent policy list
  nylas agent policy list --all
  nylas agent policy list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolicyList(jsonOutput, allPolicies)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&allPolicies, "all", false, "List all policies referenced by provider=nylas accounts")

	return cmd
}

func runPolicyList(jsonOutput, allPolicies bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		if allPolicies {
			policies, err := client.ListPolicies(ctx)
			if err != nil {
				return struct{}{}, common.WrapListError("policies", err)
			}

			accounts, err := client.ListAgentAccounts(ctx)
			if err != nil {
				return struct{}{}, common.WrapListError("agent accounts", err)
			}
			refsByPolicyID := buildPolicyAccountRefs(accounts)
			policies = filterPoliciesWithAgentAccounts(policies, refsByPolicyID)

			if jsonOutput {
				return struct{}{}, common.PrintJSON(policies)
			}

			if len(policies) == 0 {
				common.PrintEmptyStateWithHint("policies attached to nylas agent accounts", "Create or update a provider=nylas account with a policy_id to see it here")
				return struct{}{}, nil
			}

			_, _ = common.BoldWhite.Printf("Policies (%d)\n\n", len(policies))
			for i, policy := range policies {
				printPolicySummary(policy, i, refsByPolicyID[policy.ID])
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

		policyID := strings.TrimSpace(account.Settings.PolicyID)
		if policyID == "" {
			if jsonOutput {
				fmt.Println("[]")
				return struct{}{}, nil
			}
			common.PrintEmptyStateWithHint(
				"policy on the default agent account",
				"Use 'nylas agent policy list --all' to inspect all agent-attached policies",
			)
			return struct{}{}, nil
		}

		policy, err := client.GetPolicy(ctx, policyID)
		if err != nil {
			return struct{}{}, common.WrapGetError("policy", err)
		}
		policies := []domain.Policy{*policy}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(policies)
		}

		_, _ = common.BoldWhite.Printf("Policies (%d)\n\n", len(policies))
		printPolicySummary(*policy, 0, []policyAgentAccountRef{{
			GrantID: account.ID,
			Email:   account.Email,
		}})
		fmt.Println()
		return struct{}{}, nil
	})

	return err
}

func newPolicyGetCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "get <policy-id>",
		Short: "Show a policy",
		Long: `Show details for a single policy.

Examples:
  nylas agent policy get <policy-id>
  nylas agent policy get <policy-id> --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolicyGet(args[0], jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func newPolicyReadCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "read <policy-id>",
		Short: "Read a policy",
		Long: `Read details for a single policy.

Examples:
  nylas agent policy read <policy-id>
  nylas agent policy read <policy-id> --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolicyGet(args[0], jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

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
