package mcp

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// maxBodyLen caps email body length to reduce token usage.
const maxBodyLen = 10000

// reHTMLTag matches HTML tags for stripping.
var reHTMLTag = regexp.MustCompile(`<[^>]*>`)

// stripHTML removes HTML tags and normalizes whitespace to produce plain text.
func stripHTML(s string) string {
	// Replace common block elements with newlines for readability.
	s = strings.NewReplacer(
		"<br>", "\n", "<br/>", "\n", "<br />", "\n",
		"<BR>", "\n", "<BR/>", "\n", "<BR />", "\n",
		"</p>", "\n", "</P>", "\n",
		"</div>", "\n", "</DIV>", "\n",
		"</tr>", "\n", "</TR>", "\n",
		"</li>", "\n", "</LI>", "\n",
	).Replace(s)

	// Strip all remaining HTML tags.
	s = reHTMLTag.ReplaceAllString(s, "")

	// Decode common HTML entities.
	s = strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
		"&apos;", "'",
		"&nbsp;", " ",
	).Replace(s)

	// Collapse runs of blank lines to at most two newlines.
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(s)
}

// maxSnippetLen caps snippet length to reduce token usage in list responses.
const maxSnippetLen = 120

// cleanSnippet removes invisible padding characters and trims to a reasonable length.
func cleanSnippet(s string) string {
	// Remove zero-width non-joiners, zero-width spaces, and other invisible chars.
	s = strings.NewReplacer(
		"\u200c", "", // zero-width non-joiner (‌)
		"\u200b", "", // zero-width space
		"\u034f", "", // combining grapheme joiner (͏)
		"\r\n", " ",
		"\r", " ",
		"\n", " ",
	).Replace(s)

	// Collapse multiple spaces.
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	s = strings.TrimSpace(s)

	runes := []rune(s)
	if len(runes) > maxSnippetLen {
		s = string(runes[:maxSnippetLen]) + "..."
	}
	return s
}

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
		Limit:          clampLimit(args, "limit", 10),
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
	if pageToken := getString(args, "page_token", ""); pageToken != "" {
		params.PageToken = pageToken
	}

	resp, err := s.client.GetMessagesWithCursor(ctx, grantID, params)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	result := make([]map[string]any, 0, len(resp.Data))
	for _, msg := range resp.Data {
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
			"snippet":   cleanSnippet(msg.Snippet),
			"thread_id": msg.ThreadID,
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

// executeGetMessage retrieves a specific email message.
func (s *Server) executeGetMessage(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	messageID := getString(args, "message_id", "")
	if messageID == "" {
		return toolError("message_id is required")
	}

	msg, err := s.client.GetMessage(ctx, grantID, messageID)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	body := stripHTML(msg.Body)
	if bodyRunes := []rune(body); len(bodyRunes) > maxBodyLen {
		body = string(bodyRunes[:maxBodyLen])
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
		return toolError(sanitizeError(err))
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
		return toolError(sanitizeError(err))
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
		return toolError(sanitizeError(err))
	}

	return toolSuccessText("Deleted message " + messageID)
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
		return toolError(sanitizeError(err))
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
		return toolError(sanitizeError(err))
	}

	return toolSuccess(map[string]any{
		"suggestion": suggestion.Suggestion,
	})
}

// executeListDrafts lists email drafts.
func (s *Server) executeListDrafts(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	limit := clampLimit(args, "limit", 10)

	drafts, err := s.client.GetDrafts(ctx, grantID, limit)
	if err != nil {
		return toolError(sanitizeError(err))
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
		return toolError(sanitizeError(err))
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
		return toolError(sanitizeError(err))
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
		return toolError(sanitizeError(err))
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
		return toolError(sanitizeError(err))
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
		return toolError(sanitizeError(err))
	}

	return toolSuccessText("Deleted draft " + draftID)
}
