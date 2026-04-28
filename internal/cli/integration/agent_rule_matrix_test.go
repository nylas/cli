//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

type ruleMatrixScope struct {
	env    map[string]string
	client interface {
		CreatePolicy(context.Context, map[string]any) (*domain.Policy, error)
		DeletePolicy(context.Context, string) error
		DeleteAgentAccount(context.Context, string) error
		DeleteRule(context.Context, string) error
	}
	policyID   string
	accountID  string
	createdIDs []string
}

type ruleConditionMatrixCase struct {
	name          string
	trigger       string
	field         string
	operator      string
	rawValue      string
	expectedValue any
}

type ruleActionMatrixCase struct {
	name          string
	trigger       string
	actionArg     string
	expectedType  string
	expectedValue any
}

func TestCLI_AgentRuleMatrix_CreateAllSupportedConditionsAndActions(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	scope := setupRuleMatrixScope(t, "rule-matrix-create")
	placeholder := createRuleForTest(t, getTestClient(), "it-rule-matrix-create-placeholder")
	scope.trackRule(placeholder.ID)
	attachRuleToPolicyForTest(t, getTestClient(), scope.policyID, placeholder.ID)

	for _, tc := range buildRuleConditionMatrixCases() {
		t.Run("create-"+tc.name, func(t *testing.T) {
			rule := runAgentRuleCreateJSON(t, scope.env, scope.policyID,
				"--name", fmt.Sprintf("it-%s-%d", tc.name, time.Now().UnixNano()),
				"--trigger", tc.trigger,
				"--match-operator", "all",
				"--condition", buildConditionArg(tc.field, tc.operator, tc.rawValue),
				"--action", "archive",
			)
			scope.trackRule(rule.ID)
			assertRuleTrigger(t, rule, tc.trigger)
			assertRuleMatchOperator(t, rule, "all")
			assertRuleCondition(t, rule, tc.field, tc.operator, tc.expectedValue)
			assertRuleAction(t, rule, "archive", nil)
		})
	}

	for _, tc := range buildRuleActionMatrixCases() {
		t.Run("create-action-"+tc.name, func(t *testing.T) {
			rule := runAgentRuleCreateJSON(t, scope.env, scope.policyID,
				"--name", fmt.Sprintf("it-action-%s-%d", tc.name, time.Now().UnixNano()),
				"--trigger", tc.trigger,
				"--condition", representativeCondition(tc.trigger),
				"--action", tc.actionArg,
			)
			scope.trackRule(rule.ID)
			assertRuleTrigger(t, rule, tc.trigger)
			assertRuleAction(t, rule, tc.expectedType, tc.expectedValue)
		})
	}

	inboundStateRule := runAgentRuleCreateJSON(t, scope.env, scope.policyID,
		"--name", fmt.Sprintf("it-state-inbound-%d", time.Now().UnixNano()),
		"--priority", "3",
		"--disabled",
		"--match-operator", "any",
		"--condition", "from.domain,is,alpha.example",
		"--condition", "from.domain,is,beta.example",
		"--action", "archive",
	)
	scope.trackRule(inboundStateRule.ID)
	assertRuleTrigger(t, inboundStateRule, "inbound")
	assertRuleEnabled(t, inboundStateRule, false)
	assertRulePriority(t, inboundStateRule, 3)
	assertRuleMatchOperator(t, inboundStateRule, "any")
	assertRuleConditionCount(t, inboundStateRule, 2)

	outboundStateRule := runAgentRuleCreateJSON(t, scope.env, scope.policyID,
		"--name", fmt.Sprintf("it-state-outbound-%d", time.Now().UnixNano()),
		"--trigger", "outbound",
		"--priority", "4",
		"--disabled",
		"--match-operator", "any",
		"--condition", "recipient.domain,is,alpha.example",
		"--condition", "outbound.type,is,compose",
		"--action", "archive",
	)
	scope.trackRule(outboundStateRule.ID)
	assertRuleTrigger(t, outboundStateRule, "outbound")
	assertRuleEnabled(t, outboundStateRule, false)
	assertRulePriority(t, outboundStateRule, 4)
	assertRuleMatchOperator(t, outboundStateRule, "any")
	assertRuleConditionCount(t, outboundStateRule, 2)
}

func TestCLI_AgentRuleMatrix_UpdateAllSupportedConditionsAndActions(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	scope := setupRuleMatrixScope(t, "rule-matrix-update")
	client := getTestClient()

	placeholder := createRuleForTest(t, client, "it-rule-matrix-update-placeholder")
	scope.trackRule(placeholder.ID)
	attachRuleToPolicyForTest(t, client, scope.policyID, placeholder.ID)

	inboundBase := createMatrixRuleForTest(t, client, "inbound", "it-rule-matrix-update-inbound")
	scope.trackRule(inboundBase.ID)
	attachRuleToPolicyForTest(t, client, scope.policyID, inboundBase.ID)

	outboundBase := createMatrixRuleForTest(t, client, "outbound", "it-rule-matrix-update-outbound")
	scope.trackRule(outboundBase.ID)
	attachRuleToPolicyForTest(t, client, scope.policyID, outboundBase.ID)

	for _, tc := range buildRuleConditionMatrixCases() {
		t.Run("update-condition-"+tc.name, func(t *testing.T) {
			ruleID := inboundBase.ID
			if tc.trigger == "outbound" {
				ruleID = outboundBase.ID
			}

			rule := runAgentRuleUpdateJSON(t, scope.env, scope.policyID, ruleID,
				"--trigger", tc.trigger,
				"--match-operator", "all",
				"--condition", buildConditionArg(tc.field, tc.operator, tc.rawValue),
				"--action", "archive",
			)
			assertRuleTrigger(t, rule, tc.trigger)
			assertRuleMatchOperator(t, rule, "all")
			assertRuleCondition(t, rule, tc.field, tc.operator, tc.expectedValue)
			assertRuleAction(t, rule, "archive", nil)
		})
	}

	for _, tc := range buildRuleActionMatrixCases() {
		t.Run("update-action-"+tc.name, func(t *testing.T) {
			ruleID := inboundBase.ID
			if tc.trigger == "outbound" {
				ruleID = outboundBase.ID
			}

			rule := runAgentRuleUpdateJSON(t, scope.env, scope.policyID, ruleID,
				"--trigger", tc.trigger,
				"--match-operator", "all",
				"--condition", representativeCondition(tc.trigger),
				"--action", tc.actionArg,
			)
			assertRuleTrigger(t, rule, tc.trigger)
			assertRuleAction(t, rule, tc.expectedType, tc.expectedValue)
		})
	}

	inboundAny := runAgentRuleUpdateJSON(t, scope.env, scope.policyID, inboundBase.ID,
		"--trigger", "inbound",
		"--priority", "7",
		"--disabled",
		"--match-operator", "any",
		"--condition", "from.address,is,alpha@example.com",
		"--condition", "from.domain,is,beta.example",
		"--action", "mark_as_read",
	)
	assertRuleTrigger(t, inboundAny, "inbound")
	assertRulePriority(t, inboundAny, 7)
	assertRuleEnabled(t, inboundAny, false)
	assertRuleMatchOperator(t, inboundAny, "any")
	assertRuleConditionCount(t, inboundAny, 2)
	assertRuleAction(t, inboundAny, "mark_as_read", nil)

	outboundAny := runAgentRuleUpdateJSON(t, scope.env, scope.policyID, outboundBase.ID,
		"--trigger", "outbound",
		"--priority", "8",
		"--disabled",
		"--match-operator", "any",
		"--condition", "recipient.address,is,alpha@example.com",
		"--condition", "outbound.type,is,reply",
		"--action", "mark_as_starred",
	)
	assertRuleTrigger(t, outboundAny, "outbound")
	assertRulePriority(t, outboundAny, 8)
	assertRuleEnabled(t, outboundAny, false)
	assertRuleMatchOperator(t, outboundAny, "any")
	assertRuleConditionCount(t, outboundAny, 2)
	assertRuleAction(t, outboundAny, "mark_as_starred", nil)

	flippedOutbound := runAgentRuleUpdateJSON(t, scope.env, scope.policyID, inboundBase.ID,
		"--trigger", "outbound",
		"--condition", "recipient.domain,is,example.org",
		"--condition", "outbound.type,is,reply",
		"--action", "archive",
	)
	assertRuleTrigger(t, flippedOutbound, "outbound")
	assertRuleCondition(t, flippedOutbound, "recipient.domain", "is", "example.org")

	flippedInbound := runAgentRuleUpdateJSON(t, scope.env, scope.policyID, outboundBase.ID,
		"--trigger", "inbound",
		"--condition", "from.domain,is,example.org",
		"--action", "archive",
	)
	assertRuleTrigger(t, flippedInbound, "inbound")
	assertRuleCondition(t, flippedInbound, "from.domain", "is", "example.org")
}

func setupRuleMatrixScope(t *testing.T, prefix string) *ruleMatrixScope {
	t.Helper()

	env := newAgentSandboxEnv(t)
	client := getTestClient()

	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	policy, err := client.CreatePolicy(ctx, map[string]any{"name": newPolicyTestName(prefix)})
	cancel()
	if err != nil {
		t.Fatalf("failed to create policy for %s: %v", prefix, err)
	}

	email := newAgentTestEmail(t, prefix)
	account := createAgentWithPolicyForTest(t, email, policy.ID)
	env["NYLAS_GRANT_ID"] = account.ID

	scope := &ruleMatrixScope{
		env:       env,
		client:    client,
		policyID:  policy.ID,
		accountID: account.ID,
	}

	t.Cleanup(func() {
		seen := make(map[string]struct{}, len(scope.createdIDs))
		for _, ruleID := range scope.createdIDs {
			if _, ok := seen[ruleID]; ok || strings.TrimSpace(ruleID) == "" {
				continue
			}
			seen[ruleID] = struct{}{}

			removeRuleFromPolicyForTest(t, client, scope.policyID, ruleID)
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := client.DeleteRule(ctx, ruleID); err != nil {
				t.Logf("cleanup delete rule %s: %v", ruleID, err)
			}
			cancel()
		}

		acquireRateLimit(t)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := client.DeleteAgentAccount(ctx, scope.accountID); err != nil {
			t.Logf("cleanup delete agent account %s: %v", scope.accountID, err)
		}
		cancel()

		acquireRateLimit(t)
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		if err := client.DeletePolicy(ctx, scope.policyID); err != nil {
			t.Logf("cleanup delete policy %s: %v", scope.policyID, err)
		}
		cancel()
	})

	return scope
}

func (s *ruleMatrixScope) trackRule(ruleID string) {
	s.createdIDs = append(s.createdIDs, ruleID)
}

func runAgentRuleCreateJSON(t *testing.T, env map[string]string, policyID string, args ...string) domain.Rule {
	t.Helper()

	cmdArgs := append([]string{"agent", "rule", "create"}, args...)
	cmdArgs = append(cmdArgs, "--policy-id", policyID, "--json")
	stdout, stderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, cmdArgs...)
	if err != nil {
		t.Fatalf("agent rule create failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var rule domain.Rule
	if err := json.Unmarshal([]byte(stdout), &rule); err != nil {
		t.Fatalf("failed to parse create JSON: %v\noutput: %s", err, stdout)
	}
	if rule.ID == "" {
		t.Fatalf("expected created rule ID, got: %s", stdout)
	}

	return rule
}

func runAgentRuleUpdateJSON(t *testing.T, env map[string]string, policyID, ruleID string, args ...string) domain.Rule {
	t.Helper()

	cmdArgs := append([]string{"agent", "rule", "update", ruleID}, args...)
	cmdArgs = append(cmdArgs, "--policy-id", policyID, "--json")
	stdout, stderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, cmdArgs...)
	if err != nil {
		t.Fatalf("agent rule update failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var rule domain.Rule
	if err := json.Unmarshal([]byte(stdout), &rule); err != nil {
		t.Fatalf("failed to parse update JSON: %v\noutput: %s", err, stdout)
	}
	if rule.ID != ruleID {
		t.Fatalf("updated rule ID = %q, want %q", rule.ID, ruleID)
	}

	return rule
}

func createMatrixRuleForTest(t *testing.T, client interface {
	CreateRule(context.Context, map[string]any) (*domain.Rule, error)
}, trigger, name string) *domain.Rule {
	t.Helper()

	payload := map[string]any{
		"name":    name,
		"enabled": true,
		"trigger": trigger,
		"match": map[string]any{
			"operator": "all",
			"conditions": []map[string]any{{
				"field":    representativeField(trigger),
				"operator": "is",
				"value":    representativeValue(trigger),
			}},
		},
		"actions": []map[string]any{{
			"type": "archive",
		}},
	}

	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rule, err := client.CreateRule(ctx, payload)
	if err != nil {
		t.Fatalf("failed to create %s matrix rule %q: %v", trigger, name, err)
	}

	return rule
}

func representativeCondition(trigger string) string {
	return buildConditionArg(representativeField(trigger), "is", representativeValue(trigger))
}

func representativeField(trigger string) string {
	if trigger == "outbound" {
		return "recipient.domain"
	}
	return "from.domain"
}

func representativeValue(trigger string) string {
	if trigger == "outbound" {
		return "example.com"
	}
	return "example.com"
}

func buildConditionArg(field, operator, rawValue string) string {
	return fmt.Sprintf("%s,%s,%s", field, operator, rawValue)
}

func buildRuleConditionMatrixCases() []ruleConditionMatrixCase {
	cases := make([]ruleConditionMatrixCase, 0, 38)

	appendStringFieldCases := func(trigger, field, exactValue, containsValue, listPrefix string) {
		cases = append(cases,
			ruleConditionMatrixCase{
				name:          fmt.Sprintf("%s-%s-is", trigger, strings.ReplaceAll(field, ".", "-")),
				trigger:       trigger,
				field:         field,
				operator:      "is",
				rawValue:      exactValue,
				expectedValue: exactValue,
			},
			ruleConditionMatrixCase{
				name:          fmt.Sprintf("%s-%s-is-not", trigger, strings.ReplaceAll(field, ".", "-")),
				trigger:       trigger,
				field:         field,
				operator:      "is_not",
				rawValue:      exactValue,
				expectedValue: exactValue,
			},
			ruleConditionMatrixCase{
				name:          fmt.Sprintf("%s-%s-contains", trigger, strings.ReplaceAll(field, ".", "-")),
				trigger:       trigger,
				field:         field,
				operator:      "contains",
				rawValue:      containsValue,
				expectedValue: containsValue,
			},
			ruleConditionMatrixCase{
				name:          fmt.Sprintf("%s-%s-in-list", trigger, strings.ReplaceAll(field, ".", "-")),
				trigger:       trigger,
				field:         field,
				operator:      "in_list",
				rawValue:      fmt.Sprintf("%s-1,%s-2", listPrefix, listPrefix),
				expectedValue: []string{fmt.Sprintf("%s-1", listPrefix), fmt.Sprintf("%s-2", listPrefix)},
			},
		)
	}

	appendStringFieldCases("inbound", "from.address", "sender@example.com", "sender@", "inbound-address-list")
	appendStringFieldCases("inbound", "from.domain", "example.com", "ample", "inbound-domain-list")
	appendStringFieldCases("inbound", "from.tld", "com", "o", "inbound-tld-list")

	appendStringFieldCases("outbound", "from.address", "sender@example.com", "sender@", "outbound-from-address-list")
	appendStringFieldCases("outbound", "from.domain", "example.com", "ample", "outbound-from-domain-list")
	appendStringFieldCases("outbound", "from.tld", "com", "o", "outbound-from-tld-list")
	appendStringFieldCases("outbound", "recipient.address", "recipient@example.net", "recipient@", "outbound-recipient-address-list")
	appendStringFieldCases("outbound", "recipient.domain", "example.net", "ample", "outbound-recipient-domain-list")
	appendStringFieldCases("outbound", "recipient.tld", "net", "e", "outbound-recipient-tld-list")

	cases = append(cases,
		ruleConditionMatrixCase{
			name:          "outbound-outbound-type-is",
			trigger:       "outbound",
			field:         "outbound.type",
			operator:      "is",
			rawValue:      "compose",
			expectedValue: "compose",
		},
		ruleConditionMatrixCase{
			name:          "outbound-outbound-type-is-not",
			trigger:       "outbound",
			field:         "outbound.type",
			operator:      "is_not",
			rawValue:      "reply",
			expectedValue: "reply",
		},
	)

	return cases
}

func buildRuleActionMatrixCases() []ruleActionMatrixCase {
	base := []ruleActionMatrixCase{
		{name: "block", actionArg: "block", expectedType: "block"},
		{name: "mark-as-spam", actionArg: "mark_as_spam", expectedType: "mark_as_spam"},
		{name: "assign-to-folder", actionArg: "assign_to_folder=folder-123", expectedType: "assign_to_folder", expectedValue: "folder-123"},
		{name: "mark-as-read", actionArg: "mark_as_read", expectedType: "mark_as_read"},
		{name: "mark-as-starred", actionArg: "mark_as_starred", expectedType: "mark_as_starred"},
		{name: "archive", actionArg: "archive", expectedType: "archive"},
		{name: "trash", actionArg: "trash", expectedType: "trash"},
	}

	cases := make([]ruleActionMatrixCase, 0, len(base)*2)
	for _, trigger := range []string{"inbound", "outbound"} {
		for _, tc := range base {
			cases = append(cases, ruleActionMatrixCase{
				name:          trigger + "-" + tc.name,
				trigger:       trigger,
				actionArg:     tc.actionArg,
				expectedType:  tc.expectedType,
				expectedValue: tc.expectedValue,
			})
		}
	}

	return cases
}

func assertRuleTrigger(t *testing.T, rule domain.Rule, want string) {
	t.Helper()
	if rule.Trigger != want {
		t.Fatalf("rule trigger = %q, want %q", rule.Trigger, want)
	}
}

func assertRuleEnabled(t *testing.T, rule domain.Rule, want bool) {
	t.Helper()
	if rule.Enabled == nil || *rule.Enabled != want {
		t.Fatalf("rule enabled = %v, want %t", rule.Enabled, want)
	}
}

func assertRulePriority(t *testing.T, rule domain.Rule, want int) {
	t.Helper()
	if rule.Priority == nil || *rule.Priority != want {
		t.Fatalf("rule priority = %v, want %d", rule.Priority, want)
	}
}

func assertRuleMatchOperator(t *testing.T, rule domain.Rule, want string) {
	t.Helper()
	if rule.Match == nil || rule.Match.Operator != want {
		t.Fatalf("rule match operator = %q, want %q", rule.Match.Operator, want)
	}
}

func assertRuleConditionCount(t *testing.T, rule domain.Rule, want int) {
	t.Helper()
	if rule.Match == nil || len(rule.Match.Conditions) != want {
		t.Fatalf("rule condition count = %d, want %d", len(rule.Match.Conditions), want)
	}
}

func assertRuleCondition(t *testing.T, rule domain.Rule, field, operator string, value any) {
	t.Helper()
	if rule.Match == nil || len(rule.Match.Conditions) == 0 {
		t.Fatalf("rule has no conditions: %+v", rule)
	}
	condition := rule.Match.Conditions[0]
	if condition.Field != field {
		t.Fatalf("condition field = %q, want %q", condition.Field, field)
	}
	if condition.Operator != operator {
		t.Fatalf("condition operator = %q, want %q", condition.Operator, operator)
	}
	if !ruleValueEqual(condition.Value, value) {
		t.Fatalf("condition value = %#v, want %#v", condition.Value, value)
	}
}

func assertRuleAction(t *testing.T, rule domain.Rule, actionType string, value any) {
	t.Helper()
	if len(rule.Actions) == 0 {
		t.Fatalf("rule has no actions: %+v", rule)
	}
	action := rule.Actions[0]
	if action.Type != actionType {
		t.Fatalf("action type = %q, want %q", action.Type, actionType)
	}
	if !ruleValueEqual(action.Value, value) {
		t.Fatalf("action value = %#v, want %#v", action.Value, value)
	}
}

func ruleValueEqual(got, want any) bool {
	if want == nil {
		return got == nil
	}

	wantSlice, wantIsSlice := want.([]string)
	if !wantIsSlice {
		return reflect.DeepEqual(got, want)
	}

	gotSlice, ok := got.([]any)
	if !ok {
		return false
	}
	if len(gotSlice) != len(wantSlice) {
		return false
	}
	for i := range gotSlice {
		gotValue, ok := gotSlice[i].(string)
		if !ok || gotValue != wantSlice[i] {
			return false
		}
	}
	return true
}
