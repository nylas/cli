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
			removeRuleFromPolicyForTest(t, client, createdPolicy.ID, createdRule.ID)
		}
		if createdRule != nil && createdRule.ID != "" {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeleteRule(ctx, createdRule.ID)
		}
		if createdPolicy != nil && placeholderRule != nil && placeholderRule.ID != "" {
			removeRuleFromPolicyForTest(t, client, createdPolicy.ID, placeholderRule.ID)
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
	if exists, _ := waitForAgentByEmail(t, client, email, true); !exists {
		t.Fatalf("created agent account %q did not appear in list", email)
	}
	env["NYLAS_GRANT_ID"] = createdAccount.ID

	placeholderRule = createRuleForTest(t, client, "it-rule-placeholder")
	attachRuleToPolicyForTest(t, client, createdPolicy.ID, placeholderRule.ID)
	assertPolicyContainsRuleForTest(t, client, createdPolicy.ID, placeholderRule.ID)

	createStdout, createStderr, err := runCLIWithOverridesAndRateLimit(
		t,
		2*time.Minute,
		env,
		"agent",
		"rule",
		"create",
		"--name", ruleName,
		"--description", "Blocks example.com",
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
	if rule.Description != "Blocks example.com" {
		t.Fatalf("created rule description = %q, want %q", rule.Description, "Blocks example.com")
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
	createdRule = &rule

	assertPolicyContainsRuleForTest(t, client, createdPolicy.ID, createdRule.ID)

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
	if !strings.Contains(readTextStdout, "Match:") || !strings.Contains(readTextStdout, "Actions:") || !strings.Contains(readTextStdout, createdPolicy.Name) {
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

	allStdout, allStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "rule", "list", "--all")
	if err != nil {
		t.Fatalf("rule list --all failed: %v\nstdout: %s\nstderr: %s", err, allStdout, allStderr)
	}
	if !strings.Contains(allStdout, createdRule.Name) || !strings.Contains(allStdout, createdPolicy.Name) || !strings.Contains(allStdout, createdAccount.Email) {
		t.Fatalf("rule list --all output missing expected references\noutput: %s", allStdout)
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
		"--description", "Blocks example.org",
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
	if updatedRule.Description != "Blocks example.org" {
		t.Fatalf("updated rule description = %q, want %q", updatedRule.Description, "Blocks example.org")
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

	deleteStdout, deleteStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "rule", "delete", createdRule.ID, "--yes")
	if err != nil {
		t.Fatalf("rule delete failed: %v\nstdout: %s\nstderr: %s", err, deleteStdout, deleteStderr)
	}
	if !strings.Contains(strings.ToLower(deleteStdout), "deleted") {
		t.Fatalf("expected delete confirmation in stdout, got: %s", deleteStdout)
	}

	assertPolicyMissingRuleForTest(t, client, createdPolicy.ID, createdRule.ID)
	createdRule = nil
}

func TestCLI_AgentRuleDelete_RejectsLastRuleOnPolicy(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	client := getTestClient()
	email := newAgentTestEmail(t, "rule-delete-last")
	policyName := newPolicyTestName("rule-delete-last")
	ruleName := fmt.Sprintf("it-rule-last-%d", time.Now().UnixNano())

	var createdPolicy *domain.Policy
	var createdAccount *domain.AgentAccount
	var createdRule *domain.Rule

	t.Cleanup(func() {
		if createdPolicy != nil && createdRule != nil && createdRule.ID != "" {
			removeRuleFromPolicyForTest(t, client, createdPolicy.ID, createdRule.ID)
		}
		if createdRule != nil && createdRule.ID != "" {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeleteRule(ctx, createdRule.ID)
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
		t.Fatalf("failed to create policy for delete-last test: %v", err)
	}
	createdPolicy = policy

	createdAccount = createAgentWithPolicyForTest(t, email, createdPolicy.ID)
	if exists, _ := waitForAgentByEmail(t, client, email, true); !exists {
		t.Fatalf("created agent account %q did not appear in list", email)
	}
	env["NYLAS_GRANT_ID"] = createdAccount.ID

	createStdout, createStderr, err := runCLIWithOverridesAndRateLimit(
		t,
		2*time.Minute,
		env,
		"agent",
		"rule",
		"create",
		"--name", ruleName,
		"--condition", "from.domain,is,example.com",
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
	createdRule = &rule
	assertPolicyContainsRuleForTest(t, client, createdPolicy.ID, createdRule.ID)

	deleteStdout, deleteStderr, err := runCLIWithOverridesAndRateLimit(
		t,
		2*time.Minute,
		env,
		"agent",
		"rule",
		"delete",
		createdRule.ID,
		"--yes",
	)
	if err == nil {
		t.Fatalf("expected deleting the last rule on a policy to fail\nstdout: %s\nstderr: %s", deleteStdout, deleteStderr)
	}
	if !strings.Contains(strings.ToLower(deleteStderr), "cannot delete the last rule") {
		t.Fatalf("expected last-rule delete error, got stderr: %s", deleteStderr)
	}

	assertPolicyContainsRuleForTest(t, client, createdPolicy.ID, createdRule.ID)
}

func TestCLI_AgentRuleCommands_RejectMixedScopeRule(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	client := getTestClient()
	sharedPolicyID := findNonAgentOnlyPolicyIDForTest(t, client)
	if sharedPolicyID == "" {
		t.Skip("no non-agent-only policy available to build mixed scope in this environment")
	}

	email := newAgentTestEmail(t, "rule-mixed")
	createdAccount := createAgentWithPolicyForTest(t, email, sharedPolicyID)
	createdRule := createRuleForTest(t, client, fmt.Sprintf("it-rule-mixed-%d", time.Now().UnixNano()))
	attachRuleToPolicyForTest(t, client, sharedPolicyID, createdRule.ID)
	assertPolicyContainsRuleForTest(t, client, sharedPolicyID, createdRule.ID)

	t.Cleanup(func() {
		if createdRule != nil && createdRule.ID != "" {
			removeRuleFromPolicyForTest(t, client, sharedPolicyID, createdRule.ID)
		}
		if createdRule != nil && createdRule.ID != "" {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeleteRule(ctx, createdRule.ID)
		}
		if createdAccount != nil && createdAccount.ID != "" {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeleteAgentAccount(ctx, createdAccount.ID)
		}
	})
	if exists, _ := waitForAgentByEmail(t, client, email, true); !exists {
		t.Fatalf("created agent account %q did not appear in list", email)
	}

	updateStdout, updateStderr, err := runCLIWithOverridesAndRateLimit(
		t,
		2*time.Minute,
		env,
		"agent",
		"rule",
		"update",
		createdRule.ID,
		"--policy-id", sharedPolicyID,
		"--name", fmt.Sprintf("reject-mixed-%d", time.Now().UnixNano()),
		"--json",
	)
	if err == nil {
		t.Fatalf("expected rule update to fail for mixed-scope rule\nstdout: %s\nstderr: %s", updateStdout, updateStderr)
	}
	if !strings.Contains(strings.ToLower(updateStderr), "shared with a non-agent policy") {
		t.Fatalf("expected mixed-scope rejection, got stderr: %s", updateStderr)
	}

	deleteStdout, deleteStderr, err := runCLIWithOverridesAndRateLimit(
		t,
		2*time.Minute,
		env,
		"agent",
		"rule",
		"delete",
		createdRule.ID,
		"--policy-id", sharedPolicyID,
		"--yes",
	)
	if err == nil {
		t.Fatalf("expected rule delete to fail for mixed-scope rule\nstdout: %s\nstderr: %s", deleteStdout, deleteStderr)
	}
	if !strings.Contains(strings.ToLower(deleteStderr), "shared with a non-agent policy") {
		t.Fatalf("expected mixed-scope rejection, got stderr: %s", deleteStderr)
	}
}

func assertPolicyContainsRuleForTest(t *testing.T, client interface {
	GetPolicy(context.Context, string) (*domain.Policy, error)
}, policyID, ruleID string) {
	t.Helper()

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		acquireRateLimit(t)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		policy, err := client.GetPolicy(ctx, policyID)
		cancel()
		if err == nil && containsString(policy.Rules, ruleID) {
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("policy %q does not include rule %q", policyID, ruleID)
}

func assertPolicyMissingRuleForTest(t *testing.T, client interface {
	GetPolicy(context.Context, string) (*domain.Policy, error)
}, policyID, ruleID string) {
	t.Helper()

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		acquireRateLimit(t)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		policy, err := client.GetPolicy(ctx, policyID)
		cancel()
		if err == nil && !containsString(policy.Rules, ruleID) {
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("policy %q still includes deleted rule %q", policyID, ruleID)
}

func removeRuleFromPolicyForTest(t *testing.T, client interface {
	GetPolicy(context.Context, string) (*domain.Policy, error)
	UpdatePolicy(context.Context, string, map[string]any) (*domain.Policy, error)
}, policyID, ruleID string) {
	t.Helper()

	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	policy, err := client.GetPolicy(ctx, policyID)
	if err != nil || !containsString(policy.Rules, ruleID) {
		return
	}

	updatedRules := make([]string, 0, len(policy.Rules))
	for _, existingRuleID := range policy.Rules {
		if existingRuleID == ruleID {
			continue
		}
		updatedRules = append(updatedRules, existingRuleID)
	}

	_, _ = client.UpdatePolicy(ctx, policyID, map[string]any{"rules": updatedRules})
}

func attachRuleToPolicyForTest(t *testing.T, client interface {
	GetPolicy(context.Context, string) (*domain.Policy, error)
	UpdatePolicy(context.Context, string, map[string]any) (*domain.Policy, error)
}, policyID, ruleID string) {
	t.Helper()

	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	policy, err := client.GetPolicy(ctx, policyID)
	if err != nil {
		t.Fatalf("failed to get policy %q: %v", policyID, err)
	}
	if containsString(policy.Rules, ruleID) {
		return
	}

	updatedRules := append(append([]string(nil), policy.Rules...), ruleID)
	if _, err := client.UpdatePolicy(ctx, policyID, map[string]any{"rules": updatedRules}); err != nil {
		t.Fatalf("failed to attach rule %q to policy %q: %v", ruleID, policyID, err)
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
