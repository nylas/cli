package air

import (
	"cmp"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// Email Templates Handlers
// =============================================================================

// handleTemplates handles template CRUD operations.
func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listTemplates(w, r)
	case http.MethodPost:
		s.createTemplate(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTemplateByID handles single template operations.
func (s *Server) handleTemplateByID(w http.ResponseWriter, r *http.Request) {
	// Parse template ID from path: /api/templates/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/templates/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Template ID required", http.StatusBadRequest)
		return
	}
	templateID := parts[0]

	// Handle /api/templates/{id}/expand
	if len(parts) > 1 && parts[1] == "expand" {
		s.expandTemplate(w, r, templateID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getTemplate(w, r, templateID)
	case http.MethodPut:
		s.updateTemplate(w, r, templateID)
	case http.MethodDelete:
		s.deleteTemplate(w, r, templateID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listTemplates returns all templates.
func (s *Server) listTemplates(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	s.templatesMu.RLock()
	templates := make([]EmailTemplate, 0, len(s.emailTemplates))
	for _, t := range s.emailTemplates {
		if category == "" || t.Category == category {
			templates = append(templates, t)
		}
	}
	s.templatesMu.RUnlock()

	// Sort by usage count (most used first), then by name
	slices.SortFunc(templates, func(a, b EmailTemplate) int {
		if a.UsageCount != b.UsageCount {
			return cmp.Compare(b.UsageCount, a.UsageCount) // descending
		}
		return cmp.Compare(a.Name, b.Name) // ascending
	})

	// Add default templates if none exist
	if len(templates) == 0 {
		templates = defaultTemplates()
	}

	writeJSON(w, http.StatusOK, TemplateListResponse{
		Templates: templates,
		Total:     len(templates),
	})
}

// createTemplate creates a new template.
func (s *Server) createTemplate(w http.ResponseWriter, r *http.Request) {
	var template EmailTemplate
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&template); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if template.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Template name required"})
		return
	}
	if template.Body == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Template body required"})
		return
	}

	// Generate ID
	template.ID = "tmpl-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	template.CreatedAt = time.Now().Unix()
	template.UpdatedAt = template.CreatedAt
	template.UsageCount = 0

	// Extract variables from both body and subject, then deduplicate
	allVars := extractTemplateVariables(template.Body)
	if template.Subject != "" {
		allVars = append(allVars, extractTemplateVariables(template.Subject)...)
	}
	template.Variables = deduplicateStrings(allVars)

	func() {
		s.templatesMu.Lock()
		defer s.templatesMu.Unlock()
		if s.emailTemplates == nil {
			s.emailTemplates = make(map[string]EmailTemplate)
		}
		s.emailTemplates[template.ID] = template
	}()

	writeJSON(w, http.StatusCreated, template)
}

// getTemplate returns a single template.
func (s *Server) getTemplate(w http.ResponseWriter, _ *http.Request, templateID string) {
	s.templatesMu.RLock()
	template, exists := s.emailTemplates[templateID]
	s.templatesMu.RUnlock()

	if !exists {
		// Check default templates
		for _, t := range defaultTemplates() {
			if t.ID == templateID {
				writeJSON(w, http.StatusOK, t)
				return
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Template not found"})
		return
	}

	writeJSON(w, http.StatusOK, template)
}

// updateTemplate updates an existing template.
func (s *Server) updateTemplate(w http.ResponseWriter, r *http.Request, templateID string) {
	var update EmailTemplate
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&update); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	s.templatesMu.Lock()
	defer s.templatesMu.Unlock()

	template, exists := s.emailTemplates[templateID]
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Template not found"})
		return
	}

	// Update fields
	if update.Name != "" {
		template.Name = update.Name
	}
	if update.Subject != "" {
		template.Subject = update.Subject
	}
	if update.Body != "" {
		template.Body = update.Body
		template.Variables = extractTemplateVariables(template.Body)
	}
	if update.Shortcut != "" {
		template.Shortcut = update.Shortcut
	}
	if update.Category != "" {
		template.Category = update.Category
	}
	template.UpdatedAt = time.Now().Unix()

	s.emailTemplates[templateID] = template
	writeJSON(w, http.StatusOK, template)
}

// deleteTemplate deletes a template.
func (s *Server) deleteTemplate(w http.ResponseWriter, _ *http.Request, templateID string) {
	func() {
		s.templatesMu.Lock()
		defer s.templatesMu.Unlock()
		delete(s.emailTemplates, templateID)
	}()

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"id":      templateID,
	})
}

// expandTemplate expands a template with variables.
func (s *Server) expandTemplate(w http.ResponseWriter, r *http.Request, templateID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Variables map[string]string `json:"variables"`
	}
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	// Find template
	s.templatesMu.RLock()
	template, exists := s.emailTemplates[templateID]
	s.templatesMu.RUnlock()

	if !exists {
		// Check default templates
		for _, t := range defaultTemplates() {
			if t.ID == templateID {
				template = t
				exists = true
				break
			}
		}
	}

	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Template not found"})
		return
	}

	// Expand variables
	body := template.Body
	subject := template.Subject
	for key, value := range req.Variables {
		placeholder := "{{" + key + "}}"
		body = strings.ReplaceAll(body, placeholder, value)
		subject = strings.ReplaceAll(subject, placeholder, value)
	}

	// Increment usage count
	func() {
		s.templatesMu.Lock()
		defer s.templatesMu.Unlock()
		if t, ok := s.emailTemplates[templateID]; ok {
			t.UsageCount++
			s.emailTemplates[templateID] = t
		}
	}()

	writeJSON(w, http.StatusOK, map[string]any{
		"subject": subject,
		"body":    body,
	})
}
