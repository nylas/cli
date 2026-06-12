package domain

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
type UpdateWorkspaceRequest struct {
	PolicyID *string   `json:"policy_id,omitempty"`
	RulesIDs *[]string `json:"rule_ids,omitempty"`
}
