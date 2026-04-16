package air

import (
	"net/http"
	"strings"

	"github.com/nylas/cli/internal/air/cache"
	"github.com/nylas/cli/internal/domain"
)

// handleContactGroups returns contact groups.
func (s *Server) handleContactGroups(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	grantID := s.withAuthGrant(w, ContactGroupsResponse{Groups: demoContactGroups()})
	if grantID == "" {
		return
	}

	// Fetch contact groups from Nylas API
	ctx, cancel := s.withTimeout(r)
	defer cancel()

	groups, err := s.nylasClient.GetContactGroups(ctx, grantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch contact groups: " + err.Error(),
		})
		return
	}

	// Convert to response format
	resp := ContactGroupsResponse{
		Groups: make([]ContactGroupResponse, 0, len(groups)),
	}
	for _, g := range groups {
		resp.Groups = append(resp.Groups, ContactGroupResponse{
			ID:   g.ID,
			Name: g.Name,
			Path: g.Path,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleContactSearch searches contacts with text query.
func (s *Server) handleContactSearch(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	query := NewQueryParams(r.URL.Query())
	q := query.Get("q")

	// Special demo mode: filter mock contacts by query
	if s.demoMode {
		contacts := demoContacts()
		if q != "" {
			qLower := strings.ToLower(q)
			filtered := make([]ContactResponse, 0)
			for _, c := range contacts {
				if strings.Contains(strings.ToLower(c.DisplayName), qLower) ||
					strings.Contains(strings.ToLower(c.GivenName), qLower) ||
					strings.Contains(strings.ToLower(c.Surname), qLower) ||
					strings.Contains(strings.ToLower(c.CompanyName), qLower) ||
					containsEmail(c.Emails, qLower) {
					filtered = append(filtered, c)
				}
			}
			contacts = filtered
		}
		writeJSON(w, http.StatusOK, ContactsResponse{
			Contacts: contacts,
			HasMore:  false,
		})
		return
	}
	grantID := s.withAuthGrant(w, nil) // Demo mode already handled above
	if grantID == "" {
		return
	}

	// Parse query parameters
	params := &domain.ContactQueryParams{
		Limit:     query.GetLimit(50),
		PageToken: query.Get("cursor"),
	}

	// Set email filter if query looks like email
	if strings.Contains(q, "@") {
		params.Email = q
	}

	// Get account email for cache search
	accountEmail := s.getAccountEmail(grantID)

	// Try cache search first
	if q != "" && s.cacheAvailable() {
		var cached []*cache.CachedContact
		if err := s.withContactStore(accountEmail, func(store *cache.ContactStore) error {
			var err error
			cached, err = store.Search(q, params.Limit)
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

	// Filter by query if provided (for non-email queries)
	var contacts []ContactResponse
	for _, c := range result.Data {
		contact := contactToResponse(c)
		if q == "" || matchesContactQuery(contact, q) {
			contacts = append(contacts, contact)
		}
	}

	resp := ContactsResponse{
		Contacts:   contacts,
		NextCursor: result.Pagination.NextCursor,
		HasMore:    result.Pagination.HasMore,
	}

	writeJSON(w, http.StatusOK, resp)
}
