package domain

import (
	"testing"
)

// =============================================================================
// ContactGroup Tests
// =============================================================================

func TestContactGroup_Creation(t *testing.T) {
	group := ContactGroup{
		ID:      "group-123",
		GrantID: "grant-456",
		Name:    "Work Contacts",
		Path:    "Work/Colleagues",
	}

	if group.ID != "group-123" {
		t.Errorf("ContactGroup.ID = %q, want %q", group.ID, "group-123")
	}
	if group.Name != "Work Contacts" {
		t.Errorf("ContactGroup.Name = %q, want %q", group.Name, "Work Contacts")
	}
	if group.Path != "Work/Colleagues" {
		t.Errorf("ContactGroup.Path = %q, want %q", group.Path, "Work/Colleagues")
	}
}

// =============================================================================
// ContactQueryParams Tests
// =============================================================================

func TestContactQueryParams_Creation(t *testing.T) {
	params := ContactQueryParams{
		Limit:          50,
		PageToken:      "token-123",
		Email:          "john@example.com",
		PhoneNumber:    "+1-555-123-4567",
		Source:         "address_book",
		Group:          "group-123",
		Recurse:        true,
		ProfilePicture: true,
	}

	if params.Limit != 50 {
		t.Errorf("ContactQueryParams.Limit = %d, want 50", params.Limit)
	}
	if params.Source != "address_book" {
		t.Errorf("ContactQueryParams.Source = %q, want %q", params.Source, "address_book")
	}
	if !params.Recurse {
		t.Error("ContactQueryParams.Recurse should be true")
	}
	if !params.ProfilePicture {
		t.Error("ContactQueryParams.ProfilePicture should be true")
	}
}

// =============================================================================
// CreateContactRequest Tests
// =============================================================================

func TestCreateContactRequest_Creation(t *testing.T) {
	req := CreateContactRequest{
		GivenName:   "John",
		MiddleName:  "William",
		Surname:     "Doe",
		Suffix:      "Jr.",
		Nickname:    "Johnny",
		Birthday:    "1990-01-15",
		CompanyName: "Acme Corp",
		JobTitle:    "Engineer",
		ManagerName: "Jane Manager",
		Notes:       "Met at conference",
		Emails: []ContactEmail{
			{Email: "john@example.com", Type: "work"},
		},
		PhoneNumbers: []ContactPhone{
			{Number: "+1-555-123-4567", Type: "mobile"},
		},
		WebPages: []ContactWebPage{
			{URL: "https://johndoe.com", Type: "profile"},
		},
		IMAddresses: []ContactIM{
			{IMAddress: "johndoe", Type: "skype"},
		},
		PhysicalAddresses: []ContactAddress{
			{Type: "work", City: "San Francisco"},
		},
		Groups: []ContactGroupInfo{
			{ID: "group-123"},
		},
	}

	if req.GivenName != "John" {
		t.Errorf("CreateContactRequest.GivenName = %q, want %q", req.GivenName, "John")
	}
	if req.Surname != "Doe" {
		t.Errorf("CreateContactRequest.Surname = %q, want %q", req.Surname, "Doe")
	}
	if len(req.Emails) != 1 {
		t.Errorf("CreateContactRequest.Emails length = %d, want 1", len(req.Emails))
	}
	if len(req.PhoneNumbers) != 1 {
		t.Errorf("CreateContactRequest.PhoneNumbers length = %d, want 1", len(req.PhoneNumbers))
	}
}

// =============================================================================
// UpdateContactRequest Tests
// =============================================================================

func TestUpdateContactRequest_Creation(t *testing.T) {
	givenName := "John"
	surname := "Smith"

	req := UpdateContactRequest{
		GivenName: &givenName,
		Surname:   &surname,
		Emails: []ContactEmail{
			{Email: "john.smith@example.com", Type: "work"},
		},
	}

	if req.GivenName == nil || *req.GivenName != "John" {
		t.Errorf("UpdateContactRequest.GivenName = %v, want %q", req.GivenName, "John")
	}
	if req.Surname == nil || *req.Surname != "Smith" {
		t.Errorf("UpdateContactRequest.Surname = %v, want %q", req.Surname, "Smith")
	}
	if len(req.Emails) != 1 {
		t.Errorf("UpdateContactRequest.Emails length = %d, want 1", len(req.Emails))
	}
}

// =============================================================================
// ContactWebPage Tests
// =============================================================================

func TestContactWebPage_Creation(t *testing.T) {
	tests := []struct {
		name    string
		webPage ContactWebPage
	}{
		{
			name: "creates profile webpage",
			webPage: ContactWebPage{
				URL:  "https://linkedin.com/in/johndoe",
				Type: "profile",
			},
		},
		{
			name: "creates work webpage",
			webPage: ContactWebPage{
				URL:  "https://acme.com",
				Type: "work",
			},
		},
		{
			name: "creates blog webpage",
			webPage: ContactWebPage{
				URL:  "https://johndoe.blog",
				Type: "blog",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.webPage.URL == "" {
				t.Error("ContactWebPage.URL should not be empty")
			}
		})
	}
}

// =============================================================================
// ContactIM Tests
// =============================================================================

func TestContactIM_Creation(t *testing.T) {
	tests := []struct {
		name string
		im   ContactIM
	}{
		{
			name: "creates skype IM",
			im: ContactIM{
				IMAddress: "johndoe.skype",
				Type:      "skype",
			},
		},
		{
			name: "creates slack IM",
			im: ContactIM{
				IMAddress: "@johndoe",
				Type:      "other",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.im.IMAddress == "" {
				t.Error("ContactIM.IMAddress should not be empty")
			}
		})
	}
}

// =============================================================================
// ContactGroupInfo Tests
// =============================================================================

func TestContactGroupInfo_Creation(t *testing.T) {
	info := ContactGroupInfo{
		ID: "group-456",
	}

	if info.ID != "group-456" {
		t.Errorf("ContactGroupInfo.ID = %q, want %q", info.ID, "group-456")
	}
}

// =============================================================================
// ContactListResponse Tests
// =============================================================================

func TestContactListResponse_Creation(t *testing.T) {
	resp := ContactListResponse{
		Data: []Contact{
			{ID: "contact-1", GivenName: "John"},
			{ID: "contact-2", GivenName: "Jane"},
		},
		Pagination: Pagination{
			NextCursor: "cursor-123",
			HasMore:    true,
		},
	}

	if len(resp.Data) != 2 {
		t.Errorf("ContactListResponse.Data length = %d, want 2", len(resp.Data))
	}
	if !resp.Pagination.HasMore {
		t.Error("ContactListResponse.Pagination.HasMore should be true")
	}
	if resp.Pagination.NextCursor != "cursor-123" {
		t.Errorf("Pagination.NextCursor = %q, want %q", resp.Pagination.NextCursor, "cursor-123")
	}
}

// =============================================================================
// CreateContactGroupRequest Tests
// =============================================================================

func TestCreateContactGroupRequest_Creation(t *testing.T) {
	req := CreateContactGroupRequest{
		Name: "VIP Customers",
	}

	if req.Name != "VIP Customers" {
		t.Errorf("CreateContactGroupRequest.Name = %q, want %q", req.Name, "VIP Customers")
	}
}

// =============================================================================
// UpdateContactGroupRequest Tests
// =============================================================================

func TestUpdateContactGroupRequest_Creation(t *testing.T) {
	name := "Updated Group Name"
	req := UpdateContactGroupRequest{
		Name: &name,
	}

	if req.Name == nil || *req.Name != "Updated Group Name" {
		t.Errorf("UpdateContactGroupRequest.Name = %v, want %q", req.Name, "Updated Group Name")
	}
}
