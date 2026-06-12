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

func TestHandlePolicyPatch_DefaultPolicyImmutable(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	// policy-1 is attached to the default workspace (workspace-1) in the mock,
	// making it the plan-ceiling policy: edits must be rejected server-side.
	w := doJSON(t, server.routePolicies, http.MethodPatch, "/api/policies/policy-1", `{"name":"Renamed"}`)

	if w.Code != http.StatusForbidden {
		t.Fatalf("default policy is the plan ceiling and must be immutable: expected 403, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestHandlePolicyDelete_DefaultPolicyImmutable(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routePolicies, http.MethodDelete, "/api/policies/policy-1", "")

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 deleting the plan-ceiling policy, got %d", w.Code)
	}
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

// ceilingClient gives the plan-ceiling policy (policy-1) explicit limits so
// validation against them can be exercised.
type ceilingClient struct {
	*nylasmock.MockClient
}

func (c *ceilingClient) GetPolicy(ctx context.Context, policyID string) (*domain.Policy, error) {
	daily := int64(500)
	return &domain.Policy{
		ID:   policyID,
		Name: "Default Policy",
		Limits: &domain.PolicyLimits{
			LimitCountDailyMessagePerGrant: &daily,
		},
	}, nil
}

func TestHandlePolicyCreate_RejectsLimitsAboveCeiling(t *testing.T) {
	t.Parallel()
	server := NewServer("127.0.0.1:0", &ceilingClient{MockClient: nylasmock.NewMockClient()})

	w := doJSON(t, server.routePolicies, http.MethodPost, "/api/policies",
		`{"name":"Over","limits":{"limit_count_daily_message_per_grant":600}}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("limits above the plan ceiling must be rejected: expected 400, got %d (body: %s)", w.Code, w.Body.String())
	}
	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "above_plan_ceiling" {
		t.Fatalf("expected structured above_plan_ceiling error, got %q", resp["error"])
	}
}

func TestHandlePolicyCreate_AllowsLimitsWithinCeiling(t *testing.T) {
	t.Parallel()
	server := NewServer("127.0.0.1:0", &ceilingClient{MockClient: nylasmock.NewMockClient()})

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
