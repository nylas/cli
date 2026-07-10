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

type fakeAdminClient struct {
	ports.AdminClient

	listApplications      func(context.Context) ([]domain.Application, error)
	createCallbackURI     func(context.Context, *domain.CreateCallbackURIRequest) (*domain.CallbackURI, error)
	updateConnector       func(context.Context, string, *domain.UpdateConnectorRequest) (*domain.Connector, error)
	listConnectors        func(context.Context) ([]domain.Connector, error)
	listCredentials       func(context.Context, string) ([]domain.ConnectorCredential, error)
	getCredential         func(context.Context, string, string) (*domain.ConnectorCredential, error)
	getWorkspace          func(context.Context, string) (*domain.Workspace, error)
	assignWorkspaceGrants func(context.Context, string, *domain.WorkspaceAssignRequest) (*domain.WorkspaceAssignResult, error)
	listAllGrants         func(context.Context, *domain.GrantsQueryParams) ([]domain.Grant, error)
	getGrantStats         func(context.Context) (*domain.GrantStats, error)
}

func (f *fakeAdminClient) ListApplications(ctx context.Context) ([]domain.Application, error) {
	if f.listApplications == nil {
		return nil, errors.New("unexpected ListApplications")
	}
	return f.listApplications(ctx)
}

func (f *fakeAdminClient) CreateCallbackURI(ctx context.Context, req *domain.CreateCallbackURIRequest) (*domain.CallbackURI, error) {
	if f.createCallbackURI == nil {
		return nil, errors.New("unexpected CreateCallbackURI")
	}
	return f.createCallbackURI(ctx, req)
}

func (f *fakeAdminClient) UpdateConnector(ctx context.Context, connectorID string, req *domain.UpdateConnectorRequest) (*domain.Connector, error) {
	if f.updateConnector == nil {
		return nil, errors.New("unexpected UpdateConnector")
	}
	return f.updateConnector(ctx, connectorID, req)
}

func (f *fakeAdminClient) ListConnectors(ctx context.Context) ([]domain.Connector, error) {
	if f.listConnectors == nil {
		return nil, errors.New("unexpected ListConnectors")
	}
	return f.listConnectors(ctx)
}

func (f *fakeAdminClient) ListCredentials(ctx context.Context, connectorID string) ([]domain.ConnectorCredential, error) {
	if f.listCredentials == nil {
		return nil, errors.New("unexpected ListCredentials")
	}
	return f.listCredentials(ctx, connectorID)
}

func (f *fakeAdminClient) GetCredential(ctx context.Context, connectorID, credentialID string) (*domain.ConnectorCredential, error) {
	if f.getCredential == nil {
		return nil, errors.New("unexpected GetCredential")
	}
	return f.getCredential(ctx, connectorID, credentialID)
}

func (f *fakeAdminClient) GetWorkspace(ctx context.Context, workspaceID string) (*domain.Workspace, error) {
	if f.getWorkspace == nil {
		return nil, errors.New("unexpected GetWorkspace")
	}
	return f.getWorkspace(ctx, workspaceID)
}

func (f *fakeAdminClient) AssignWorkspaceGrants(ctx context.Context, workspaceID string, req *domain.WorkspaceAssignRequest) (*domain.WorkspaceAssignResult, error) {
	if f.assignWorkspaceGrants == nil {
		return nil, errors.New("unexpected AssignWorkspaceGrants")
	}
	return f.assignWorkspaceGrants(ctx, workspaceID, req)
}

func (f *fakeAdminClient) ListAllGrants(ctx context.Context, params *domain.GrantsQueryParams) ([]domain.Grant, error) {
	if f.listAllGrants == nil {
		return nil, errors.New("unexpected ListAllGrants")
	}
	return f.listAllGrants(ctx, params)
}

func (f *fakeAdminClient) GetGrantStats(ctx context.Context) (*domain.GrantStats, error) {
	if f.getGrantStats == nil {
		return nil, errors.New("unexpected GetGrantStats")
	}
	return f.getGrantStats(ctx)
}

func TestRegisterAdminHandlers_RegistersAllMethods(t *testing.T) {
	d := NewDispatcher()
	RegisterAdminHandlers(d, &fakeAdminClient{})

	methods := []string{
		"admin.app.list",
		"admin.app.get",
		"admin.app.create",
		"admin.app.update",
		"admin.app.delete",
		"admin.callbackUri.list",
		"admin.callbackUri.get",
		"admin.callbackUri.create",
		"admin.callbackUri.update",
		"admin.callbackUri.delete",
		"admin.connector.list",
		"admin.connector.get",
		"admin.connector.create",
		"admin.connector.update",
		"admin.connector.delete",
		"admin.credential.list",
		"admin.credential.get",
		"admin.credential.create",
		"admin.credential.update",
		"admin.credential.delete",
		"workspace.list",
		"workspace.get",
		"workspace.create",
		"workspace.update",
		"workspace.delete",
		"workspace.assignGrants",
		"admin.grants.listAll",
		"admin.grants.stats",
	}

	if len(d.handlers) != len(methods) {
		t.Fatalf("registered handlers = %d, want %d", len(d.handlers), len(methods))
	}
	for _, method := range methods {
		if d.handlers[method] == nil {
			t.Fatalf("handler %q not registered", method)
		}
	}
}

func TestRegisterAdminHandlers(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name   string
		method string
		params string
		client *fakeAdminClient
		assert func(*testing.T, rpcTestResponse)
	}{
		{
			name:   "admin.app.list returns applications",
			method: "admin.app.list",
			params: `{}`,
			client: &fakeAdminClient{
				listApplications: func(ctx context.Context) ([]domain.Application, error) {
					return []domain.Application{{ID: "app-1"}}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var result struct {
					Applications []domain.Application `json:"applications"`
				}
				unmarshalResult(t, resp, &result)
				if len(result.Applications) != 1 || result.Applications[0].ID != "app-1" {
					t.Fatalf("applications = %#v, want app-1", result.Applications)
				}
			},
		},
		{
			name:   "admin.app.get missing app_id returns invalid params",
			method: "admin.app.get",
			params: `{}`,
			client: &fakeAdminClient{},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:   "admin.app.list client error maps to internal error",
			method: "admin.app.list",
			params: `{}`,
			client: &fakeAdminClient{
				listApplications: func(ctx context.Context) ([]domain.Application, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
		{
			name:   "admin.callbackUri.create embeds request fields",
			method: "admin.callbackUri.create",
			params: `{"url":"http://localhost/callback","platform":"web"}`,
			client: &fakeAdminClient{
				createCallbackURI: func(ctx context.Context, req *domain.CreateCallbackURIRequest) (*domain.CallbackURI, error) {
					if req.URL != "http://localhost/callback" || req.Platform != "web" {
						t.Fatalf("request = %#v, want callback URL and web platform", req)
					}
					return &domain.CallbackURI{ID: "uri-1", URL: req.URL, Platform: req.Platform}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var uri domain.CallbackURI
				unmarshalResult(t, resp, &uri)
				if uri.ID != "uri-1" {
					t.Fatalf("callback URI = %#v, want uri-1", uri)
				}
			},
		},
		{
			name:   "admin.callbackUri.get missing uri_id returns invalid params",
			method: "admin.callbackUri.get",
			params: `{}`,
			client: &fakeAdminClient{},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:   "admin.callbackUri.create client error maps to internal error",
			method: "admin.callbackUri.create",
			params: `{"url":"http://localhost/callback","platform":"web"}`,
			client: &fakeAdminClient{
				createCallbackURI: func(ctx context.Context, req *domain.CreateCallbackURIRequest) (*domain.CallbackURI, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
		{
			name:   "admin.connector.update embeds request fields",
			method: "admin.connector.update",
			params: `{"connector_id":"connector-1","name":"Google"}`,
			client: &fakeAdminClient{
				updateConnector: func(ctx context.Context, connectorID string, req *domain.UpdateConnectorRequest) (*domain.Connector, error) {
					if connectorID != "connector-1" {
						t.Fatalf("connectorID = %q, want connector-1", connectorID)
					}
					if req.Name == nil || *req.Name != "Google" {
						t.Fatalf("Name = %#v, want Google", req.Name)
					}
					return &domain.Connector{ID: connectorID, Name: *req.Name, Provider: "google"}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var connector domain.Connector
				unmarshalResult(t, resp, &connector)
				if connector.ID != "connector-1" || connector.Name != "Google" {
					t.Fatalf("connector = %#v, want connector-1 Google", connector)
				}
			},
		},
		{
			name:   "admin.connector.update missing connector_id returns invalid params",
			method: "admin.connector.update",
			params: `{"name":"Google"}`,
			client: &fakeAdminClient{},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:   "admin.connector.update client error maps to internal error",
			method: "admin.connector.update",
			params: `{"connector_id":"connector-1","name":"Google"}`,
			client: &fakeAdminClient{
				updateConnector: func(ctx context.Context, connectorID string, req *domain.UpdateConnectorRequest) (*domain.Connector, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
		{
			name:   "admin.credential.list maps legacy connector_id to provider",
			method: "admin.credential.list",
			params: `{"connector_id":"connector-1"}`,
			client: &fakeAdminClient{
				listConnectors: func(ctx context.Context) ([]domain.Connector, error) {
					return []domain.Connector{{ID: "connector-1", Provider: "google"}}, nil
				},
				listCredentials: func(ctx context.Context, connectorID string) ([]domain.ConnectorCredential, error) {
					if connectorID != "google" {
						t.Fatalf("connectorID = %q, want provider google", connectorID)
					}
					return []domain.ConnectorCredential{{ID: "credential-1", Name: "Google Credential"}}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var result struct {
					Credentials []domain.ConnectorCredential `json:"credentials"`
				}
				unmarshalResult(t, resp, &result)
				if len(result.Credentials) != 1 || result.Credentials[0].ID != "credential-1" {
					t.Fatalf("credentials = %#v, want credential-1", result.Credentials)
				}
			},
		},
		{
			name:   "admin.credential.list missing connector_id with no unique connector returns invalid params",
			method: "admin.credential.list",
			params: `{}`,
			client: &fakeAdminClient{
				listConnectors: func(ctx context.Context) ([]domain.Connector, error) {
					// Zero (or multiple) connectors → cannot auto-detect.
					return nil, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:   "admin.credential.list auto-detects the sole connector when connector_id omitted",
			method: "admin.credential.list",
			params: `{}`,
			client: &fakeAdminClient{
				listConnectors: func(ctx context.Context) ([]domain.Connector, error) {
					return []domain.Connector{{ID: "connector-1", Provider: "google"}}, nil
				},
				listCredentials: func(ctx context.Context, connectorID string) ([]domain.ConnectorCredential, error) {
					if connectorID != "google" {
						t.Fatalf("connectorID = %q, want auto-detected provider google", connectorID)
					}
					return []domain.ConnectorCredential{{ID: "credential-1"}}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
			},
		},
		{
			name:   "admin.credential.list omitted connector_id with multiple connectors names them",
			method: "admin.credential.list",
			params: `{}`,
			client: &fakeAdminClient{
				listConnectors: func(ctx context.Context) ([]domain.Connector, error) {
					return []domain.Connector{
						{ID: "c1", Provider: "google"},
						{ID: "c2", Provider: "microsoft"},
					}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if !strings.Contains(resp.Error.Message, "google") || !strings.Contains(resp.Error.Message, "microsoft") {
					t.Fatalf("error %q should name the candidate providers google and microsoft", resp.Error.Message)
				}
			},
		},
		{
			name:   "admin.credential.get missing credential_id returns invalid params",
			method: "admin.credential.get",
			params: `{"connector_id":"connector-1"}`,
			client: &fakeAdminClient{},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:   "admin.credential.get client error maps to internal error",
			method: "admin.credential.get",
			params: `{"connector_id":"google","credential_id":"credential-1"}`,
			client: &fakeAdminClient{
				getCredential: func(ctx context.Context, connectorID, credentialID string) (*domain.ConnectorCredential, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
		{
			name:   "workspace.assignGrants embeds request fields",
			method: "workspace.assignGrants",
			params: `{"workspace_id":"workspace-1","assign_grants":["grant-1"],"remove_grants":["grant-2"]}`,
			client: &fakeAdminClient{
				assignWorkspaceGrants: func(ctx context.Context, workspaceID string, req *domain.WorkspaceAssignRequest) (*domain.WorkspaceAssignResult, error) {
					if workspaceID != "workspace-1" {
						t.Fatalf("workspaceID = %q, want workspace-1", workspaceID)
					}
					if len(req.AssignGrants) != 1 || req.AssignGrants[0] != "grant-1" {
						t.Fatalf("AssignGrants = %#v, want grant-1", req.AssignGrants)
					}
					if len(req.RemoveGrants) != 1 || req.RemoveGrants[0] != "grant-2" {
						t.Fatalf("RemoveGrants = %#v, want grant-2", req.RemoveGrants)
					}
					return &domain.WorkspaceAssignResult{WorkspaceID: workspaceID, GrantsAssigned: req.AssignGrants, GrantsRemoved: req.RemoveGrants}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var result domain.WorkspaceAssignResult
				unmarshalResult(t, resp, &result)
				if result.WorkspaceID != "workspace-1" || len(result.GrantsAssigned) != 1 || result.GrantsAssigned[0] != "grant-1" {
					t.Fatalf("assign result = %#v, want workspace-1 grant-1", result)
				}
			},
		},
		{
			name:   "workspace.assignGrants missing workspace_id returns invalid params",
			method: "workspace.assignGrants",
			params: `{"assign_grants":["grant-1"]}`,
			client: &fakeAdminClient{},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:   "workspace.get client error maps to internal error",
			method: "workspace.get",
			params: `{"workspace_id":"workspace-1"}`,
			client: &fakeAdminClient{
				getWorkspace: func(ctx context.Context, workspaceID string) (*domain.Workspace, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
		{
			name:   "admin.grants.listAll returns grants and forwards params",
			method: "admin.grants.listAll",
			params: `{"limit":10,"offset":5,"connector_id":"connector-1","status":"valid"}`,
			client: &fakeAdminClient{
				listAllGrants: func(ctx context.Context, params *domain.GrantsQueryParams) ([]domain.Grant, error) {
					if params.Limit != 10 || params.Offset != 5 || params.ConnectorID != "connector-1" || params.Status != "valid" {
						t.Fatalf("params = %#v, want list all query params", params)
					}
					return []domain.Grant{{ID: "grant-1", Email: "user@example.com", GrantStatus: "valid"}}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var result struct {
					Grants []domain.Grant `json:"grants"`
				}
				unmarshalResult(t, resp, &result)
				if len(result.Grants) != 1 || result.Grants[0].ID != "grant-1" {
					t.Fatalf("grants = %#v, want grant-1", result.Grants)
				}
			},
		},
		{
			name:   "admin.grants.stats returns stats",
			method: "admin.grants.stats",
			params: `{}`,
			client: &fakeAdminClient{
				getGrantStats: func(ctx context.Context) (*domain.GrantStats, error) {
					return &domain.GrantStats{Total: 2, Valid: 1, Invalid: 1}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var stats domain.GrantStats
				unmarshalResult(t, resp, &stats)
				if stats.Total != 2 || stats.Valid != 1 || stats.Invalid != 1 {
					t.Fatalf("stats = %#v, want total 2 valid 1 invalid 1", stats)
				}
			},
		},
		{
			name:   "admin.grants.stats client error maps to internal error",
			method: "admin.grants.stats",
			params: `{}`,
			client: &fakeAdminClient{
				getGrantStats: func(ctx context.Context) (*domain.GrantStats, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterAdminHandlers(d, tt.client)

			resp := dispatchAdminRequest(t, d, tt.method, tt.params)
			tt.assert(t, resp)
		})
	}
}

func dispatchAdminRequest(t *testing.T, d *Dispatcher, method, params string) rpcTestResponse {
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
