//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestCLI_AgentRuleLifecycle_OutboundCreateUpdateDelete(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	client := getTestClient()
	email := newAgentTestEmail(t, "rule-outbound")
	policyName := newPolicyTestName("rule-outbound")

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
		t.Fatalf("failed to create policy for outbound rule lifecycle: %v", err)
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
		"--name", "Block Replies",
		"--trigger", "outbound",
		"--condition", "outbound.type,is,reply",
		"--action", "block",
		"--json",
	)
	if err != nil {
		if outboundTriggerUnsupported(createStderr) {
			t.Skip("outbound rule trigger not supported by this API environment yet")
		}
		t.Fatalf("outbound rule create failed: %v\nstdout: %s\nstderr: %s", err, createStdout, createStderr)
	}

	var created domain.Rule
	if err := json.Unmarshal([]byte(createStdout), &created); err != nil {
		t.Fatalf("failed to parse outbound rule create JSON: %v\noutput: %s", err, createStdout)
	}
	if created.ID == "" {
		t.Fatalf("expected created outbound rule ID, got output: %s", createStdout)
	}
	if created.Trigger != "outbound" {
		t.Fatalf("created outbound rule trigger = %q, want %q", created.Trigger, "outbound")
	}
	createdRule = &created

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
	if !strings.Contains(readStdout, "Trigger:      outbound") {
		t.Fatalf("expected outbound trigger in read output\noutput: %s", readStdout)
	}

	updateStdout, updateStderr, err := runCLIWithOverridesAndRateLimit(
		t,
		2*time.Minute,
		env,
		"agent",
		"rule",
		"update",
		createdRule.ID,
		"--condition", "recipient.domain,is,example.org",
		"--action", "archive",
		"--json",
	)
	if err != nil {
		t.Fatalf("outbound rule update failed: %v\nstdout: %s\nstderr: %s", err, updateStdout, updateStderr)
	}

	var updated domain.Rule
	if err := json.Unmarshal([]byte(updateStdout), &updated); err != nil {
		t.Fatalf("failed to parse outbound rule update JSON: %v\noutput: %s", err, updateStdout)
	}
	if updated.Trigger != "outbound" {
		t.Fatalf("updated outbound rule trigger = %q, want %q", updated.Trigger, "outbound")
	}
	if updated.Match == nil || len(updated.Match.Conditions) != 1 || updated.Match.Conditions[0].Field != "recipient.domain" {
		t.Fatalf("updated outbound rule conditions = %#v, want recipient.domain", updated.Match)
	}
	if len(updated.Actions) != 1 || updated.Actions[0].Type != "archive" {
		t.Fatalf("updated outbound rule actions = %#v, want archive", updated.Actions)
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

func outboundTriggerUnsupported(stderr string) bool {
	stderr = strings.ToLower(stderr)
	if strings.Contains(stderr, "outbound rules are not enabled on this api environment") {
		return true
	}
	return strings.Contains(stderr, "invalid rule trigger") && strings.Contains(stderr, "must be 'inbound'")
}
