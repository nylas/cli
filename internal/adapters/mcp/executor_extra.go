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

// executeUpdateThread updates a thread's read/star/folder state.
func (s *Server) executeUpdateThread(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	threadID := getString(args, "thread_id", "")
	if threadID == "" {
		return toolError("thread_id is required")
	}

	req := &domain.UpdateMessageRequest{
		Unread:  getBool(args, "unread"),
		Starred: getBool(args, "starred"),
		Folders: getStringSlice(args, "folders"),
	}

	t, err := s.client.UpdateThread(ctx, grantID, threadID, req)
	if err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"id":     t.ID,
		"status": "updated",
	})
}

// executeDeleteThread deletes an email thread.
func (s *Server) executeDeleteThread(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	threadID := getString(args, "thread_id", "")
	if threadID == "" {
		return toolError("thread_id is required")
	}

	if err := s.client.DeleteThread(ctx, grantID, threadID); err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"status":    "deleted",
		"thread_id": threadID,
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

// executeUpdateFolder updates an email folder.
func (s *Server) executeUpdateFolder(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	folderID := getString(args, "folder_id", "")
	if folderID == "" {
		return toolError("folder_id is required")
	}

	req := &domain.UpdateFolderRequest{
		Name:            getString(args, "name", ""),
		ParentID:        getString(args, "parent_id", ""),
		BackgroundColor: getString(args, "background_color", ""),
		TextColor:       getString(args, "text_color", ""),
	}

	folder, err := s.client.UpdateFolder(ctx, grantID, folderID, req)
	if err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"id":     folder.ID,
		"name":   folder.Name,
		"status": "updated",
	})
}

// executeDeleteFolder deletes an email folder.
func (s *Server) executeDeleteFolder(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	folderID := getString(args, "folder_id", "")
	if folderID == "" {
		return toolError("folder_id is required")
	}

	if err := s.client.DeleteFolder(ctx, grantID, folderID); err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"status":    "deleted",
		"folder_id": folderID,
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

