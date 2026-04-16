package air

import (
	"net/http"
	"strings"

	"github.com/nylas/cli/internal/domain"
)

const rulesPolicyUnsupportedMessage = "Policy & Rules are only available for Nylas-managed accounts."

func (s *Server) handleListPolicies(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if s.handleDemoMode(w, PoliciesResponse{Policies: demoPolicies()}) {
		return
	}
	if !s.requireConfig(w) {
		return
	}

	grant, ok := s.requireDefaultGrantInfo(w)
	if !ok {
		return
	}
	if grant.Provider != domain.ProviderNylas {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": rulesPolicyUnsupportedMessage})
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	account, err := s.nylasClient.GetAgentAccount(ctx, grant.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch default agent account: " + err.Error(),
		})
		return
	}

	policyID := strings.TrimSpace(account.Settings.PolicyID)
	if policyID == "" {
		writeJSON(w, http.StatusOK, PoliciesResponse{Policies: []domain.Policy{}})
		return
	}

	policy, err := s.nylasClient.GetPolicy(ctx, policyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch policy: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, PoliciesResponse{Policies: []domain.Policy{*policy}})
}

func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if s.handleDemoMode(w, RulesResponse{Rules: demoRules()}) {
		return
	}
	if !s.requireConfig(w) {
		return
	}

	grant, ok := s.requireDefaultGrantInfo(w)
	if !ok {
		return
	}
	if grant.Provider != domain.ProviderNylas {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": rulesPolicyUnsupportedMessage})
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	account, err := s.nylasClient.GetAgentAccount(ctx, grant.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch default agent account: " + err.Error(),
		})
		return
	}

	policyID := strings.TrimSpace(account.Settings.PolicyID)
	if policyID == "" {
		writeJSON(w, http.StatusOK, RulesResponse{Rules: []domain.Rule{}})
		return
	}

	policy, err := s.nylasClient.GetPolicy(ctx, policyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch policy for rules: " + err.Error(),
		})
		return
	}

	ruleIDs := make(map[string]struct{}, len(policy.Rules))
	for _, ruleID := range policy.Rules {
		ruleID = strings.TrimSpace(ruleID)
		if ruleID == "" {
			continue
		}
		ruleIDs[ruleID] = struct{}{}
	}
	if len(ruleIDs) == 0 {
		writeJSON(w, http.StatusOK, RulesResponse{Rules: []domain.Rule{}})
		return
	}

	allRules, err := s.nylasClient.ListRules(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch rules: " + err.Error(),
		})
		return
	}

	rules := make([]domain.Rule, 0, len(ruleIDs))
	for _, rule := range allRules {
		if _, ok := ruleIDs[rule.ID]; !ok {
			continue
		}
		rules = append(rules, rule)
	}

	writeJSON(w, http.StatusOK, RulesResponse{Rules: rules})
}

func demoPolicies() []domain.Policy {
	return []domain.Policy{
		{
			ID:             "policy-demo-default",
			Name:           "Default Tenant Policy",
			ApplicationID:  "app-demo",
			OrganizationID: "org-demo",
			Rules:          []string{"rule-demo-inbound"},
		},
	}
}

func demoRules() []domain.Rule {
	enabled := true

	return []domain.Rule{
		{
			ID:          "rule-demo-inbound",
			Name:        "Block risky inbound senders",
			Description: "Flags inbound messages from blocked domains before they reach the inbox.",
			Enabled:     &enabled,
			Trigger:     "inbound",
			Match: &domain.RuleMatch{
				Operator: "all",
				Conditions: []domain.RuleCondition{{
					Field:    "from.domain",
					Operator: "is",
					Value:    "blocked.example",
				}},
			},
			Actions: []domain.RuleAction{{
				Type: "mark_as_spam",
			}},
		},
	}
}
