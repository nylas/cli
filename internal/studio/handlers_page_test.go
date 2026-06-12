package studio

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleIndex_ServesStudioPage(t *testing.T) {
	t.Parallel()

	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Agent Studio") {
		t.Fatal("index page must render the Agent Studio shell")
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected text/html content type, got %q", ct)
	}
	// webguard's CSP is script-src 'self' with no 'unsafe-inline': any inline
	// script would be silently blocked in the browser.
	if strings.Contains(body, "<script>") {
		t.Fatal("index page must not use inline scripts (blocked by CSP script-src 'self')")
	}
	for _, src := range []string{"/static/js/dom.js", "/static/js/api.js", "/static/js/modal.js", "/static/js/dragdrop.js", "/static/js/drawers.js", "/static/js/builders.js", "/static/js/accounts.js", "/static/js/board.js", "/static/css/studio.css"} {
		if !strings.Contains(body, src) {
			t.Fatalf("index page must reference %s", src)
		}
	}
	// Mount points the JS modules require.
	for _, id := range []string{`id="palette"`, `id="board"`, `id="accountsView"`, `id="drawer"`, `id="totals"`, `id="statusbar"`, `id="toast"`, `id="modal"`, `id="newMenu"`, `id="newBtn"`} {
		if !strings.Contains(body, id) {
			t.Fatalf("index page must contain %s", id)
		}
	}
	// The view switcher needs both tabs.
	for _, tab := range []string{`data-view="board"`, `data-view="accounts"`} {
		if !strings.Contains(body, tab) {
			t.Fatalf("index page must contain view tab %s", tab)
		}
	}
}

func TestHandleIndex_NotFoundForOtherPaths(t *testing.T) {
	t.Parallel()

	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for unknown path, got %d", w.Code)
	}
}

func TestHandleStatic_ServesAssets(t *testing.T) {
	t.Parallel()

	server := newTestServer()

	tests := []struct {
		path        string
		contentType string
		marker      string
	}{
		{"/static/js/board.js", "javascript", "/api/board"},
		{"/static/js/dom.js", "javascript", "textContent"},
		{"/static/js/api.js", "javascript", "/api/board"},
		{"/static/js/drawers.js", "javascript", "StudioDrawer"},
		{"/static/css/studio.css", "text/css", "Agent Studio"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			server.handleStatic(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200 for %s, got %d", tt.path, w.Code)
			}
			if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, tt.contentType) {
				t.Fatalf("expected %s content type for %s, got %q", tt.contentType, tt.path, ct)
			}
			if !strings.Contains(w.Body.String(), tt.marker) {
				t.Fatalf("%s must contain %q", tt.path, tt.marker)
			}
		})
	}
}
