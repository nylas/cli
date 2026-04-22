package air

import (
	"html/template"
	"net/http"
	"sync"
)

// handleIndex renders the main page.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Only handle root path
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Fall back to static file if templates not loaded
	if s.templates == nil {
		http.Error(w, "Templates not loaded", http.StatusInternalServerError)
		return
	}

	// Build page data - use real data when available, fall back to mock
	data := s.buildPageData()

	// Render template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// buildPageData gathers all data needed for the page.
func (s *Server) buildPageData() PageData {
	// Start with mock data as base
	data := buildMockPageData()

	// Demo mode: return mock data
	if s.demoMode {
		return data
	}

	// Non-demo mode: clear mock data so JavaScript loads real data
	// This prevents the "flash" of mock data before real data loads
	data.Emails = nil
	data.SelectedEmail = nil
	data.Events = nil
	data.Calendars = nil
	data.Contacts = nil

	// Get real config status - use same logic as /api/config handler
	// s.hasAPIKey is set during server initialization from keyring
	status, err := s.configSvc.GetStatus()
	if err == nil {
		data.ClientID = status.ClientID
		data.Region = status.Region
		data.HasAPIKey = s.hasAPIKey || status.HasAPIKey
	}

	// Get real grants filtered to providers supported by Air.
	supportedGrants, err := s.listSupportedGrants()
	data.Configured = data.HasAPIKey && err == nil && len(supportedGrants) > 0

	if len(supportedGrants) > 0 {
		defaultGrant, defaultErr := s.resolveDefaultGrantInfo()

		if defaultErr == nil {
			data.UserEmail = defaultGrant.Email
			data.UserName = extractName(defaultGrant.Email)
			data.UserAvatar = initials(defaultGrant.Email)
			data.DefaultGrantID = defaultGrant.ID
			data.Provider = string(defaultGrant.Provider)
		}

		// Build grants list for UI (supported providers only)
		data.Grants = make([]GrantInfo, 0, len(supportedGrants))
		for _, g := range supportedGrants {
			data.Grants = append(data.Grants, GrantInfo{
				ID:        g.ID,
				Email:     g.Email,
				Provider:  string(g.Provider),
				IsDefault: defaultErr == nil && g.ID == defaultGrant.ID,
			})
		}
		data.AccountsCount = len(supportedGrants)
	}

	return data
}

// extractName extracts a display name from an email address.
func extractName(email string) string {
	// Simple extraction: use the part before @ and capitalize
	for i, c := range email {
		if c == '@' {
			name := email[:i]
			// Capitalize first letter
			if len(name) > 0 {
				return string(name[0]-32) + name[1:]
			}
			return name
		}
	}
	return email
}

// initials returns the initials from an email address.
func initials(email string) string {
	// Get first letter of email
	if len(email) == 0 {
		return "?"
	}
	// Uppercase first letter
	c := email[0]
	if c >= 'a' && c <= 'z' {
		c -= 32
	}
	return string(c)
}

// loadTemplates parses all template files. Uses sync.Once since templates are
// embedded via embed.FS and cannot change at runtime; errors are cached permanently.
func loadTemplates() (*template.Template, error) {
	templatesOnce.Do(func() {
		parsedTemplates, parsedTemplatesErr = template.New("").Funcs(templateFuncs).ParseFS(
			templateFiles,
			"templates/*.gohtml",
			"templates/partials/*.gohtml",
			"templates/pages/*.gohtml",
		)
	})

	return parsedTemplates, parsedTemplatesErr
}

// Template functions.
var templateFuncs = template.FuncMap{
	"safeHTML": func(s string) template.HTML {
		//nolint:gosec // G203: We control the input, this is for rendering pre-defined HTML
		return template.HTML(s)
	},
}

var (
	templatesOnce      sync.Once
	parsedTemplates    *template.Template
	parsedTemplatesErr error
)
