package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (d *DemoClient) ListRemoteTemplates(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID string,
	params *domain.CursorListParams,
) (*domain.RemoteTemplateListResponse, error) {
	return &domain.RemoteTemplateListResponse{
		Data: []domain.RemoteTemplate{
			{
				ID:      "demo-template-001",
				Engine:  "mustache",
				Name:    "Demo Booking Confirmation",
				Subject: "Booking confirmed for {{user.name}}",
				Body:    "<p>Hello {{user.name}}, your demo booking is confirmed.</p>",
			},
		},
	}, nil
}

func (d *DemoClient) GetRemoteTemplate(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, templateID string,
) (*domain.RemoteTemplate, error) {
	return &domain.RemoteTemplate{
		ID:      templateID,
		Engine:  "mustache",
		Name:    "Demo Booking Confirmation",
		Subject: "Booking confirmed for {{user.name}}",
		Body:    "<p>Hello {{user.name}}, your demo booking is confirmed.</p>",
	}, nil
}

func (d *DemoClient) CreateRemoteTemplate(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID string,
	req *domain.CreateRemoteTemplateRequest,
) (*domain.RemoteTemplate, error) {
	return &domain.RemoteTemplate{
		ID:      "demo-template-new",
		Engine:  req.Engine,
		Name:    req.Name,
		Subject: req.Subject,
		Body:    req.Body,
	}, nil
}

func (d *DemoClient) UpdateRemoteTemplate(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, templateID string,
	req *domain.UpdateRemoteTemplateRequest,
) (*domain.RemoteTemplate, error) {
	return &domain.RemoteTemplate{ID: templateID, Name: "Updated Demo Template"}, nil
}

func (d *DemoClient) DeleteRemoteTemplate(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, templateID string,
) error {
	return nil
}

func (d *DemoClient) RenderRemoteTemplate(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, templateID string,
	req *domain.TemplateRenderRequest,
) (domain.TemplateRenderResult, error) {
	return domain.TemplateRenderResult{
		"subject": "Demo booking confirmation",
		"body":    "<p>Hello Demo User</p>",
	}, nil
}

func (d *DemoClient) RenderRemoteTemplateHTML(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID string,
	req *domain.TemplateRenderHTMLRequest,
) (domain.TemplateRenderResult, error) {
	return domain.TemplateRenderResult{
		"html": "<p>Hello Demo User</p>",
	}, nil
}

func (d *DemoClient) ListWorkflows(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID string,
	params *domain.CursorListParams,
) (*domain.RemoteWorkflowListResponse, error) {
	return &domain.RemoteWorkflowListResponse{
		Data: []domain.RemoteWorkflow{
			{
				ID:           "demo-workflow-001",
				Name:         "Demo Booking Workflow",
				TriggerEvent: "booking.created",
				TemplateID:   "demo-template-001",
				Delay:        5,
				IsEnabled:    true,
			},
		},
	}, nil
}

func (d *DemoClient) GetWorkflow(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, workflowID string,
) (*domain.RemoteWorkflow, error) {
	return &domain.RemoteWorkflow{
		ID:           workflowID,
		Name:         "Demo Booking Workflow",
		TriggerEvent: "booking.created",
		TemplateID:   "demo-template-001",
		Delay:        5,
		IsEnabled:    true,
	}, nil
}

func (d *DemoClient) CreateWorkflow(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID string,
	req *domain.CreateRemoteWorkflowRequest,
) (*domain.RemoteWorkflow, error) {
	enabled := true
	if req.IsEnabled != nil {
		enabled = *req.IsEnabled
	}
	return &domain.RemoteWorkflow{
		ID:           "demo-workflow-new",
		Name:         req.Name,
		TriggerEvent: req.TriggerEvent,
		TemplateID:   req.TemplateID,
		Delay:        req.Delay,
		IsEnabled:    enabled,
		From:         req.From,
	}, nil
}

func (d *DemoClient) UpdateWorkflow(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, workflowID string,
	req *domain.UpdateRemoteWorkflowRequest,
) (*domain.RemoteWorkflow, error) {
	return &domain.RemoteWorkflow{ID: workflowID, Name: "Updated Demo Workflow"}, nil
}

func (d *DemoClient) DeleteWorkflow(
	ctx context.Context,
	scope domain.RemoteScope,
	grantID, workflowID string,
) error {
	return nil
}
