package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// TemplateWorkflowClient defines the interface for hosted templates and workflows.
type TemplateWorkflowClient interface {
	// Template operations
	ListRemoteTemplates(ctx context.Context, scope domain.RemoteScope, grantID string, params *domain.CursorListParams) (*domain.RemoteTemplateListResponse, error)
	GetRemoteTemplate(ctx context.Context, scope domain.RemoteScope, grantID, templateID string) (*domain.RemoteTemplate, error)
	CreateRemoteTemplate(ctx context.Context, scope domain.RemoteScope, grantID string, req *domain.CreateRemoteTemplateRequest) (*domain.RemoteTemplate, error)
	UpdateRemoteTemplate(ctx context.Context, scope domain.RemoteScope, grantID, templateID string, req *domain.UpdateRemoteTemplateRequest) (*domain.RemoteTemplate, error)
	DeleteRemoteTemplate(ctx context.Context, scope domain.RemoteScope, grantID, templateID string) error
	RenderRemoteTemplate(ctx context.Context, scope domain.RemoteScope, grantID, templateID string, req *domain.TemplateRenderRequest) (domain.TemplateRenderResult, error)
	RenderRemoteTemplateHTML(ctx context.Context, scope domain.RemoteScope, grantID string, req *domain.TemplateRenderHTMLRequest) (domain.TemplateRenderResult, error)

	// Workflow operations
	ListWorkflows(ctx context.Context, scope domain.RemoteScope, grantID string, params *domain.CursorListParams) (*domain.RemoteWorkflowListResponse, error)
	GetWorkflow(ctx context.Context, scope domain.RemoteScope, grantID, workflowID string) (*domain.RemoteWorkflow, error)
	CreateWorkflow(ctx context.Context, scope domain.RemoteScope, grantID string, req *domain.CreateRemoteWorkflowRequest) (*domain.RemoteWorkflow, error)
	UpdateWorkflow(ctx context.Context, scope domain.RemoteScope, grantID, workflowID string, req *domain.UpdateRemoteWorkflowRequest) (*domain.RemoteWorkflow, error)
	DeleteWorkflow(ctx context.Context, scope domain.RemoteScope, grantID, workflowID string) error
}
