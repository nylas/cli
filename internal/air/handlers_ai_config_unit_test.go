package air

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandleGetAIUsage_ZeroBudget verifies that a zero usage budget does not
// produce +Inf percentUsed (which encoding/json would refuse to marshal,
// surfacing as a 500 to the client).
func TestHandleGetAIUsage_ZeroBudget(t *testing.T) {
	srv := &Server{}

	aiStore.mu.Lock()
	prevBudget := aiStore.config.UsageBudget
	prevSpent := aiStore.config.UsageSpent
	aiStore.config.UsageBudget = 0
	aiStore.config.UsageSpent = 1.0
	aiStore.mu.Unlock()
	t.Cleanup(func() {
		aiStore.mu.Lock()
		aiStore.config.UsageBudget = prevBudget
		aiStore.config.UsageSpent = prevSpent
		aiStore.mu.Unlock()
	})

	req := httptest.NewRequest(http.MethodGet, "/api/ai/usage", nil)
	w := httptest.NewRecorder()
	srv.handleGetAIUsage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	pct, ok := resp["percentUsed"].(float64)
	if !ok {
		t.Fatalf("percentUsed missing or not a number: %#v", resp["percentUsed"])
	}
	if math.IsInf(pct, 0) || math.IsNaN(pct) {
		t.Fatalf("percentUsed should be finite for zero budget, got %v", pct)
	}
	if pct != 0 {
		t.Errorf("expected 0 percent used when budget is zero, got %v", pct)
	}
}

// TestHandleUpdateAIConfig_RejectsMaskedKey verifies that posting back the
// masked API key returned by GET (e.g. "***1234") does NOT overwrite the
// stored real key. Previously the check was "!= \"***\"", which let any
// "***xxxx" payload through.
func TestHandleUpdateAIConfig_RejectsMaskedKey(t *testing.T) {
	srv := &Server{}

	aiStore.mu.Lock()
	prev := aiStore.config.APIKey
	aiStore.config.APIKey = "secret-real-key-1234"
	aiStore.mu.Unlock()
	t.Cleanup(func() {
		aiStore.mu.Lock()
		aiStore.config.APIKey = prev
		aiStore.mu.Unlock()
	})

	body := `{"apiKey": "***1234"}`
	req := httptest.NewRequest(http.MethodPut, "/api/ai/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleUpdateAIConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	aiStore.mu.RLock()
	defer aiStore.mu.RUnlock()
	if aiStore.config.APIKey != "secret-real-key-1234" {
		t.Errorf("masked payload overwrote real API key: now %q", aiStore.config.APIKey)
	}
}

// TestHandleUpdateAIConfig_AcceptsRealKey verifies that a non-masked key
// still updates the stored value.
func TestHandleUpdateAIConfig_AcceptsRealKey(t *testing.T) {
	srv := &Server{}

	aiStore.mu.Lock()
	prev := aiStore.config.APIKey
	aiStore.config.APIKey = "old-key"
	aiStore.mu.Unlock()
	t.Cleanup(func() {
		aiStore.mu.Lock()
		aiStore.config.APIKey = prev
		aiStore.mu.Unlock()
	})

	body := `{"apiKey": "new-real-key-abcd"}`
	req := httptest.NewRequest(http.MethodPut, "/api/ai/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleUpdateAIConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	aiStore.mu.RLock()
	defer aiStore.mu.RUnlock()
	if aiStore.config.APIKey != "new-real-key-abcd" {
		t.Errorf("real key was not stored: got %q", aiStore.config.APIKey)
	}
}
