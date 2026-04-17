package agent

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newPolicyCreateCmd() *cobra.Command {
	var (
		name       string
		data       string
		dataFile   string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a policy",
		Long: `Create a new policy.

Use --name for a simple policy, or pass a full request body with --data or
--data-file to set limits, rules, options, and spam detection.

Examples:
  nylas agent policy create --name "Strict Policy"
  nylas agent policy create --data '{"name":"Strict Policy","rules":["rule-123"]}'
  nylas agent policy create --data-file policy.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := loadPolicyPayload(data, dataFile, name, true)
			if err != nil {
				return err
			}
			return runPolicyCreate(payload, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Policy name")
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON request body")
	cmd.Flags().StringVar(&dataFile, "data-file", "", "Path to a JSON request body file")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func runPolicyCreate(payload map[string]any, jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		policy, err := client.CreatePolicy(ctx, payload)
		if err != nil {
			return struct{}{}, common.WrapCreateError("policy", err)
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(policy)
		}

		printSuccess("Policy created successfully!")
		fmt.Println()
		printPolicyDetails(*policy)
		return struct{}{}, nil
	})

	return err
}

func newPolicyUpdateCmd() *cobra.Command {
	var (
		name       string
		data       string
		dataFile   string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "update <policy-id>",
		Short: "Update a policy",
		Long: `Update an existing policy.

Use --name for a simple rename, or pass a partial JSON request body with
--data or --data-file to update nested fields.

Examples:
  nylas agent policy update <policy-id> --name "Updated Policy"
  nylas agent policy update <policy-id> --data '{"spam_detection":{"spam_sensitivity":2}}'
  nylas agent policy update <policy-id> --data-file update.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := loadPolicyPayload(data, dataFile, name, false)
			if err != nil {
				return err
			}
			if len(payload) == 0 {
				return common.NewUserError("policy update requires at least one field", "Use --name, --data, or --data-file")
			}
			return runPolicyUpdate(args[0], payload, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Updated policy name")
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON request body")
	cmd.Flags().StringVar(&dataFile, "data-file", "", "Path to a JSON request body file")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func runPolicyUpdate(policyID string, payload map[string]any, jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		scope, err := loadAgentPolicyScope(ctx, client)
		if err != nil {
			return struct{}{}, err
		}

		if _, err := resolvePolicyForAgentOps(scope, policyID); err != nil {
			return struct{}{}, err
		}

		policy, err := client.UpdatePolicy(ctx, policyID, payload)
		if err != nil {
			return struct{}{}, common.WrapUpdateError("policy", err)
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(policy)
		}

		common.PrintUpdateSuccess("policy", policy.Name)
		fmt.Println()
		printPolicyDetails(*policy)
		return struct{}{}, nil
	})

	return err
}

func newPolicyDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <policy-id>",
		Short: "Delete a policy",
		Long: `Delete a policy permanently.

Examples:
  nylas agent policy delete <policy-id> --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return common.NewUserError("deletion requires confirmation", "Re-run with --yes to delete the policy")
			}
			return runPolicyDelete(args[0])
		},
	}

	common.AddYesFlag(cmd, &yes)

	return cmd
}

func runPolicyDelete(policyID string) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		scope, err := loadAgentPolicyScope(ctx, client)
		if err != nil {
			return struct{}{}, err
		}

		resolved, err := resolvePolicyForAgentOps(scope, policyID)
		if err != nil {
			return struct{}{}, err
		}

		attachedAccounts := resolved.AgentAccounts
		if len(attachedAccounts) > 0 {
			accountSummary := formatPolicyAgentAccounts(attachedAccounts)
			return struct{}{}, common.NewUserError(
				fmt.Sprintf("policy is attached to agent accounts: %s", accountSummary),
				fmt.Sprintf("Detach or move the listed accounts to another policy before deleting %q", policyID),
			)
		}

		if err := client.DeletePolicy(ctx, policyID); err != nil {
			return struct{}{}, common.WrapDeleteError("policy", err)
		}
		common.PrintSuccess("Policy deleted")
		return struct{}{}, nil
	})

	return err
}
