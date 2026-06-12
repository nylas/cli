package domain

import "encoding/json"

// Workspace represents a grant workspace. For provider=nylas accounts, this is
// the attachment point for policy and rule relationships.
type Workspace struct {
	ID            string   `json:"workspace_id,omitempty"`
	ApplicationID string   `json:"application_id,omitempty"`
	Name          string   `json:"name,omitempty"`
	Domain        *string  `json:"domain,omitempty"`
	AutoGroup     bool     `json:"auto_group,omitempty"`
	Default       bool     `json:"default"`
	PolicyID      string   `json:"policy_id,omitempty"`
	RulesIDs      []string `json:"rule_ids,omitempty"`
	CreatedAt     UnixTime `json:"created_at,omitempty"`
	UpdatedAt     UnixTime `json:"updated_at,omitempty"`
}

// CreateWorkspaceRequest creates a new workspace.
type CreateWorkspaceRequest struct {
	Name      string   `json:"name"`
	Domain    string   `json:"domain,omitempty"`
	AutoGroup *bool    `json:"auto_group,omitempty"`
	PolicyID  string   `json:"policy_id,omitempty"`
	RulesIDs  []string `json:"rule_ids,omitempty"`
}

// UpdateWorkspaceRequest updates workspace policy/rule attachments.
// PolicyID semantics: nil leaves the policy untouched; a pointer to the empty
// string detaches it (the API accepts a UUID or null, never "").
type UpdateWorkspaceRequest struct {
	PolicyID *string   `json:"policy_id,omitempty"`
	RulesIDs *[]string `json:"rule_ids,omitempty"`
}

// MarshalJSON serializes a PolicyID pointing at the empty string as JSON null,
// which is the only detach signal the API accepts.
func (r UpdateWorkspaceRequest) MarshalJSON() ([]byte, error) {
	out := make(map[string]any, 2)
	if r.PolicyID != nil {
		if *r.PolicyID == "" {
			out["policy_id"] = nil
		} else {
			out["policy_id"] = *r.PolicyID
		}
	}
	if r.RulesIDs != nil {
		out["rule_ids"] = *r.RulesIDs
	}
	return json.Marshal(out)
}

// WorkspaceAssignRequest moves grants into or out of a workspace via the
// manual-assign endpoint. The API requires these exact field names; assigning
// a grant moves it even if it currently belongs to another workspace, while
// removing leaves it in no workspace (not the default one).
type WorkspaceAssignRequest struct {
	AssignGrants []string `json:"assign_grants,omitempty"`
	RemoveGrants []string `json:"remove_grants,omitempty"`
}

// WorkspaceAssignResult reports which grants the API assigned or removed.
type WorkspaceAssignResult struct {
	ApplicationID  string   `json:"application_id,omitempty"`
	WorkspaceID    string   `json:"workspace_id,omitempty"`
	Domain         string   `json:"domain,omitempty"`
	GrantsAssigned []string `json:"grants_assigned,omitempty"`
	GrantsRemoved  []string `json:"grants_removed,omitempty"`
}
