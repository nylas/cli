package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) ListRemoteTemplates(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID string,
	params *domain.CursorListParams,
) (*domain.RemoteTemplateListResponse, error) {
	m.ListRemoteTemplatesCalled = true
	m.LastGrantID = grantID
	if m.ListRemoteTemplatesFunc != nil {
		return m.ListRemoteTemplatesFunc(ctx, scope, grantID, params)
	}

	return &domain.RemoteTemplateListResponse{
		Data: []domain.RemoteTemplate{
			{
				ID:      "tpl-1",
				Engine:  "mustache",
				Name:    "Welcome Template",
				Subject: "Welcome {{user.name}}",
				Body:    "<p>Hello {{user.name}}</p>",
			},
		},
	}, nil
}

func (m *MockClient) GetRemoteTemplate(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, templateID string,
) (*domain.RemoteTemplate, error) {
	m.GetRemoteTemplateCalled = true
	m.LastGrantID = grantID
	m.LastTemplateID = templateID
	if m.GetRemoteTemplateFunc != nil {
		return m.GetRemoteTemplateFunc(ctx, scope, grantID, templateID)
	}

	return &domain.RemoteTemplate{
		ID:      templateID,
		Engine:  "mustache",
		Name:    "Welcome Template",
		Subject: "Welcome {{user.name}}",
		Body:    "<p>Hello {{user.name}}</p>",
	}, nil
}

func (m *MockClient) CreateRemoteTemplate(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID string,
	req *domain.CreateRemoteTemplateRequest,
) (*domain.RemoteTemplate, error) {
	m.CreateRemoteTemplateCalled = true
	m.LastGrantID = grantID
	if m.CreateRemoteTemplateFunc != nil {
		return m.CreateRemoteTemplateFunc(ctx, scope, grantID, req)
	}

	return &domain.RemoteTemplate{
		ID:      "tpl-new",
		Engine:  req.Engine,
		Name:    req.Name,
		Subject: req.Subject,
		Body:    req.Body,
	}, nil
}

func (m *MockClient) UpdateRemoteTemplate(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, templateID string,
	req *domain.UpdateRemoteTemplateRequest,
) (*domain.RemoteTemplate, error) {
	m.UpdateRemoteTemplateCalled = true
	m.LastGrantID = grantID
	m.LastTemplateID = templateID
	if m.UpdateRemoteTemplateFunc != nil {
		return m.UpdateRemoteTemplateFunc(ctx, scope, grantID, templateID, req)
	}

	template := &domain.RemoteTemplate{ID: templateID, Engine: "mustache", Name: "Updated Template"}
	if req.Name != nil {
		template.Name = *req.Name
	}
	if req.Subject != nil {
		template.Subject = *req.Subject
	}
	if req.Body != nil {
		template.Body = *req.Body
	}
	if req.Engine != nil {
		template.Engine = *req.Engine
	}
	return template, nil
}

func (m *MockClient) DeleteRemoteTemplate(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, templateID string,
) error {
	m.DeleteRemoteTemplateCalled = true
	m.LastGrantID = grantID
	m.LastTemplateID = templateID
	if m.DeleteRemoteTemplateFunc != nil {
		return m.DeleteRemoteTemplateFunc(ctx, scope, grantID, templateID)
	}
	return nil
}

func (m *MockClient) RenderRemoteTemplate(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, templateID string,
	req *domain.TemplateRenderRequest,
) (domain.TemplateRenderResult, error) {
	m.RenderRemoteTemplateCalled = true
	m.LastGrantID = grantID
	m.LastTemplateID = templateID
	if m.RenderRemoteTemplateFunc != nil {
		return m.RenderRemoteTemplateFunc(ctx, scope, grantID, templateID, req)
	}

	return domain.TemplateRenderResult{
		"subject": "Welcome Nylas",
		"body":    "<p>Hello Nylas</p>",
	}, nil
}

func (m *MockClient) RenderRemoteTemplateHTML(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID string,
	req *domain.TemplateRenderHTMLRequest,
) (domain.TemplateRenderResult, error) {
	m.RenderTemplateHTMLCalled = true
	m.LastGrantID = grantID
	if m.RenderTemplateHTMLFunc != nil {
		return m.RenderTemplateHTMLFunc(ctx, scope, grantID, req)
	}

	return domain.TemplateRenderResult{
		"html": "<p>Hello Nylas</p>",
	}, nil
}

func (m *MockClient) ListWorkflows(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID string,
	params *domain.CursorListParams,
) (*domain.RemoteWorkflowListResponse, error) {
	m.ListWorkflowsCalled = true
	m.LastGrantID = grantID
	if m.ListWorkflowsFunc != nil {
		return m.ListWorkflowsFunc(ctx, scope, grantID, params)
	}

	return &domain.RemoteWorkflowListResponse{
		Data: []domain.RemoteWorkflow{
			{
				ID:           "wf-1",
				Name:         "Booking Confirmation",
				TemplateID:   "tpl-1",
				TriggerEvent: "booking.created",
				Delay:        1,
				IsEnabled:    true,
			},
		},
	}, nil
}

func (m *MockClient) GetWorkflow(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, workflowID string,
) (*domain.RemoteWorkflow, error) {
	m.GetWorkflowCalled = true
	m.LastGrantID = grantID
	m.LastWorkflowID = workflowID
	if m.GetWorkflowFunc != nil {
		return m.GetWorkflowFunc(ctx, scope, grantID, workflowID)
	}

	return &domain.RemoteWorkflow{
		ID:           workflowID,
		Name:         "Booking Confirmation",
		TemplateID:   "tpl-1",
		TriggerEvent: "booking.created",
		Delay:        1,
		IsEnabled:    true,
	}, nil
}

func (m *MockClient) CreateWorkflow(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID string,
	req *domain.CreateRemoteWorkflowRequest,
) (*domain.RemoteWorkflow, error) {
	m.CreateWorkflowCalled = true
	m.LastGrantID = grantID
	if m.CreateWorkflowFunc != nil {
		return m.CreateWorkflowFunc(ctx, scope, grantID, req)
	}

	enabled := true
	if req.IsEnabled != nil {
		enabled = *req.IsEnabled
	}

	return &domain.RemoteWorkflow{
		ID:           "wf-new",
		Name:         req.Name,
		TemplateID:   req.TemplateID,
		TriggerEvent: req.TriggerEvent,
		Delay:        req.Delay,
		IsEnabled:    enabled,
		From:         req.From,
	}, nil
}

func (m *MockClient) UpdateWorkflow(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, workflowID string,
	req *domain.UpdateRemoteWorkflowRequest,
) (*domain.RemoteWorkflow, error) {
	m.UpdateWorkflowCalled = true
	m.LastGrantID = grantID
	m.LastWorkflowID = workflowID
	if m.UpdateWorkflowFunc != nil {
		return m.UpdateWorkflowFunc(ctx, scope, grantID, workflowID, req)
	}

	workflow := &domain.RemoteWorkflow{ID: workflowID, Name: "Updated Workflow"}
	if req.Name != nil {
		workflow.Name = *req.Name
	}
	if req.TemplateID != nil {
		workflow.TemplateID = *req.TemplateID
	}
	if req.TriggerEvent != nil {
		workflow.TriggerEvent = *req.TriggerEvent
	}
	if req.Delay != nil {
		workflow.Delay = *req.Delay
	}
	if req.IsEnabled != nil {
		workflow.IsEnabled = *req.IsEnabled
	}
	if req.From != nil {
		workflow.From = req.From
	}
	return workflow, nil
}

func (m *MockClient) DeleteWorkflow(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, workflowID string,
) error {
	m.DeleteWorkflowCalled = true
	m.LastGrantID = grantID
	m.LastWorkflowID = workflowID
	if m.DeleteWorkflowFunc != nil {
		return m.DeleteWorkflowFunc(ctx, scope, grantID, workflowID)
	}
	return nil
}
