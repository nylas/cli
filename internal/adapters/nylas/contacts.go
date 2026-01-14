package nylas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/util"
)

// contactResponse represents a contact from the API.
type contactResponse struct {
	ID                string                    `json:"id"`
	GrantID           string                    `json:"grant_id"`
	Object            string                    `json:"object"`
	GivenName         string                    `json:"given_name"`
	MiddleName        string                    `json:"middle_name"`
	Surname           string                    `json:"surname"`
	Suffix            string                    `json:"suffix"`
	Nickname          string                    `json:"nickname"`
	Birthday          string                    `json:"birthday"`
	CompanyName       string                    `json:"company_name"`
	JobTitle          string                    `json:"job_title"`
	ManagerName       string                    `json:"manager_name"`
	Notes             string                    `json:"notes"`
	PictureURL        string                    `json:"picture_url"`
	Picture           string                    `json:"picture"`
	Emails            []domain.ContactEmail     `json:"emails"`
	PhoneNumbers      []domain.ContactPhone     `json:"phone_numbers"`
	WebPages          []domain.ContactWebPage   `json:"web_pages"`
	IMAddresses       []domain.ContactIM        `json:"im_addresses"`
	PhysicalAddresses []domain.ContactAddress   `json:"physical_addresses"`
	Groups            []domain.ContactGroupInfo `json:"groups"`
	Source            string                    `json:"source"`
}

// contactGroupResponse represents a contact group from the API.
type contactGroupResponse struct {
	ID      string `json:"id"`
	GrantID string `json:"grant_id"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Object  string `json:"object"`
}

// GetContacts retrieves contacts for a grant.
func (c *HTTPClient) GetContacts(ctx context.Context, grantID string, params *domain.ContactQueryParams) ([]domain.Contact, error) {
	result, err := c.GetContactsWithCursor(ctx, grantID, params)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetContactsWithCursor retrieves contacts with pagination cursor.
func (c *HTTPClient) GetContactsWithCursor(ctx context.Context, grantID string, params *domain.ContactQueryParams) (*domain.ContactListResponse, error) {
	baseURL := fmt.Sprintf("%s/v3/grants/%s/contacts", c.baseURL, grantID)

	qb := NewQueryBuilder()
	if params != nil {
		qb.AddInt("limit", params.Limit).
			Add("page_token", params.PageToken).
			Add("email", params.Email).
			Add("phone_number", params.PhoneNumber).
			Add("source", params.Source).
			Add("group", params.Group).
			AddBool("recurse", params.Recurse).
			AddBool("profile_picture", params.ProfilePicture)
	}
	queryURL := qb.BuildURL(baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}
	c.setAuthHeader(req)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result struct {
		Data       []contactResponse `json:"data"`
		NextCursor string            `json:"next_cursor,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &domain.ContactListResponse{
		Data: util.Map(result.Data, convertContact),
		Pagination: domain.Pagination{
			NextCursor: result.NextCursor,
			HasMore:    result.NextCursor != "",
		},
	}, nil
}

// GetContact retrieves a single contact by ID.
func (c *HTTPClient) GetContact(ctx context.Context, grantID, contactID string) (*domain.Contact, error) {
	return c.GetContactWithPicture(ctx, grantID, contactID, false)
}

// GetContactWithPicture retrieves a single contact by ID with optional profile picture.
func (c *HTTPClient) GetContactWithPicture(ctx context.Context, grantID, contactID string, includePicture bool) (*domain.Contact, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/contacts/%s", c.baseURL, grantID, contactID)

	if includePicture {
		queryURL += "?profile_picture=true"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}
	c.setAuthHeader(req)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, domain.ErrContactNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result struct {
		Data contactResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	contact := convertContact(result.Data)
	return &contact, nil
}

// CreateContact creates a new contact.
func (c *HTTPClient) CreateContact(ctx context.Context, grantID string, req *domain.CreateContactRequest) (*domain.Contact, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/contacts", c.baseURL, grantID)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data contactResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	contact := convertContact(result.Data)
	return &contact, nil
}

// UpdateContact updates an existing contact.
func (c *HTTPClient) UpdateContact(ctx context.Context, grantID, contactID string, req *domain.UpdateContactRequest) (*domain.Contact, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/contacts/%s", c.baseURL, grantID, contactID)

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, req, http.StatusOK)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data contactResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	contact := convertContact(result.Data)
	return &contact, nil
}

// DeleteContact deletes a contact.
func (c *HTTPClient) DeleteContact(ctx context.Context, grantID, contactID string) error {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/contacts/%s", c.baseURL, grantID, contactID)
	return c.doDelete(ctx, queryURL)
}

// GetContactGroups retrieves contact groups for a grant.
func (c *HTTPClient) GetContactGroups(ctx context.Context, grantID string) ([]domain.ContactGroup, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/contacts/groups", c.baseURL, grantID)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}
	c.setAuthHeader(req)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result struct {
		Data []contactGroupResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return util.Map(result.Data, func(g contactGroupResponse) domain.ContactGroup {
		return domain.ContactGroup{
			ID:      g.ID,
			GrantID: g.GrantID,
			Name:    g.Name,
			Path:    g.Path,
			Object:  g.Object,
		}
	}), nil
}

// GetContactGroup retrieves a single contact group by ID.
func (c *HTTPClient) GetContactGroup(ctx context.Context, grantID, groupID string) (*domain.ContactGroup, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/contacts/groups/%s", c.baseURL, grantID, groupID)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}
	c.setAuthHeader(req)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: contact group not found", domain.ErrAPIError)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result struct {
		Data contactGroupResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	group := domain.ContactGroup{
		ID:      result.Data.ID,
		GrantID: result.Data.GrantID,
		Name:    result.Data.Name,
		Path:    result.Data.Path,
		Object:  result.Data.Object,
	}
	return &group, nil
}

// CreateContactGroup creates a new contact group.
func (c *HTTPClient) CreateContactGroup(ctx context.Context, grantID string, req *domain.CreateContactGroupRequest) (*domain.ContactGroup, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/contacts/groups", c.baseURL, grantID)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data contactGroupResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	group := domain.ContactGroup{
		ID:      result.Data.ID,
		GrantID: result.Data.GrantID,
		Name:    result.Data.Name,
		Path:    result.Data.Path,
		Object:  result.Data.Object,
	}
	return &group, nil
}

// UpdateContactGroup updates an existing contact group.
func (c *HTTPClient) UpdateContactGroup(ctx context.Context, grantID, groupID string, req *domain.UpdateContactGroupRequest) (*domain.ContactGroup, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/contacts/groups/%s", c.baseURL, grantID, groupID)

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, req, http.StatusOK)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data contactGroupResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	group := domain.ContactGroup{
		ID:      result.Data.ID,
		GrantID: result.Data.GrantID,
		Name:    result.Data.Name,
		Path:    result.Data.Path,
		Object:  result.Data.Object,
	}
	return &group, nil
}

// DeleteContactGroup deletes a contact group.
func (c *HTTPClient) DeleteContactGroup(ctx context.Context, grantID, groupID string) error {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/contacts/groups/%s", c.baseURL, grantID, groupID)
	return c.doDelete(ctx, queryURL)
}

// convertContact converts an API contact response to domain model.
func convertContact(c contactResponse) domain.Contact {
	return domain.Contact{
		ID:                c.ID,
		GrantID:           c.GrantID,
		Object:            c.Object,
		GivenName:         c.GivenName,
		MiddleName:        c.MiddleName,
		Surname:           c.Surname,
		Suffix:            c.Suffix,
		Nickname:          c.Nickname,
		Birthday:          c.Birthday,
		CompanyName:       c.CompanyName,
		JobTitle:          c.JobTitle,
		ManagerName:       c.ManagerName,
		Notes:             c.Notes,
		PictureURL:        c.PictureURL,
		Picture:           c.Picture,
		Emails:            c.Emails,
		PhoneNumbers:      c.PhoneNumbers,
		WebPages:          c.WebPages,
		IMAddresses:       c.IMAddresses,
		PhysicalAddresses: c.PhysicalAddresses,
		Groups:            c.Groups,
		Source:            c.Source,
	}
}
