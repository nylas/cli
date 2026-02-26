package mcp

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// THREAD TOOLS
// ============================================================================

// executeListThreads lists email threads with optional filters.
func (s *Server) executeListThreads(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	params := &domain.ThreadQueryParams{
		Limit:   getInt(args, "limit", 10),
		Subject: getString(args, "subject", ""),
		From:    getString(args, "from", ""),
		To:      getString(args, "to", ""),
		Unread:  getBool(args, "unread"),
	}

	threads, err := s.client.GetThreads(ctx, grantID, params)
	if err != nil {
		return toolError(err.Error())
	}

	result := make([]map[string]any, 0, len(threads))
	for _, t := range threads {
		result = append(result, map[string]any{
			"id":                           t.ID,
			"subject":                      t.Subject,
			"snippet":                      t.Snippet,
			"unread":                       t.Unread,
			"starred":                      t.Starred,
			"message_ids":                  t.MessageIDs,
			"latest_message_received_date": t.LatestMessageRecvDate.Format(time.RFC3339),
			"participants":                 formatParticipants(t.Participants),
		})
	}
	return toolSuccess(result)
}

// executeGetThread retrieves a specific email thread.
func (s *Server) executeGetThread(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	threadID := getString(args, "thread_id", "")
	if threadID == "" {
		return toolError("thread_id is required")
	}

	t, err := s.client.GetThread(ctx, grantID, threadID)
	if err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"id":                           t.ID,
		"subject":                      t.Subject,
		"snippet":                      t.Snippet,
		"unread":                       t.Unread,
		"starred":                      t.Starred,
		"message_ids":                  t.MessageIDs,
		"draft_ids":                    t.DraftIDs,
		"folders":                      t.FolderIDs,
		"participants":                 formatParticipants(t.Participants),
		"has_attachments":              t.HasAttachments,
		"earliest_message_date":        t.EarliestMessageDate.Format(time.RFC3339),
		"latest_message_received_date": t.LatestMessageRecvDate.Format(time.RFC3339),
	})
}

// ============================================================================
// FOLDER TOOLS
// ============================================================================

// executeListFolders lists all email folders for a grant.
func (s *Server) executeListFolders(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)

	folders, err := s.client.GetFolders(ctx, grantID)
	if err != nil {
		return toolError(err.Error())
	}

	result := make([]map[string]any, 0, len(folders))
	for _, f := range folders {
		result = append(result, map[string]any{
			"id":            f.ID,
			"name":          f.Name,
			"system_folder": f.SystemFolder,
			"total_count":   f.TotalCount,
			"unread_count":  f.UnreadCount,
		})
	}
	return toolSuccess(result)
}

// executeGetFolder retrieves a specific email folder.
func (s *Server) executeGetFolder(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	folderID := getString(args, "folder_id", "")
	if folderID == "" {
		return toolError("folder_id is required")
	}

	f, err := s.client.GetFolder(ctx, grantID, folderID)
	if err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"id":            f.ID,
		"name":          f.Name,
		"system_folder": f.SystemFolder,
		"total_count":   f.TotalCount,
		"unread_count":  f.UnreadCount,
		"parent_id":     f.ParentID,
		"child_ids":     f.ChildIDs,
		"attributes":    f.Attributes,
	})
}

// executeCreateFolder creates a new email folder.
func (s *Server) executeCreateFolder(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	name := getString(args, "name", "")
	if name == "" {
		return toolError("name is required")
	}

	req := &domain.CreateFolderRequest{
		Name:            name,
		ParentID:        getString(args, "parent_id", ""),
		BackgroundColor: getString(args, "background_color", ""),
		TextColor:       getString(args, "text_color", ""),
	}

	folder, err := s.client.CreateFolder(ctx, grantID, req)
	if err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"id":     folder.ID,
		"name":   folder.Name,
		"status": "created",
	})
}

// ============================================================================
// ATTACHMENT TOOLS
// ============================================================================

// executeListAttachments lists all attachments for a specific message.
func (s *Server) executeListAttachments(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	messageID := getString(args, "message_id", "")
	if messageID == "" {
		return toolError("message_id is required")
	}

	attachments, err := s.client.ListAttachments(ctx, grantID, messageID)
	if err != nil {
		return toolError(err.Error())
	}

	result := make([]map[string]any, 0, len(attachments))
	for _, a := range attachments {
		result = append(result, map[string]any{
			"id":           a.ID,
			"filename":     a.Filename,
			"content_type": a.ContentType,
			"size":         a.Size,
			"is_inline":    a.IsInline,
		})
	}
	return toolSuccess(result)
}

// executeGetAttachment retrieves attachment metadata (no binary download).
func (s *Server) executeGetAttachment(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	messageID := getString(args, "message_id", "")
	if messageID == "" {
		return toolError("message_id is required")
	}
	attachmentID := getString(args, "attachment_id", "")
	if attachmentID == "" {
		return toolError("attachment_id is required")
	}

	att, err := s.client.GetAttachment(ctx, grantID, messageID, attachmentID)
	if err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"id":           att.ID,
		"filename":     att.Filename,
		"content_type": att.ContentType,
		"size":         att.Size,
		"content_id":   att.ContentID,
		"is_inline":    att.IsInline,
	})
}

// ============================================================================
// SCHEDULED MESSAGE TOOLS
// ============================================================================

// executeListScheduledMessages lists all scheduled (send-later) messages.
func (s *Server) executeListScheduledMessages(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)

	msgs, err := s.client.ListScheduledMessages(ctx, grantID)
	if err != nil {
		return toolError(err.Error())
	}

	result := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, map[string]any{
			"schedule_id": m.ScheduleID,
			"status":      m.Status,
			"close_time":  m.CloseTime,
		})
	}
	return toolSuccess(result)
}

// executeCancelScheduledMessage cancels a scheduled message.
func (s *Server) executeCancelScheduledMessage(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	scheduleID := getString(args, "schedule_id", "")
	if scheduleID == "" {
		return toolError("schedule_id is required")
	}

	if err := s.client.CancelScheduledMessage(ctx, grantID, scheduleID); err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"schedule_id": scheduleID,
		"status":      "cancelled",
	})
}

// ============================================================================
// CONTACT TOOLS
// ============================================================================

// executeListContacts lists or searches contacts.
func (s *Server) executeListContacts(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	params := &domain.ContactQueryParams{
		Limit:       getInt(args, "limit", 10),
		Email:       getString(args, "email", ""),
		PhoneNumber: getString(args, "phone_number", ""),
		Source:      getString(args, "source", ""),
		Group:       getString(args, "group", ""),
	}

	contacts, err := s.client.GetContacts(ctx, grantID, params)
	if err != nil {
		return toolError(err.Error())
	}

	result := make([]map[string]any, 0, len(contacts))
	for _, c := range contacts {
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
		return toolError(err.Error())
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
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"id":           contact.ID,
		"display_name": contact.DisplayName(),
		"status":       "created",
	})
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
	})
}

// executeEpochToDatetime converts a Unix timestamp to a human-readable datetime.
func (s *Server) executeEpochToDatetime(args map[string]any) *ToolResponse {
	epoch := getInt64(args, "epoch", 0)
	if epoch == 0 {
		return toolError("epoch is required")
	}

	loc, err := resolveLocation(getString(args, "timezone", ""))
	if err != nil {
		return toolError("invalid timezone: " + getString(args, "timezone", ""))
	}

	t := time.Unix(epoch, 0).In(loc)
	return toolSuccess(map[string]any{
		"datetime":       t.Format(time.RFC3339),
		"timezone":       loc.String(),
		"unix_timestamp": epoch,
		"human_readable": t.Format("Monday, January 2, 2006 3:04 PM MST"),
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
