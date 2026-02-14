package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// requireSlack checks if Slack integration is available and returns an error result if not.
func (e *ToolExecutor) requireSlack(toolName string) (ToolResult, bool) {
	if e.slack == nil {
		return ToolResult{
			Name:  toolName,
			Error: "Slack integration not configured",
		}, false
	}
	return ToolResult{}, true
}

// resolveSlackChannel converts a channel name or ID to a channel ID.
// If input is already an ID (starts with C/G/D and length 9+), returns it directly.
// Otherwise tries: exact match, then prefix match, then search fallback.
func (e *ToolExecutor) resolveSlackChannel(ctx context.Context, input string) (string, error) {
	// Check if input is already a channel ID
	if isChannelID(input) {
		return input, nil
	}

	// Treat as channel name - strip # prefix and normalize
	name := strings.TrimPrefix(input, "#")
	name = strings.ToLower(name)

	// Paginate through user's channels (excludes DMs/IMs for efficient pagination)
	// Track best prefix match in case exact match isn't found
	var prefixMatch string
	cursor := ""
	for range 5 {
		resp, err := e.slack.ListMyChannels(ctx, &domain.SlackChannelQueryParams{
			Limit:           1000,
			ExcludeArchived: true,
			Cursor:          cursor,
		})
		if err != nil {
			return "", fmt.Errorf("failed to list channels: %w", err)
		}

		for _, ch := range resp.Channels {
			chName := strings.ToLower(ch.Name)
			if chName == name {
				return ch.ID, nil
			}
			// Track prefix match (e.g. "incident-foo" matches "incident-foo-748")
			if prefixMatch == "" && strings.HasPrefix(chName, name) {
				prefixMatch = ch.ID
			}
		}

		if resp.NextCursor == "" {
			break
		}
		cursor = resp.NextCursor
	}

	// Use prefix match if found
	if prefixMatch != "" {
		return prefixMatch, nil
	}

	// Fallback: search for a message in the channel to discover its ID
	results, err := e.slack.SearchMessages(ctx, "in:#"+name, 1)
	if err == nil && len(results) > 0 && results[0].ChannelID != "" {
		return results[0].ChannelID, nil
	}

	return "", fmt.Errorf("channel not found: %q", input)
}

// listSlackChannels returns a list of Slack channels accessible to the user.
func (e *ToolExecutor) listSlackChannels(ctx context.Context, args map[string]any) ToolResult {
	if result, ok := e.requireSlack("list_slack_channels"); !ok {
		return result
	}

	// Parse limit
	limit := 20
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	// Get channels
	resp, err := e.slack.ListMyChannels(ctx, &domain.SlackChannelQueryParams{
		Limit:           limit,
		ExcludeArchived: true,
	})
	if err != nil {
		return ToolResult{
			Name:  "list_slack_channels",
			Error: fmt.Sprintf("failed to list channels: %v", err),
		}
	}

	// Build response
	type channelSummary struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		MemberCount int    `json:"member_count"`
		Topic       string `json:"topic,omitempty"`
	}

	channels := make([]channelSummary, len(resp.Channels))
	for i, ch := range resp.Channels {
		channels[i] = channelSummary{
			ID:          ch.ID,
			Name:        ch.ChannelDisplayName(),
			Type:        ch.ChannelType(),
			MemberCount: ch.MemberCount,
			Topic:       ch.Topic,
		}
	}

	return ToolResult{
		Name: "list_slack_channels",
		Data: channels,
	}
}

// readSlackMessages returns messages from a Slack channel with threads expanded inline.
// Falls back to search API when conversations.history returns too few results
// (some private channels have limited history access but full search access).
func (e *ToolExecutor) readSlackMessages(ctx context.Context, args map[string]any) ToolResult {
	if result, ok := e.requireSlack("read_slack_messages"); !ok {
		return result
	}

	// Get required channel parameter
	channel, ok := args["channel"].(string)
	if !ok || channel == "" {
		return ToolResult{
			Name:  "read_slack_messages",
			Error: "missing required parameter: channel",
		}
	}

	// Resolve channel name to ID
	channelID, err := e.resolveSlackChannel(ctx, channel)
	if err != nil {
		return ToolResult{
			Name:  "read_slack_messages",
			Error: err.Error(),
		}
	}

	// Parse limit
	limit := 500
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	// Get messages via conversations.history
	resp, err := e.slack.GetMessages(ctx, &domain.SlackMessageQueryParams{
		ChannelID: channelID,
		Limit:     limit,
	})
	if err != nil {
		return ToolResult{
			Name:  "read_slack_messages",
			Error: fmt.Sprintf("failed to get messages: %v", err),
		}
	}

	// If history returned very few messages, fall back to search API
	// (search API has broader access for some channel types like Rootly incident channels)
	if len(resp.Messages) < 5 {
		searchQuery := buildChannelSearchQuery(channel, channelID)
		searchResults, searchErr := e.slack.SearchMessages(ctx, searchQuery, limit)
		if searchErr == nil && len(searchResults) > len(resp.Messages) {
			return e.formatSearchMessages(searchResults)
		}
	}

	return e.formatHistoryMessages(ctx, channelID, resp.Messages)
}

// buildChannelSearchQuery creates a Slack search query to find messages in a channel.
func buildChannelSearchQuery(channel, channelID string) string {
	// If original input is a channel name, use it directly
	name := strings.TrimPrefix(channel, "#")
	if !isChannelID(name) {
		return "in:#" + strings.ToLower(name)
	}
	// For channel IDs, use the Slack channel link format
	return "in:<#" + channelID + ">"
}

// isChannelID returns true if the string looks like a Slack channel ID (C/G/D prefix + 9+ chars).
func isChannelID(s string) bool {
	if len(s) < 9 {
		return false
	}
	return s[0] == 'C' || s[0] == 'G' || s[0] == 'D'
}

// formatSearchMessages formats search results as a ToolResult.
func (e *ToolExecutor) formatSearchMessages(results []domain.SlackMessage) ToolResult {
	type msgSummary struct {
		ID        string `json:"id"`
		Username  string `json:"username"`
		Text      string `json:"text"`
		Timestamp string `json:"timestamp"`
	}

	messages := make([]msgSummary, 0, len(results))
	for _, msg := range results {
		text := msg.Text
		if len(text) > 500 {
			text = text[:500] + "..."
		}
		messages = append(messages, msgSummary{
			ID:        msg.ID,
			Username:  msg.Username,
			Text:      text,
			Timestamp: msg.Timestamp.Format(time.RFC3339),
		})
	}

	return ToolResult{
		Name: "read_slack_messages",
		Data: messages,
	}
}

// formatHistoryMessages formats conversation history results with thread expansion.
func (e *ToolExecutor) formatHistoryMessages(ctx context.Context, channelID string, msgs []domain.SlackMessage) ToolResult {
	type msgSummary struct {
		ID        string `json:"id"`
		Username  string `json:"username"`
		Text      string `json:"text"`
		Timestamp string `json:"timestamp"`
		IsReply   bool   `json:"is_reply,omitempty"`
	}

	var messages []msgSummary
	for _, msg := range msgs {
		text := msg.Text
		if len(text) > 500 {
			text = text[:500] + "..."
		}

		messages = append(messages, msgSummary{
			ID:        msg.ID,
			Username:  msg.Username,
			Text:      text,
			Timestamp: msg.Timestamp.Format(time.RFC3339),
		})

		// Expand thread replies inline
		if msg.ReplyCount > 0 {
			replies, replyErr := e.slack.GetThreadReplies(ctx, channelID, msg.ID, 100)
			if replyErr == nil && len(replies) > 1 {
				// Skip first reply (parent message already included)
				for _, reply := range replies[1:] {
					replyText := reply.Text
					if len(replyText) > 500 {
						replyText = replyText[:500] + "..."
					}
					messages = append(messages, msgSummary{
						ID:        reply.ID,
						Username:  reply.Username,
						Text:      replyText,
						Timestamp: reply.Timestamp.Format(time.RFC3339),
						IsReply:   true,
					})
				}
			}
		}
	}

	return ToolResult{
		Name: "read_slack_messages",
		Data: messages,
	}
}

// readSlackThread returns messages from a Slack thread.
func (e *ToolExecutor) readSlackThread(ctx context.Context, args map[string]any) ToolResult {
	if result, ok := e.requireSlack("read_slack_thread"); !ok {
		return result
	}

	// Get required parameters
	channel, ok := args["channel"].(string)
	if !ok || channel == "" {
		return ToolResult{
			Name:  "read_slack_thread",
			Error: "missing required parameter: channel",
		}
	}

	threadTS, ok := args["thread_ts"].(string)
	if !ok || threadTS == "" {
		return ToolResult{
			Name:  "read_slack_thread",
			Error: "missing required parameter: thread_ts",
		}
	}

	// Resolve channel name to ID
	channelID, err := e.resolveSlackChannel(ctx, channel)
	if err != nil {
		return ToolResult{
			Name:  "read_slack_thread",
			Error: err.Error(),
		}
	}

	// Parse limit
	limit := 20
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	// Get thread replies
	replies, err := e.slack.GetThreadReplies(ctx, channelID, threadTS, limit)
	if err != nil {
		return ToolResult{
			Name:  "read_slack_thread",
			Error: fmt.Sprintf("failed to get thread replies: %v", err),
		}
	}

	// Build response
	type replySummary struct {
		ID        string `json:"id"`
		Username  string `json:"username"`
		Text      string `json:"text"`
		Timestamp string `json:"timestamp"`
		IsReply   bool   `json:"is_reply"`
	}

	messages := make([]replySummary, len(replies))
	for i, msg := range replies {
		text := msg.Text
		if len(text) > 500 {
			text = text[:500] + "..."
		}

		messages[i] = replySummary{
			ID:        msg.ID,
			Username:  msg.Username,
			Text:      text,
			Timestamp: msg.Timestamp.Format(time.RFC3339),
			IsReply:   msg.IsReply,
		}
	}

	return ToolResult{
		Name: "read_slack_thread",
		Data: messages,
	}
}

// searchSlack searches for messages matching a query.
func (e *ToolExecutor) searchSlack(ctx context.Context, args map[string]any) ToolResult {
	if result, ok := e.requireSlack("search_slack"); !ok {
		return result
	}

	// Get required query parameter
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return ToolResult{
			Name:  "search_slack",
			Error: "missing required parameter: query",
		}
	}

	// Parse limit
	limit := 10
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	// Search messages
	results, err := e.slack.SearchMessages(ctx, query, limit)
	if err != nil {
		return ToolResult{
			Name:  "search_slack",
			Error: fmt.Sprintf("failed to search messages: %v", err),
		}
	}

	// Build response
	type searchResult struct {
		ID        string `json:"id"`
		ChannelID string `json:"channel_id"`
		Username  string `json:"username"`
		Text      string `json:"text"`
		Timestamp string `json:"timestamp"`
	}

	messages := make([]searchResult, len(results))
	for i, msg := range results {
		text := msg.Text
		if len(text) > 500 {
			text = text[:500] + "..."
		}

		messages[i] = searchResult{
			ID:        msg.ID,
			ChannelID: msg.ChannelID,
			Username:  msg.Username,
			Text:      text,
			Timestamp: msg.Timestamp.Format(time.RFC3339),
		}
	}

	return ToolResult{
		Name: "search_slack",
		Data: messages,
	}
}

// sendSlackMessage sends a message to a Slack channel.
func (e *ToolExecutor) sendSlackMessage(ctx context.Context, args map[string]any) ToolResult {
	if result, ok := e.requireSlack("send_slack_message"); !ok {
		return result
	}

	// Get required parameters
	channel, ok := args["channel"].(string)
	if !ok || channel == "" {
		return ToolResult{
			Name:  "send_slack_message",
			Error: "missing required parameter: channel",
		}
	}

	text, ok := args["text"].(string)
	if !ok || text == "" {
		return ToolResult{
			Name:  "send_slack_message",
			Error: "missing required parameter: text",
		}
	}

	// Resolve channel name to ID
	channelID, err := e.resolveSlackChannel(ctx, channel)
	if err != nil {
		return ToolResult{
			Name:  "send_slack_message",
			Error: err.Error(),
		}
	}

	// Get optional thread_ts
	threadTS, _ := args["thread_ts"].(string)

	// Send message
	req := &domain.SlackSendMessageRequest{
		ChannelID: channelID,
		Text:      text,
		ThreadTS:  threadTS,
	}

	msg, err := e.slack.SendMessage(ctx, req)
	if err != nil {
		return ToolResult{
			Name:  "send_slack_message",
			Error: fmt.Sprintf("failed to send message: %v", err),
		}
	}

	// Build response
	response := map[string]any{
		"id":     msg.ID,
		"status": "sent",
	}

	return ToolResult{
		Name: "send_slack_message",
		Data: response,
	}
}

// listSlackUsers returns a list of Slack workspace users.
func (e *ToolExecutor) listSlackUsers(ctx context.Context, args map[string]any) ToolResult {
	if result, ok := e.requireSlack("list_slack_users"); !ok {
		return result
	}

	// Parse limit
	limit := 20
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	// Get users
	resp, err := e.slack.ListUsers(ctx, limit, "")
	if err != nil {
		return ToolResult{
			Name:  "list_slack_users",
			Error: fmt.Sprintf("failed to list users: %v", err),
		}
	}

	// Build response
	type userSummary struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Title       string `json:"title,omitempty"`
		IsBot       bool   `json:"is_bot,omitempty"`
	}

	users := make([]userSummary, len(resp.Users))
	for i, user := range resp.Users {
		users[i] = userSummary{
			ID:          user.ID,
			Name:        user.Name,
			DisplayName: user.BestDisplayName(),
			Title:       user.Title,
			IsBot:       user.IsBot,
		}
	}

	return ToolResult{
		Name: "list_slack_users",
		Data: users,
	}
}
