package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeTemplateWorkflowClient struct {
	ports.TemplateWorkflowClient

	listRemoteTemplates      func(context.Context, domain.RemoteScope, string, *domain.CursorListParams) (*domain.RemoteTemplateListResponse, error)
	getRemoteTemplate        func(context.Context, domain.RemoteScope, string, string) (*domain.RemoteTemplate, error)
	createRemoteTemplate     func(context.Context, domain.RemoteScope, string, *domain.CreateRemoteTemplateRequest) (*domain.RemoteTemplate, error)
	updateRemoteTemplate     func(context.Context, domain.RemoteScope, string, string, *domain.UpdateRemoteTemplateRequest) (*domain.RemoteTemplate, error)
	deleteRemoteTemplate     func(context.Context, domain.RemoteScope, string, string) error
	renderRemoteTemplate     func(context.Context, domain.RemoteScope, string, string, *domain.TemplateRenderRequest) (domain.TemplateRenderResult, error)
	renderRemoteTemplateHTML func(context.Context, domain.RemoteScope, string, *domain.TemplateRenderHTMLRequest) (domain.TemplateRenderResult, error)
	listWorkflows            func(context.Context, domain.RemoteScope, string, *domain.CursorListParams) (*domain.RemoteWorkflowListResponse, error)
	getWorkflow              func(context.Context, domain.RemoteScope, string, string) (*domain.RemoteWorkflow, error)
	createWorkflow           func(context.Context, domain.RemoteScope, string, *domain.CreateRemoteWorkflowRequest) (*domain.RemoteWorkflow, error)
	updateWorkflow           func(context.Context, domain.RemoteScope, string, string, *domain.UpdateRemoteWorkflowRequest) (*domain.RemoteWorkflow, error)
	deleteWorkflow           func(context.Context, domain.RemoteScope, string, string) error
}

func (f *fakeTemplateWorkflowClient) ListRemoteTemplates(ctx context.Context, scope domain.RemoteScope, grantID string, params *domain.CursorListParams) (*domain.RemoteTemplateListResponse, error) {
	if f.listRemoteTemplates == nil {
		return nil, errors.New("unexpected ListRemoteTemplates")
	}
	return f.listRemoteTemplates(ctx, scope, grantID, params)
}

func (f *fakeTemplateWorkflowClient) GetRemoteTemplate(ctx context.Context, scope domain.RemoteScope, grantID, templateID string) (*domain.RemoteTemplate, error) {
	if f.getRemoteTemplate == nil {
		return nil, errors.New("unexpected GetRemoteTemplate")
	}
	return f.getRemoteTemplate(ctx, scope, grantID, templateID)
}

func (f *fakeTemplateWorkflowClient) CreateRemoteTemplate(ctx context.Context, scope domain.RemoteScope, grantID string, req *domain.CreateRemoteTemplateRequest) (*domain.RemoteTemplate, error) {
	if f.createRemoteTemplate == nil {
		return nil, errors.New("unexpected CreateRemoteTemplate")
	}
	return f.createRemoteTemplate(ctx, scope, grantID, req)
}

func (f *fakeTemplateWorkflowClient) UpdateRemoteTemplate(ctx context.Context, scope domain.RemoteScope, grantID, templateID string, req *domain.UpdateRemoteTemplateRequest) (*domain.RemoteTemplate, error) {
	if f.updateRemoteTemplate == nil {
		return nil, errors.New("unexpected UpdateRemoteTemplate")
	}
	return f.updateRemoteTemplate(ctx, scope, grantID, templateID, req)
}

func (f *fakeTemplateWorkflowClient) DeleteRemoteTemplate(ctx context.Context, scope domain.RemoteScope, grantID, templateID string) error {
	if f.deleteRemoteTemplate == nil {
		return errors.New("unexpected DeleteRemoteTemplate")
	}
	return f.deleteRemoteTemplate(ctx, scope, grantID, templateID)
}

func (f *fakeTemplateWorkflowClient) RenderRemoteTemplate(ctx context.Context, scope domain.RemoteScope, grantID, templateID string, req *domain.TemplateRenderRequest) (domain.TemplateRenderResult, error) {
	if f.renderRemoteTemplate == nil {
		return nil, errors.New("unexpected RenderRemoteTemplate")
	}
	return f.renderRemoteTemplate(ctx, scope, grantID, templateID, req)
}

func (f *fakeTemplateWorkflowClient) RenderRemoteTemplateHTML(ctx context.Context, scope domain.RemoteScope, grantID string, req *domain.TemplateRenderHTMLRequest) (domain.TemplateRenderResult, error) {
	if f.renderRemoteTemplateHTML == nil {
		return nil, errors.New("unexpected RenderRemoteTemplateHTML")
	}
	return f.renderRemoteTemplateHTML(ctx, scope, grantID, req)
}

func (f *fakeTemplateWorkflowClient) ListWorkflows(ctx context.Context, scope domain.RemoteScope, grantID string, params *domain.CursorListParams) (*domain.RemoteWorkflowListResponse, error) {
	if f.listWorkflows == nil {
		return nil, errors.New("unexpected ListWorkflows")
	}
	return f.listWorkflows(ctx, scope, grantID, params)
}

func (f *fakeTemplateWorkflowClient) GetWorkflow(ctx context.Context, scope domain.RemoteScope, grantID, workflowID string) (*domain.RemoteWorkflow, error) {
	if f.getWorkflow == nil {
		return nil, errors.New("unexpected GetWorkflow")
	}
	return f.getWorkflow(ctx, scope, grantID, workflowID)
}

func (f *fakeTemplateWorkflowClient) CreateWorkflow(ctx context.Context, scope domain.RemoteScope, grantID string, req *domain.CreateRemoteWorkflowRequest) (*domain.RemoteWorkflow, error) {
	if f.createWorkflow == nil {
		return nil, errors.New("unexpected CreateWorkflow")
	}
	return f.createWorkflow(ctx, scope, grantID, req)
}

func (f *fakeTemplateWorkflowClient) UpdateWorkflow(ctx context.Context, scope domain.RemoteScope, grantID, workflowID string, req *domain.UpdateRemoteWorkflowRequest) (*domain.RemoteWorkflow, error) {
	if f.updateWorkflow == nil {
		return nil, errors.New("unexpected UpdateWorkflow")
	}
	return f.updateWorkflow(ctx, scope, grantID, workflowID, req)
}

func (f *fakeTemplateWorkflowClient) DeleteWorkflow(ctx context.Context, scope domain.RemoteScope, grantID, workflowID string) error {
	if f.deleteWorkflow == nil {
		return errors.New("unexpected DeleteWorkflow")
	}
	return f.deleteWorkflow(ctx, scope, grantID, workflowID)
}

func TestRegisterTemplateWorkflowHandlers(t *testing.T) {
	clientErr := errors.New("client unavailable")
	enabled := true

	tests := []struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeTemplateWorkflowClient
		assert       func(*testing.T, rpcTestResponse)
	}{
		{
			name:   "template.list app scope succeeds without default grant",
			method: "template.list",
			params: `{"scope":"app","limit":2,"page_token":"cursor-1"}`,
			client: &fakeTemplateWorkflowClient{listRemoteTemplates: func(ctx context.Context, scope domain.RemoteScope, grantID string, params *domain.CursorListParams) (*domain.RemoteTemplateListResponse, error) {
				if scope != domain.ScopeApplication || grantID != "" || params.Limit != 2 || params.PageToken != "cursor-1" {
					t.Fatalf("list args = %q %q %+v, want app empty grant cursor params", scope, grantID, params)
				}
				return &domain.RemoteTemplateListResponse{Data: []domain.RemoteTemplate{{ID: "tpl-1"}}, NextCursor: "cursor-2"}, nil
			}},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result templateListResult
				unmarshalResult(t, resp, &result)
				if len(result.Templates) != 1 || result.Templates[0].ID != "tpl-1" || result.NextCursor != "cursor-2" {
					t.Fatalf("result = %+v, want tpl-1 cursor-2", result)
				}
			},
		},
		{
			name:   "template.list grant scope requires grant_id",
			method: "template.list",
			params: `{"scope":"grant"}`,
			client: &fakeTemplateWorkflowClient{},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "template.get returns template",
			method:       "template.get",
			params:       `{"scope":"grant","grant_id":"grant-1","template_id":"tpl-1"}`,
			defaultGrant: "default-grant",
			client: &fakeTemplateWorkflowClient{getRemoteTemplate: func(ctx context.Context, scope domain.RemoteScope, grantID, templateID string) (*domain.RemoteTemplate, error) {
				if scope != domain.ScopeGrant || grantID != "grant-1" || templateID != "tpl-1" {
					t.Fatalf("get args = %q %q %q, want grant grant-1 tpl-1", scope, grantID, templateID)
				}
				return &domain.RemoteTemplate{ID: templateID, Name: "Welcome"}, nil
			}},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var template domain.RemoteTemplate
				unmarshalResult(t, resp, &template)
				if template.ID != "tpl-1" || template.Name != "Welcome" {
					t.Fatalf("template = %+v, want tpl-1 Welcome", template)
				}
			},
		},
		{
			name:         "template.create forwards request",
			method:       "template.create",
			params:       `{"grant_id":"grant-1","name":"Welcome","engine":"handlebars","subject":"Hi","body":"Hello {{name}}"}`,
			defaultGrant: "default-grant",
			client: &fakeTemplateWorkflowClient{createRemoteTemplate: func(ctx context.Context, scope domain.RemoteScope, grantID string, req *domain.CreateRemoteTemplateRequest) (*domain.RemoteTemplate, error) {
				if scope != domain.ScopeApplication || grantID != "grant-1" || req.Name != "Welcome" || req.Body != "Hello {{name}}" {
					t.Fatalf("create args = %q %q %+v, want forwarded template request", scope, grantID, req)
				}
				return &domain.RemoteTemplate{ID: "tpl-1", Name: req.Name}, nil
			}},
			assert: requireNoRPCError,
		},
		{
			name:         "template.update forwards request",
			method:       "template.update",
			params:       `{"template_id":"tpl-1","name":"Updated"}`,
			defaultGrant: "default-grant",
			client: &fakeTemplateWorkflowClient{updateRemoteTemplate: func(ctx context.Context, scope domain.RemoteScope, grantID, templateID string, req *domain.UpdateRemoteTemplateRequest) (*domain.RemoteTemplate, error) {
				if scope != domain.ScopeApplication || grantID != "" || templateID != "tpl-1" || req.Name == nil || *req.Name != "Updated" {
					t.Fatalf("update args = %q %q %q %+v, want forwarded update", scope, grantID, templateID, req)
				}
				return &domain.RemoteTemplate{ID: templateID, Name: *req.Name}, nil
			}},
			assert: requireNoRPCError,
		},
		{
			name:         "template.delete returns deleted",
			method:       "template.delete",
			params:       `{"template_id":"tpl-1"}`,
			defaultGrant: "default-grant",
			client: &fakeTemplateWorkflowClient{deleteRemoteTemplate: func(ctx context.Context, scope domain.RemoteScope, grantID, templateID string) error {
				if scope != domain.ScopeApplication || grantID != "" || templateID != "tpl-1" {
					t.Fatalf("delete args = %q %q %q, want app empty grant tpl-1", scope, grantID, templateID)
				}
				return nil
			}},
			assert: assertDeleted,
		},
		{
			name:         "template.render returns result",
			method:       "template.render",
			params:       `{"template_id":"tpl-1","variables":{"name":"Ada"},"strict":true}`,
			defaultGrant: "default-grant",
			client: &fakeTemplateWorkflowClient{renderRemoteTemplate: func(ctx context.Context, scope domain.RemoteScope, grantID, templateID string, req *domain.TemplateRenderRequest) (domain.TemplateRenderResult, error) {
				if scope != domain.ScopeApplication || grantID != "" || templateID != "tpl-1" || req.Strict == nil || !*req.Strict || req.Variables["name"] != "Ada" {
					t.Fatalf("render args = %q %q %q %+v, want forwarded render request", scope, grantID, templateID, req)
				}
				return domain.TemplateRenderResult{"body": "Hello Ada"}, nil
			}},
			assert: assertRenderBody("Hello Ada"),
		},
		{
			name:         "template.renderHTML returns result",
			method:       "template.renderHTML",
			params:       `{"body":"Hello {{name}}","engine":"handlebars","variables":{"name":"Ada"}}`,
			defaultGrant: "default-grant",
			client: &fakeTemplateWorkflowClient{renderRemoteTemplateHTML: func(ctx context.Context, scope domain.RemoteScope, grantID string, req *domain.TemplateRenderHTMLRequest) (domain.TemplateRenderResult, error) {
				if scope != domain.ScopeApplication || grantID != "" || req.Body != "Hello {{name}}" || req.Engine != "handlebars" || req.Variables["name"] != "Ada" {
					t.Fatalf("renderHTML args = %q %q %+v, want forwarded render HTML request", scope, grantID, req)
				}
				return domain.TemplateRenderResult{"body": "Hello Ada"}, nil
			}},
			assert: assertRenderBody("Hello Ada"),
		},
		{
			name:         "workflow.list returns workflows",
			method:       "workflow.list",
			params:       `{"limit":2}`,
			defaultGrant: "default-grant",
			client: &fakeTemplateWorkflowClient{listWorkflows: func(ctx context.Context, scope domain.RemoteScope, grantID string, params *domain.CursorListParams) (*domain.RemoteWorkflowListResponse, error) {
				if scope != domain.ScopeApplication || grantID != "" || params.Limit != 2 {
					t.Fatalf("workflow list args = %q %q %+v, want app empty grant limit 2", scope, grantID, params)
				}
				return &domain.RemoteWorkflowListResponse{Data: []domain.RemoteWorkflow{{ID: "wf-1"}}, NextCursor: "cursor-2"}, nil
			}},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result workflowListResult
				unmarshalResult(t, resp, &result)
				if len(result.Workflows) != 1 || result.Workflows[0].ID != "wf-1" || result.NextCursor != "cursor-2" {
					t.Fatalf("result = %+v, want wf-1 cursor-2", result)
				}
			},
		},
		{
			name:         "workflow.get returns workflow",
			method:       "workflow.get",
			params:       `{"scope":"grant","grant_id":"grant-1","workflow_id":"wf-1"}`,
			defaultGrant: "default-grant",
			client: &fakeTemplateWorkflowClient{getWorkflow: func(ctx context.Context, scope domain.RemoteScope, grantID, workflowID string) (*domain.RemoteWorkflow, error) {
				if scope != domain.ScopeGrant || grantID != "grant-1" || workflowID != "wf-1" {
					t.Fatalf("workflow get args = %q %q %q, want grant grant-1 wf-1", scope, grantID, workflowID)
				}
				return &domain.RemoteWorkflow{ID: workflowID, Name: "Reminder"}, nil
			}},
			assert: requireNoRPCError,
		},
		{
			name:         "workflow.create forwards request",
			method:       "workflow.create",
			params:       `{"name":"Reminder","template_id":"tpl-1","trigger_event":"booking.created","is_enabled":true}`,
			defaultGrant: "default-grant",
			client: &fakeTemplateWorkflowClient{createWorkflow: func(ctx context.Context, scope domain.RemoteScope, grantID string, req *domain.CreateRemoteWorkflowRequest) (*domain.RemoteWorkflow, error) {
				if scope != domain.ScopeApplication || grantID != "" || req.Name != "Reminder" || req.TemplateID != "tpl-1" || req.IsEnabled == nil || !*req.IsEnabled {
					t.Fatalf("workflow create args = %q %q %+v, want forwarded request", scope, grantID, req)
				}
				return &domain.RemoteWorkflow{ID: "wf-1", Name: req.Name}, nil
			}},
			assert: requireNoRPCError,
		},
		{
			name:         "workflow.update forwards request",
			method:       "workflow.update",
			params:       `{"workflow_id":"wf-1","is_enabled":true}`,
			defaultGrant: "default-grant",
			client: &fakeTemplateWorkflowClient{updateWorkflow: func(ctx context.Context, scope domain.RemoteScope, grantID, workflowID string, req *domain.UpdateRemoteWorkflowRequest) (*domain.RemoteWorkflow, error) {
				if scope != domain.ScopeApplication || grantID != "" || workflowID != "wf-1" || req.IsEnabled == nil || *req.IsEnabled != enabled {
					t.Fatalf("workflow update args = %q %q %q %+v, want forwarded update", scope, grantID, workflowID, req)
				}
				return &domain.RemoteWorkflow{ID: workflowID, IsEnabled: *req.IsEnabled}, nil
			}},
			assert: requireNoRPCError,
		},
		{
			name:         "workflow.delete returns deleted",
			method:       "workflow.delete",
			params:       `{"workflow_id":"wf-1"}`,
			defaultGrant: "default-grant",
			client: &fakeTemplateWorkflowClient{deleteWorkflow: func(ctx context.Context, scope domain.RemoteScope, grantID, workflowID string) error {
				if scope != domain.ScopeApplication || grantID != "" || workflowID != "wf-1" {
					t.Fatalf("workflow delete args = %q %q %q, want app empty grant wf-1", scope, grantID, workflowID)
				}
				return nil
			}},
			assert: assertDeleted,
		},
	}

	for _, spec := range templateWorkflowErrorSpecs(clientErr) {
		tests = append(tests, spec...)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterTemplateWorkflowHandlers(d, tt.client, tt.defaultGrant)

			resp := dispatchTemplateWorkflowRequest(t, d, tt.method, tt.params)
			tt.assert(t, resp)
		})
	}
}

func templateWorkflowErrorSpecs(clientErr error) [][]struct {
	name         string
	method       string
	params       string
	defaultGrant string
	client       *fakeTemplateWorkflowClient
	assert       func(*testing.T, rpcTestResponse)
} {
	invalidParams := func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) }
	internalError := func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InternalError) }
	return [][]struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeTemplateWorkflowClient
		assert       func(*testing.T, rpcTestResponse)
	}{
		errorSpec("template.list", `{"limit":1}`, `{"scope":"grant"}`, &fakeTemplateWorkflowClient{listRemoteTemplates: func(context.Context, domain.RemoteScope, string, *domain.CursorListParams) (*domain.RemoteTemplateListResponse, error) {
			return nil, clientErr
		}}, invalidParams, internalError),
		errorSpec("template.get", `{"template_id":"tpl-1"}`, `{}`, &fakeTemplateWorkflowClient{getRemoteTemplate: func(context.Context, domain.RemoteScope, string, string) (*domain.RemoteTemplate, error) {
			return nil, clientErr
		}}, invalidParams, internalError),
		errorSpec("template.create", `{"name":"Welcome"}`, `{"scope":"grant"}`, &fakeTemplateWorkflowClient{createRemoteTemplate: func(context.Context, domain.RemoteScope, string, *domain.CreateRemoteTemplateRequest) (*domain.RemoteTemplate, error) {
			return nil, clientErr
		}}, invalidParams, internalError),
		errorSpec("template.update", `{"template_id":"tpl-1","name":"Updated"}`, `{}`, &fakeTemplateWorkflowClient{updateRemoteTemplate: func(context.Context, domain.RemoteScope, string, string, *domain.UpdateRemoteTemplateRequest) (*domain.RemoteTemplate, error) {
			return nil, clientErr
		}}, invalidParams, internalError),
		errorSpec("template.delete", `{"template_id":"tpl-1"}`, `{}`, &fakeTemplateWorkflowClient{deleteRemoteTemplate: func(context.Context, domain.RemoteScope, string, string) error {
			return clientErr
		}}, invalidParams, internalError),
		errorSpec("template.render", `{"template_id":"tpl-1"}`, `{}`, &fakeTemplateWorkflowClient{renderRemoteTemplate: func(context.Context, domain.RemoteScope, string, string, *domain.TemplateRenderRequest) (domain.TemplateRenderResult, error) {
			return nil, clientErr
		}}, invalidParams, internalError),
		errorSpec("template.renderHTML", `{"body":"Hello","engine":"handlebars"}`, `{"scope":"grant"}`, &fakeTemplateWorkflowClient{renderRemoteTemplateHTML: func(context.Context, domain.RemoteScope, string, *domain.TemplateRenderHTMLRequest) (domain.TemplateRenderResult, error) {
			return nil, clientErr
		}}, invalidParams, internalError),
		errorSpec("workflow.list", `{"limit":1}`, `{"scope":"grant"}`, &fakeTemplateWorkflowClient{listWorkflows: func(context.Context, domain.RemoteScope, string, *domain.CursorListParams) (*domain.RemoteWorkflowListResponse, error) {
			return nil, clientErr
		}}, invalidParams, internalError),
		errorSpec("workflow.get", `{"workflow_id":"wf-1"}`, `{}`, &fakeTemplateWorkflowClient{getWorkflow: func(context.Context, domain.RemoteScope, string, string) (*domain.RemoteWorkflow, error) {
			return nil, clientErr
		}}, invalidParams, internalError),
		errorSpec("workflow.create", `{"name":"Reminder"}`, `{"scope":"grant"}`, &fakeTemplateWorkflowClient{createWorkflow: func(context.Context, domain.RemoteScope, string, *domain.CreateRemoteWorkflowRequest) (*domain.RemoteWorkflow, error) {
			return nil, clientErr
		}}, invalidParams, internalError),
		errorSpec("workflow.update", `{"workflow_id":"wf-1","name":"Updated"}`, `{}`, &fakeTemplateWorkflowClient{updateWorkflow: func(context.Context, domain.RemoteScope, string, string, *domain.UpdateRemoteWorkflowRequest) (*domain.RemoteWorkflow, error) {
			return nil, clientErr
		}}, invalidParams, internalError),
		errorSpec("workflow.delete", `{"workflow_id":"wf-1"}`, `{}`, &fakeTemplateWorkflowClient{deleteWorkflow: func(context.Context, domain.RemoteScope, string, string) error {
			return clientErr
		}}, invalidParams, internalError),
	}
}

func errorSpec(
	method string,
	clientErrorParams string,
	missingParams string,
	client *fakeTemplateWorkflowClient,
	invalidParams func(*testing.T, rpcTestResponse),
	internalError func(*testing.T, rpcTestResponse),
) []struct {
	name         string
	method       string
	params       string
	defaultGrant string
	client       *fakeTemplateWorkflowClient
	assert       func(*testing.T, rpcTestResponse)
} {
	badScopeParams := strings.Replace(clientErrorParams, "{", `{"scope":"bad",`, 1)

	return []struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeTemplateWorkflowClient
		assert       func(*testing.T, rpcTestResponse)
	}{
		{name: method + " missing required param returns invalid params", method: method, params: missingParams, client: &fakeTemplateWorkflowClient{}, assert: invalidParams},
		{name: method + " bad scope returns invalid params", method: method, params: badScopeParams, defaultGrant: "default-grant", client: &fakeTemplateWorkflowClient{}, assert: invalidParams},
		{name: method + " client error maps to internal error", method: method, params: clientErrorParams, defaultGrant: "default-grant", client: client, assert: internalError},
	}
}

func assertDeleted(t *testing.T, resp rpcTestResponse) {
	t.Helper()

	requireNoRPCError(t, resp)
	var result deletedResult
	unmarshalResult(t, resp, &result)
	if !result.Deleted {
		t.Fatal("deleted = false, want true")
	}
}

func assertRenderBody(want string) func(*testing.T, rpcTestResponse) {
	return func(t *testing.T, resp rpcTestResponse) {
		t.Helper()

		requireNoRPCError(t, resp)
		var result domain.TemplateRenderResult
		unmarshalResult(t, resp, &result)
		if result["body"] != want {
			t.Fatalf("body = %v, want %q", result["body"], want)
		}
	}
}

func dispatchTemplateWorkflowRequest(t *testing.T, d *Dispatcher, method, params string) rpcTestResponse {
	t.Helper()

	raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`)
	got := d.Dispatch(context.Background(), raw)
	if got == nil {
		t.Fatal("Dispatch() = nil, want response")
	}

	var resp rpcTestResponse
	if err := json.Unmarshal(got, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.JSONRPC != "2.0" {
		t.Fatalf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
	}
	return resp
}
