package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type templateWorkflowListParams struct {
	Scope   string `json:"scope,omitempty"`
	GrantID string `json:"grant_id,omitempty"`
	domain.CursorListParams
}

type templateListResult struct {
	Templates  []domain.RemoteTemplate `json:"templates"`
	NextCursor string                  `json:"next_cursor,omitempty"`
}

type templateGetParams struct {
	Scope      string `json:"scope,omitempty"`
	GrantID    string `json:"grant_id,omitempty"`
	TemplateID string `json:"template_id"`
}

type templateCreateParams struct {
	Scope   string `json:"scope,omitempty"`
	GrantID string `json:"grant_id,omitempty"`
	domain.CreateRemoteTemplateRequest
}

type templateUpdateParams struct {
	Scope      string `json:"scope,omitempty"`
	GrantID    string `json:"grant_id,omitempty"`
	TemplateID string `json:"template_id"`
	domain.UpdateRemoteTemplateRequest
}

type templateDeleteParams struct {
	Scope      string `json:"scope,omitempty"`
	GrantID    string `json:"grant_id,omitempty"`
	TemplateID string `json:"template_id"`
}

type templateRenderParams struct {
	Scope      string `json:"scope,omitempty"`
	GrantID    string `json:"grant_id,omitempty"`
	TemplateID string `json:"template_id"`
	domain.TemplateRenderRequest
}

type templateRenderHTMLParams struct {
	Scope   string `json:"scope,omitempty"`
	GrantID string `json:"grant_id,omitempty"`
	domain.TemplateRenderHTMLRequest
}

type workflowListResult struct {
	Workflows  []domain.RemoteWorkflow `json:"workflows"`
	NextCursor string                  `json:"next_cursor,omitempty"`
}

type workflowGetParams struct {
	Scope      string `json:"scope,omitempty"`
	GrantID    string `json:"grant_id,omitempty"`
	WorkflowID string `json:"workflow_id"`
}

type workflowCreateParams struct {
	Scope   string `json:"scope,omitempty"`
	GrantID string `json:"grant_id,omitempty"`
	domain.CreateRemoteWorkflowRequest
}

type workflowUpdateParams struct {
	Scope      string `json:"scope,omitempty"`
	GrantID    string `json:"grant_id,omitempty"`
	WorkflowID string `json:"workflow_id"`
	domain.UpdateRemoteWorkflowRequest
}

type workflowDeleteParams struct {
	Scope      string `json:"scope,omitempty"`
	GrantID    string `json:"grant_id,omitempty"`
	WorkflowID string `json:"workflow_id"`
}

func RegisterTemplateWorkflowHandlers(d *Dispatcher, client ports.TemplateWorkflowClient, defaultGrant string) {
	d.Register("template.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p templateWorkflowListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		scope, err := parseRPCRemoteScope(p.Scope)
		if err != nil {
			return nil, err
		}
		grantID := p.GrantID
		if scope == domain.ScopeGrant {
			grantID, err = resolveGrant(p.GrantID, defaultGrant)
			if err != nil {
				return nil, err
			}
		}

		resp, err := client.ListRemoteTemplates(ctx, scope, grantID, &p.CursorListParams)
		if err != nil {
			return nil, fmt.Errorf("template.list: %w", err)
		}
		return templateListResult{Templates: resp.Data, NextCursor: resp.NextCursor}, nil
	})

	d.Register("template.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p templateGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.TemplateID == "" {
			return nil, NewRPCError(InvalidParams, "template_id required", nil)
		}

		scope, err := parseRPCRemoteScope(p.Scope)
		if err != nil {
			return nil, err
		}
		grantID := p.GrantID
		if scope == domain.ScopeGrant {
			grantID, err = resolveGrant(p.GrantID, defaultGrant)
			if err != nil {
				return nil, err
			}
		}

		template, err := client.GetRemoteTemplate(ctx, scope, grantID, p.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("template.get: %w", err)
		}
		return template, nil
	})

	d.Register("template.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p templateCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		scope, err := parseRPCRemoteScope(p.Scope)
		if err != nil {
			return nil, err
		}
		grantID := p.GrantID
		if scope == domain.ScopeGrant {
			grantID, err = resolveGrant(p.GrantID, defaultGrant)
			if err != nil {
				return nil, err
			}
		}

		template, err := client.CreateRemoteTemplate(ctx, scope, grantID, &p.CreateRemoteTemplateRequest)
		if err != nil {
			return nil, fmt.Errorf("template.create: %w", err)
		}
		return template, nil
	})

	d.Register("template.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p templateUpdateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.TemplateID == "" {
			return nil, NewRPCError(InvalidParams, "template_id required", nil)
		}

		scope, err := parseRPCRemoteScope(p.Scope)
		if err != nil {
			return nil, err
		}
		grantID := p.GrantID
		if scope == domain.ScopeGrant {
			grantID, err = resolveGrant(p.GrantID, defaultGrant)
			if err != nil {
				return nil, err
			}
		}

		template, err := client.UpdateRemoteTemplate(ctx, scope, grantID, p.TemplateID, &p.UpdateRemoteTemplateRequest)
		if err != nil {
			return nil, fmt.Errorf("template.update: %w", err)
		}
		return template, nil
	})

	d.Register("template.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p templateDeleteParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.TemplateID == "" {
			return nil, NewRPCError(InvalidParams, "template_id required", nil)
		}

		scope, err := parseRPCRemoteScope(p.Scope)
		if err != nil {
			return nil, err
		}
		grantID := p.GrantID
		if scope == domain.ScopeGrant {
			grantID, err = resolveGrant(p.GrantID, defaultGrant)
			if err != nil {
				return nil, err
			}
		}

		if err := client.DeleteRemoteTemplate(ctx, scope, grantID, p.TemplateID); err != nil {
			return nil, fmt.Errorf("template.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})

	d.Register("template.render", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p templateRenderParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.TemplateID == "" {
			return nil, NewRPCError(InvalidParams, "template_id required", nil)
		}

		scope, err := parseRPCRemoteScope(p.Scope)
		if err != nil {
			return nil, err
		}
		grantID := p.GrantID
		if scope == domain.ScopeGrant {
			grantID, err = resolveGrant(p.GrantID, defaultGrant)
			if err != nil {
				return nil, err
			}
		}

		result, err := client.RenderRemoteTemplate(ctx, scope, grantID, p.TemplateID, &p.TemplateRenderRequest)
		if err != nil {
			return nil, fmt.Errorf("template.render: %w", err)
		}
		return result, nil
	})

	d.Register("template.renderHTML", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p templateRenderHTMLParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		scope, err := parseRPCRemoteScope(p.Scope)
		if err != nil {
			return nil, err
		}
		grantID := p.GrantID
		if scope == domain.ScopeGrant {
			grantID, err = resolveGrant(p.GrantID, defaultGrant)
			if err != nil {
				return nil, err
			}
		}

		result, err := client.RenderRemoteTemplateHTML(ctx, scope, grantID, &p.TemplateRenderHTMLRequest)
		if err != nil {
			return nil, fmt.Errorf("template.renderHTML: %w", err)
		}
		return result, nil
	})

	d.Register("workflow.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p templateWorkflowListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		scope, err := parseRPCRemoteScope(p.Scope)
		if err != nil {
			return nil, err
		}
		grantID := p.GrantID
		if scope == domain.ScopeGrant {
			grantID, err = resolveGrant(p.GrantID, defaultGrant)
			if err != nil {
				return nil, err
			}
		}

		resp, err := client.ListWorkflows(ctx, scope, grantID, &p.CursorListParams)
		if err != nil {
			return nil, fmt.Errorf("workflow.list: %w", err)
		}
		return workflowListResult{Workflows: resp.Data, NextCursor: resp.NextCursor}, nil
	})

	d.Register("workflow.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p workflowGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.WorkflowID == "" {
			return nil, NewRPCError(InvalidParams, "workflow_id required", nil)
		}

		scope, err := parseRPCRemoteScope(p.Scope)
		if err != nil {
			return nil, err
		}
		grantID := p.GrantID
		if scope == domain.ScopeGrant {
			grantID, err = resolveGrant(p.GrantID, defaultGrant)
			if err != nil {
				return nil, err
			}
		}

		workflow, err := client.GetWorkflow(ctx, scope, grantID, p.WorkflowID)
		if err != nil {
			return nil, fmt.Errorf("workflow.get: %w", err)
		}
		return workflow, nil
	})

	d.Register("workflow.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p workflowCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		scope, err := parseRPCRemoteScope(p.Scope)
		if err != nil {
			return nil, err
		}
		grantID := p.GrantID
		if scope == domain.ScopeGrant {
			grantID, err = resolveGrant(p.GrantID, defaultGrant)
			if err != nil {
				return nil, err
			}
		}

		workflow, err := client.CreateWorkflow(ctx, scope, grantID, &p.CreateRemoteWorkflowRequest)
		if err != nil {
			return nil, fmt.Errorf("workflow.create: %w", err)
		}
		return workflow, nil
	})

	d.Register("workflow.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p workflowUpdateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.WorkflowID == "" {
			return nil, NewRPCError(InvalidParams, "workflow_id required", nil)
		}

		scope, err := parseRPCRemoteScope(p.Scope)
		if err != nil {
			return nil, err
		}
		grantID := p.GrantID
		if scope == domain.ScopeGrant {
			grantID, err = resolveGrant(p.GrantID, defaultGrant)
			if err != nil {
				return nil, err
			}
		}

		workflow, err := client.UpdateWorkflow(ctx, scope, grantID, p.WorkflowID, &p.UpdateRemoteWorkflowRequest)
		if err != nil {
			return nil, fmt.Errorf("workflow.update: %w", err)
		}
		return workflow, nil
	})

	d.Register("workflow.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p workflowDeleteParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.WorkflowID == "" {
			return nil, NewRPCError(InvalidParams, "workflow_id required", nil)
		}

		scope, err := parseRPCRemoteScope(p.Scope)
		if err != nil {
			return nil, err
		}
		grantID := p.GrantID
		if scope == domain.ScopeGrant {
			grantID, err = resolveGrant(p.GrantID, defaultGrant)
			if err != nil {
				return nil, err
			}
		}

		if err := client.DeleteWorkflow(ctx, scope, grantID, p.WorkflowID); err != nil {
			return nil, fmt.Errorf("workflow.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})
}

func parseRPCRemoteScope(scope string) (domain.RemoteScope, error) {
	if scope == "" {
		return domain.ScopeApplication, nil
	}
	parsed, err := domain.ParseRemoteScope(scope)
	if err != nil {
		return "", NewRPCError(InvalidParams, "invalid scope", nil)
	}
	return parsed, nil
}
