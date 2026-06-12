package air

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

func newRulesPolicyTestServer(provider domain.Provider) *Server {
	return &Server{
		grantStore: &testGrantStore{
			grants: []domain.GrantInfo{{
				ID:       "grant-123",
				Email:    "managed@example.com",
				Provider: provider,
			}},
			defaultGrant: "grant-123",
		},
		nylasClient: nylasmock.NewMockClient(),
	}
}

func TestHandleListPolicies_NylasProvider(t *testing.T) {
	t.Parallel()

	server := newRulesPolicyTestServer(domain.ProviderNylas)

	req := httptest.NewRequest(http.MethodGet, "/api/policies", nil)
	w := httptest.NewRecorder()

	server.handleListPolicies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp PoliciesResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Policies) == 0 {
		t.Fatal("expected at least one policy")
	}
	if len(resp.Policies) != 1 {
		t.Fatalf("expected exactly one policy for the default agent account, got %d", len(resp.Policies))
	}
	if resp.Policies[0].ID != "policy-1" {
		t.Fatalf("expected policy-1, got %q", resp.Policies[0].ID)
	}
	if resp.Policies[0].Name == "" {
		t.Fatal("expected policy name to be populated")
	}
}

func TestHandleListRules_NylasProvider(t *testing.T) {
	t.Parallel()

	server := newRulesPolicyTestServer(domain.ProviderNylas)

	req := httptest.NewRequest(http.MethodGet, "/api/rules", nil)
	w := httptest.NewRecorder()

	server.handleListRules(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp RulesResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Rules) == 0 {
		t.Fatal("expected at least one rule")
	}
	if len(resp.Rules) != 1 {
		t.Fatalf("expected exactly one rule linked to the default policy, got %d", len(resp.Rules))
	}
	if resp.Rules[0].ID != "rule-1" {
		t.Fatalf("expected rule-1, got %q", resp.Rules[0].ID)
	}
	if resp.Rules[0].Trigger == "" {
		t.Fatal("expected rule trigger to be populated")
	}
}

// policyNotFoundClient simulates a workspace whose policy_id references a
// deleted policy: GetPolicy returns ErrPolicyNotFound.
type policyNotFoundClient struct {
	*nylasmock.MockClient
}

func (c *policyNotFoundClient) GetPolicy(ctx context.Context, policyID string) (*domain.Policy, error) {
	return nil, domain.ErrPolicyNotFound
}

func TestHandleListPolicies_DanglingPolicyReturnsEmpty(t *testing.T) {
	t.Parallel()

	server := newRulesPolicyTestServer(domain.ProviderNylas)
	server.nylasClient = &policyNotFoundClient{MockClient: nylasmock.NewMockClient()}

	req := httptest.NewRequest(http.MethodGet, "/api/policies", nil)
	w := httptest.NewRecorder()

	server.handleListPolicies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("dangling policy_id should render as empty state, not an error: expected status 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	var resp PoliciesResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Policies) != 0 {
		t.Fatalf("expected no policies for a dangling policy reference, got %d", len(resp.Policies))
	}
}

// workspaceNotFoundClient simulates an agent account whose workspace was
// deleted: GetWorkspace returns ErrWorkspaceNotFound.
type workspaceNotFoundClient struct {
	*nylasmock.MockClient
}

func (c *workspaceNotFoundClient) GetWorkspace(ctx context.Context, workspaceID string) (*domain.Workspace, error) {
	return nil, domain.ErrWorkspaceNotFound
}

func TestHandleRulesPolicy_DanglingWorkspaceDegradesGracefully(t *testing.T) {
	t.Parallel()

	server := newRulesPolicyTestServer(domain.ProviderNylas)
	server.nylasClient = &workspaceNotFoundClient{MockClient: nylasmock.NewMockClient()}

	t.Run("policies fall back to empty state", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/policies", nil)
		w := httptest.NewRecorder()

		server.handleListPolicies(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("a deleted workspace must not break the policies endpoint: expected status 200, got %d (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("rules fall back to empty state", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/rules", nil)
		w := httptest.NewRecorder()

		server.handleListRules(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("a deleted workspace must not break the rules endpoint: expected status 200, got %d (body: %s)", w.Code, w.Body.String())
		}
	})
}

// policyErrorClient simulates a transient failure fetching the policy.
type policyErrorClient struct {
	*nylasmock.MockClient
}

func (c *policyErrorClient) GetPolicy(ctx context.Context, policyID string) (*domain.Policy, error) {
	return nil, context.DeadlineExceeded
}

func TestHandleListPolicies_PolicyFetchErrorStays500(t *testing.T) {
	t.Parallel()

	server := newRulesPolicyTestServer(domain.ProviderNylas)
	server.nylasClient = &policyErrorClient{MockClient: nylasmock.NewMockClient()}

	req := httptest.NewRequest(http.MethodGet, "/api/policies", nil)
	w := httptest.NewRecorder()

	server.handleListPolicies(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("only ErrPolicyNotFound may degrade to an empty state; other errors must surface: expected status 500, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestHandleAgentWorkspace_NylasProvider(t *testing.T) {
	t.Parallel()

	server := newRulesPolicyTestServer(domain.ProviderNylas)

	req := httptest.NewRequest(http.MethodGet, "/api/workspace", nil)
	w := httptest.NewRecorder()

	server.handleAgentWorkspace(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	var resp WorkspaceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Workspace == nil {
		t.Fatal("expected workspace to be populated")
	}
	if resp.Workspace.ID != "workspace-1" {
		t.Fatalf("expected workspace-1, got %q", resp.Workspace.ID)
	}
	if resp.Workspace.PolicyID == "" {
		t.Fatal("expected workspace policy ID to be populated")
	}
	if len(resp.Workspace.RulesIDs) == 0 {
		t.Fatal("expected workspace rule IDs to be populated")
	}
}

func TestHandleAgentLists_NylasProvider(t *testing.T) {
	t.Parallel()

	server := newRulesPolicyTestServer(domain.ProviderNylas)

	req := httptest.NewRequest(http.MethodGet, "/api/lists", nil)
	w := httptest.NewRecorder()

	server.handleAgentLists(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	var resp AgentListsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Lists) != 1 {
		t.Fatalf("expected exactly one list, got %d", len(resp.Lists))
	}
	if resp.Lists[0].ID != "list-1" {
		t.Fatalf("expected list-1, got %q", resp.Lists[0].ID)
	}
	if resp.Lists[0].Type == "" {
		t.Fatal("expected list type to be populated")
	}
}

func TestHandleRulesPolicy_RejectsNonNylasProvider(t *testing.T) {
	t.Parallel()

	server := newRulesPolicyTestServer(domain.ProviderGoogle)

	tests := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		path    string
	}{
		{name: "policies", handler: server.handleListPolicies, path: "/api/policies"},
		{name: "rules", handler: server.handleListRules, path: "/api/rules"},
		{name: "workspace", handler: server.handleAgentWorkspace, path: "/api/workspace"},
		{name: "lists", handler: server.handleAgentLists, path: "/api/lists"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			tt.handler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", w.Code)
			}

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if resp["error"] != rulesPolicyUnsupportedMessage {
				t.Fatalf("expected unsupported provider message %q, got %q", rulesPolicyUnsupportedMessage, resp["error"])
			}
		})
	}
}

func TestBaseTemplate_PolicyRulesEntryIsEmailScoped(t *testing.T) {
	t.Parallel()

	templates, err := loadTemplates()
	if err != nil {
		t.Fatalf("load templates: %v", err)
	}

	tests := []struct {
		name        string
		provider    string
		expectEntry bool
		expectView  bool
	}{
		{name: "nylas provider", provider: string(domain.ProviderNylas), expectEntry: true, expectView: true},
		{name: "google provider", provider: string(domain.ProviderGoogle), expectEntry: false, expectView: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			data := PageData{
				Configured: true,
				Provider:   tt.provider,
				UserAvatar: "N",
				UserEmail:  "managed@example.com",
			}

			if err := templates.ExecuteTemplate(&out, "base", data); err != nil {
				t.Fatalf("render template: %v", err)
			}

			html := out.String()
			hasEntry := strings.Contains(html, `data-testid="email-policy-rules-trigger"`)
			hasNavTab := strings.Contains(html, `data-testid="nav-tab-rules-policy"`)
			hasView := strings.Contains(html, `data-testid="rules-policy-view"`)
			hasLabel := strings.Contains(html, `Policy &amp; Rules`)
			hasAccountEmailAttr := strings.Contains(html, `data-account-email="`+data.UserEmail+`"`)
			hasGrantIDAttr := strings.Contains(html, `data-grant-id="`+data.DefaultGrantID+`"`)

			if hasEntry != tt.expectEntry {
				t.Fatalf("expected email entry presence %t, got %t", tt.expectEntry, hasEntry)
			}
			if hasNavTab {
				t.Fatal("expected Policy & Rules to stay out of top-level navigation")
			}
			if hasView != tt.expectView {
				t.Fatalf("expected view presence %t, got %t", tt.expectView, hasView)
			}
			if hasLabel != tt.expectEntry {
				t.Fatalf("expected Policy & Rules label presence %t, got %t", tt.expectEntry, hasLabel)
			}
			if hasAccountEmailAttr != tt.expectView {
				t.Fatalf("expected account email data attribute presence %t, got %t", tt.expectView, hasAccountEmailAttr)
			}
			if hasGrantIDAttr != tt.expectView {
				t.Fatalf("expected grant id data attribute presence %t, got %t", tt.expectView, hasGrantIDAttr)
			}
		})
	}
}
