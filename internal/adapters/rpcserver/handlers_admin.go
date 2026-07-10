package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type appListResult struct {
	Applications []domain.Application `json:"applications"`
}

type appGetParams struct {
	AppID string `json:"app_id"`
}

type appCreateParams struct {
	domain.CreateApplicationRequest
}

type appUpdateParams struct {
	AppID string `json:"app_id"`
	domain.UpdateApplicationRequest
}

type appDeleteParams struct {
	AppID string `json:"app_id"`
}

type callbackURIListResult struct {
	CallbackURIs []domain.CallbackURI `json:"callback_uris"`
}

type callbackURIGetParams struct {
	URIID string `json:"uri_id"`
}

type callbackURICreateParams struct {
	domain.CreateCallbackURIRequest
}

type callbackURIUpdateParams struct {
	URIID string `json:"uri_id"`
	domain.UpdateCallbackURIRequest
}

type callbackURIDeleteParams struct {
	URIID string `json:"uri_id"`
}

type connectorListResult struct {
	Connectors []domain.Connector `json:"connectors"`
}

type connectorGetParams struct {
	ConnectorID string `json:"connector_id"`
}

type connectorCreateParams struct {
	domain.CreateConnectorRequest
}

type connectorUpdateParams struct {
	ConnectorID string `json:"connector_id"`
	domain.UpdateConnectorRequest
}

type connectorDeleteParams struct {
	ConnectorID string `json:"connector_id"`
}

type credentialListParams struct {
	ConnectorID string `json:"connector_id"`
}

type credentialListResult struct {
	Credentials []domain.ConnectorCredential `json:"credentials"`
}

type credentialGetParams struct {
	ConnectorID  string `json:"connector_id"`
	CredentialID string `json:"credential_id"`
}

type credentialCreateParams struct {
	ConnectorID string `json:"connector_id"`
	domain.CreateCredentialRequest
}

type credentialUpdateParams struct {
	ConnectorID  string `json:"connector_id"`
	CredentialID string `json:"credential_id"`
	domain.UpdateCredentialRequest
}

type credentialDeleteParams struct {
	ConnectorID  string `json:"connector_id"`
	CredentialID string `json:"credential_id"`
}

type workspaceListResult struct {
	Workspaces []domain.Workspace `json:"workspaces"`
}

type workspaceGetParams struct {
	WorkspaceID string `json:"workspace_id"`
}

type workspaceCreateParams struct {
	domain.CreateWorkspaceRequest
}

type workspaceUpdateParams struct {
	WorkspaceID string `json:"workspace_id"`
	domain.UpdateWorkspaceRequest
}

type workspaceDeleteParams struct {
	WorkspaceID string `json:"workspace_id"`
}

type workspaceAssignParams struct {
	WorkspaceID string `json:"workspace_id"`
	domain.WorkspaceAssignRequest
}

type grantsListAllParams struct {
	domain.GrantsQueryParams
}

type grantsListAllResult struct {
	Grants []domain.Grant `json:"grants"`
}

// resolveConnectorID returns the explicit connector provider (rejecting
// deprecated ones), or auto-detects it when the application has exactly one
// connector — mirroring the CLI's `credentials` commands. Credentials live under
// /v3/connectors/{provider}/creds, so a provider is always required. The
// resolution policy is shared via domain.ResolveConnectorProvider so this RPC
// surface and the CLI cannot diverge; this wrapper only supplies the connector
// list and maps errors to RPC errors.
func resolveConnectorID(ctx context.Context, client ports.AdminClient, explicit string) (string, error) {
	connectors, listErr := client.ListConnectors(ctx)

	provider, err := domain.ResolveConnectorProvider(connectors, explicit)
	if err == nil {
		return provider, nil
	}

	// Discovery failed and nothing was named to fall back on: surface the real
	// listing error rather than a misleading "no connectors found".
	if listErr != nil && explicit == "" {
		return "", fmt.Errorf("resolve connector: %w", listErr)
	}
	// Same masking guard as the CLI: a discovery failure must not surface as a
	// misleading "unknown connector" when an explicit legacy ID couldn't be
	// mapped only because the connector list was unavailable.
	if listErr != nil && errors.Is(err, domain.ErrUnknownConnector) {
		return "", fmt.Errorf("resolve connector: %w", listErr)
	}

	var multi *domain.MultipleConnectorsError
	switch {
	case errors.As(err, &multi):
		return "", NewRPCError(InvalidParams,
			fmt.Sprintf("connector_id required: multiple connectors (%s)", strings.Join(multi.Providers, ", ")), nil)
	case errors.Is(err, domain.ErrDeprecatedConnector):
		return "", NewRPCError(InvalidParams,
			fmt.Sprintf("connector provider %q is no longer supported", explicit), nil)
	case errors.Is(err, domain.ErrUnknownConnector):
		return "", NewRPCError(InvalidParams,
			fmt.Sprintf("unknown connector provider %q", explicit), nil)
	default: // domain.ErrNoConnectors
		return "", NewRPCError(InvalidParams, "connector_id required: no connectors found", nil)
	}
}

func RegisterAdminHandlers(d *Dispatcher, client ports.AdminClient) {
	d.Register("admin.app.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		apps, err := client.ListApplications(ctx)
		if err != nil {
			return nil, fmt.Errorf("admin.app.list: %w", err)
		}
		return appListResult{Applications: apps}, nil
	})

	d.Register("admin.app.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p appGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.AppID == "" {
			return nil, NewRPCError(InvalidParams, "app_id required", nil)
		}

		app, err := client.GetApplication(ctx, p.AppID)
		if err != nil {
			return nil, fmt.Errorf("admin.app.get: %w", err)
		}
		return app, nil
	})

	d.Register("admin.app.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p appCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		app, err := client.CreateApplication(ctx, &p.CreateApplicationRequest)
		if err != nil {
			return nil, fmt.Errorf("admin.app.create: %w", err)
		}
		return app, nil
	})

	d.Register("admin.app.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p appUpdateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.AppID == "" {
			return nil, NewRPCError(InvalidParams, "app_id required", nil)
		}

		app, err := client.UpdateApplication(ctx, p.AppID, &p.UpdateApplicationRequest)
		if err != nil {
			return nil, fmt.Errorf("admin.app.update: %w", err)
		}
		return app, nil
	})

	d.Register("admin.app.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p appDeleteParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.AppID == "" {
			return nil, NewRPCError(InvalidParams, "app_id required", nil)
		}

		if err := client.DeleteApplication(ctx, p.AppID); err != nil {
			return nil, fmt.Errorf("admin.app.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})

	d.Register("admin.callbackUri.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		uris, err := client.ListCallbackURIs(ctx)
		if err != nil {
			return nil, fmt.Errorf("admin.callbackUri.list: %w", err)
		}
		return callbackURIListResult{CallbackURIs: uris}, nil
	})

	d.Register("admin.callbackUri.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p callbackURIGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.URIID == "" {
			return nil, NewRPCError(InvalidParams, "uri_id required", nil)
		}

		uri, err := client.GetCallbackURI(ctx, p.URIID)
		if err != nil {
			return nil, fmt.Errorf("admin.callbackUri.get: %w", err)
		}
		return uri, nil
	})

	d.Register("admin.callbackUri.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p callbackURICreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		uri, err := client.CreateCallbackURI(ctx, &p.CreateCallbackURIRequest)
		if err != nil {
			return nil, fmt.Errorf("admin.callbackUri.create: %w", err)
		}
		return uri, nil
	})

	d.Register("admin.callbackUri.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p callbackURIUpdateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.URIID == "" {
			return nil, NewRPCError(InvalidParams, "uri_id required", nil)
		}

		uri, err := client.UpdateCallbackURI(ctx, p.URIID, &p.UpdateCallbackURIRequest)
		if err != nil {
			return nil, fmt.Errorf("admin.callbackUri.update: %w", err)
		}
		return uri, nil
	})

	d.Register("admin.callbackUri.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p callbackURIDeleteParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.URIID == "" {
			return nil, NewRPCError(InvalidParams, "uri_id required", nil)
		}

		if err := client.DeleteCallbackURI(ctx, p.URIID); err != nil {
			return nil, fmt.Errorf("admin.callbackUri.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})

	d.Register("admin.connector.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		connectors, err := client.ListConnectors(ctx)
		if err != nil {
			return nil, fmt.Errorf("admin.connector.list: %w", err)
		}
		return connectorListResult{Connectors: connectors}, nil
	})

	d.Register("admin.connector.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p connectorGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.ConnectorID == "" {
			return nil, NewRPCError(InvalidParams, "connector_id required", nil)
		}

		connector, err := client.GetConnector(ctx, p.ConnectorID)
		if err != nil {
			return nil, fmt.Errorf("admin.connector.get: %w", err)
		}
		return connector, nil
	})

	d.Register("admin.connector.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p connectorCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		connector, err := client.CreateConnector(ctx, &p.CreateConnectorRequest)
		if err != nil {
			return nil, fmt.Errorf("admin.connector.create: %w", err)
		}
		return connector, nil
	})

	d.Register("admin.connector.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p connectorUpdateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.ConnectorID == "" {
			return nil, NewRPCError(InvalidParams, "connector_id required", nil)
		}

		connector, err := client.UpdateConnector(ctx, p.ConnectorID, &p.UpdateConnectorRequest)
		if err != nil {
			return nil, fmt.Errorf("admin.connector.update: %w", err)
		}
		return connector, nil
	})

	d.Register("admin.connector.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p connectorDeleteParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.ConnectorID == "" {
			return nil, NewRPCError(InvalidParams, "connector_id required", nil)
		}

		if err := client.DeleteConnector(ctx, p.ConnectorID); err != nil {
			return nil, fmt.Errorf("admin.connector.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})

	d.Register("admin.credential.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p credentialListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		connectorID, err := resolveConnectorID(ctx, client, p.ConnectorID)
		if err != nil {
			return nil, err
		}

		credentials, err := client.ListCredentials(ctx, connectorID)
		if err != nil {
			return nil, fmt.Errorf("admin.credential.list: %w", err)
		}
		return credentialListResult{Credentials: credentials}, nil
	})

	d.Register("admin.credential.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p credentialGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.CredentialID == "" {
			return nil, NewRPCError(InvalidParams, "credential_id required", nil)
		}
		connectorID, err := resolveConnectorID(ctx, client, p.ConnectorID)
		if err != nil {
			return nil, err
		}

		credential, err := client.GetCredential(ctx, connectorID, p.CredentialID)
		if err != nil {
			return nil, fmt.Errorf("admin.credential.get: %w", err)
		}
		return credential, nil
	})

	d.Register("admin.credential.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p credentialCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		connectorID, err := resolveConnectorID(ctx, client, p.ConnectorID)
		if err != nil {
			return nil, err
		}

		credential, err := client.CreateCredential(ctx, connectorID, &p.CreateCredentialRequest)
		if err != nil {
			return nil, fmt.Errorf("admin.credential.create: %w", err)
		}
		return credential, nil
	})

	d.Register("admin.credential.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p credentialUpdateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.CredentialID == "" {
			return nil, NewRPCError(InvalidParams, "credential_id required", nil)
		}
		connectorID, err := resolveConnectorID(ctx, client, p.ConnectorID)
		if err != nil {
			return nil, err
		}

		credential, err := client.UpdateCredential(ctx, connectorID, p.CredentialID, &p.UpdateCredentialRequest)
		if err != nil {
			return nil, fmt.Errorf("admin.credential.update: %w", err)
		}
		return credential, nil
	})

	d.Register("admin.credential.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p credentialDeleteParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.CredentialID == "" {
			return nil, NewRPCError(InvalidParams, "credential_id required", nil)
		}
		connectorID, err := resolveConnectorID(ctx, client, p.ConnectorID)
		if err != nil {
			return nil, err
		}

		if err := client.DeleteCredential(ctx, connectorID, p.CredentialID); err != nil {
			return nil, fmt.Errorf("admin.credential.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})

	d.Register("workspace.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		workspaces, err := client.ListWorkspaces(ctx)
		if err != nil {
			return nil, fmt.Errorf("workspace.list: %w", err)
		}
		return workspaceListResult{Workspaces: workspaces}, nil
	})

	d.Register("workspace.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p workspaceGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.WorkspaceID == "" {
			return nil, NewRPCError(InvalidParams, "workspace_id required", nil)
		}

		workspace, err := client.GetWorkspace(ctx, p.WorkspaceID)
		if err != nil {
			return nil, fmt.Errorf("workspace.get: %w", err)
		}
		return workspace, nil
	})

	d.Register("workspace.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p workspaceCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		workspace, err := client.CreateWorkspace(ctx, &p.CreateWorkspaceRequest)
		if err != nil {
			return nil, fmt.Errorf("workspace.create: %w", err)
		}
		return workspace, nil
	})

	d.Register("workspace.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p workspaceUpdateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.WorkspaceID == "" {
			return nil, NewRPCError(InvalidParams, "workspace_id required", nil)
		}

		workspace, err := client.UpdateWorkspace(ctx, p.WorkspaceID, &p.UpdateWorkspaceRequest)
		if err != nil {
			return nil, fmt.Errorf("workspace.update: %w", err)
		}
		return workspace, nil
	})

	d.Register("workspace.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p workspaceDeleteParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.WorkspaceID == "" {
			return nil, NewRPCError(InvalidParams, "workspace_id required", nil)
		}

		if err := client.DeleteWorkspace(ctx, p.WorkspaceID); err != nil {
			return nil, fmt.Errorf("workspace.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})

	d.Register("workspace.assignGrants", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p workspaceAssignParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.WorkspaceID == "" {
			return nil, NewRPCError(InvalidParams, "workspace_id required", nil)
		}

		result, err := client.AssignWorkspaceGrants(ctx, p.WorkspaceID, &p.WorkspaceAssignRequest)
		if err != nil {
			return nil, fmt.Errorf("workspace.assignGrants: %w", err)
		}
		return result, nil
	})

	d.Register("admin.grants.listAll", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p grantsListAllParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grants, err := client.ListAllGrants(ctx, &p.GrantsQueryParams)
		if err != nil {
			return nil, fmt.Errorf("admin.grants.listAll: %w", err)
		}
		return grantsListAllResult{Grants: grants}, nil
	})

	d.Register("admin.grants.stats", func(ctx context.Context, params json.RawMessage) (any, error) {
		stats, err := client.GetGrantStats(ctx)
		if err != nil {
			return nil, fmt.Errorf("admin.grants.stats: %w", err)
		}
		return stats, nil
	})
}
