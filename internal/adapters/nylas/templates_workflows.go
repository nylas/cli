package nylas

import (
	"context"
	"fmt"
	"net/url"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/util"
)

type remoteTemplateResponse struct {
	ID        string  `json:"id"`
	GrantID   string  `json:"grant_id"`
	AppID     *string `json:"app_id"`
	Engine    string  `json:"engine"`
	Name      string  `json:"name"`
	Subject   string  `json:"subject"`
	Body      string  `json:"body"`
	CreatedAt int64   `json:"created_at"`
	UpdatedAt int64   `json:"updated_at"`
	Object    string  `json:"object"`
}

type workflowSenderResponse struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type remoteWorkflowResponse struct {
	ID           string                  `json:"id"`
	GrantID      string                  `json:"grant_id"`
	AppID        *string                 `json:"app_id"`
	IsEnabled    bool                    `json:"is_enabled"`
	Name         string                  `json:"name"`
	TriggerEvent string                  `json:"trigger_event"`
	Delay        int                     `json:"delay"`
	TemplateID   string                  `json:"template_id"`
	From         *workflowSenderResponse `json:"from"`
	DateCreated  int64                   `json:"date_created"`
	Object       string                  `json:"object"`
}

func (c *HTTPClient) ListRemoteTemplates(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID string,
	params *domain.CursorListParams,
) (*domain.RemoteTemplateListResponse, error) {
	baseURL, err := c.templatesBaseURL(scope, grantID)
	if err != nil {
		return nil, err
	}

	params = normalizeCursorListParams(params)
	queryURL := NewQueryBuilder().
		AddInt("limit", params.Limit).
		Add("page_token", params.PageToken).
		BuildURL(baseURL)

	var result struct {
		Data       []remoteTemplateResponse `json:"data"`
		NextCursor string                   `json:"next_cursor,omitempty"`
		RequestID  string                   `json:"request_id,omitempty"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}

	return &domain.RemoteTemplateListResponse{
		Data:       util.Map(result.Data, convertRemoteTemplate),
		NextCursor: result.NextCursor,
		RequestID:  result.RequestID,
	}, nil
}

func (c *HTTPClient) GetRemoteTemplate(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, templateID string,
) (*domain.RemoteTemplate, error) {
	if err := validateRequired("template ID", templateID); err != nil {
		return nil, err
	}

	queryURL, err := c.templateURL(scope, grantID, templateID)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data remoteTemplateResponse `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrTemplateNotFound); err != nil {
		return nil, err
	}

	template := convertRemoteTemplate(result.Data)
	return &template, nil
}

func (c *HTTPClient) CreateRemoteTemplate(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID string,
	req *domain.CreateRemoteTemplateRequest,
) (*domain.RemoteTemplate, error) {
	queryURL, err := c.templatesBaseURL(scope, grantID)
	if err != nil {
		return nil, err
	}

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data remoteTemplateResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	template := convertRemoteTemplate(result.Data)
	return &template, nil
}

func (c *HTTPClient) UpdateRemoteTemplate(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, templateID string,
	req *domain.UpdateRemoteTemplateRequest,
) (*domain.RemoteTemplate, error) {
	if err := validateRequired("template ID", templateID); err != nil {
		return nil, err
	}

	queryURL, err := c.templateURL(scope, grantID, templateID)
	if err != nil {
		return nil, err
	}

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data remoteTemplateResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	template := convertRemoteTemplate(result.Data)
	return &template, nil
}

func (c *HTTPClient) DeleteRemoteTemplate(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, templateID string,
) error {
	if err := validateRequired("template ID", templateID); err != nil {
		return err
	}

	queryURL, err := c.templateURL(scope, grantID, templateID)
	if err != nil {
		return err
	}

	return c.doDelete(ctx, queryURL)
}

func (c *HTTPClient) RenderRemoteTemplate(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, templateID string,
	req *domain.TemplateRenderRequest,
) (domain.TemplateRenderResult, error) {
	if err := validateRequired("template ID", templateID); err != nil {
		return nil, err
	}

	queryURL, err := c.templateURL(scope, grantID, templateID)
	if err != nil {
		return nil, err
	}
	queryURL += "/render"

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.TemplateRenderResult `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *HTTPClient) RenderRemoteTemplateHTML(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID string,
	req *domain.TemplateRenderHTMLRequest,
) (domain.TemplateRenderResult, error) {
	baseURL, err := c.templatesBaseURL(scope, grantID)
	if err != nil {
		return nil, err
	}

	resp, err := c.doJSONRequest(ctx, "POST", baseURL+"/render", req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.TemplateRenderResult `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *HTTPClient) ListWorkflows(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID string,
	params *domain.CursorListParams,
) (*domain.RemoteWorkflowListResponse, error) {
	baseURL, err := c.workflowsBaseURL(scope, grantID)
	if err != nil {
		return nil, err
	}

	params = normalizeCursorListParams(params)
	queryURL := NewQueryBuilder().
		AddInt("limit", params.Limit).
		Add("page_token", params.PageToken).
		BuildURL(baseURL)

	var result struct {
		Data       []remoteWorkflowResponse `json:"data"`
		NextCursor string                   `json:"next_cursor,omitempty"`
		RequestID  string                   `json:"request_id,omitempty"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}

	return &domain.RemoteWorkflowListResponse{
		Data:       util.Map(result.Data, convertRemoteWorkflow),
		NextCursor: result.NextCursor,
		RequestID:  result.RequestID,
	}, nil
}

func (c *HTTPClient) GetWorkflow(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, workflowID string,
) (*domain.RemoteWorkflow, error) {
	if err := validateRequired("workflow ID", workflowID); err != nil {
		return nil, err
	}

	queryURL, err := c.workflowURL(scope, grantID, workflowID)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data remoteWorkflowResponse `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrWorkflowNotFound); err != nil {
		return nil, err
	}

	workflow := convertRemoteWorkflow(result.Data)
	return &workflow, nil
}

func (c *HTTPClient) CreateWorkflow(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID string,
	req *domain.CreateRemoteWorkflowRequest,
) (*domain.RemoteWorkflow, error) {
	queryURL, err := c.workflowsBaseURL(scope, grantID)
	if err != nil {
		return nil, err
	}

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data remoteWorkflowResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	workflow := convertRemoteWorkflow(result.Data)
	return &workflow, nil
}

func (c *HTTPClient) UpdateWorkflow(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, workflowID string,
	req *domain.UpdateRemoteWorkflowRequest,
) (*domain.RemoteWorkflow, error) {
	if err := validateRequired("workflow ID", workflowID); err != nil {
		return nil, err
	}

	queryURL, err := c.workflowURL(scope, grantID, workflowID)
	if err != nil {
		return nil, err
	}

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data remoteWorkflowResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	workflow := convertRemoteWorkflow(result.Data)
	return &workflow, nil
}

func (c *HTTPClient) DeleteWorkflow(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, workflowID string,
) error {
	if err := validateRequired("workflow ID", workflowID); err != nil {
		return err
	}

	queryURL, err := c.workflowURL(scope, grantID, workflowID)
	if err != nil {
		return err
	}

	return c.doDelete(ctx, queryURL)
}

func (c *HTTPClient) templatesBaseURL(scope domain.RemoteScope, grantID string) (string, error) {
	switch scope {
	case domain.ScopeApplication:
		return fmt.Sprintf("%s/v3/templates", c.baseURL), nil
	case domain.ScopeGrant:
		if err := validateRequired("grant ID", grantID); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s/v3/grants/%s/templates", c.baseURL, url.PathEscape(grantID)), nil
	default:
		return "", fmt.Errorf("%w: invalid scope %q", domain.ErrInvalidInput, scope)
	}
}

func (c *HTTPClient) templateURL(scope domain.RemoteScope, grantID, templateID string) (string, error) {
	baseURL, err := c.templatesBaseURL(scope, grantID)
	if err != nil {
		return "", err
	}
	return baseURL + "/" + url.PathEscape(templateID), nil
}

func (c *HTTPClient) workflowsBaseURL(scope domain.RemoteScope, grantID string) (string, error) {
	switch scope {
	case domain.ScopeApplication:
		return fmt.Sprintf("%s/v3/workflows", c.baseURL), nil
	case domain.ScopeGrant:
		if err := validateRequired("grant ID", grantID); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s/v3/grants/%s/workflows", c.baseURL, url.PathEscape(grantID)), nil
	default:
		return "", fmt.Errorf("%w: invalid scope %q", domain.ErrInvalidInput, scope)
	}
}

func (c *HTTPClient) workflowURL(scope domain.RemoteScope, grantID, workflowID string) (string, error) {
	baseURL, err := c.workflowsBaseURL(scope, grantID)
	if err != nil {
		return "", err
	}
	return baseURL + "/" + url.PathEscape(workflowID), nil
}

func normalizeCursorListParams(params *domain.CursorListParams) *domain.CursorListParams {
	if params == nil {
		return &domain.CursorListParams{Limit: 50}
	}
	if params.Limit <= 0 {
		params.Limit = 50
	}
	return params
}

func convertRemoteTemplate(t remoteTemplateResponse) domain.RemoteTemplate {
	return domain.RemoteTemplate{
		ID:        t.ID,
		GrantID:   t.GrantID,
		AppID:     t.AppID,
		Engine:    t.Engine,
		Name:      t.Name,
		Subject:   t.Subject,
		Body:      t.Body,
		CreatedAt: unixToTime(t.CreatedAt),
		UpdatedAt: unixToTime(t.UpdatedAt),
		Object:    t.Object,
	}
}

func convertRemoteWorkflow(w remoteWorkflowResponse) domain.RemoteWorkflow {
	var from *domain.WorkflowSender
	if w.From != nil {
		from = &domain.WorkflowSender{
			Name:  w.From.Name,
			Email: w.From.Email,
		}
	}

	return domain.RemoteWorkflow{
		ID:           w.ID,
		GrantID:      w.GrantID,
		AppID:        w.AppID,
		IsEnabled:    w.IsEnabled,
		Name:         w.Name,
		TriggerEvent: w.TriggerEvent,
		Delay:        w.Delay,
		TemplateID:   w.TemplateID,
		From:         from,
		DateCreated:  unixToTime(w.DateCreated),
		Object:       w.Object,
	}
}
