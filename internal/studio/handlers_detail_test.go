package studio

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestHandlePolicyGet(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routePolicies, http.MethodGet, "/api/policies/policy-1", "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	var policy domain.Policy
	if err := json.NewDecoder(w.Body).Decode(&policy); err != nil {
		t.Fatalf("decode policy: %v", err)
	}
	if policy.ID != "policy-1" {
		t.Fatalf("expected policy-1, got %q", policy.ID)
	}
}

func TestHandleListItemsGet(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.routeListItems, http.MethodGet, "/api/lists/list-1/items", "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	var resp struct {
		Items []string `json:"items"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode items: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items from mock, got %d", len(resp.Items))
	}
}

func TestHandleTestEmail(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.handleTestEmail, http.MethodPost, "/api/actions/test-email", `{"grant_id":"agent-1"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "sent" {
		t.Fatalf("expected sent status, got %q", resp["status"])
	}
}

func TestHandleTestEmail_CooldownPerGrant(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	first := doJSON(t, server.handleTestEmail, http.MethodPost, "/api/actions/test-email", `{"grant_id":"agent-1"}`)
	if first.Code != http.StatusOK {
		t.Fatalf("first send should succeed, got %d", first.Code)
	}

	second := doJSON(t, server.handleTestEmail, http.MethodPost, "/api/actions/test-email", `{"grant_id":"agent-1"}`)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("immediate repeat send must hit the cooldown: expected 429, got %d", second.Code)
	}

	other := doJSON(t, server.handleTestEmail, http.MethodPost, "/api/actions/test-email", `{"grant_id":"agent-2"}`)
	if other.Code != http.StatusOK {
		t.Fatalf("cooldown is per grant; a different grant should succeed, got %d", other.Code)
	}
}

func TestHandleTestEmail_RequiresGrantID(t *testing.T) {
	t.Parallel()
	server := newTestServer()

	w := doJSON(t, server.handleTestEmail, http.MethodPost, "/api/actions/test-email", `{}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
