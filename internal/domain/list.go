package domain

// AgentList represents a /v3/lists resource. Lists hold normalized values
// (domains, TLDs, or email addresses) referenced by rule in_list conditions.
type AgentList struct {
	ID             string   `json:"id,omitempty"`
	Name           string   `json:"name,omitempty"`
	Description    string   `json:"description,omitempty"`
	Type           string   `json:"type,omitempty"`
	ItemsCount     int      `json:"items_count"`
	ApplicationID  string   `json:"application_id,omitempty"`
	OrganizationID string   `json:"organization_id,omitempty"`
	CreatedAt      UnixTime `json:"created_at,omitzero"`
	UpdatedAt      UnixTime `json:"updated_at,omitzero"`
}

// AgentListTypes enumerates the immutable list types accepted by /v3/lists.
var AgentListTypes = []string{"domain", "tld", "address"}
