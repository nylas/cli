package domain

// Rule represents a policy rule resource.
type Rule struct {
	ID             string       `json:"id,omitempty"`
	Name           string       `json:"name,omitempty"`
	Description    string       `json:"description,omitempty"`
	Priority       *int         `json:"priority,omitempty"`
	Enabled        *bool        `json:"enabled,omitempty"`
	Trigger        string       `json:"trigger,omitempty"`
	Match          *RuleMatch   `json:"match,omitempty"`
	Actions        []RuleAction `json:"actions,omitempty"`
	ApplicationID  string       `json:"application_id,omitempty"`
	OrganizationID string       `json:"organization_id,omitempty"`
	CreatedAt      UnixTime     `json:"created_at,omitempty"`
	UpdatedAt      UnixTime     `json:"updated_at,omitempty"`
}

// RuleMatch describes how rule conditions are evaluated.
type RuleMatch struct {
	Operator   string          `json:"operator,omitempty"`
	Conditions []RuleCondition `json:"conditions,omitempty"`
}

// RuleCondition represents a single condition in a rule match expression.
type RuleCondition struct {
	Field    string `json:"field,omitempty"`
	Operator string `json:"operator,omitempty"`
	Value    any    `json:"value,omitempty"`
}

// RuleAction represents an action executed when a rule matches.
type RuleAction struct {
	Type  string `json:"type,omitempty"`
	Value any    `json:"value,omitempty"`
}
