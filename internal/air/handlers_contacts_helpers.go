package air

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"

	"github.com/nylas/cli/internal/air/cache"
	"github.com/nylas/cli/internal/domain"
)

// handleContactPhoto returns the contact's profile picture as an image.
// Photos are cached locally for 30 days to reduce API calls.
func (s *Server) handleContactPhoto(w http.ResponseWriter, r *http.Request, contactID string) {
	// Demo mode: return a placeholder image
	if s.demoMode {
		// Return a 1x1 transparent PNG as placeholder
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		// 1x1 transparent PNG
		transparentPNG := []byte{
			0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
			0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
			0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
			0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
			0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
		}
		_, _ = w.Write(transparentPNG)
		return
	}

	// Check if configured
	if !s.requireConfig(w) {
		return
	}

	// Try to serve from cache first
	if s.hasPhotoStore() {
		var (
			imageData   []byte
			contentType string
		)
		if err := s.withPhotoStore(func(store *cache.PhotoStore) error {
			var err error
			imageData, contentType, err = store.Get(contactID)
			return err
		}); err == nil && imageData != nil {
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Cache-Control", "public, max-age=86400")
			w.Header().Set("Content-Length", strconv.Itoa(len(imageData)))
			w.Header().Set("X-Cache", "HIT")
			_, _ = w.Write(imageData)
			return
		}
	}

	// Get default grant
	grantID, ok := s.requireDefaultGrant(w)
	if !ok {
		return
	}

	// Fetch contact with picture from Nylas API
	ctx, cancel := s.withTimeout(r)
	defer cancel()

	contact, err := s.nylasClient.GetContactWithPicture(ctx, grantID, contactID, true)
	if err != nil {
		http.Error(w, "Failed to fetch contact photo", http.StatusInternalServerError)
		return
	}

	// Check if contact has a picture
	if contact.Picture == "" {
		// No picture available - return 404
		http.Error(w, "No photo available", http.StatusNotFound)
		return
	}

	// Parse the base64 data URL (format: data:image/jpeg;base64,/9j/4AAQ...)
	pictureData := contact.Picture
	var contentType string
	var imageData []byte

	if strings.HasPrefix(pictureData, "data:") {
		// Parse data URL
		parts := strings.SplitN(pictureData, ",", 2)
		if len(parts) != 2 {
			http.Error(w, "Invalid image data format", http.StatusInternalServerError)
			return
		}
		// Extract content type from "data:image/jpeg;base64"
		metaParts := strings.SplitN(parts[0], ";", 2)
		contentType = strings.TrimPrefix(metaParts[0], "data:")
		if contentType == "" {
			contentType = "image/jpeg"
		}
		// Decode base64
		imageData, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			http.Error(w, "Failed to decode image data", http.StatusInternalServerError)
			return
		}
	} else {
		// Assume raw base64
		contentType = "image/jpeg"
		imageData, err = base64.StdEncoding.DecodeString(pictureData)
		if err != nil {
			http.Error(w, "Failed to decode image data", http.StatusInternalServerError)
			return
		}
	}

	// Cache the photo for future requests (30 days)
	if s.hasPhotoStore() {
		_ = s.withPhotoStore(func(store *cache.PhotoStore) error {
			return store.Put(contactID, contentType, imageData)
		})
	}

	// Set headers and write image
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400") // Browser cache for 1 day
	w.Header().Set("Content-Length", strconv.Itoa(len(imageData)))
	w.Header().Set("X-Cache", "MISS")
	_, _ = w.Write(imageData)
}

// contactToResponse converts a domain contact to an API response.
func contactToResponse(c domain.Contact) ContactResponse {
	resp := ContactResponse{
		ID:          c.ID,
		GivenName:   c.GivenName,
		Surname:     c.Surname,
		DisplayName: c.DisplayName(),
		Nickname:    c.Nickname,
		CompanyName: c.CompanyName,
		JobTitle:    c.JobTitle,
		Birthday:    c.Birthday,
		Notes:       c.Notes,
		PictureURL:  c.PictureURL,
		Source:      c.Source,
	}

	// Convert emails
	for _, e := range c.Emails {
		resp.Emails = append(resp.Emails, ContactEmailResponse{
			Email: e.Email,
			Type:  e.Type,
		})
	}

	// Convert phone numbers
	for _, p := range c.PhoneNumbers {
		resp.PhoneNumbers = append(resp.PhoneNumbers, ContactPhoneResponse{
			Number: p.Number,
			Type:   p.Type,
		})
	}

	// Convert addresses
	for _, a := range c.PhysicalAddresses {
		resp.Addresses = append(resp.Addresses, ContactAddressResponse{
			Type:          a.Type,
			StreetAddress: a.StreetAddress,
			City:          a.City,
			State:         a.State,
			PostalCode:    a.PostalCode,
			Country:       a.Country,
		})
	}

	return resp
}

// cachedContactToResponse converts a cached contact to response format.
func cachedContactToResponse(c *cache.CachedContact) ContactResponse {
	return ContactResponse{
		ID:          c.ID,
		GivenName:   c.GivenName,
		Surname:     c.Surname,
		DisplayName: c.DisplayName,
		Emails: []ContactEmailResponse{
			{Email: c.Email, Type: "personal"},
		},
		PhoneNumbers: []ContactPhoneResponse{
			{Number: c.Phone, Type: "mobile"},
		},
		CompanyName: c.Company,
		JobTitle:    c.JobTitle,
		Notes:       c.Notes,
	}
}

// containsEmail checks if any email in the list contains the query.
func containsEmail(emails []ContactEmailResponse, q string) bool {
	for _, e := range emails {
		if strings.Contains(strings.ToLower(e.Email), q) {
			return true
		}
	}
	return false
}

// matchesContactQuery checks if a contact matches the search query.
func matchesContactQuery(c ContactResponse, q string) bool {
	q = strings.ToLower(q)
	if strings.Contains(strings.ToLower(c.DisplayName), q) ||
		strings.Contains(strings.ToLower(c.GivenName), q) ||
		strings.Contains(strings.ToLower(c.Surname), q) ||
		strings.Contains(strings.ToLower(c.CompanyName), q) ||
		strings.Contains(strings.ToLower(c.Notes), q) {
		return true
	}
	return containsEmail(c.Emails, q)
}

// demoContacts returns demo contact data.
func demoContacts() []ContactResponse {
	return []ContactResponse{
		{
			ID:          "demo-contact-001",
			GivenName:   "Sarah",
			Surname:     "Chen",
			DisplayName: "Sarah Chen",
			CompanyName: "Nylas Inc",
			JobTitle:    "Product Manager",
			Emails: []ContactEmailResponse{
				{Email: "sarah.chen@company.com", Type: "work"},
				{Email: "sarah@personal.com", Type: "home"},
			},
			PhoneNumbers: []ContactPhoneResponse{
				{Number: "+1-555-123-4567", Type: "mobile"},
			},
		},
		{
			ID:          "demo-contact-002",
			GivenName:   "Alex",
			Surname:     "Johnson",
			DisplayName: "Alex Johnson",
			CompanyName: "Example Corp",
			JobTitle:    "Senior Engineer",
			Emails: []ContactEmailResponse{
				{Email: "demo@example.com", Type: "work"},
			},
			PhoneNumbers: []ContactPhoneResponse{
				{Number: "+1-555-234-5678", Type: "work"},
			},
		},
		{
			ID:          "demo-contact-003",
			GivenName:   "Maria",
			Surname:     "Garcia",
			DisplayName: "Maria Garcia",
			CompanyName: "Acme Corp",
			JobTitle:    "VP of Sales",
			Emails: []ContactEmailResponse{
				{Email: "maria.g@acme.com", Type: "work"},
			},
			PhoneNumbers: []ContactPhoneResponse{
				{Number: "+1-555-345-6789", Type: "mobile"},
				{Number: "+1-555-345-0000", Type: "work"},
			},
			Addresses: []ContactAddressResponse{
				{
					Type:          "work",
					StreetAddress: "123 Business St",
					City:          "San Francisco",
					State:         "CA",
					PostalCode:    "94107",
					Country:       "USA",
				},
			},
		},
		{
			ID:          "demo-contact-004",
			GivenName:   "James",
			Surname:     "Wilson",
			DisplayName: "James Wilson",
			CompanyName: "Tech Solutions",
			JobTitle:    "CTO",
			Emails: []ContactEmailResponse{
				{Email: "jwilson@techsolutions.io", Type: "work"},
			},
		},
		{
			ID:          "demo-contact-005",
			GivenName:   "Emily",
			Surname:     "Brown",
			DisplayName: "Emily Brown",
			Nickname:    "Em",
			Birthday:    "1990-03-15",
			Emails: []ContactEmailResponse{
				{Email: "emily.brown@email.com", Type: "home"},
			},
			PhoneNumbers: []ContactPhoneResponse{
				{Number: "+1-555-456-7890", Type: "mobile"},
			},
		},
	}
}

// demoContactGroups returns demo contact group data.
func demoContactGroups() []ContactGroupResponse {
	return []ContactGroupResponse{
		{ID: "group-001", Name: "Work", Path: "/Work"},
		{ID: "group-002", Name: "Family", Path: "/Family"},
		{ID: "group-003", Name: "Friends", Path: "/Friends"},
		{ID: "group-004", Name: "VIP Clients", Path: "/Work/VIP Clients"},
	}
}
