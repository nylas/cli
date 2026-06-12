package studio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nylasmock "github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
)

// mutationResponse is the common shape of every successful write: the fresh
// board plus the affected resource ID.
type mutationResponse struct {
	ID    string          `json:"id,omitempty"`
	Board json.RawMessage `json:"board"`
}

func doJSON(t *testing.T, handler http.HandlerFunc, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

func decodeMutation(t *testing.T, w *httptest.ResponseRecorder) mutationResponse {
	t.Helper()
	var resp mutationResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode mutation response: %v (body: %s)", err, w.Body.String())
	}
	if len(resp.Board) == 0 {
		t.Fatalf("every mutation must return fresh board state (body: %s)", w.Body.String())
	}
	return resp
}

// --- Workspaces ---

func TestHandleWorkspaceCreate(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routeWorkspaces, http.MethodPost, "/api/workspaces", `{"name":"Sales workspace"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", w.Code, w.Body.String())
	}
	resp := decodeMutation(t, w)
	if resp.ID != "workspace-new" {
		t.Fatalf("expected created workspace ID, got %q", resp.ID)
	}
}

func TestHandleWorkspaceCreate_RequiresName(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routeWorkspaces, http.MethodPost, "/api/workspaces", `{"name":"  "}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for blank name, got %d", w.Code)
	}
}

type workspaceUpdateRecorder struct {
	*nylasmock.MockClient
	gotPolicyID *string
	gotRuleIDs  *[]string
}

func (c *workspaceUpdateRecorder) UpdateWorkspace(ctx context.Context, workspaceID string, req *domain.UpdateWorkspaceRequest) (*domain.Workspace, error) {
	c.gotPolicyID = req.PolicyID
	c.gotRuleIDs = req.RulesIDs
	return c.MockClient.UpdateWorkspace(ctx, workspaceID, req)
}

func TestHandleWorkspacePatch_SetPolicy(t *testing.T) {
	t.Parallel()
	rec := &workspaceUpdateRecorder{MockClient: nylasmock.NewMockClient()}
	server := NewServer("127.0.0.1:0", rec)

	// workspace-2 is not the default workspace in the mock.
	w := doJSON(t, server.routeWorkspaces, http.MethodPatch, "/api/workspaces/workspace-2", `{"policy_id":"policy-9"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	decodeMutation(t, w)
	if rec.gotPolicyID == nil || *rec.gotPolicyID != "policy-9" {
		t.Fatalf("expected policy_id forwarded to UpdateWorkspace, got %v", rec.gotPolicyID)
	}
}

func TestHandleWorkspacePatch_DefaultWorkspacePolicyLocked(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	// workspace-1 is the default workspace: its policy defines the plan
	// ceiling, so swapping it would bypass the ceiling-policy protection.
	w := doJSON(t, server.routeWorkspaces, http.MethodPatch, "/api/workspaces/workspace-1", `{"policy_id":"policy-9"}`)

	if w.Code != http.StatusForbidden {
		t.Fatalf("default workspace policy swap must be rejected: expected 403, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestHandleWorkspacePatch_DefaultWorkspaceRuleAttachAllowed(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	// Only the policy slot is locked on the default workspace; rule
	// attachments remain legitimate.
	w := doJSON(t, server.routeWorkspaces, http.MethodPatch, "/api/workspaces/workspace-1", `{"add_rule_ids":["rule-2"]}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 attaching rules to default workspace, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestHandleWorkspacePatch_AddAndRemoveRules(t *testing.T) {
	t.Parallel()
	rec := &workspaceUpdateRecorder{MockClient: nylasmock.NewMockClient()}
	server := NewServer("127.0.0.1:0", rec)

	// Mock workspace has rule-1 attached; add rule-2 and remove rule-1.
	w := doJSON(t, server.routeWorkspaces, http.MethodPatch, "/api/workspaces/workspace-1",
		`{"add_rule_ids":["rule-2"],"remove_rule_ids":["rule-1"]}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	if rec.gotRuleIDs == nil {
		t.Fatal("expected rule_ids update sent to UpdateWorkspace")
	}
	if len(*rec.gotRuleIDs) != 1 || (*rec.gotRuleIDs)[0] != "rule-2" {
		t.Fatalf("read-modify-write must apply add+remove against current rule_ids; got %v", *rec.gotRuleIDs)
	}
}

func TestHandleWorkspaceDelete(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routeWorkspaces, http.MethodDelete, "/api/workspaces/workspace-2", "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	decodeMutation(t, w)
}

// --- Accounts ---

func TestHandleAccountCreate(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routeAccounts, http.MethodPost, "/api/accounts",
		`{"email":"bot@app.nylas.email","app_password":"ValidAgentPass123ABC!"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", w.Code, w.Body.String())
	}
	resp := decodeMutation(t, w)
	if resp.ID != "agent-new" {
		t.Fatalf("expected created account ID, got %q", resp.ID)
	}
}

func TestHandleAccountCreate_RequiresEmail(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routeAccounts, http.MethodPost, "/api/accounts", `{"email":""}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty email, got %d", w.Code)
	}
}

func TestHandleAccountPatch_RotatePassword(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routeAccounts, http.MethodPatch, "/api/accounts/agent-1",
		`{"app_password":"NewValidAgentPass456DEF!"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	decodeMutation(t, w)
}

func TestHandleAccountDelete(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routeAccounts, http.MethodDelete, "/api/accounts/agent-1", "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	decodeMutation(t, w)
}

// moveRecorder captures the manual-assign call so tests can assert the move
// is a single assign (the API moves a grant out of its old workspace itself).
type moveRecorder struct {
	*nylasmock.MockClient
	gotWorkspaceID string
	gotAssign      []string
	gotRemove      []string
}

func (c *moveRecorder) AssignWorkspaceGrants(ctx context.Context, workspaceID string, req *domain.WorkspaceAssignRequest) (*domain.WorkspaceAssignResult, error) {
	c.gotWorkspaceID = workspaceID
	c.gotAssign = req.AssignGrants
	c.gotRemove = req.RemoveGrants
	return c.MockClient.AssignWorkspaceGrants(ctx, workspaceID, req)
}

func TestHandleAccountMove(t *testing.T) {
	t.Parallel()
	client := &moveRecorder{MockClient: nylasmock.NewMockClient()}
	server := NewServer("127.0.0.1:0", client)

	w := doJSON(t, server.routeAccounts, http.MethodPost, "/api/accounts/agent-1/move",
		`{"workspace_id":"workspace-2"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	resp := decodeMutation(t, w)
	if resp.ID != "agent-1" {
		t.Fatalf("expected moved account ID, got %q", resp.ID)
	}
	if client.gotWorkspaceID != "workspace-2" {
		t.Fatalf("expected assign on target workspace-2, got %q", client.gotWorkspaceID)
	}
	if len(client.gotAssign) != 1 || client.gotAssign[0] != "agent-1" {
		t.Fatalf("expected single assign of agent-1, got %v", client.gotAssign)
	}
	if len(client.gotRemove) != 0 {
		t.Fatalf("a move must not remove grants (removal strands them in no workspace), got %v", client.gotRemove)
	}
}

func TestHandleAccountMove_RequiresWorkspaceID(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routeAccounts, http.MethodPost, "/api/accounts/agent-1/move", `{"workspace_id":" "}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing workspace_id, got %d", w.Code)
	}
}

type moveFailClient struct {
	*nylasmock.MockClient
}

func (c *moveFailClient) AssignWorkspaceGrants(ctx context.Context, workspaceID string, req *domain.WorkspaceAssignRequest) (*domain.WorkspaceAssignResult, error) {
	return nil, &domain.APIError{StatusCode: http.StatusBadRequest, Message: "upstream rejected"}
}

func TestHandleAccountMove_UpstreamError(t *testing.T) {
	t.Parallel()
	server := NewServer("127.0.0.1:0", &moveFailClient{MockClient: nylasmock.NewMockClient()})

	w := doJSON(t, server.routeAccounts, http.MethodPost, "/api/accounts/agent-1/move",
		`{"workspace_id":"workspace-2"}`)

	if w.Code < http.StatusBadRequest {
		t.Fatalf("expected error status, got %d", w.Code)
	}
}

// --- Plan-cap translation ---

type planLimitClient struct {
	*nylasmock.MockClient
}

func (c *planLimitClient) CreateRule(ctx context.Context, payload map[string]any) (*domain.Rule, error) {
	return nil, &domain.APIError{StatusCode: http.StatusForbidden, Message: "plan limit exceeded: max 5 rules"}
}

func TestHandleRuleCreate_TranslatesPlanLimit(t *testing.T) {
	t.Parallel()
	server := NewServer("127.0.0.1:0", &planLimitClient{MockClient: nylasmock.NewMockClient()})

	w := doJSON(t, server.routeRules, http.MethodPost, "/api/rules", `{"name":"r","trigger":"inbound"}`)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["error"] != "plan_limit" {
		t.Fatalf("upstream 403s must be translated to a structured plan_limit error, got %q", resp["error"])
	}
}
