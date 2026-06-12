package ui

import (
	"io/fs"
	"log/slog"
	"net/http"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Only handle root path
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Fall back to static file if templates not loaded
	if s.templates == nil {
		staticFS, _ := fs.Sub(staticFiles, "static")
		http.FileServer(http.FS(staticFS)).ServeHTTP(w, r)
		return
	}

	// Build page data
	data := s.buildPageData()

	// Render template. Template errors can include data field paths and
	// snippets of upstream content; log the raw error and return a
	// generic message to the client.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "base", data); err != nil {
		slog.Error("template render failed", "template", "base", "err", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// buildPageData gathers all data needed for the page.
func (s *Server) buildPageData() PageData {
	data := PageData{
		Commands: GetDefaultCommands(),
		DemoMode: s.demoMode,
	}

	// Demo mode: return sample data
	if s.demoMode {
		data.Configured = true
		data.ClientID = "demo-client-id"
		data.Region = "us"
		data.HasAPIKey = true
		data.DefaultGrant = demoDefaultGrant()
		data.Grants = demoGrants()
		data.DefaultGrantEmail = "alice@example.com"
		return data
	}

	// Get config status
	status, err := s.configSvc.GetStatus()
	if err == nil && status.IsConfigured {
		data.Configured = true
		data.ClientID = status.ClientID
		data.Region = status.Region
		data.HasAPIKey = status.HasAPIKey
		data.DefaultGrant = status.DefaultGrant
	}

	// Get grants and default grant from store
	grants, err := s.grantStore.ListGrants()
	if err == nil {
		// Get default grant ID from store (more reliable than config)
		defaultID, _ := s.grantStore.GetDefaultGrant()
		if defaultID != "" {
			data.DefaultGrant = defaultID
		}

		for _, g := range grants {
			data.Grants = append(data.Grants, grantFromDomain(g))

			// Set default grant email
			if g.ID == data.DefaultGrant {
				data.DefaultGrantEmail = g.Email
			}
		}
	}

	return data
}

// ConfigStatusResponse represents the config status API response.
