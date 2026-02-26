package mcp

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// parseParticipants extracts email participants from tool arguments.
// Accepts an array of objects with "email" and optional "name" fields.
func parseParticipants(args map[string]any, key string) []domain.EmailParticipant {
	val, ok := args[key]
	if !ok {
		return nil
	}
	arr, ok := val.([]any)
	if !ok {
		return nil
	}
	var result []domain.EmailParticipant
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		email, _ := m["email"].(string)
		if email == "" {
			continue
		}
		name, _ := m["name"].(string)
		result = append(result, domain.EmailParticipant{Email: email, Name: name})
	}
	return result
}

// formatParticipants formats email participants for display.
func formatParticipants(participants []domain.EmailParticipant) []string {
	result := make([]string, 0, len(participants))
	for _, p := range participants {
		if p.Name != "" {
			result = append(result, p.Name+" <"+p.Email+">")
		} else {
			result = append(result, p.Email)
		}
	}
	return result
}

// executeListMessages lists email messages with optional filters.
func (s *Server) executeListMessages(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	params := &domain.MessageQueryParams{
		Limit:          getInt(args, "limit", 10),
		Subject:        getString(args, "subject", ""),
		From:           getString(args, "from", ""),
		To:             getString(args, "to", ""),
		ReceivedBefore: getInt64(args, "received_before", 0),
		ReceivedAfter:  getInt64(args, "received_after", 0),
		SearchQuery:    getString(args, "query", ""),
		Unread:         getBool(args, "unread"),
		Starred:        getBool(args, "starred"),
		HasAttachment:  getBool(args, "has_attachment"),
	}
	if folderID := getString(args, "folder_id", ""); folderID != "" {
		params.In = []string{folderID}
	}

	messages, err := s.client.GetMessagesWithParams(ctx, grantID, params)
	if err != nil {
		return toolError(err.Error())
	}

	result := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		from := ""
		if len(msg.From) > 0 {
			from = msg.From[0].String()
		}
		result = append(result, map[string]any{
			"id":        msg.ID,
			"subject":   msg.Subject,
			"from":      from,
			"date":      msg.Date.Format(time.RFC3339),
			"unread":    msg.Unread,
			"starred":   msg.Starred,
			"snippet":   msg.Snippet,
			"thread_id": msg.ThreadID,
			"folders":   msg.Folders,
		})
	}
	return toolSuccess(result)
}

// executeGetMessage retrieves a specific email message.
func (s *Server) executeGetMessage(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	messageID := getString(args, "message_id", "")
	if messageID == "" {
		return toolError("message_id is required")
	}

	msg, err := s.client.GetMessage(ctx, grantID, messageID)
	if err != nil {
		return toolError(err.Error())
	}

	body := msg.Body
	if len(body) > 10000 {
		body = body[:10000]
	}

	from := ""
	if len(msg.From) > 0 {
		from = msg.From[0].String()
	}

	return toolSuccess(map[string]any{
		"id":          msg.ID,
		"subject":     msg.Subject,
		"from":        from,
		"to":          formatParticipants(msg.To),
		"cc":          formatParticipants(msg.Cc),
		"bcc":         formatParticipants(msg.Bcc),
		"date":        msg.Date.Format(time.RFC3339),
		"body":        body,
		"unread":      msg.Unread,
		"starred":     msg.Starred,
		"thread_id":   msg.ThreadID,
		"folders":     msg.Folders,
		"attachments": len(msg.Attachments),
	})
}

// executeSendMessage sends a new email message.
func (s *Server) executeSendMessage(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)

	to := parseParticipants(args, "to")
	if len(to) == 0 {
		return toolError("to is required")
	}
	subject := getString(args, "subject", "")
	if subject == "" {
		return toolError("subject is required")
	}
	body := getString(args, "body", "")
	if body == "" {
		return toolError("body is required")
	}

	req := &domain.SendMessageRequest{
		To:           to,
		Cc:           parseParticipants(args, "cc"),
		Bcc:          parseParticipants(args, "bcc"),
		Subject:      subject,
		Body:         body,
		ReplyToMsgID: getString(args, "reply_to_message_id", ""),
	}

	msg, err := s.client.SendMessage(ctx, grantID, req)
	if err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"id":        msg.ID,
		"thread_id": msg.ThreadID,
		"status":    "sent",
	})
}

// executeUpdateMessage updates message properties (read status, starred, folders).
func (s *Server) executeUpdateMessage(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	messageID := getString(args, "message_id", "")
	if messageID == "" {
		return toolError("message_id is required")
	}

	req := &domain.UpdateMessageRequest{
		Unread:  getBool(args, "unread"),
		Starred: getBool(args, "starred"),
		Folders: getStringSlice(args, "folders"),
	}

	msg, err := s.client.UpdateMessage(ctx, grantID, messageID, req)
	if err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"id":      msg.ID,
		"unread":  msg.Unread,
		"starred": msg.Starred,
		"folders": msg.Folders,
	})
}

// executeDeleteMessage deletes an email message.
func (s *Server) executeDeleteMessage(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	messageID := getString(args, "message_id", "")
	if messageID == "" {
		return toolError("message_id is required")
	}

	if err := s.client.DeleteMessage(ctx, grantID, messageID); err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"status":     "deleted",
		"message_id": messageID,
	})
}

// executeSmartCompose generates an AI-powered email draft.
func (s *Server) executeSmartCompose(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	prompt := getString(args, "prompt", "")
	if prompt == "" {
		return toolError("prompt is required")
	}

	req := &domain.SmartComposeRequest{Prompt: prompt}
	suggestion, err := s.client.SmartCompose(ctx, grantID, req)
	if err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"suggestion": suggestion.Suggestion,
	})
}

// executeSmartComposeReply generates an AI-powered reply suggestion.
func (s *Server) executeSmartComposeReply(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	messageID := getString(args, "message_id", "")
	if messageID == "" {
		return toolError("message_id is required")
	}
	prompt := getString(args, "prompt", "")
	if prompt == "" {
		return toolError("prompt is required")
	}

	req := &domain.SmartComposeRequest{Prompt: prompt}
	suggestion, err := s.client.SmartComposeReply(ctx, grantID, messageID, req)
	if err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"suggestion": suggestion.Suggestion,
	})
}

// executeListDrafts lists email drafts.
func (s *Server) executeListDrafts(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	limit := getInt(args, "limit", 10)

	drafts, err := s.client.GetDrafts(ctx, grantID, limit)
	if err != nil {
		return toolError(err.Error())
	}

	result := make([]map[string]any, 0, len(drafts))
	for _, d := range drafts {
		result = append(result, map[string]any{
			"id":      d.ID,
			"subject": d.Subject,
			"to":      formatParticipants(d.To),
			"date":    d.CreatedAt.Format(time.RFC3339),
		})
	}
	return toolSuccess(result)
}

// executeGetDraft retrieves a specific email draft.
func (s *Server) executeGetDraft(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	draftID := getString(args, "draft_id", "")
	if draftID == "" {
		return toolError("draft_id is required")
	}

	draft, err := s.client.GetDraft(ctx, grantID, draftID)
	if err != nil {
		return toolError(err.Error())
	}

	from := ""
	if len(draft.From) > 0 {
		from = draft.From[0].String()
	}

	return toolSuccess(map[string]any{
		"id":                  draft.ID,
		"subject":             draft.Subject,
		"from":                from,
		"to":                  formatParticipants(draft.To),
		"cc":                  formatParticipants(draft.Cc),
		"bcc":                 formatParticipants(draft.Bcc),
		"body":                draft.Body,
		"reply_to_message_id": draft.ReplyToMsgID,
	})
}

// executeCreateDraft creates a new email draft.
func (s *Server) executeCreateDraft(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)

	req := &domain.CreateDraftRequest{
		Subject:      getString(args, "subject", ""),
		Body:         getString(args, "body", ""),
		To:           parseParticipants(args, "to"),
		Cc:           parseParticipants(args, "cc"),
		Bcc:          parseParticipants(args, "bcc"),
		ReplyToMsgID: getString(args, "reply_to_message_id", ""),
	}

	draft, err := s.client.CreateDraft(ctx, grantID, req)
	if err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"id":      draft.ID,
		"subject": draft.Subject,
		"status":  "created",
	})
}

// executeUpdateDraft updates an existing email draft.
func (s *Server) executeUpdateDraft(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	draftID := getString(args, "draft_id", "")
	if draftID == "" {
		return toolError("draft_id is required")
	}

	req := &domain.CreateDraftRequest{
		Subject:      getString(args, "subject", ""),
		Body:         getString(args, "body", ""),
		To:           parseParticipants(args, "to"),
		Cc:           parseParticipants(args, "cc"),
		Bcc:          parseParticipants(args, "bcc"),
		ReplyToMsgID: getString(args, "reply_to_message_id", ""),
	}

	draft, err := s.client.UpdateDraft(ctx, grantID, draftID, req)
	if err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"id":      draft.ID,
		"subject": draft.Subject,
		"status":  "updated",
	})
}

// executeSendDraft sends a draft as an email message.
func (s *Server) executeSendDraft(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	draftID := getString(args, "draft_id", "")
	if draftID == "" {
		return toolError("draft_id is required")
	}

	msg, err := s.client.SendDraft(ctx, grantID, draftID)
	if err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"id":        msg.ID,
		"thread_id": msg.ThreadID,
		"status":    "sent",
	})
}

// executeDeleteDraft deletes an email draft.
func (s *Server) executeDeleteDraft(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	draftID := getString(args, "draft_id", "")
	if draftID == "" {
		return toolError("draft_id is required")
	}

	if err := s.client.DeleteDraft(ctx, grantID, draftID); err != nil {
		return toolError(err.Error())
	}

	return toolSuccess(map[string]any{
		"status":   "deleted",
		"draft_id": draftID,
	})
}
