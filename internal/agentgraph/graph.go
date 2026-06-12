// Package agentgraph joins the five agent resource collections (accounts,
// workspaces, policies, rules, lists) into one account-rooted graph with
// referential-health flags. It is the shared engine behind both the
// `nylas agent overview` command and the Agent Studio board.
package agentgraph

import (
	"strings"

	"github.com/nylas/cli/internal/domain"
)

// List is a list referenced by a rule's in_list condition.
type List struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	Type       string `json:"type,omitempty"`
	ItemsCount int    `json:"items_count"`
	Missing    bool   `json:"missing"`
}

// Rule is a rule attached to a workspace, with the lists it references.
type Rule struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Trigger string `json:"trigger,omitempty"`
	Enabled bool   `json:"enabled"`
	Missing bool   `json:"missing"`
	Lists   []List `json:"lists,omitempty"`
}

// Policy is the policy governing an account, resolved through its workspace
// with the account-level setting as fallback.
type Policy struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Missing bool   `json:"missing"`
}

// Account is one agent account with its resolved workspace attachments.
type Account struct {
	ID               string  `json:"id"`
	Email            string  `json:"email"`
	Status           string  `json:"status,omitempty"`
	WorkspaceID      string  `json:"workspace_id,omitempty"`
	WorkspaceName    string  `json:"workspace_name,omitempty"`
	WorkspaceMissing bool    `json:"workspace_missing"`
	AutoGroup        bool    `json:"auto_group"`
	Default          bool    `json:"default"`
	SharedWith       int     `json:"shared_with"`
	Policy           *Policy `json:"policy,omitempty"`
	Rules            []Rule  `json:"rules,omitempty"`
}

// AccountRef identifies an account from a workspace's point of view.
type AccountRef struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Status string `json:"status,omitempty"`
}

// WorkspaceView is the workspace-centric slice of the graph: every workspace
// (including ones with no member accounts) with its resolved attachments.
type WorkspaceView struct {
	ID        string       `json:"id"`
	Name      string       `json:"name,omitempty"`
	AutoGroup bool         `json:"auto_group"`
	Default   bool         `json:"default"`
	Policy    *Policy      `json:"policy,omitempty"`
	Rules     []Rule       `json:"rules,omitempty"`
	Accounts  []AccountRef `json:"accounts,omitempty"`
}

// Overview is the full agent resource graph.
type Overview struct {
	Accounts       []Account       `json:"accounts"`
	Workspaces     []WorkspaceView `json:"workspaces"`
	OrphanPolicies []Policy        `json:"orphan_policies,omitempty"`
	OrphanRules    []Rule          `json:"orphan_rules,omitempty"`
	UnusedLists    []List          `json:"unused_lists,omitempty"`
	Totals         map[string]int  `json:"totals"`
}

// Build joins the five agent resource collections into one account-rooted
// graph, flagging references to resources that no longer exist.
func Build(
	accounts []domain.AgentAccount,
	workspaces []domain.Workspace,
	policies []domain.Policy,
	rules []domain.Rule,
	lists []domain.AgentList,
) *Overview {
	workspaceByID := make(map[string]domain.Workspace, len(workspaces))
	for _, ws := range workspaces {
		workspaceByID[ws.ID] = ws
	}
	policyByID := make(map[string]domain.Policy, len(policies))
	for _, p := range policies {
		policyByID[p.ID] = p
	}
	ruleByID := make(map[string]domain.Rule, len(rules))
	for _, r := range rules {
		ruleByID[r.ID] = r
	}
	listByID := make(map[string]domain.AgentList, len(lists))
	for _, l := range lists {
		listByID[l.ID] = l
	}

	accountsPerWorkspace := make(map[string]int, len(accounts))
	for _, acct := range accounts {
		if wsID := strings.TrimSpace(acct.WorkspaceID); wsID != "" {
			accountsPerWorkspace[wsID]++
		}
	}

	attachedPolicies := make(map[string]bool)
	attachedRules := make(map[string]bool)
	referencedLists := make(map[string]bool)
	for _, ws := range workspaces {
		if ws.PolicyID != "" {
			attachedPolicies[ws.PolicyID] = true
		}
		for _, id := range ws.RulesIDs {
			attachedRules[id] = true
		}
	}
	for _, acct := range accounts {
		if id := strings.TrimSpace(acct.Settings.PolicyID); id != "" {
			attachedPolicies[id] = true
		}
	}
	// A list counts as referenced when ANY existing rule names it, including
	// rules attached to no workspace — deleting such a list would still break
	// the rule the moment it is re-attached.
	for _, rule := range rules {
		for _, id := range ruleListIDs(rule) {
			referencedLists[id] = true
		}
	}

	overview := &Overview{
		Accounts: make([]Account, 0, len(accounts)),
		Totals: map[string]int{
			"accounts":   len(accounts),
			"workspaces": len(workspaces),
			"policies":   len(policies),
			"rules":      len(rules),
			"lists":      len(lists),
		},
	}

	for _, acct := range accounts {
		entry := Account{
			ID:          acct.ID,
			Email:       acct.Email,
			Status:      acct.GrantStatus,
			WorkspaceID: strings.TrimSpace(acct.WorkspaceID),
		}

		// The effective policy is the workspace's, falling back to the policy
		// attached on the account itself (same resolution as `agent policy`).
		effectivePolicyID := strings.TrimSpace(acct.Settings.PolicyID)

		if entry.WorkspaceID != "" {
			ws, ok := workspaceByID[entry.WorkspaceID]
			if !ok {
				entry.WorkspaceMissing = true
			} else {
				entry.WorkspaceName = ws.Name
				entry.AutoGroup = ws.AutoGroup
				entry.Default = ws.Default
				entry.SharedWith = accountsPerWorkspace[entry.WorkspaceID] - 1
				if wsPolicyID := strings.TrimSpace(ws.PolicyID); wsPolicyID != "" {
					effectivePolicyID = wsPolicyID
				}
				entry.Rules = resolveRules(ws.RulesIDs, ruleByID, listByID)
			}
		}
		entry.Policy = resolvePolicy(effectivePolicyID, policyByID)

		overview.Accounts = append(overview.Accounts, entry)
	}

	overview.Workspaces = make([]WorkspaceView, 0, len(workspaces))
	for _, ws := range workspaces {
		view := WorkspaceView{
			ID:        ws.ID,
			Name:      ws.Name,
			AutoGroup: ws.AutoGroup,
			Default:   ws.Default,
			Policy:    resolvePolicy(ws.PolicyID, policyByID),
			Rules:     resolveRules(ws.RulesIDs, ruleByID, listByID),
		}
		for _, acct := range accounts {
			if strings.TrimSpace(acct.WorkspaceID) == ws.ID {
				view.Accounts = append(view.Accounts, AccountRef{
					ID: acct.ID, Email: acct.Email, Status: acct.GrantStatus,
				})
			}
		}
		overview.Workspaces = append(overview.Workspaces, view)
	}

	for _, p := range policies {
		if !attachedPolicies[p.ID] {
			overview.OrphanPolicies = append(overview.OrphanPolicies, Policy{ID: p.ID, Name: p.Name})
		}
	}
	for _, r := range rules {
		if !attachedRules[r.ID] {
			overview.OrphanRules = append(overview.OrphanRules, Rule{
				ID: r.ID, Name: r.Name, Trigger: r.Trigger, Enabled: r.Enabled == nil || *r.Enabled,
			})
		}
	}
	for _, l := range lists {
		if !referencedLists[l.ID] {
			overview.UnusedLists = append(overview.UnusedLists, List{
				ID: l.ID, Name: l.Name, Type: l.Type, ItemsCount: l.ItemsCount,
			})
		}
	}

	return overview
}

func resolvePolicy(policyID string, policyByID map[string]domain.Policy) *Policy {
	policyID = strings.TrimSpace(policyID)
	if policyID == "" {
		return nil
	}
	if p, ok := policyByID[policyID]; ok {
		return &Policy{ID: p.ID, Name: p.Name}
	}
	return &Policy{ID: policyID, Missing: true}
}

func resolveRules(ruleIDs []string, ruleByID map[string]domain.Rule, listByID map[string]domain.AgentList) []Rule {
	var out []Rule
	for _, id := range ruleIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		rule, ok := ruleByID[id]
		if !ok {
			out = append(out, Rule{ID: id, Missing: true})
			continue
		}

		entry := Rule{
			ID:      rule.ID,
			Name:    rule.Name,
			Trigger: rule.Trigger,
			Enabled: rule.Enabled == nil || *rule.Enabled,
		}
		for _, listID := range ruleListIDs(rule) {
			if list, ok := listByID[listID]; ok {
				entry.Lists = append(entry.Lists, List{
					ID: list.ID, Name: list.Name, Type: list.Type, ItemsCount: list.ItemsCount,
				})
			} else {
				entry.Lists = append(entry.Lists, List{ID: listID, Missing: true})
			}
		}
		out = append(out, entry)
	}
	return out
}

// ruleListIDs collects the list IDs referenced by a rule's in_list conditions.
func ruleListIDs(rule domain.Rule) []string {
	if rule.Match == nil {
		return nil
	}
	var ids []string
	seen := make(map[string]bool)
	for _, condition := range rule.Match.Conditions {
		for _, id := range conditionListIDs(condition) {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// conditionListIDs extracts list IDs from an in_list condition value. The API
// returns the value as an array of IDs, but a bare string is tolerated.
func conditionListIDs(condition domain.RuleCondition) []string {
	if condition.Operator != "in_list" {
		return nil
	}
	switch value := condition.Value.(type) {
	case string:
		if value = strings.TrimSpace(value); value != "" {
			return []string{value}
		}
	case []string:
		var ids []string
		for _, v := range value {
			if v = strings.TrimSpace(v); v != "" {
				ids = append(ids, v)
			}
		}
		return ids
	case []any:
		var ids []string
		for _, v := range value {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				ids = append(ids, strings.TrimSpace(s))
			}
		}
		return ids
	}
	return nil
}
