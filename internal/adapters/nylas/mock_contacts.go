package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) GetContacts(ctx context.Context, grantID string, params *domain.ContactQueryParams) ([]domain.Contact, error) {
	return []domain.Contact{
		{
			ID:        "contact-1",
			GivenName: "John",
			Surname:   "Doe",
			Emails:    []domain.ContactEmail{{Email: "john@example.com", Type: "work"}},
		},
	}, nil
}

// GetContactsWithCursor retrieves contacts with pagination.
func (m *MockClient) GetContactsWithCursor(ctx context.Context, grantID string, params *domain.ContactQueryParams) (*domain.ContactListResponse, error) {
	return &domain.ContactListResponse{
		Data: []domain.Contact{
			{
				ID:        "contact-1",
				GivenName: "John",
				Surname:   "Doe",
				Emails:    []domain.ContactEmail{{Email: "john@example.com", Type: "work"}},
			},
		},
	}, nil
}

// GetContact retrieves a single contact.
func (m *MockClient) GetContact(ctx context.Context, grantID, contactID string) (*domain.Contact, error) {
	return &domain.Contact{
		ID:        contactID,
		GivenName: "John",
		Surname:   "Doe",
		Emails:    []domain.ContactEmail{{Email: "john@example.com", Type: "work"}},
	}, nil
}

// GetContactWithPicture retrieves a single contact with optional profile picture.
func (m *MockClient) GetContactWithPicture(ctx context.Context, grantID, contactID string, includePicture bool) (*domain.Contact, error) {
	contact := &domain.Contact{
		ID:        contactID,
		GivenName: "John",
		Surname:   "Doe",
		Emails:    []domain.ContactEmail{{Email: "john@example.com", Type: "work"}},
	}
	if includePicture {
		contact.Picture = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	}
	return contact, nil
}

// CreateContact creates a new contact.
func (m *MockClient) CreateContact(ctx context.Context, grantID string, req *domain.CreateContactRequest) (*domain.Contact, error) {
	return &domain.Contact{
		ID:        "new-contact-id",
		GivenName: req.GivenName,
		Surname:   req.Surname,
		Emails:    req.Emails,
	}, nil
}

// UpdateContact updates an existing contact.
func (m *MockClient) UpdateContact(ctx context.Context, grantID, contactID string, req *domain.UpdateContactRequest) (*domain.Contact, error) {
	contact := &domain.Contact{ID: contactID}
	if req.GivenName != nil {
		contact.GivenName = *req.GivenName
	}
	if req.Surname != nil {
		contact.Surname = *req.Surname
	}
	contact.Emails = req.Emails
	return contact, nil
}

// DeleteContact deletes a contact.
func (m *MockClient) DeleteContact(ctx context.Context, grantID, contactID string) error {
	return nil
}

// GetContactGroups retrieves contact groups.
func (m *MockClient) GetContactGroups(ctx context.Context, grantID string) ([]domain.ContactGroup, error) {
	return []domain.ContactGroup{
		{ID: "group-1", Name: "Contacts"},
	}, nil
}

// GetContactGroup retrieves a single contact group.
func (m *MockClient) GetContactGroup(ctx context.Context, grantID, groupID string) (*domain.ContactGroup, error) {
	return &domain.ContactGroup{
		ID:      groupID,
		GrantID: grantID,
		Name:    "Test Group",
	}, nil
}

// CreateContactGroup creates a new contact group.
func (m *MockClient) CreateContactGroup(ctx context.Context, grantID string, req *domain.CreateContactGroupRequest) (*domain.ContactGroup, error) {
	return &domain.ContactGroup{
		ID:      "new-group-id",
		GrantID: grantID,
		Name:    req.Name,
	}, nil
}

// UpdateContactGroup updates an existing contact group.
func (m *MockClient) UpdateContactGroup(ctx context.Context, grantID, groupID string, req *domain.UpdateContactGroupRequest) (*domain.ContactGroup, error) {
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

// DeleteContactGroup deletes a contact group.
func (m *MockClient) DeleteContactGroup(ctx context.Context, grantID, groupID string) error {
	return nil
}

// ListWebhooks lists all webhooks.
