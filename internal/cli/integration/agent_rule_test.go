//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestCLI_AgentRuleLifecycle_CreateReadListUpdateDelete(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	client := getTestClient()
	email := newAgentTestEmail(t, "rule-lifecycle")
	policyName := newPolicyTestName("rule-policy")
	ruleName := fmt.Sprintf("it-rule-%d", time.Now().UnixNano())
	updatedRuleName := fmt.Sprintf("it-rule-updated-%d", time.Now().UnixNano())

	var createdPolicy *domain.Policy
	var createdAccount *domain.AgentAccount
	var createdRule *domain.Rule
	var placeholderRule *domain.Rule

	t.Cleanup(func() {
		if createdPolicy != nil && createdRule != nil && createdRule.ID != "" {
			removeRuleFromWorkspaceForTest(t, client, createdAccount.WorkspaceID, createdRule.ID)
		}
		if createdRule != nil && createdRule.ID != "" {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeleteRule(ctx, createdRule.ID)
		}
		if createdPolicy != nil && placeholderRule != nil && placeholderRule.ID != "" {
			removeRuleFromWorkspaceForTest(t, client, createdAccount.WorkspaceID, placeholderRule.ID)
		}
		if placeholderRule != nil && placeholderRule.ID != "" {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeleteRule(ctx, placeholderRule.ID)
		}
		if createdAccount != nil && createdAccount.ID != "" {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeleteAgentAccount(ctx, createdAccount.ID)
		}
		if createdPolicy != nil && createdPolicy.ID != "" {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeletePolicy(ctx, createdPolicy.ID)
		}
	})

	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	policy, err := client.CreatePolicy(ctx, map[string]any{"name": policyName})
	cancel()
	if err != nil {
		t.Fatalf("failed to create policy for rule lifecycle: %v", err)
	}
	createdPolicy = policy

	createdAccount = createAgentWithPolicyForTest(t, email, createdPolicy.ID)
	// Subsequent CLI calls use NYLAS_GRANT_ID directly (no email→ID
	// resolution), so the consistent GET-by-id check is sufficient.
	if exists, _ := waitForAgentByID(t, client, createdAccount.ID, true); !exists {
		t.Fatalf("created agent account %q not retrievable by id", email)
	}
	env["NYLAS_GRANT_ID"] = createdAccount.ID

	placeholderRule = createRuleForTest(t, client, "it-rule-placeholder")
	attachRuleToWorkspaceForTest(t, client, createdAccount.WorkspaceID, placeholderRule.ID)
	assertWorkspaceContainsRuleForTest(t, client, createdAccount.WorkspaceID, placeholderRule.ID)

	createStdout, createStderr, err := runCLIWithOverridesAndRateLimit(
		t,
		2*time.Minute,
		env,
		"agent",
		"rule",
		"create",
		"--name", ruleName,
		"--description", "Marks inbound mail from example.com as spam",
		"--priority", "10",
		"--disabled",
		"--match-operator", "any",
		"--condition", "from.domain,is,example.com",
		"--condition", "from.tld,is,com",
		"--action", "mark_as_spam",
		"--json",
	)
	if err != nil {
		t.Fatalf("rule create failed: %v\nstdout: %s\nstderr: %s", err, createStdout, createStderr)
	}

	var rule domain.Rule
	if err := json.Unmarshal([]byte(createStdout), &rule); err != nil {
		t.Fatalf("failed to parse rule create JSON: %v\noutput: %s", err, createStdout)
	}
	if rule.ID == "" {
		t.Fatalf("expected created rule ID, got output: %s", createStdout)
	}
	if rule.Name != ruleName {
		t.Fatalf("created rule name = %q, want %q", rule.Name, ruleName)
	}
	if rule.Description != "Marks inbound mail from example.com as spam" {
		t.Fatalf("created rule description = %q, want %q", rule.Description, "Marks inbound mail from example.com as spam")
	}
	if rule.Priority == nil || *rule.Priority != 10 {
		t.Fatalf("created rule priority = %v, want %d", rule.Priority, 10)
	}
	if rule.Enabled == nil || *rule.Enabled {
		t.Fatalf("created rule enabled = %v, want false", rule.Enabled)
	}
	if rule.Match == nil || rule.Match.Operator != "any" {
		t.Fatalf("created rule operator = %q, want %q", rule.Match.Operator, "any")
	}
	if rule.Trigger != "inbound" {
		t.Fatalf("created rule trigger = %q, want %q", rule.Trigger, "inbound")
	}
	createdRule = &rule

	assertWorkspaceContainsRuleForTest(t, client, createdAccount.WorkspaceID, createdRule.ID)

	readStdout, readStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "rule", "read", createdRule.ID, "--json")
	if err != nil {
		t.Fatalf("rule read failed: %v\nstdout: %s\nstderr: %s", err, readStdout, readStderr)
	}

	var readRule domain.Rule
	if err := json.Unmarshal([]byte(readStdout), &readRule); err != nil {
		t.Fatalf("failed to parse rule read JSON: %v\noutput: %s", err, readStdout)
	}
	if readRule.ID != createdRule.ID {
		t.Fatalf("rule read returned ID %q, want %q", readRule.ID, createdRule.ID)
	}

	readTextStdout, readTextStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "rule", "read", createdRule.ID)
	if err != nil {
		t.Fatalf("rule read text failed: %v\nstdout: %s\nstderr: %s", err, readTextStdout, readTextStderr)
	}
	if !strings.Contains(readTextStdout, "Match:") ||
		!strings.Contains(readTextStdout, "Actions:") ||
		!strings.Contains(readTextStdout, "Workspaces:") ||
		!strings.Contains(readTextStdout, "from.domain is example.com") ||
		!strings.Contains(readTextStdout, "from.tld is com") ||
		!strings.Contains(readTextStdout, "mark_as_spam") {
		t.Fatalf("rule read text output missing expected sections\noutput: %s", readTextStdout)
	}

	listStdout, listStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "rule", "list", "--json")
	if err != nil {
		t.Fatalf("rule list failed: %v\nstdout: %s\nstderr: %s", err, listStdout, listStderr)
	}

	var listedRules []domain.Rule
	if err := json.Unmarshal([]byte(listStdout), &listedRules); err != nil {
		t.Fatalf("failed to parse rule list JSON: %v\noutput: %s", err, listStdout)
	}
	foundCreatedRule := false
	for _, listedRule := range listedRules {
		if listedRule.ID == createdRule.ID {
			foundCreatedRule = true
			break
		}
	}
	if !foundCreatedRule {
		t.Fatalf("rule list did not return the created rule\noutput: %s", listStdout)
	}

	updateStdout, updateStderr, err := runCLIWithOverridesAndRateLimit(
		t,
		2*time.Minute,
		env,
		"agent",
		"rule",
		"update",
		createdRule.ID,
		"--name", updatedRuleName,
		"--description", "Marks inbound mail from example.org as spam",
		"--priority", "20",
		"--enabled",
		"--condition", "from.domain,is,example.org",
		"--action", "mark_as_spam",
		"--json",
	)
	if err != nil {
		t.Fatalf("rule update failed: %v\nstdout: %s\nstderr: %s", err, updateStdout, updateStderr)
	}

	var updatedRule domain.Rule
	if err := json.Unmarshal([]byte(updateStdout), &updatedRule); err != nil {
		t.Fatalf("failed to parse rule update JSON: %v\noutput: %s", err, updateStdout)
	}
	if updatedRule.Name != updatedRuleName {
		t.Fatalf("updated rule name = %q, want %q", updatedRule.Name, updatedRuleName)
	}
	if updatedRule.Description != "Marks inbound mail from example.org as spam" {
		t.Fatalf("updated rule description = %q, want %q", updatedRule.Description, "Marks inbound mail from example.org as spam")
	}
	if updatedRule.Priority == nil || *updatedRule.Priority != 20 {
		t.Fatalf("updated rule priority = %v, want %d", updatedRule.Priority, 20)
	}
	if updatedRule.Enabled == nil || !*updatedRule.Enabled {
		t.Fatalf("updated rule enabled = %v, want true", updatedRule.Enabled)
	}
	if updatedRule.Match == nil || updatedRule.Match.Operator != "any" {
		t.Fatalf("updated rule operator = %q, want %q", updatedRule.Match.Operator, "any")
	}
	if updatedRule.Trigger != "inbound" {
		t.Fatalf("updated rule trigger = %q, want %q", updatedRule.Trigger, "inbound")
	}

	deleteStdout, deleteStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "rule", "delete", createdRule.ID, "--yes")
	if err != nil {
		t.Fatalf("rule delete failed: %v\nstdout: %s\nstderr: %s", err, deleteStdout, deleteStderr)
	}
	if !strings.Contains(strings.ToLower(deleteStdout), "deleted") {
		t.Fatalf("expected delete confirmation in stdout, got: %s", deleteStdout)
	}

	assertWorkspaceMissingRuleForTest(t, client, createdAccount.WorkspaceID, createdRule.ID)
	createdRule = nil
}

func assertWorkspaceContainsRuleForTest(t *testing.T, client interface {
	GetWorkspace(context.Context, string) (*domain.Workspace, error)
}, workspaceID, ruleID string) {
	t.Helper()

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		acquireRateLimit(t)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		workspace, err := client.GetWorkspace(ctx, workspaceID)
		cancel()
		if err == nil && workspace != nil && containsString(workspace.RulesIDs, ruleID) {
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("workspace %q does not include rule %q", workspaceID, ruleID)
}

func assertWorkspaceMissingRuleForTest(t *testing.T, client interface {
	GetWorkspace(context.Context, string) (*domain.Workspace, error)
}, workspaceID, ruleID string) {
	t.Helper()

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		acquireRateLimit(t)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		workspace, err := client.GetWorkspace(ctx, workspaceID)
		cancel()
		if err == nil && workspace != nil && !containsString(workspace.RulesIDs, ruleID) {
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("workspace %q still includes deleted rule %q", workspaceID, ruleID)
}

func removeRuleFromWorkspaceForTest(t *testing.T, client interface {
	GetWorkspace(context.Context, string) (*domain.Workspace, error)
	UpdateWorkspace(context.Context, string, *domain.UpdateWorkspaceRequest) (*domain.Workspace, error)
}, workspaceID, ruleID string) {
	t.Helper()

	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	workspace, err := client.GetWorkspace(ctx, workspaceID)
	if err != nil {
		t.Logf("cleanup: get workspace %q: %v", workspaceID, err)
		return
	}
	if workspace == nil || !containsString(workspace.RulesIDs, ruleID) {
		return
	}

	updatedRules := make([]string, 0, len(workspace.RulesIDs))
	for _, existingRuleID := range workspace.RulesIDs {
		if existingRuleID == ruleID {
			continue
		}
		updatedRules = append(updatedRules, existingRuleID)
	}

	if _, err := client.UpdateWorkspace(ctx, workspaceID, &domain.UpdateWorkspaceRequest{RulesIDs: &updatedRules}); err != nil {
		t.Logf("cleanup: remove rule %q from workspace %q: %v", ruleID, workspaceID, err)
	}
}

func attachRuleToWorkspaceForTest(t *testing.T, client interface {
	GetWorkspace(context.Context, string) (*domain.Workspace, error)
	UpdateWorkspace(context.Context, string, *domain.UpdateWorkspaceRequest) (*domain.Workspace, error)
}, workspaceID, ruleID string) {
	t.Helper()

	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	workspace, err := client.GetWorkspace(ctx, workspaceID)
	if err != nil {
		t.Fatalf("failed to get workspace %q: %v", workspaceID, err)
	}
	if workspace == nil {
		t.Fatalf("workspace %q not found", workspaceID)
	}
	if containsString(workspace.RulesIDs, ruleID) {
		return
	}

	updatedRules := append(append([]string(nil), workspace.RulesIDs...), ruleID)
	if _, err := client.UpdateWorkspace(ctx, workspaceID, &domain.UpdateWorkspaceRequest{RulesIDs: &updatedRules}); err != nil {
		t.Fatalf("failed to attach rule %q to workspace %q: %v", ruleID, workspaceID, err)
	}
}

func createRuleForTest(t *testing.T, client interface {
	CreateRule(context.Context, map[string]any) (*domain.Rule, error)
}, name string) *domain.Rule {
	t.Helper()

	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rule, err := client.CreateRule(ctx, map[string]any{
		"name":    name,
		"enabled": true,
		"trigger": "inbound",
		"match": map[string]any{
			"operator": "all",
			"conditions": []map[string]any{{
				"field":    "from.domain",
				"operator": "is",
				"value":    "placeholder.example",
			}},
		},
		"actions": []map[string]any{{
			"type": "mark_as_spam",
		}},
	})
	if err != nil {
		t.Fatalf("failed to create placeholder rule %q: %v", name, err)
	}

	return rule
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
