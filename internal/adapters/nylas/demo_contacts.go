package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// GetContacts returns demo contacts.
func (d *DemoClient) GetContacts(ctx context.Context, grantID string, params *domain.ContactQueryParams) ([]domain.Contact, error) {
	return d.getDemoContacts(), nil
}

// GetContactsWithCursor returns demo contacts with pagination.
func (d *DemoClient) GetContactsWithCursor(ctx context.Context, grantID string, params *domain.ContactQueryParams) (*domain.ContactListResponse, error) {
	return &domain.ContactListResponse{Data: d.getDemoContacts()}, nil
}

func (d *DemoClient) getDemoContacts() []domain.Contact {
	return []domain.Contact{
		{
			ID:           "contact-001",
			GivenName:    "Sarah",
			Surname:      "Chen",
			Emails:       []domain.ContactEmail{{Email: "sarah.chen@company.com", Type: "work"}},
			PhoneNumbers: []domain.ContactPhone{{Number: "+1-555-0101", Type: "mobile"}},
			CompanyName:  "Acme Corp",
			JobTitle:     "Engineering Manager",
		},
		{
			ID:           "contact-002",
			GivenName:    "Mike",
			Surname:      "Johnson",
			Emails:       []domain.ContactEmail{{Email: "mike.j@gmail.com", Type: "personal"}},
			PhoneNumbers: []domain.ContactPhone{{Number: "+1-555-0102", Type: "mobile"}},
		},
		{
			ID:           "contact-003",
			GivenName:    "Emily",
			Surname:      "Williams",
			Emails:       []domain.ContactEmail{{Email: "emily.w@startup.io", Type: "work"}},
			PhoneNumbers: []domain.ContactPhone{{Number: "+1-555-0103", Type: "work"}},
			CompanyName:  "TechStart Inc",
			JobTitle:     "CEO",
		},
		{
			ID:          "contact-004",
			GivenName:   "Alex",
			Surname:     "Kumar",
			Emails:      []domain.ContactEmail{{Email: "alex.kumar@dev.com", Type: "work"}},
			CompanyName: "DevOps Solutions",
			JobTitle:    "Senior Developer",
		},
		{
			ID:           "contact-005",
			GivenName:    "Jessica",
			Surname:      "Martinez",
			Emails:       []domain.ContactEmail{{Email: "jess.m@design.co", Type: "work"}},
			PhoneNumbers: []domain.ContactPhone{{Number: "+1-555-0105", Type: "mobile"}},
			CompanyName:  "Creative Design Co",
			JobTitle:     "Lead Designer",
		},
		{
			ID:          "contact-006",
			GivenName:   "David",
			Surname:     "Brown",
			Emails:      []domain.ContactEmail{{Email: "david.b@consulting.com", Type: "work"}},
			CompanyName: "Brown Consulting",
			JobTitle:    "Consultant",
		},
	}
}

// GetContact returns a demo contact.
func (d *DemoClient) GetContact(ctx context.Context, grantID, contactID string) (*domain.Contact, error) {
	contacts := d.getDemoContacts()
	for _, contact := range contacts {
		if contact.ID == contactID {
			return &contact, nil
		}
	}
	return &contacts[0], nil
}

// GetContactWithPicture returns a demo contact with optional profile picture.
func (d *DemoClient) GetContactWithPicture(ctx context.Context, grantID, contactID string, includePicture bool) (*domain.Contact, error) {
	contact, err := d.GetContact(ctx, grantID, contactID)
	if err != nil {
		return nil, err
	}
	if includePicture {
		// Demo base64-encoded 1x1 pixel image
		contact.Picture = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	}
	return contact, nil
}

// CreateContact simulates creating a contact.
func (d *DemoClient) CreateContact(ctx context.Context, grantID string, req *domain.CreateContactRequest) (*domain.Contact, error) {
	return &domain.Contact{ID: "new-contact", GivenName: req.GivenName, Surname: req.Surname}, nil
}

// UpdateContact simulates updating a contact.
func (d *DemoClient) UpdateContact(ctx context.Context, grantID, contactID string, req *domain.UpdateContactRequest) (*domain.Contact, error) {
	contact := &domain.Contact{ID: contactID}
	if req.GivenName != nil {
		contact.GivenName = *req.GivenName
	}
	if req.Surname != nil {
		contact.Surname = *req.Surname
	}
	return contact, nil
}

// DeleteContact simulates deleting a contact.
func (d *DemoClient) DeleteContact(ctx context.Context, grantID, contactID string) error {
	return nil
}

// GetContactGroups returns demo contact groups.
func (d *DemoClient) GetContactGroups(ctx context.Context, grantID string) ([]domain.ContactGroup, error) {
	return []domain.ContactGroup{
		{ID: "group-001", Name: "Coworkers"},
		{ID: "group-002", Name: "Friends"},
		{ID: "group-003", Name: "Family"},
		{ID: "group-004", Name: "VIP"},
	}, nil
}

// GetContactGroup returns a demo contact group.
func (d *DemoClient) GetContactGroup(ctx context.Context, grantID, groupID string) (*domain.ContactGroup, error) {
	return &domain.ContactGroup{
		ID:      groupID,
		GrantID: grantID,
		Name:    "Demo Group",
	}, nil
}

// CreateContactGroup creates a demo contact group.
func (d *DemoClient) CreateContactGroup(ctx context.Context, grantID string, req *domain.CreateContactGroupRequest) (*domain.ContactGroup, error) {
	return &domain.ContactGroup{
		ID:      "group-new",
		GrantID: grantID,
		Name:    req.Name,
	}, nil
}

// UpdateContactGroup updates a demo contact group.
func (d *DemoClient) UpdateContactGroup(ctx context.Context, grantID, groupID string, req *domain.UpdateContactGroupRequest) (*domain.ContactGroup, error) {
	name := "Updated Group"
	if req.Name != nil {
		name = *req.Name
	}
	return &domain.ContactGroup{
		ID:      groupID,
		GrantID: grantID,
		Name:    name,
	}, nil
}

// DeleteContactGroup deletes a demo contact group.
func (d *DemoClient) DeleteContactGroup(ctx context.Context, grantID, groupID string) error {
	return nil
}
