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

func TestCLI_AgentRuleLifecycle_CreateReadUpdateDeleteOutbound(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	client := getTestClient()
	email := newAgentTestEmail(t, "rule-outbound")
	policyName := newPolicyTestName("rule-outbound")
	ruleName := fmt.Sprintf("it-rule-outbound-%d", time.Now().UnixNano())
	updatedRuleName := fmt.Sprintf("it-rule-outbound-updated-%d", time.Now().UnixNano())

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
		t.Fatalf("failed to create policy for outbound rule lifecycle: %v", err)
	}
	createdPolicy = policy

	createdAccount = createAgentWithPolicyForTest(t, email, createdPolicy.ID)
	env["NYLAS_GRANT_ID"] = createdAccount.ID

	placeholderRule = createRuleForTest(t, client, "it-rule-outbound-placeholder")
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
		"--description", "Archives outbound compose mail to example.com",
		"--trigger", "outbound",
		"--match-operator", "all",
		"--condition", "recipient.domain,is,example.com",
		"--condition", "outbound.type,is,compose",
		"--action", "archive",
		"--json",
	)
	if err != nil {
		t.Fatalf("outbound rule create failed: %v\nstdout: %s\nstderr: %s", err, createStdout, createStderr)
	}

	var rule domain.Rule
	if err := json.Unmarshal([]byte(createStdout), &rule); err != nil {
		t.Fatalf("failed to parse outbound rule create JSON: %v\noutput: %s", err, createStdout)
	}
	if rule.ID == "" {
		t.Fatalf("expected created outbound rule ID, got output: %s", createStdout)
	}
	if rule.Name != ruleName {
		t.Fatalf("created outbound rule name = %q, want %q", rule.Name, ruleName)
	}
	if rule.Trigger != "outbound" {
		t.Fatalf("created outbound rule trigger = %q, want %q", rule.Trigger, "outbound")
	}
	createdRule = &rule

	assertPolicyContainsRuleForTest(t, client, createdPolicy.ID, createdRule.ID)

	readStdout, readStderr, err := runCLIWithOverridesAndRateLimit(
		t,
		2*time.Minute,
		env,
		"agent",
		"rule",
		"read",
		createdRule.ID,
	)
	if err != nil {
		t.Fatalf("outbound rule read failed: %v\nstdout: %s\nstderr: %s", err, readStdout, readStderr)
	}
	if !strings.Contains(readStdout, "recipient.domain is example.com") ||
		!strings.Contains(readStdout, "outbound.type is compose") ||
		!strings.Contains(readStdout, "archive") {
		t.Fatalf("outbound rule read output missing expected sections\noutput: %s", readStdout)
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
		"--description", "Archives outbound reply mail to example.org",
		"--condition", "recipient.domain,is,example.org",
		"--condition", "outbound.type,is,reply",
		"--action", "archive",
		"--json",
	)
	if err != nil {
		t.Fatalf("outbound rule update failed: %v\nstdout: %s\nstderr: %s", err, updateStdout, updateStderr)
	}

	var updatedRule domain.Rule
	if err := json.Unmarshal([]byte(updateStdout), &updatedRule); err != nil {
		t.Fatalf("failed to parse outbound rule update JSON: %v\noutput: %s", err, updateStdout)
	}
	if updatedRule.Name != updatedRuleName {
		t.Fatalf("updated outbound rule name = %q, want %q", updatedRule.Name, updatedRuleName)
	}
	if updatedRule.Trigger != "outbound" {
		t.Fatalf("updated outbound rule trigger = %q, want %q", updatedRule.Trigger, "outbound")
	}

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
	if err != nil {
		t.Fatalf("outbound rule delete failed: %v\nstdout: %s\nstderr: %s", err, deleteStdout, deleteStderr)
	}
	if !strings.Contains(strings.ToLower(deleteStdout), "deleted") {
		t.Fatalf("expected delete confirmation in stdout, got: %s", deleteStdout)
	}

	assertPolicyMissingRuleForTest(t, client, createdPolicy.ID, createdRule.ID)
	createdRule = nil
}
