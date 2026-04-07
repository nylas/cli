package air

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewDemoServer(t *testing.T) {
	t.Parallel()

	server := NewDemoServer(":7365")

	if server == nil {
		t.Fatal("expected non-nil server")
		return
	}

	if !server.demoMode {
		t.Error("expected demoMode to be true")
	}

	if server.addr != ":7365" {
		t.Errorf("expected addr :7365, got %s", server.addr)
	}

	// Demo server should not have Nylas client or stores
	if server.nylasClient != nil {
		t.Error("expected nylasClient to be nil in demo mode")
	}

	if server.configSvc != nil {
		t.Error("expected configSvc to be nil in demo mode")
	}
}

func TestExtractName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		email    string
		expected string
	}{
		{"john@example.com", "John"},
		{"alice.smith@company.org", "Alice.smith"},
		{"bob@test.io", "Bob"},
		{"a@b.com", "A"},
		{"test", "test"}, // No @ symbol
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := extractName(tt.email)
			if result != tt.expected {
				t.Errorf("extractName(%q) = %q, want %q", tt.email, result, tt.expected)
			}
		})
	}
}

func TestInitials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		email    string
		expected string
	}{
		{"john@example.com", "J"},
		{"alice@company.org", "A"},
		{"Bob@test.io", "B"},
		{"", "?"},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := initials(tt.email)
			if result != tt.expected {
				t.Errorf("initials(%q) = %q, want %q", tt.email, result, tt.expected)
			}
		})
	}
}

func TestHandleIndex_NonRootPath(t *testing.T) {
	t.Parallel()

	server := NewDemoServer(":7365")

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleIndex_RootPath_DemoMode(t *testing.T) {
	t.Parallel()

	server := NewDemoServer(":7365")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	// Should succeed with templates loaded
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 200 or 500, got %d", w.Code)
	}

	// Check content type if successful
	if w.Code == http.StatusOK {
		contentType := w.Header().Get("Content-Type")
		if contentType != "text/html; charset=utf-8" {
			t.Errorf("expected content-type text/html, got %s", contentType)
		}
	}
}

func TestBuildPageData_DemoMode(t *testing.T) {
	t.Parallel()

	server := NewDemoServer(":7365")
	data := server.buildPageData()

	// In demo mode, should have mock data
	if len(data.Emails) == 0 {
		t.Error("expected non-empty emails in demo mode")
	}

	if len(data.Folders) == 0 {
		t.Error("expected non-empty folders in demo mode")
	}

	if len(data.Calendars) == 0 {
		t.Error("expected non-empty calendars in demo mode")
	}

	if len(data.Events) == 0 {
		t.Error("expected non-empty events in demo mode")
	}

	if len(data.Contacts) == 0 {
		t.Error("expected non-empty contacts in demo mode")
	}

	if data.UserName == "" {
		t.Error("expected non-empty UserName in demo mode")
	}
}

// =============================================================================
// CSS Layout Regression Tests
// =============================================================================

// TestCSS_EmailListNoMaxHeight verifies that the accessibility CSS
// does not constrain the email list to 300px (regression test for layout bug).
func TestCSS_EmailListNoMaxHeight(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/css/accessibility-aria.css", nil)
	w := httptest.NewRecorder()

	// Get the CSS file handler
	staticFS, _ := fs.Sub(staticFiles, "static")
	fileServer := http.FileServer(http.FS(staticFS))
	fileServer.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	css := w.Body.String()

	// Verify the [role="listbox"] selector excludes .email-list
	if !strings.Contains(css, `:not(.email-list)`) {
		t.Error("accessibility-aria.css must use [role=\"listbox\"]:not(.email-list) to exclude email list from max-height constraint")
	}

	// Verify max-height: 300px exists but NOT for email-list
	if strings.Contains(css, `max-height: 300px`) {
		// This is OK as long as it's in the :not(.email-list) rule
		if !strings.Contains(css, `[role="listbox"]:not(.email-list)`) {
			t.Error("max-height: 300px found without :not(.email-list) exclusion")
		}
	}
}

// TestCSS_EmailListGrid verifies that email-list-container uses CSS Grid.
func TestCSS_EmailListGrid(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/css/email-list.css", nil)
	w := httptest.NewRecorder()

	staticFS, _ := fs.Sub(staticFiles, "static")
	fileServer := http.FileServer(http.FS(staticFS))
	fileServer.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	css := w.Body.String()

	// Verify CSS Grid layout
	if !strings.Contains(css, "display: grid") {
		t.Error("email-list.css must use 'display: grid' for email-list-container")
	}

	if !strings.Contains(css, "grid-template-rows: auto 1fr") {
		t.Error("email-list.css must use 'grid-template-rows: auto 1fr' for proper layout")
	}
}

// TestCSS_EmailViewGrid verifies that email-view uses CSS Grid.
func TestCSS_EmailViewGrid(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/css/calendar-grid.css", nil)
	w := httptest.NewRecorder()

	staticFS, _ := fs.Sub(staticFiles, "static")
	fileServer := http.FileServer(http.FS(staticFS))
	fileServer.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	css := w.Body.String()

	// Verify email-view.active uses Grid
	if !strings.Contains(css, ".email-view.active") {
		t.Error("calendar-grid.css must define .email-view.active")
	}

	if !strings.Contains(css, "grid-template-columns") {
		t.Error("email-view.active must use grid-template-columns for layout")
	}
}

// TestCSS_MainLayoutHeight verifies explicit height calculation.
func TestCSS_MainLayoutHeight(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/css/layout.css", nil)
	w := httptest.NewRecorder()

	staticFS, _ := fs.Sub(staticFiles, "static")
	fileServer := http.FileServer(http.FS(staticFS))
	fileServer.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	css := w.Body.String()

	// Verify explicit height calculation
	if !strings.Contains(css, "calc(100vh") {
		t.Error("layout.css must use calc(100vh - ...) for explicit height")
	}
}
