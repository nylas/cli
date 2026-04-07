package domain

import "time"

// RemoteScope identifies whether an operation targets app-level or grant-level resources.
type RemoteScope string

const (
	ScopeApplication RemoteScope = "app"
	ScopeGrant       RemoteScope = "grant"
)

// ParseRemoteScope validates and converts a CLI scope string.
func ParseRemoteScope(scope string) (RemoteScope, error) {
	switch RemoteScope(scope) {
	case ScopeApplication, ScopeGrant:
		return RemoteScope(scope), nil
	default:
		return "", ErrInvalidInput
	}
}

// TemplateEngines returns the supported template engines from the Nylas API.
func TemplateEngines() []string {
	return []string{"handlebars", "mustache", "nunjucks", "twig"}
}

// WorkflowTriggerEvents returns the supported workflow trigger events from the Nylas API.
func WorkflowTriggerEvents() []string {
	return []string{
		"booking.cancelled",
		"booking.created",
		"booking.pending",
		"booking.reminder",
		"booking.rescheduled",
	}
}

// CursorListParams configures cursor-based list requests.
type CursorListParams struct {
	Limit     int    `json:"limit,omitempty"`
	PageToken string `json:"page_token,omitempty"`
}

// RemoteTemplate represents a Nylas-hosted template.
type RemoteTemplate struct {
	ID        string    `json:"id"`
	GrantID   string    `json:"grant_id,omitempty"`
	AppID     *string   `json:"app_id,omitempty"`
	Engine    string    `json:"engine"`
	Name      string    `json:"name"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	Object    string    `json:"object,omitempty"`
}

// QuietField returns the field used by quiet output mode.
func (t RemoteTemplate) QuietField() string {
	return t.ID
}

// RemoteTemplateListResponse contains a page of hosted templates.
type RemoteTemplateListResponse struct {
	Data       []RemoteTemplate `json:"data"`
	NextCursor string           `json:"next_cursor,omitempty"`
	RequestID  string           `json:"request_id,omitempty"`
}

// CreateRemoteTemplateRequest creates a hosted template.
type CreateRemoteTemplateRequest struct {
	Body    string `json:"body"`
	Engine  string `json:"engine,omitempty"`
	Name    string `json:"name"`
	Subject string `json:"subject"`
}

// UpdateRemoteTemplateRequest updates a hosted template.
type UpdateRemoteTemplateRequest struct {
	Body    *string `json:"body,omitempty"`
	Engine  *string `json:"engine,omitempty"`
	Name    *string `json:"name,omitempty"`
	Subject *string `json:"subject,omitempty"`
}

// TemplateRenderRequest renders a stored template with variables.
type TemplateRenderRequest struct {
	Strict    *bool          `json:"strict,omitempty"`
	Variables map[string]any `json:"variables,omitempty"`
}

// TemplateRenderHTMLRequest renders arbitrary template HTML with variables.
type TemplateRenderHTMLRequest struct {
	Body      string         `json:"body"`
	Engine    string         `json:"engine"`
	Strict    *bool          `json:"strict,omitempty"`
	Variables map[string]any `json:"variables,omitempty"`
}

// TemplateRenderResult contains the raw render output from the Nylas API.
type TemplateRenderResult map[string]any

// WorkflowSender configures the transactional sender for a workflow.
type WorkflowSender struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// RemoteWorkflow represents a Nylas-hosted workflow.
type RemoteWorkflow struct {
	ID           string          `json:"id"`
	GrantID      string          `json:"grant_id,omitempty"`
	AppID        *string         `json:"app_id,omitempty"`
	IsEnabled    bool            `json:"is_enabled"`
	Name         string          `json:"name"`
	TriggerEvent string          `json:"trigger_event"`
	Delay        int             `json:"delay"`
	TemplateID   string          `json:"template_id"`
	From         *WorkflowSender `json:"from,omitempty"`
	DateCreated  time.Time       `json:"date_created,omitempty"`
	Object       string          `json:"object,omitempty"`
}

// QuietField returns the field used by quiet output mode.
func (w RemoteWorkflow) QuietField() string {
	return w.ID
}

// RemoteWorkflowListResponse contains a page of hosted workflows.
type RemoteWorkflowListResponse struct {
	Data       []RemoteWorkflow `json:"data"`
	NextCursor string           `json:"next_cursor,omitempty"`
	RequestID  string           `json:"request_id,omitempty"`
}

// CreateRemoteWorkflowRequest creates a hosted workflow.
type CreateRemoteWorkflowRequest struct {
	Delay        int             `json:"delay,omitempty"`
	IsEnabled    *bool           `json:"is_enabled,omitempty"`
	Name         string          `json:"name"`
	TemplateID   string          `json:"template_id"`
	TriggerEvent string          `json:"trigger_event"`
	From         *WorkflowSender `json:"from,omitempty"`
}

// UpdateRemoteWorkflowRequest updates a hosted workflow.
type UpdateRemoteWorkflowRequest struct {
	Delay        *int            `json:"delay,omitempty"`
	IsEnabled    *bool           `json:"is_enabled,omitempty"`
	Name         *string         `json:"name,omitempty"`
	TemplateID   *string         `json:"template_id,omitempty"`
	TriggerEvent *string         `json:"trigger_event,omitempty"`
	From         *WorkflowSender `json:"from,omitempty"`
}
