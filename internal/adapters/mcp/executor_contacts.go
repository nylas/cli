package mcp

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// CONTACT TOOLS
// ============================================================================

// executeListContacts lists or searches contacts.
func (s *Server) executeListContacts(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	params := &domain.ContactQueryParams{
		Limit:       clampLimit(args, "limit", 10),
		Email:       getString(args, "email", ""),
		PhoneNumber: getString(args, "phone_number", ""),
		Source:      getString(args, "source", ""),
		Group:       getString(args, "group", ""),
	}
	if pageToken := getString(args, "page_token", ""); pageToken != "" {
		params.PageToken = pageToken
	}

	resp, err := s.client.GetContactsWithCursor(ctx, grantID, params)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	result := make([]map[string]any, 0, len(resp.Data))
	for _, c := range resp.Data {
		result = append(result, map[string]any{
			"id":           c.ID,
			"given_name":   c.GivenName,
			"surname":      c.Surname,
			"display_name": c.DisplayName(),
			"email":        c.PrimaryEmail(),
			"phone":        c.PrimaryPhone(),
			"company_name": c.CompanyName,
			"job_title":    c.JobTitle,
		})
	}

	if resp.Pagination.NextCursor != "" {
		return toolSuccess(map[string]any{
			"data":        result,
			"next_cursor": resp.Pagination.NextCursor,
		})
	}
	return toolSuccess(result)
}

// executeGetContact retrieves full detail for a specific contact.
func (s *Server) executeGetContact(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	contactID := getString(args, "contact_id", "")
	if contactID == "" {
		return toolError("contact_id is required")
	}

	contact, err := s.client.GetContact(ctx, grantID, contactID)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	return toolSuccess(map[string]any{
		"id":                 contact.ID,
		"given_name":         contact.GivenName,
		"surname":            contact.Surname,
		"middle_name":        contact.MiddleName,
		"nickname":           contact.Nickname,
		"birthday":           contact.Birthday,
		"company_name":       contact.CompanyName,
		"job_title":          contact.JobTitle,
		"emails":             contact.Emails,
		"phone_numbers":      contact.PhoneNumbers,
		"web_pages":          contact.WebPages,
		"physical_addresses": contact.PhysicalAddresses,
		"notes":              contact.Notes,
		"groups":             contact.Groups,
	})
}

// executeCreateContact creates a new contact.
func (s *Server) executeCreateContact(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)

	req := &domain.CreateContactRequest{
		GivenName:    getString(args, "given_name", ""),
		Surname:      getString(args, "surname", ""),
		Nickname:     getString(args, "nickname", ""),
		CompanyName:  getString(args, "company_name", ""),
		JobTitle:     getString(args, "job_title", ""),
		Notes:        getString(args, "notes", ""),
		Emails:       parseContactEmails(args),
		PhoneNumbers: parseContactPhones(args),
	}

	contact, err := s.client.CreateContact(ctx, grantID, req)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	return toolSuccess(map[string]any{
		"id":           contact.ID,
		"display_name": contact.DisplayName(),
		"status":       "created",
	})
}

// executeUpdateContact updates a contact's information.
func (s *Server) executeUpdateContact(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	contactID := getString(args, "contact_id", "")
	if contactID == "" {
		return toolError("contact_id is required")
	}

	req := &domain.UpdateContactRequest{}
	if v := getString(args, "given_name", ""); v != "" {
		req.GivenName = &v
	}
	if v := getString(args, "surname", ""); v != "" {
		req.Surname = &v
	}
	if v := getString(args, "nickname", ""); v != "" {
		req.Nickname = &v
	}
	if v := getString(args, "company_name", ""); v != "" {
		req.CompanyName = &v
	}
	if v := getString(args, "job_title", ""); v != "" {
		req.JobTitle = &v
	}
	if v := getString(args, "notes", ""); v != "" {
		req.Notes = &v
	}
	if emails := parseContactEmails(args); len(emails) > 0 {
		req.Emails = emails
	}
	if phones := parseContactPhones(args); len(phones) > 0 {
		req.PhoneNumbers = phones
	}

	contact, err := s.client.UpdateContact(ctx, grantID, contactID, req)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	return toolSuccess(map[string]any{
		"id":           contact.ID,
		"display_name": contact.DisplayName(),
		"status":       "updated",
	})
}

// executeDeleteContact deletes a contact.
func (s *Server) executeDeleteContact(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	contactID := getString(args, "contact_id", "")
	if contactID == "" {
		return toolError("contact_id is required")
	}

	if err := s.client.DeleteContact(ctx, grantID, contactID); err != nil {
		return toolError(sanitizeError(err))
	}

	return toolSuccessText("Deleted contact " + contactID)
}

// parseContactEmails extracts contact emails from tool arguments.
func parseContactEmails(args map[string]any) []domain.ContactEmail {
	val, ok := args["emails"]
	if !ok {
		return nil
	}
	arr, ok := val.([]any)
	if !ok {
		return nil
	}
	var result []domain.ContactEmail
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		email, _ := m["email"].(string)
		if email == "" {
			continue
		}
		typ, _ := m["type"].(string)
		result = append(result, domain.ContactEmail{Email: email, Type: typ})
	}
	return result
}

// parseContactPhones extracts contact phone numbers from tool arguments.
func parseContactPhones(args map[string]any) []domain.ContactPhone {
	val, ok := args["phone_numbers"]
	if !ok {
		return nil
	}
	arr, ok := val.([]any)
	if !ok {
		return nil
	}
	var result []domain.ContactPhone
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		number, _ := m["number"].(string)
		if number == "" {
			continue
		}
		typ, _ := m["type"].(string)
		result = append(result, domain.ContactPhone{Number: number, Type: typ})
	}
	return result
}

// ============================================================================
// UTILITY TOOLS (no API call, no context needed)
// ============================================================================

// executeCurrentTime returns the current date and time in an optional timezone.
func (s *Server) executeCurrentTime(args map[string]any) *ToolResponse {
	loc, err := resolveLocation(getString(args, "timezone", ""))
	if err != nil {
		return toolError("invalid timezone: " + getString(args, "timezone", ""))
	}

	now := time.Now().In(loc)
	return toolSuccess(map[string]any{
		"datetime":       now.Format(time.RFC3339),
		"timezone":       loc.String(),
		"unix_timestamp": now.Unix(),
		"epoch":          now.Unix(),
	})
}

// executeEpochToDatetime converts a Unix timestamp to a human-readable datetime.
func (s *Server) executeEpochToDatetime(args map[string]any) *ToolResponse {
	epochVal, ok := args["epoch"]
	if !ok {
		return toolError("epoch is required")
	}
	epoch, ok := toInt64(epochVal)
	if !ok {
		return toolError("epoch must be a number")
	}

	loc, err := resolveLocation(getString(args, "timezone", ""))
	if err != nil {
		return toolError("invalid timezone: " + getString(args, "timezone", ""))
	}

	t := time.Unix(epoch, 0).In(loc)
	return toolSuccess(map[string]any{
		"datetime":       t.Format(time.RFC3339),
		"formatted":      t.Format("Monday, January 2, 2006 3:04 PM MST"),
		"timezone":       loc.String(),
		"unix_timestamp": epoch,
		"epoch":          epoch,
	})
}

// executeDatetimeToEpoch converts a datetime string to a Unix timestamp.
func (s *Server) executeDatetimeToEpoch(args map[string]any) *ToolResponse {
	dt := getString(args, "datetime", "")
	if dt == "" {
		return toolError("datetime is required")
	}

	loc, err := resolveLocation(getString(args, "timezone", ""))
	if err != nil {
		return toolError("invalid timezone: " + getString(args, "timezone", ""))
	}

	var t time.Time
	var parseErr error
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		t, parseErr = time.ParseInLocation(layout, dt, loc)
		if parseErr == nil {
			break
		}
	}
	if parseErr != nil {
		return toolError("could not parse datetime: " + dt)
	}

	return toolSuccess(map[string]any{
		"unix_timestamp": t.Unix(),
		"epoch":          t.Unix(),
		"datetime":       t.Format(time.RFC3339),
		"timezone":       loc.String(),
	})
}

// resolveLocation returns the *time.Location for an IANA timezone string.
// Returns time.Local if tz is empty, or an error if the timezone is invalid.
func resolveLocation(tz string) (*time.Location, error) {
	if tz == "" {
		return time.Local, nil
	}
	return time.LoadLocation(tz)
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	default:
		return 0, false
	}
}
