package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// ToolExecutor dispatches tool calls to the Nylas API.
type ToolExecutor struct {
	client  ports.NylasClient
	grantID string
}

// NewToolExecutor creates a new ToolExecutor.
func NewToolExecutor(client ports.NylasClient, grantID string) *ToolExecutor {
	return &ToolExecutor{client: client, grantID: grantID}
}

// Execute runs a tool call and returns the result.
func (e *ToolExecutor) Execute(ctx context.Context, call ToolCall) ToolResult {
	switch call.Name {
	case "list_emails":
		return e.listEmails(ctx, call.Args)
	case "read_email":
		return e.readEmail(ctx, call.Args)
	case "search_emails":
		return e.searchEmails(ctx, call.Args)
	case "send_email":
		return e.sendEmail(ctx, call.Args)
	case "list_events":
		return e.listEvents(ctx, call.Args)
	case "create_event":
		return e.createEvent(ctx, call.Args)
	case "list_contacts":
		return e.listContacts(ctx, call.Args)
	case "list_folders":
		return e.listFolders(ctx)
	default:
		return ToolResult{Name: call.Name, Error: fmt.Sprintf("unknown tool: %s", call.Name)}
	}
}

func (e *ToolExecutor) listEmails(ctx context.Context, args map[string]any) ToolResult {
	params := &domain.MessageQueryParams{Limit: 10}

	if v, ok := args["limit"]; ok {
		if f, ok := v.(float64); ok {
			params.Limit = int(f)
		}
	}
	if v, ok := args["subject"]; ok {
		if s, ok := v.(string); ok {
			params.Subject = s
		}
	}
	if v, ok := args["from"]; ok {
		if s, ok := v.(string); ok {
			params.From = s
		}
	}
	if v, ok := args["unread"]; ok {
		if b, ok := v.(bool); ok {
			params.Unread = &b
		}
	}

	messages, err := e.client.GetMessagesWithParams(ctx, e.grantID, params)
	if err != nil {
		return ToolResult{Name: "list_emails", Error: err.Error()}
	}

	// Return simplified message list
	type emailSummary struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
		From    string `json:"from"`
		Date    string `json:"date"`
		Unread  bool   `json:"unread"`
		Snippet string `json:"snippet"`
	}

	var results []emailSummary
	for _, m := range messages {
		from := ""
		if len(m.From) > 0 {
			from = m.From[0].Email
			if m.From[0].Name != "" {
				from = m.From[0].Name + " <" + m.From[0].Email + ">"
			}
		}
		results = append(results, emailSummary{
			ID:      m.ID,
			Subject: m.Subject,
			From:    from,
			Date:    m.Date.Format(time.RFC3339),
			Unread:  m.Unread,
			Snippet: m.Snippet,
		})
	}

	return ToolResult{Name: "list_emails", Data: results}
}

func (e *ToolExecutor) readEmail(ctx context.Context, args map[string]any) ToolResult {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return ToolResult{Name: "read_email", Error: "id parameter is required"}
	}

	msg, err := e.client.GetMessage(ctx, e.grantID, id)
	if err != nil {
		return ToolResult{Name: "read_email", Error: err.Error()}
	}

	type emailDetail struct {
		ID      string   `json:"id"`
		Subject string   `json:"subject"`
		From    string   `json:"from"`
		To      []string `json:"to"`
		Date    string   `json:"date"`
		Body    string   `json:"body"`
	}

	from := ""
	if len(msg.From) > 0 {
		from = msg.From[0].Email
		if msg.From[0].Name != "" {
			from = msg.From[0].Name + " <" + msg.From[0].Email + ">"
		}
	}

	var to []string
	for _, t := range msg.To {
		to = append(to, t.Email)
	}

	body := msg.Body
	if len(body) > 5000 {
		body = body[:5000] + "\n... [truncated]"
	}

	return ToolResult{Name: "read_email", Data: emailDetail{
		ID:      msg.ID,
		Subject: msg.Subject,
		From:    from,
		To:      to,
		Date:    msg.Date.Format(time.RFC3339),
		Body:    body,
	}}
}

func (e *ToolExecutor) searchEmails(ctx context.Context, args map[string]any) ToolResult {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return ToolResult{Name: "search_emails", Error: "query parameter is required"}
	}

	params := &domain.MessageQueryParams{
		Limit:       10,
		SearchQuery: query,
	}

	if v, ok := args["limit"]; ok {
		if f, ok := v.(float64); ok {
			params.Limit = int(f)
		}
	}

	messages, err := e.client.GetMessagesWithParams(ctx, e.grantID, params)
	if err != nil {
		return ToolResult{Name: "search_emails", Error: err.Error()}
	}

	type emailSummary struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
		From    string `json:"from"`
		Date    string `json:"date"`
		Snippet string `json:"snippet"`
	}

	var results []emailSummary
	for _, m := range messages {
		from := ""
		if len(m.From) > 0 {
			from = m.From[0].Email
		}
		results = append(results, emailSummary{
			ID:      m.ID,
			Subject: m.Subject,
			From:    from,
			Date:    m.Date.Format(time.RFC3339),
			Snippet: m.Snippet,
		})
	}

	return ToolResult{Name: "search_emails", Data: results}
}

func (e *ToolExecutor) sendEmail(ctx context.Context, args map[string]any) ToolResult {
	to, _ := args["to"].(string)
	subject, _ := args["subject"].(string)
	body, _ := args["body"].(string)

	if to == "" || subject == "" || body == "" {
		return ToolResult{Name: "send_email", Error: "to, subject, and body are required"}
	}

	req := &domain.SendMessageRequest{
		To:      []domain.EmailParticipant{{Email: to}},
		Subject: subject,
		Body:    body,
	}

	msg, err := e.client.SendMessage(ctx, e.grantID, req)
	if err != nil {
		return ToolResult{Name: "send_email", Error: err.Error()}
	}

	return ToolResult{Name: "send_email", Data: map[string]string{
		"id":     msg.ID,
		"status": "sent",
	}}
}

func (e *ToolExecutor) listEvents(ctx context.Context, args map[string]any) ToolResult {
	calendarID := "primary"
	if v, ok := args["calendar_id"].(string); ok && v != "" {
		calendarID = v
	}

	params := &domain.EventQueryParams{Limit: 10}
	if v, ok := args["limit"]; ok {
		if f, ok := v.(float64); ok {
			params.Limit = int(f)
		}
	}

	events, err := e.client.GetEvents(ctx, e.grantID, calendarID, params)
	if err != nil {
		return ToolResult{Name: "list_events", Error: err.Error()}
	}

	type eventSummary struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Start string `json:"start"`
		End   string `json:"end"`
	}

	var results []eventSummary
	for _, ev := range events {
		start := ""
		end := ""
		if ev.When.StartTime > 0 {
			start = time.Unix(ev.When.StartTime, 0).Format(time.RFC3339)
		}
		if ev.When.EndTime > 0 {
			end = time.Unix(ev.When.EndTime, 0).Format(time.RFC3339)
		}
		results = append(results, eventSummary{
			ID:    ev.ID,
			Title: ev.Title,
			Start: start,
			End:   end,
		})
	}

	return ToolResult{Name: "list_events", Data: results}
}

func (e *ToolExecutor) createEvent(ctx context.Context, args map[string]any) ToolResult {
	title, _ := args["title"].(string)
	startStr, _ := args["start_time"].(string)
	endStr, _ := args["end_time"].(string)

	if title == "" || startStr == "" || endStr == "" {
		return ToolResult{Name: "create_event", Error: "title, start_time, and end_time are required"}
	}

	startTime, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return ToolResult{Name: "create_event", Error: fmt.Sprintf("invalid start_time: %v", err)}
	}
	endTime, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return ToolResult{Name: "create_event", Error: fmt.Sprintf("invalid end_time: %v", err)}
	}

	calendarID := "primary"
	if v, ok := args["calendar_id"].(string); ok && v != "" {
		calendarID = v
	}

	desc, _ := args["description"].(string)

	req := &domain.CreateEventRequest{
		Title: title,
		When: domain.EventWhen{
			StartTime: startTime.Unix(),
			EndTime:   endTime.Unix(),
		},
		Description: desc,
	}

	event, err := e.client.CreateEvent(ctx, e.grantID, calendarID, req)
	if err != nil {
		return ToolResult{Name: "create_event", Error: err.Error()}
	}

	return ToolResult{Name: "create_event", Data: map[string]string{
		"id":    event.ID,
		"title": event.Title,
	}}
}

func (e *ToolExecutor) listContacts(ctx context.Context, args map[string]any) ToolResult {
	params := &domain.ContactQueryParams{Limit: 10}

	if v, ok := args["limit"]; ok {
		if f, ok := v.(float64); ok {
			params.Limit = int(f)
		}
	}
	if v, ok := args["query"].(string); ok && v != "" {
		params.Email = v
	}

	contacts, err := e.client.GetContacts(ctx, e.grantID, params)
	if err != nil {
		return ToolResult{Name: "list_contacts", Error: err.Error()}
	}

	type contactSummary struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	var results []contactSummary
	for _, c := range contacts {
		email := ""
		if len(c.Emails) > 0 {
			email = c.Emails[0].Email
		}
		name := ""
		if c.GivenName != "" {
			name = c.GivenName
			if c.Surname != "" {
				name += " " + c.Surname
			}
		}
		results = append(results, contactSummary{
			ID:    c.ID,
			Name:  name,
			Email: email,
		})
	}

	return ToolResult{Name: "list_contacts", Data: results}
}

func (e *ToolExecutor) listFolders(ctx context.Context) ToolResult {
	folders, err := e.client.GetFolders(ctx, e.grantID)
	if err != nil {
		return ToolResult{Name: "list_folders", Error: err.Error()}
	}

	type folderSummary struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	var results []folderSummary
	for _, f := range folders {
		results = append(results, folderSummary{
			ID:   f.ID,
			Name: f.Name,
		})
	}

	return ToolResult{Name: "list_folders", Data: results}
}
