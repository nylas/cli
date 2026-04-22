package air

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	configadapter "github.com/nylas/cli/internal/adapters/config"
	keyringadapter "github.com/nylas/cli/internal/adapters/keyring"
	authapp "github.com/nylas/cli/internal/app/auth"
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

func TestHandleListGrants_UsesSupportedFallbackWithoutMutatingDefault(t *testing.T) {
	t.Parallel()

	configStore := configadapter.NewMockConfigStore()
	configStore.SetConfig(&domain.Config{Region: "us", DefaultGrant: "grant-imap"})
	grantStore := &testGrantStore{
		grants: []domain.GrantInfo{
			{ID: "grant-imap", Email: "imap@example.com", Provider: domain.ProviderIMAP},
			{ID: "grant-google", Email: "google@example.com", Provider: domain.ProviderGoogle},
		},
		defaultGrant: "grant-imap",
	}
	server := &Server{
		grantStore:  grantStore,
		configStore: configStore,
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

	if resp.DefaultGrant != "grant-google" {
		t.Fatalf("expected supported default grant %q, got %q", "grant-google", resp.DefaultGrant)
	}
	if grantStore.defaultGrant != "grant-imap" {
		t.Fatalf("expected stored keyring default grant to remain %q, got %q", "grant-imap", grantStore.defaultGrant)
	}

	cfg, err := configStore.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DefaultGrant != "grant-imap" {
		t.Fatalf("expected config default grant to remain %q, got %q", "grant-imap", cfg.DefaultGrant)
	}
}

func TestHandleSetDefaultGrant_RejectsUnsupportedProviders(t *testing.T) {
	t.Parallel()

	server := &Server{
		grantStore: &testGrantStore{
			grants: []domain.GrantInfo{
				{ID: "grant-imap", Email: "imap@example.com", Provider: domain.ProviderIMAP},
			},
		},
		configStore: configadapter.NewMockConfigStore(),
	}

	body := `{"grant_id":"grant-imap"}`
	req := httptest.NewRequest(http.MethodPost, "/api/grants/default", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleSetDefaultGrant(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var resp SetDefaultGrantResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !strings.Contains(resp.Error, "not supported") {
		t.Fatalf("expected unsupported-provider error, got %q", resp.Error)
	}
}

func TestBuildPageData_UsesResolvedSupportedDefaultGrantWithoutPersistingIt(t *testing.T) {
	t.Parallel()

	configStore := configadapter.NewMockConfigStore()
	configStore.SetConfig(&domain.Config{Region: "us", DefaultGrant: "grant-imap"})
	secretStore := keyringadapter.NewMockSecretStore()
	grantStore := &testGrantStore{
		grants: []domain.GrantInfo{
			{ID: "grant-imap", Email: "imap@example.com", Provider: domain.ProviderIMAP},
			{ID: "grant-google", Email: "google@example.com", Provider: domain.ProviderGoogle},
		},
		defaultGrant: "grant-imap",
	}

	server := &Server{
		configSvc:   authapp.NewConfigService(configStore, secretStore),
		configStore: configStore,
		grantStore:  grantStore,
		hasAPIKey:   true,
	}

	data := server.buildPageData()

	if !data.Configured {
		t.Fatal("expected Air to be configured when a supported grant is available")
	}
	if data.DefaultGrantID != "grant-google" {
		t.Fatalf("expected resolved default grant %q, got %q", "grant-google", data.DefaultGrantID)
	}
	if data.UserEmail != "google@example.com" {
		t.Fatalf("expected resolved user email %q, got %q", "google@example.com", data.UserEmail)
	}
	if grantStore.defaultGrant != "grant-imap" {
		t.Fatalf("expected stored grant store default to remain %q, got %q", "grant-imap", grantStore.defaultGrant)
	}

	cfg, err := configStore.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DefaultGrant != "grant-imap" {
		t.Fatalf("expected config default grant to remain %q, got %q", "grant-imap", cfg.DefaultGrant)
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
