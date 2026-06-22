package agent

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newRuleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rule",
		Short: "Manage agent rules",
		Long: `Manage rules attached to agent account workspaces.

Rules are backed by the /v3/rules API. They attach to workspaces via
rule_ids[].

API reference: https://developer.nylas.com/docs/v3/agent-accounts/policies-rules-lists/

Examples:
  nylas agent rule list
  nylas agent rule read <rule-id>
  nylas agent rule create --name "Archive outbound mail" --trigger outbound --condition recipient.domain,is,example.com --action archive
  nylas agent rule update <rule-id> --name "Updated Rule"
  nylas agent rule delete <rule-id> --yes`,
	}

	cmd.AddCommand(newRuleListCmd())
	cmd.AddCommand(newRuleGetCmd())
	cmd.AddCommand(newRuleReadCmd())
	cmd.AddCommand(newRuleCreateCmd())
	cmd.AddCommand(newRuleUpdateCmd())
	cmd.AddCommand(newRuleDeleteCmd())

	return cmd
}

func resolveDefaultAgentAccount(ctx context.Context, client ports.NylasClient) (*domain.AgentAccount, error) {
	grantID, err := common.GetGrantID(nil)
	if err != nil {
		return nil, common.WrapGetError("default grant", err)
	}

	account, err := client.GetAgentAccount(ctx, grantID)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidGrant) {
			return nil, common.NewUserError(
				"default grant is not a nylas agent account",
				"Use 'nylas auth switch <grant-id>' to select a provider=nylas account",
			)
		}
		return nil, common.WrapGetError("default agent account", err)
	}

	return account, nil
}

func findPolicyByID(policies []domain.Policy, policyID string) *domain.Policy {
	for i := range policies {
		if policies[i].ID == policyID {
			return &policies[i]
		}
	}
	return nil
}

func appendUniqueString(items []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return append([]string(nil), items...)
	}

	updated := append([]string(nil), items...)
	if !slices.Contains(updated, value) {
		updated = append(updated, value)
	}
	return updated
}

func removeString(items []string, value string) []string {
	value = strings.TrimSpace(value)
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) == value {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func attachRuleToAgentWorkspaces(ctx context.Context, client interface {
	GetWorkspace(context.Context, string) (*domain.Workspace, error)
	UpdateWorkspace(context.Context, string, *domain.UpdateWorkspaceRequest) (*domain.Workspace, error)
}, accounts []policyAgentAccountRef, ruleID string) error {
	seenWorkspaceIDs := make(map[string]struct{}, len(accounts))
	for _, account := range accounts {
		workspaceID := strings.TrimSpace(account.WorkspaceID)
		if workspaceID == "" {
			continue
		}
		if _, seen := seenWorkspaceIDs[workspaceID]; seen {
			continue
		}
		seenWorkspaceIDs[workspaceID] = struct{}{}

		workspace, err := client.GetWorkspace(ctx, workspaceID)
		if err != nil {
			return err
		}
		if workspace == nil {
			return common.NewUserError("workspace not found", "The API returned an empty workspace response")
		}
		updatedRules := appendUniqueString(workspace.RulesIDs, ruleID)
		if slices.Equal(updatedRules, workspace.RulesIDs) {
			continue
		}
		if _, err := client.UpdateWorkspace(ctx, workspaceID, &domain.UpdateWorkspaceRequest{RulesIDs: &updatedRules}); err != nil {
			return err
		}
	}
	if len(seenWorkspaceIDs) == 0 {
		return common.NewUserError(
			"agent account has no workspace",
			"The selected provider=nylas account is missing a workspace to attach the rule to; reconnect the account and try again",
		)
	}
	return nil
}

func detachRuleFromAgentWorkspaces(ctx context.Context, client interface {
	GetWorkspace(context.Context, string) (*domain.Workspace, error)
	UpdateWorkspace(context.Context, string, *domain.UpdateWorkspaceRequest) (*domain.Workspace, error)
}, accounts []policyAgentAccountRef, ruleID string) (func(context.Context) error, error) {
	workspaces, err := loadReferencedWorkspaces(ctx, client, accounts)
	if err != nil {
		return nil, err
	}

	originalRulesByWorkspaceID := make(map[string][]string)
	updatedWorkspaceIDs := make([]string, 0)

	for _, workspace := range workspaces {
		if !slices.ContainsFunc(workspace.RulesIDs, func(id string) bool {
			return strings.TrimSpace(id) == strings.TrimSpace(ruleID)
		}) {
			continue
		}

		originalRulesByWorkspaceID[workspace.ID] = append([]string(nil), workspace.RulesIDs...)
		updatedRules := removeString(workspace.RulesIDs, ruleID)
		if _, err := client.UpdateWorkspace(ctx, workspace.ID, &domain.UpdateWorkspaceRequest{RulesIDs: &updatedRules}); err != nil {
			if rollbackErr := rollbackWorkspaceRuleUpdates(ctx, client, originalRulesByWorkspaceID, updatedWorkspaceIDs); rollbackErr != nil {
				return nil, fmt.Errorf("failed to detach rule from workspace %s: %w (rollback failed: %v)", workspace.ID, err, rollbackErr)
			}
			return nil, err
		}
		updatedWorkspaceIDs = append(updatedWorkspaceIDs, workspace.ID)
	}

	return func(ctx context.Context) error {
		return rollbackWorkspaceRuleUpdates(ctx, client, originalRulesByWorkspaceID, updatedWorkspaceIDs)
	}, nil
}

func loadReferencedWorkspaces(ctx context.Context, client interface {
	GetWorkspace(context.Context, string) (*domain.Workspace, error)
}, accounts []policyAgentAccountRef) ([]domain.Workspace, error) {
	seenWorkspaceIDs := make(map[string]struct{})
	workspaces := make([]domain.Workspace, 0)
	for _, account := range accounts {
		workspaceID := strings.TrimSpace(account.WorkspaceID)
		if workspaceID == "" {
			continue
		}
		if _, seen := seenWorkspaceIDs[workspaceID]; seen {
			continue
		}
		seenWorkspaceIDs[workspaceID] = struct{}{}

		workspace, err := client.GetWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		if workspace == nil {
			return nil, common.NewUserError("workspace not found", "The API returned an empty workspace response")
		}
		workspaces = append(workspaces, *workspace)
	}
	return workspaces, nil
}

func rollbackWorkspaceRuleUpdates(ctx context.Context, client interface {
	UpdateWorkspace(context.Context, string, *domain.UpdateWorkspaceRequest) (*domain.Workspace, error)
}, originalRulesByWorkspaceID map[string][]string, updatedWorkspaceIDs []string) error {
	var failures []string
	for _, workspaceID := range updatedWorkspaceIDs {
		rules := append([]string(nil), originalRulesByWorkspaceID[workspaceID]...)
		if _, err := client.UpdateWorkspace(ctx, workspaceID, &domain.UpdateWorkspaceRequest{RulesIDs: &rules}); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", workspaceID, err))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("failed to rollback workspace updates: %s", strings.Join(failures, "; "))
	}
	return nil
}
