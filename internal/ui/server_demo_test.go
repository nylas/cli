package ui

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// Demo Mode Handler Tests
// =============================================================================

func TestHandleConfigStatus_DemoMode(t *testing.T) {
	t.Parallel()

	server := NewDemoServer(":0")

	req := httptest.NewRequest(http.MethodGet, "/api/config/status", nil)
	w := httptest.NewRecorder()

	server.handleConfigStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp ConfigStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Configured {
		t.Error("Expected Configured to be true in demo mode")
	}
	if resp.Region != "us" {
		t.Errorf("Expected Region 'us', got %q", resp.Region)
	}
	if resp.ClientID != "demo-client-id" {
		t.Errorf("Expected ClientID 'demo-client-id', got %q", resp.ClientID)
	}
	if !resp.HasAPIKey {
		t.Error("Expected HasAPIKey to be true in demo mode")
	}
	if resp.GrantCount != 3 {
		t.Errorf("Expected GrantCount 3, got %d", resp.GrantCount)
	}
	if resp.DefaultGrant != "demo-grant-001" {
		t.Errorf("Expected DefaultGrant 'demo-grant-001', got %q", resp.DefaultGrant)
	}
}

func TestHandleListGrants_DemoMode(t *testing.T) {
	t.Parallel()

	server := NewDemoServer(":0")

	req := httptest.NewRequest(http.MethodGet, "/api/grants", nil)
	w := httptest.NewRecorder()

	server.handleListGrants(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp GrantsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Grants) != 3 {
		t.Errorf("Expected 3 demo grants, got %d", len(resp.Grants))
	}
	if resp.DefaultGrant != "demo-grant-001" {
		t.Errorf("Expected DefaultGrant 'demo-grant-001', got %q", resp.DefaultGrant)
	}

	// Verify grant data
	if resp.Grants[0].Email != "alice@example.com" {
		t.Errorf("Expected first grant email 'alice@example.com', got %q", resp.Grants[0].Email)
	}
}

func TestHandleSetDefaultGrant_DemoMode(t *testing.T) {
	t.Parallel()

	server := NewDemoServer(":0")

	body, _ := json.Marshal(SetDefaultGrantRequest{GrantID: "demo-grant-002"})
	req := httptest.NewRequest(http.MethodPost, "/api/grants/default", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleSetDefaultGrant(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp SetDefaultGrantResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success to be true in demo mode")
	}
	if !strings.Contains(resp.Message, "demo mode") {
		t.Errorf("Expected message to mention demo mode, got: %s", resp.Message)
	}
}

func TestHandleConfigSetup_DemoMode(t *testing.T) {
	t.Parallel()

	server := NewDemoServer(":0")

	body, _ := json.Marshal(SetupRequest{APIKey: "test-key", Region: "us"})
	req := httptest.NewRequest(http.MethodPost, "/api/config/setup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleConfigSetup(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp SetupResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success to be true in demo mode")
	}
	if !strings.Contains(resp.Message, "Demo mode") {
		t.Errorf("Expected message to mention demo mode, got: %s", resp.Message)
	}
	if resp.ClientID != "demo-client-id" {
		t.Errorf("Expected ClientID 'demo-client-id', got %q", resp.ClientID)
	}
	if len(resp.Applications) != 1 {
		t.Errorf("Expected 1 demo application, got %d", len(resp.Applications))
	}
	if len(resp.Grants) != 3 {
		t.Errorf("Expected 3 demo grants, got %d", len(resp.Grants))
	}
}

func TestHandleExecCommand_DemoMode(t *testing.T) {
	t.Parallel()

	server := NewDemoServer(":0")

	tests := []struct {
		name     string
		command  string
		contains string
	}{
		{"email list", "email list", "Demo Mode"},
		{"calendar events", "calendar events", "Demo Mode"},
		{"auth status", "auth status", "Demo Mode"},
		{"version", "version", "demo mode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(ExecRequest{Command: tt.command})
			req := httptest.NewRequest(http.MethodPost, "/api/exec", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleExecCommand(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			var resp ExecResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if resp.Error != "" {
				t.Errorf("Unexpected error: %s", resp.Error)
			}
			if !strings.Contains(resp.Output, tt.contains) {
				t.Errorf("Expected output to contain %q, got: %s", tt.contains, resp.Output)
			}
		})
	}
}

// =============================================================================
// buildPageData Tests
// =============================================================================

func TestBuildPageData_DemoMode(t *testing.T) {
	t.Parallel()

	server := NewDemoServer(":0")
	data := server.buildPageData()

	if !data.DemoMode {
		t.Error("Expected DemoMode to be true")
	}
	if !data.Configured {
		t.Error("Expected Configured to be true in demo mode")
	}
	if data.ClientID != "demo-client-id" {
		t.Errorf("Expected ClientID 'demo-client-id', got %q", data.ClientID)
	}
	if data.Region != "us" {
		t.Errorf("Expected Region 'us', got %q", data.Region)
	}
	if !data.HasAPIKey {
		t.Error("Expected HasAPIKey to be true")
	}
	if data.DefaultGrant != "demo-grant-001" {
		t.Errorf("Expected DefaultGrant 'demo-grant-001', got %q", data.DefaultGrant)
	}
	if len(data.Grants) != 3 {
		t.Errorf("Expected 3 grants, got %d", len(data.Grants))
	}
	if data.DefaultGrantEmail != "alice@example.com" {
		t.Errorf("Expected DefaultGrantEmail 'alice@example.com', got %q", data.DefaultGrantEmail)
	}

	// Verify commands are loaded
	if len(data.Commands.Auth) == 0 {
		t.Error("Expected commands to be loaded")
	}
}

// =============================================================================
// Demo Helper Function Tests
// =============================================================================

func TestDemoGrants(t *testing.T) {
	t.Parallel()

	grants := demoGrants()

	if len(grants) != 3 {
		t.Errorf("Expected 3 demo grants, got %d", len(grants))
	}

	// Check first grant
	if grants[0].ID != "demo-grant-001" {
		t.Errorf("Expected first grant ID 'demo-grant-001', got %q", grants[0].ID)
	}
	if grants[0].Email != "alice@example.com" {
		t.Errorf("Expected first grant email 'alice@example.com', got %q", grants[0].Email)
	}
	if grants[0].Provider != "google" {
		t.Errorf("Expected first grant provider 'google', got %q", grants[0].Provider)
	}

	// Check second grant has different provider
	if grants[1].Provider != "microsoft" {
		t.Errorf("Expected second grant provider 'microsoft', got %q", grants[1].Provider)
	}
}

func TestDemoDefaultGrant(t *testing.T) {
	t.Parallel()

	result := demoDefaultGrant()
	if result != "demo-grant-001" {
		t.Errorf("Expected 'demo-grant-001', got %q", result)
	}
}

// =============================================================================
// NewDemoServer Tests
// =============================================================================

func TestNewDemoServer(t *testing.T) {
	t.Parallel()

	server := NewDemoServer(":8080")

	if server == nil {
		t.Fatal("NewDemoServer returned nil")
	}
	if server.addr != ":8080" {
		t.Errorf("Expected addr ':8080', got %q", server.addr)
	}
	if !server.demoMode {
		t.Error("Expected demoMode to be true")
	}
	if server.templates == nil {
		t.Error("Expected templates to be loaded")
	}
	// Demo mode doesn't use real stores
	if server.configSvc != nil {
		t.Error("Expected configSvc to be nil in demo mode")
	}
	if server.grantStore != nil {
		t.Error("Expected grantStore to be nil in demo mode")
	}
}

// =============================================================================
// handleIndex Tests
// =============================================================================

func TestHandleIndex_RootPath(t *testing.T) {
	t.Parallel()

	server := NewDemoServer(":0")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for root path, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected Content-Type to contain 'text/html', got %q", contentType)
	}
}

func TestHandleIndex_NonRootPath(t *testing.T) {
	t.Parallel()

	server := NewDemoServer(":0")

	req := httptest.NewRequest(http.MethodGet, "/some/other/path", nil)
	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for non-root path, got %d", http.StatusNotFound, w.Code)
	}
}

// =============================================================================
// limitedBody Tests
// =============================================================================

func TestLimitedBody(t *testing.T) {
	t.Parallel()

	// Test with small body (should work)
	smallBody := strings.NewReader(`{"command": "email list"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/exec", smallBody)
	w := httptest.NewRecorder()

	reader := limitedBody(w, req)
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read limited body: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected data from limited body")
	}
}

// =============================================================================
// Static Files Embedding Tests
// =============================================================================

func TestStaticFilesEmbedded(t *testing.T) {
	t.Parallel()

	// Verify that critical static files are embedded
	expectedFiles := []string{
		"static/js/commands.js",      // Main entry point
		"static/js/commands-core.js", // Core cache and parsing
		"static/js/commands-auth.js",
		"static/js/commands-email.js",
		"static/js/commands-calendar.js",
		"static/css/base.css",
		"static/css/commands.css",
	}

	for _, path := range expectedFiles {
		t.Run(path, func(t *testing.T) {
			data, err := staticFiles.ReadFile(path)
			if err != nil {
				t.Errorf("Failed to read embedded file %q: %v", path, err)
				return
			}
			if len(data) == 0 {
				t.Errorf("Embedded file %q is empty", path)
			}
		})
	}
}

func TestCommandsJSContainsRequiredFlags(t *testing.T) {
	t.Parallel()

	// Read all command JS files and combine content
	commandFiles := []string{
		"static/js/commands-email.js",
		"static/js/commands-calendar.js",
		"static/js/commands-webhook.js",
		"static/js/commands-contacts.js",
		"static/js/commands-scheduler.js",
	}

	var allContent strings.Builder
	for _, path := range commandFiles {
		data, err := staticFiles.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", path, err)
		}
		allContent.Write(data)
	}
	content := allContent.String()

	// Verify key commands have flags defined
	flagPatterns := []struct {
		command  string
		contains string
	}{
		{"email send", "required: true"},
		{"calendar events create", "flags:"},
		{"webhook create", "flags:"},
		{"contacts create", "flags:"},
		{"scheduler configurations create", "flags:"},
		{"calendar virtual create", "email"},
	}

	for _, tt := range flagPatterns {
		t.Run(tt.command, func(t *testing.T) {
			if !strings.Contains(content, tt.contains) {
				t.Errorf("command files should contain %q for %s", tt.contains, tt.command)
			}
		})
	}
}

func TestCommandsJSContainsNoDashboardOldURL(t *testing.T) {
	t.Parallel()

	// Check all command JS files for old dashboard URL
	commandFiles := []string{
		"static/js/commands.js",
		"static/js/commands-core.js",
		"static/js/commands-auth.js",
		"static/js/commands-email.js",
		"static/js/commands-calendar.js",
		"static/js/commands-contacts.js",
		"static/js/commands-scheduler.js",
		"static/js/commands-inbound.js",
		"static/js/commands-timezone.js",
		"static/js/commands-webhook.js",
		"static/js/commands-otp.js",
		"static/js/commands-admin.js",
		"static/js/commands-notetaker.js",
	}

	for _, path := range commandFiles {
		data, err := staticFiles.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", path, err)
		}
		content := string(data)

		// Verify old dashboard URL is not present
		if strings.Contains(content, "dashboard.nylas.com") && !strings.Contains(content, "dashboard-v3.nylas.com") {
			t.Errorf("%s contains old dashboard URL (dashboard.nylas.com instead of dashboard-v3.nylas.com)", path)
		}
	}
}
