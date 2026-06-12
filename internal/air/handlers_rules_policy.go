package air

import (
	"context"
	"errors"
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
		writeUpstreamError(w, http.StatusInternalServerError, "Failed to fetch default agent account", err)
		return
	}

	policyID, err := s.resolveAccountPolicyID(ctx, account)
	if err != nil {
		writeUpstreamError(w, http.StatusInternalServerError, "Failed to resolve workspace policy", err)
		return
	}
	if policyID == "" {
		writeJSON(w, http.StatusOK, PoliciesResponse{Policies: []domain.Policy{}})
		return
	}

	policy, err := s.nylasClient.GetPolicy(ctx, policyID)
	if err != nil {
		// A workspace can reference a policy that has since been deleted;
		// render that as "no policies" rather than an error.
		if errors.Is(err, domain.ErrPolicyNotFound) {
			writeJSON(w, http.StatusOK, PoliciesResponse{Policies: []domain.Policy{}})
			return
		}
		writeUpstreamError(w, http.StatusInternalServerError, "Failed to fetch policy", err)
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
		writeUpstreamError(w, http.StatusInternalServerError, "Failed to fetch default agent account", err)
		return
	}

	ruleIDs, err := s.resolveAccountRuleIDs(ctx, account)
	if err != nil {
		writeUpstreamError(w, http.StatusInternalServerError, "Failed to resolve workspace rules", err)
		return
	}
	if len(ruleIDs) == 0 {
		writeJSON(w, http.StatusOK, RulesResponse{Rules: []domain.Rule{}})
		return
	}

	allRules, err := s.nylasClient.ListRules(ctx)
	if err != nil {
		writeUpstreamError(w, http.StatusInternalServerError, "Failed to fetch rules", err)
		return
	}

	ruleSet := make(map[string]struct{}, len(ruleIDs))
	for _, id := range ruleIDs {
		ruleSet[id] = struct{}{}
	}

	rules := make([]domain.Rule, 0, len(ruleIDs))
	for _, rule := range allRules {
		if _, ok := ruleSet[rule.ID]; !ok {
			continue
		}
		rules = append(rules, rule)
	}

	writeJSON(w, http.StatusOK, RulesResponse{Rules: rules})
}

func (s *Server) handleAgentWorkspace(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if s.handleDemoMode(w, WorkspaceResponse{Workspace: demoWorkspace()}) {
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
		writeUpstreamError(w, http.StatusInternalServerError, "Failed to fetch default agent account", err)
		return
	}

	wsID := strings.TrimSpace(account.WorkspaceID)
	if wsID == "" {
		writeJSON(w, http.StatusOK, WorkspaceResponse{})
		return
	}

	workspace, err := s.nylasClient.GetWorkspace(ctx, wsID)
	if err != nil {
		if errors.Is(err, domain.ErrWorkspaceNotFound) {
			writeJSON(w, http.StatusOK, WorkspaceResponse{})
			return
		}
		writeUpstreamError(w, http.StatusInternalServerError, "Failed to fetch workspace", err)
		return
	}

	writeJSON(w, http.StatusOK, WorkspaceResponse{Workspace: workspace})
}

func (s *Server) handleAgentLists(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if s.handleDemoMode(w, AgentListsResponse{Lists: demoAgentLists()}) {
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

	lists, err := s.nylasClient.ListLists(ctx)
	if err != nil {
		writeUpstreamError(w, http.StatusInternalServerError, "Failed to fetch lists", err)
		return
	}
	if lists == nil {
		lists = []domain.AgentList{}
	}

	writeJSON(w, http.StatusOK, AgentListsResponse{Lists: lists})
}

func (s *Server) resolveAccountPolicyID(ctx context.Context, account *domain.AgentAccount) (string, error) {
	if wsID := strings.TrimSpace(account.WorkspaceID); wsID != "" {
		ws, err := s.nylasClient.GetWorkspace(ctx, wsID)
		switch {
		case errors.Is(err, domain.ErrWorkspaceNotFound):
			// A deleted workspace must not break the endpoint; fall back to
			// the policy reference stored on the account itself.
		case err != nil:
			return "", err
		case ws != nil:
			return strings.TrimSpace(ws.PolicyID), nil
		}
	}
	return strings.TrimSpace(account.Settings.PolicyID), nil
}

func (s *Server) resolveAccountRuleIDs(ctx context.Context, account *domain.AgentAccount) ([]string, error) {
	if wsID := strings.TrimSpace(account.WorkspaceID); wsID != "" {
		ws, err := s.nylasClient.GetWorkspace(ctx, wsID)
		switch {
		case errors.Is(err, domain.ErrWorkspaceNotFound):
			// Deleted workspace: fall back to the account's policy rules.
		case err != nil:
			return nil, err
		case ws != nil:
			var ids []string
			for _, id := range ws.RulesIDs {
				if id = strings.TrimSpace(id); id != "" {
					ids = append(ids, id)
				}
			}
			return ids, nil
		}
	}
	if policyID := strings.TrimSpace(account.Settings.PolicyID); policyID != "" {
		policy, err := s.nylasClient.GetPolicy(ctx, policyID)
		if err != nil {
			if errors.Is(err, domain.ErrPolicyNotFound) {
				return nil, nil
			}
			return nil, err
		}
		return policy.Rules, nil
	}
	return nil, nil
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

func demoWorkspace() *domain.Workspace {
	return &domain.Workspace{
		ID:        "workspace-demo",
		Name:      "Demo Tenant Workspace",
		AutoGroup: true,
		Default:   true,
		PolicyID:  "policy-demo-default",
		RulesIDs:  []string{"rule-demo-inbound"},
	}
}

func demoAgentLists() []domain.AgentList {
	return []domain.AgentList{
		{
			ID:          "list-demo-domains",
			Name:        "Blocked domains",
			Description: "Domains flagged by the demo inbound rule.",
			Type:        "domain",
			ItemsCount:  2,
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
