package studio

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	nylasmock "github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
)

// --- Policies ---

func TestHandlePolicyCreate(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routePolicies, http.MethodPost, "/api/policies", `{"name":"Strict Policy"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", w.Code, w.Body.String())
	}
	resp := decodeMutation(t, w)
	if resp.ID != "policy-new" {
		t.Fatalf("expected created policy ID, got %q", resp.ID)
	}
}

func TestHandlePolicyPatch_DefaultWorkspacePolicyMutable(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	// policy-1 is attached to the default workspace (workspace-1) in the mock.
	// Per the documented model, no policy is the plan ceiling — the ceiling is
	// the billing plan, enforced by the Nylas API — so every policy is
	// editable, including the default workspace's.
	w := doJSON(t, server.routePolicies, http.MethodPatch, "/api/policies/policy-1", `{"name":"Renamed"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("every policy must be editable (plan ceiling is API-enforced): expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	decodeMutation(t, w)
}

func TestHandlePolicyDelete_DefaultWorkspacePolicyAllowed(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	// Deleting any policy is allowed: workspaces left without a policy run at
	// the billing plan's maximum limits.
	w := doJSON(t, server.routePolicies, http.MethodDelete, "/api/policies/policy-1", "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 deleting the default workspace's policy, got %d (body: %s)", w.Code, w.Body.String())
	}
	decodeMutation(t, w)
}

func TestHandlePolicyPatch_CustomPolicyAllowed(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routePolicies, http.MethodPatch, "/api/policies/policy-7", `{"name":"Renamed"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	decodeMutation(t, w)
}

// CreatePolicy and UpdatePolicy simulate the Nylas API rejecting a policy
// whose limits exceed the billing plan's maximum (the API returns 403 for
// plan caps). planLimitClient is declared in handlers_mutations_test.go.
func (c *planLimitClient) CreatePolicy(ctx context.Context, payload map[string]any) (*domain.Policy, error) {
	return nil, &domain.APIError{StatusCode: http.StatusForbidden, Message: "limit exceeds plan maximum"}
}

func (c *planLimitClient) UpdatePolicy(ctx context.Context, policyID string, payload map[string]any) (*domain.Policy, error) {
	return nil, &domain.APIError{StatusCode: http.StatusForbidden, Message: "limit exceeds plan maximum"}
}

func TestHandlePolicyCreate_SurfacesPlanLimitError(t *testing.T) {
	t.Parallel()
	server := NewServer("127.0.0.1:0", &planLimitClient{MockClient: nylasmock.NewMockClient()})

	// Plan maximums are enforced by the Nylas API, not by Studio: the handler
	// forwards the payload and must surface the API's rejection as a
	// structured plan_limit error the UI can render.
	w := doJSON(t, server.routePolicies, http.MethodPost, "/api/policies",
		`{"name":"Over","limits":{"limit_count_daily_message_per_grant":600}}`)

	if w.Code != http.StatusForbidden {
		t.Fatalf("API plan rejection must be surfaced: expected 403, got %d (body: %s)", w.Code, w.Body.String())
	}
	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "plan_limit" {
		t.Fatalf("expected structured plan_limit error, got %q", resp["error"])
	}
}

func TestHandlePolicyPatch_SurfacesPlanLimitError(t *testing.T) {
	t.Parallel()
	server := NewServer("127.0.0.1:0", &planLimitClient{MockClient: nylasmock.NewMockClient()})

	// With the client-side ceiling validation removed, the PATCH path relies
	// entirely on the API's plan enforcement — its rejection must surface as
	// the same structured plan_limit error as create.
	w := doJSON(t, server.routePolicies, http.MethodPatch, "/api/policies/policy-7",
		`{"name":"Over","limits":{"limit_count_daily_message_per_grant":600}}`)

	if w.Code != http.StatusForbidden {
		t.Fatalf("API plan rejection must be surfaced on update: expected 403, got %d (body: %s)", w.Code, w.Body.String())
	}
	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "plan_limit" {
		t.Fatalf("expected structured plan_limit error, got %q", resp["error"])
	}
}

func TestHandlePolicyCreate_ForwardsLimitsToAPI(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	// Studio performs no client-side limit validation — payloads go to the
	// API as-is and succeed unless the API rejects them.
	w := doJSON(t, server.routePolicies, http.MethodPost, "/api/policies",
		`{"name":"Within","limits":{"limit_count_daily_message_per_grant":200}}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// --- Rules ---

func TestHandleRuleCreate_AttachesToWorkspace(t *testing.T) {
	t.Parallel()
	rec := &workspaceUpdateRecorder{MockClient: nylasmock.NewMockClient()}
	server := NewServer("127.0.0.1:0", rec)

	w := doJSON(t, server.routeRules, http.MethodPost, "/api/rules",
		`{"workspace_id":"workspace-1","name":"Block spam","trigger":"inbound"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", w.Code, w.Body.String())
	}
	resp := decodeMutation(t, w)
	if resp.ID == "" {
		t.Fatal("expected created rule ID")
	}
	if rec.gotRuleIDs == nil {
		t.Fatal("rule create with workspace_id must attach the rule to the workspace")
	}
}

// attachFailClient fails the workspace attach after a successful rule create
// and records whether the compensating rule delete ran.
type attachFailClient struct {
	*nylasmock.MockClient
	deletedRuleID string
}

func (c *attachFailClient) UpdateWorkspace(ctx context.Context, workspaceID string, req *domain.UpdateWorkspaceRequest) (*domain.Workspace, error) {
	return nil, &domain.APIError{StatusCode: http.StatusBadRequest, Message: "rule_ids entry not found"}
}

func (c *attachFailClient) DeleteRule(ctx context.Context, ruleID string) error {
	c.deletedRuleID = ruleID
	return nil
}

func TestHandleRuleCreate_AttachFailureCleansUpRule(t *testing.T) {
	t.Parallel()
	client := &attachFailClient{MockClient: nylasmock.NewMockClient()}
	server := NewServer("127.0.0.1:0", client)

	w := doJSON(t, server.routeRules, http.MethodPost, "/api/rules",
		`{"workspace_id":"workspace-1","name":"Block spam","trigger":"inbound"}`)

	if w.Code == http.StatusCreated {
		t.Fatalf("attach failure must not report success, got %d", w.Code)
	}
	if client.deletedRuleID == "" {
		t.Fatal("a rule whose workspace attach failed must be deleted, not left orphaned")
	}
}

// seedFailClient fails item seeding after a successful list create and records
// whether the compensating list delete ran.
type seedFailClient struct {
	*nylasmock.MockClient
	deletedListID string
}

func (c *seedFailClient) AddListItems(ctx context.Context, listID string, items []string) (*domain.AgentList, error) {
	return nil, &domain.APIError{StatusCode: http.StatusBadRequest, Message: "invalid item for list type"}
}

func (c *seedFailClient) DeleteList(ctx context.Context, listID string) error {
	c.deletedListID = listID
	return nil
}

func TestHandleListCreate_SeedFailureCleansUpList(t *testing.T) {
	t.Parallel()
	client := &seedFailClient{MockClient: nylasmock.NewMockClient()}
	server := NewServer("127.0.0.1:0", client)

	w := doJSON(t, server.routeLists, http.MethodPost, "/api/lists",
		`{"name":"Blocked","type":"domain","items":["not a domain"]}`)

	if w.Code == http.StatusCreated {
		t.Fatalf("seed failure must not report success, got %d", w.Code)
	}
	if client.deletedListID == "" {
		t.Fatal("a list whose item seeding failed must be deleted, not left partial")
	}
}

func TestHandleRulePatch(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routeRules, http.MethodPatch, "/api/rules/rule-1", `{"name":"Renamed rule"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	decodeMutation(t, w)
}

func TestHandleRuleDelete(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routeRules, http.MethodDelete, "/api/rules/rule-1", "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	decodeMutation(t, w)
}

// --- Lists ---

func TestHandleListCreate_SeedsItems(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routeLists, http.MethodPost, "/api/lists",
		`{"name":"Blocked","type":"domain","items":["spam.com","junk.net"]}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", w.Code, w.Body.String())
	}
	resp := decodeMutation(t, w)
	if resp.ID != "list-new" {
		t.Fatalf("expected created list ID, got %q", resp.ID)
	}
}

func TestHandleListCreate_RequiresValidType(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routeLists, http.MethodPost, "/api/lists", `{"name":"Blocked","type":"bogus"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid list type, got %d", w.Code)
	}
}

func TestHandleListItems_AddAndRemove(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	add := doJSON(t, server.routeListItems, http.MethodPost, "/api/lists/list-1/items", `{"items":["new.example"]}`)
	if add.Code != http.StatusOK {
		t.Fatalf("expected 200 adding items, got %d (body: %s)", add.Code, add.Body.String())
	}
	decodeMutation(t, add)

	remove := doJSON(t, server.routeListItems, http.MethodDelete, "/api/lists/list-1/items", `{"items":["new.example"]}`)
	if remove.Code != http.StatusOK {
		t.Fatalf("expected 200 removing items, got %d (body: %s)", remove.Code, remove.Body.String())
	}
	decodeMutation(t, remove)
}

func TestHandleListDelete(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routeLists, http.MethodDelete, "/api/lists/list-1", "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	decodeMutation(t, w)
}
