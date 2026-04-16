package air

import (
	"net/http"
	"strings"

	"github.com/nylas/cli/internal/air/cache"
	"github.com/nylas/cli/internal/domain"
)

// handleContactsRoute handles /api/contacts: GET (list) and POST (create).
func (s *Server) handleContactsRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListContacts(w, r)
	case http.MethodPost:
		s.handleCreateContact(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListContacts returns contacts with optional filtering.
func (s *Server) handleListContacts(w http.ResponseWriter, r *http.Request) {
	grantID := s.withAuthGrant(w, ContactsResponse{Contacts: demoContacts(), HasMore: false})
	if grantID == "" {
		return
	}

	// Parse query parameters
	query := NewQueryParams(r.URL.Query())

	params := &domain.ContactQueryParams{
		Limit:  query.GetLimit(50),
		Email:  query.GetString("email", ""),
		Source: query.GetString("source", ""),
	}

	// Filter by group
	group := query.Get("group")
	if group != "" {
		params.Group = group
	}

	// Cursor for pagination
	cursor := query.Get("cursor")
	if cursor != "" {
		params.PageToken = cursor
	}

	// Get account email for cache lookup
	accountEmail := s.getAccountEmail(grantID)

	// Try cache first (only for first page without complex filters)
	if cursor == "" && params.Email == "" && params.Source == "" && s.cacheAvailable() {
		cacheOpts := cache.ContactListOptions{
			Group: group,
			Limit: params.Limit,
		}
		var cached []*cache.CachedContact
		if err := s.withContactStore(accountEmail, func(store *cache.ContactStore) error {
			var err error
			cached, err = store.List(cacheOpts)
			return err
		}); err == nil && len(cached) > 0 {
			resp := ContactsResponse{
				Contacts: make([]ContactResponse, 0, len(cached)),
				HasMore:  len(cached) >= params.Limit,
			}
			for _, c := range cached {
				resp.Contacts = append(resp.Contacts, cachedContactToResponse(c))
			}
			writeJSON(w, http.StatusOK, resp)
			return
		}
	}

	// Fetch contacts from Nylas API
	ctx, cancel := s.withTimeout(r)
	defer cancel()

	result, err := s.nylasClient.GetContactsWithCursor(ctx, grantID, params)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch contacts: " + err.Error(),
		})
		return
	}

	// Convert to response format
	resp := ContactsResponse{
		Contacts:   make([]ContactResponse, 0, len(result.Data)),
		NextCursor: result.Pagination.NextCursor,
		HasMore:    result.Pagination.HasMore,
	}
	for _, c := range result.Data {
		resp.Contacts = append(resp.Contacts, contactToResponse(c))
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleContactByID handles single contact operations: GET, PUT, DELETE.
func (s *Server) handleContactByID(w http.ResponseWriter, r *http.Request) {
	// Parse contact ID from path: /api/contacts/{id} or /api/contacts/{id}/photo
	path := strings.TrimPrefix(r.URL.Path, "/api/contacts/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Contact ID required", http.StatusBadRequest)
		return
	}
	contactID := parts[0]

	// Check for /photo suffix
	if len(parts) > 1 && parts[1] == "photo" {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleContactPhoto(w, r, contactID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetContact(w, r, contactID)
	case http.MethodPut:
		s.handleUpdateContact(w, r, contactID)
	case http.MethodDelete:
		s.handleDeleteContact(w, r, contactID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getAccountEmail returns the email address for a grant ID.
func (s *Server) getAccountEmail(grantID string) string {
	if s.grantStore == nil {
		return ""
	}
	grants, err := s.grantStore.ListGrants()
	if err != nil {
		return ""
	}
	for _, g := range grants {
		if g.ID == grantID {
			return g.Email
		}
	}
	// Fall back to first grant
	if len(grants) > 0 {
		return grants[0].Email
	}
	return ""
}
