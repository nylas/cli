package domain

// Contact represents a contact from Nylas.
type Contact struct {
	ID                string             `json:"id"`
	GrantID           string             `json:"grant_id"`
	Object            string             `json:"object,omitempty"`
	GivenName         string             `json:"given_name,omitempty"`
	MiddleName        string             `json:"middle_name,omitempty"`
	Surname           string             `json:"surname,omitempty"`
	Suffix            string             `json:"suffix,omitempty"`
	Nickname          string             `json:"nickname,omitempty"`
	Birthday          string             `json:"birthday,omitempty"`
	CompanyName       string             `json:"company_name,omitempty"`
	JobTitle          string             `json:"job_title,omitempty"`
	ManagerName       string             `json:"manager_name,omitempty"`
	Notes             string             `json:"notes,omitempty"`
	PictureURL        string             `json:"picture_url,omitempty"`
	Picture           string             `json:"picture,omitempty"` // Base64-encoded image data (when profile_picture=true)
	Emails            []ContactEmail     `json:"emails,omitempty"`
	PhoneNumbers      []ContactPhone     `json:"phone_numbers,omitempty"`
	WebPages          []ContactWebPage   `json:"web_pages,omitempty"`
	IMAddresses       []ContactIM        `json:"im_addresses,omitempty"`
	PhysicalAddresses []ContactAddress   `json:"physical_addresses,omitempty"`
	Groups            []ContactGroupInfo `json:"groups,omitempty"`
	Source            string             `json:"source,omitempty"`
}

// DisplayName returns a formatted display name for the contact.
func (c Contact) DisplayName() string {
	if c.GivenName != "" && c.Surname != "" {
		return c.GivenName + " " + c.Surname
	}
	if c.GivenName != "" {
		return c.GivenName
	}
	if c.Surname != "" {
		return c.Surname
	}
	if c.Nickname != "" {
		return c.Nickname
	}
	if len(c.Emails) > 0 {
		return c.Emails[0].Email
	}
	return "Unknown"
}

// PrimaryEmail returns the primary email address.
func (c Contact) PrimaryEmail() string {
	for _, e := range c.Emails {
		if e.Type == "primary" || e.Type == "work" || e.Type == "home" {
			return e.Email
		}
	}
	if len(c.Emails) > 0 {
		return c.Emails[0].Email
	}
	return ""
}

// PrimaryPhone returns the primary phone number.
func (c Contact) PrimaryPhone() string {
	for _, p := range c.PhoneNumbers {
		if p.Type == "mobile" || p.Type == "work" || p.Type == "home" {
			return p.Number
		}
	}
	if len(c.PhoneNumbers) > 0 {
		return c.PhoneNumbers[0].Number
	}
	return ""
}

// ContactEmail represents a contact's email address.
type ContactEmail struct {
	Email string `json:"email"`
	Type  string `json:"type,omitempty"` // home, work, school, other
}

// ContactPhone represents a contact's phone number.
type ContactPhone struct {
	Number string `json:"number"`
	Type   string `json:"type,omitempty"` // mobile, home, work, pager, business_fax, home_fax, other
}

// ContactWebPage represents a contact's web page.
type ContactWebPage struct {
	URL  string `json:"url"`
	Type string `json:"type,omitempty"` // profile, blog, home, work, other
}

// ContactIM represents a contact's instant messaging address.
type ContactIM struct {
	IMAddress string `json:"im_address"`
	Type      string `json:"type,omitempty"` // aim, msn, yahoo, skype, qq, google_talk, icq, jabber, other
}

// ContactAddress represents a contact's physical address.
type ContactAddress struct {
	Type          string `json:"type,omitempty"` // home, work, other
	StreetAddress string `json:"street_address,omitempty"`
	City          string `json:"city,omitempty"`
	State         string `json:"state,omitempty"`
	PostalCode    string `json:"postal_code,omitempty"`
	Country       string `json:"country,omitempty"`
}

// ContactGroupInfo represents a contact group reference.
type ContactGroupInfo struct {
	ID string `json:"id"`
}

// ContactGroup represents a contact group from Nylas.
type ContactGroup struct {
	ID      string `json:"id"`
	GrantID string `json:"grant_id"`
	Name    string `json:"name"`
	Path    string `json:"path,omitempty"`
	Object  string `json:"object,omitempty"`
}

// ContactQueryParams for filtering contacts.
type ContactQueryParams struct {
	Limit          int    `json:"limit,omitempty"`
	PageToken      string `json:"page_token,omitempty"`
	Email          string `json:"email,omitempty"`
	PhoneNumber    string `json:"phone_number,omitempty"`
	Source         string `json:"source,omitempty"` // address_book, inbox, domain
	Group          string `json:"group,omitempty"`
	Recurse        bool   `json:"recurse,omitempty"`
	ProfilePicture bool   `json:"profile_picture,omitempty"` // Include Base64-encoded profile picture
}

// CreateContactRequest for creating a new contact.
type CreateContactRequest struct {
	GivenName         string             `json:"given_name,omitempty"`
	MiddleName        string             `json:"middle_name,omitempty"`
	Surname           string             `json:"surname,omitempty"`
	Suffix            string             `json:"suffix,omitempty"`
	Nickname          string             `json:"nickname,omitempty"`
	Birthday          string             `json:"birthday,omitempty"`
	CompanyName       string             `json:"company_name,omitempty"`
	JobTitle          string             `json:"job_title,omitempty"`
	ManagerName       string             `json:"manager_name,omitempty"`
	Notes             string             `json:"notes,omitempty"`
	Emails            []ContactEmail     `json:"emails,omitempty"`
	PhoneNumbers      []ContactPhone     `json:"phone_numbers,omitempty"`
	WebPages          []ContactWebPage   `json:"web_pages,omitempty"`
	IMAddresses       []ContactIM        `json:"im_addresses,omitempty"`
	PhysicalAddresses []ContactAddress   `json:"physical_addresses,omitempty"`
	Groups            []ContactGroupInfo `json:"groups,omitempty"`
}

// UpdateContactRequest for updating a contact.
type UpdateContactRequest struct {
	GivenName         *string            `json:"given_name,omitempty"`
	MiddleName        *string            `json:"middle_name,omitempty"`
	Surname           *string            `json:"surname,omitempty"`
	Suffix            *string            `json:"suffix,omitempty"`
	Nickname          *string            `json:"nickname,omitempty"`
	Birthday          *string            `json:"birthday,omitempty"`
	CompanyName       *string            `json:"company_name,omitempty"`
	JobTitle          *string            `json:"job_title,omitempty"`
	ManagerName       *string            `json:"manager_name,omitempty"`
	Notes             *string            `json:"notes,omitempty"`
	Emails            []ContactEmail     `json:"emails,omitempty"`
	PhoneNumbers      []ContactPhone     `json:"phone_numbers,omitempty"`
	WebPages          []ContactWebPage   `json:"web_pages,omitempty"`
	IMAddresses       []ContactIM        `json:"im_addresses,omitempty"`
	PhysicalAddresses []ContactAddress   `json:"physical_addresses,omitempty"`
	Groups            []ContactGroupInfo `json:"groups,omitempty"`
}

// ContactListResponse represents a paginated contact list response.
type ContactListResponse struct {
	Data       []Contact  `json:"data"`
	Pagination Pagination `json:"pagination,omitempty"`
}

// ContactGroupListResponse represents a paginated contact group list response.
type ContactGroupListResponse struct {
	Data       []ContactGroup `json:"data"`
	Pagination Pagination     `json:"pagination,omitempty"`
}

// CreateContactGroupRequest for creating a new contact group.
type CreateContactGroupRequest struct {
	Name string `json:"name"`
}

// UpdateContactGroupRequest for updating a contact group.
type UpdateContactGroupRequest struct {
	Name *string `json:"name,omitempty"`
}
