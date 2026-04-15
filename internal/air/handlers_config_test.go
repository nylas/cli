package air

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// Helper to create demo server for handler tests
func newTestDemoServer() *Server {
	return NewDemoServer(":7365")
}

// ================================
// CONFIG HANDLER TESTS
// ================================

func TestHandleConfigStatus_DemoMode(t *testing.T) {
	t.Parallel()

	server := newTestDemoServer()

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	server.handleConfigStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp ConfigStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Configured {
		t.Error("expected Configured to be true in demo mode")
	}

	if resp.Region != "us" {
		t.Errorf("expected Region 'us', got %s", resp.Region)
	}

	if !resp.HasAPIKey {
		t.Error("expected HasAPIKey to be true in demo mode")
	}

	if resp.GrantCount != 3 {
		t.Errorf("expected GrantCount 3, got %d", resp.GrantCount)
	}
}

func TestHandleConfigStatus_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	server := newTestDemoServer()

	req := httptest.NewRequest(http.MethodPost, "/api/config", nil)
	w := httptest.NewRecorder()

	server.handleConfigStatus(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// ================================
// GRANTS HANDLER TESTS
// ================================

func TestHandleListGrants_DemoMode(t *testing.T) {
	t.Parallel()

	server := newTestDemoServer()

	req := httptest.NewRequest(http.MethodGet, "/api/grants", nil)
	w := httptest.NewRecorder()

	server.handleListGrants(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp GrantsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Grants) != 3 {
		t.Errorf("expected 3 grants, got %d", len(resp.Grants))
	}

	if resp.DefaultGrant != "demo-grant-001" {
		t.Errorf("expected default grant 'demo-grant-001', got %s", resp.DefaultGrant)
	}
}

func TestHandleSetDefaultGrant_DemoMode(t *testing.T) {
	t.Parallel()

	server := newTestDemoServer()

	body := `{"grant_id": "demo-grant-002"}`
	req := httptest.NewRequest(http.MethodPost, "/api/grants/default", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleSetDefaultGrant(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp SetDefaultGrantResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("expected Success to be true")
	}
}

func TestHandleGrants_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	server := newTestDemoServer()

	req := httptest.NewRequest(http.MethodDelete, "/api/grants", nil)
	w := httptest.NewRecorder()

	server.handleListGrants(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandleSetDefaultGrant_InvalidJSON(t *testing.T) {
	t.Parallel()

	server := newTestDemoServer()

	req := httptest.NewRequest(http.MethodPost, "/api/grants/default", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleSetDefaultGrant(w, req)

	if w.Code != http.StatusBadRequest && w.Code != http.StatusOK {
		t.Errorf("expected status 400 or 200, got %d", w.Code)
	}
}

func TestHandleListGrants_IncludesNylasProviders(t *testing.T) {
	t.Parallel()

	server := &Server{
		grantStore: &testGrantStore{
			grants: []domain.GrantInfo{
				{ID: "grant-google", Email: "google@example.com", Provider: domain.ProviderGoogle},
				{ID: "grant-nylas", Email: "nylas@example.com", Provider: domain.ProviderNylas},
				{ID: "grant-imap", Email: "imap@example.com", Provider: domain.ProviderIMAP},
			},
			defaultGrant: "grant-nylas",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/grants", nil)
	w := httptest.NewRecorder()

	server.handleListGrants(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp GrantsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Grants) != 2 {
		t.Fatalf("expected 2 supported grants, got %d", len(resp.Grants))
	}

	providers := make(map[string]bool, len(resp.Grants))
	for _, g := range resp.Grants {
		providers[g.Provider] = true
	}

	if !providers[string(domain.ProviderGoogle)] {
		t.Error("expected google grant to be included")
	}
	if !providers[string(domain.ProviderNylas)] {
		t.Error("expected nylas grant to be included")
	}
	if providers[string(domain.ProviderIMAP)] {
		t.Error("did not expect imap grant to be included")
	}

	if resp.DefaultGrant != "grant-nylas" {
		t.Errorf("expected default grant 'grant-nylas', got %s", resp.DefaultGrant)
	}
}

// ================================
// FOLDERS HANDLER TESTS
// ================================

func TestHandleListFolders_DemoMode(t *testing.T) {
	t.Parallel()

	server := newTestDemoServer()

	req := httptest.NewRequest(http.MethodGet, "/api/folders", nil)
	w := httptest.NewRecorder()

	server.handleListFolders(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp FoldersResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Folders) == 0 {
		t.Error("expected non-empty folders")
	}

	// Check for standard folders
	hasInbox := false
	for _, f := range resp.Folders {
		if f.SystemFolder == "inbox" {
			hasInbox = true
			break
		}
	}
	if !hasInbox {
		t.Error("expected inbox folder")
	}
}

func TestHandleFolders_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	server := newTestDemoServer()

	req := httptest.NewRequest(http.MethodPost, "/api/folders", nil)
	w := httptest.NewRecorder()

	server.handleListFolders(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}
