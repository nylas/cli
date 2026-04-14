package domain

// AgentAccount represents a Nylas-managed agent account grant (provider=nylas).
type AgentAccount struct {
	ID           string               `json:"id"`
	Provider     Provider             `json:"provider"`
	Email        string               `json:"email"`
	GrantStatus  string               `json:"grant_status"`
	Settings     AgentAccountSettings `json:"settings,omitempty"`
	CreatedAt    UnixTime             `json:"created_at,omitempty"`
	UpdatedAt    UnixTime             `json:"updated_at,omitempty"`
	CredentialID string               `json:"credential_id,omitempty"`
	Blocked      bool                 `json:"blocked,omitempty"`
}

// AgentAccountSettings contains provider-managed metadata for agent accounts.
type AgentAccountSettings struct {
	PolicyID string `json:"policy_id,omitempty"`
}

// IsValid returns true if the agent account grant is in a valid state.
func (a *AgentAccount) IsValid() bool {
	return a.GrantStatus == "valid"
}
