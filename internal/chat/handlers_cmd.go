package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/nylas/cli/internal/httputil"
)

// commandRequest is the body for POST /api/command.
type commandRequest struct {
	Name           string `json:"name"`
	Args           string `json:"args"`
	ConversationID string `json:"conversation_id"`
}

// commandResponse is the JSON response from a slash command.
type commandResponse struct {
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleCommand dispatches slash commands.
// POST /api/command
func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req commandRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "command name is required", http.StatusBadRequest)
		return
	}

	var resp commandResponse

	switch req.Name {
	case "status":
		resp = s.cmdStatus()
	case "email":
		resp = s.cmdEmail(r.Context(), req.Args)
	case "calendar":
		resp = s.cmdCalendar(r.Context(), req.Args)
	case "contacts":
		resp = s.cmdContacts(r.Context(), req.Args)
	default:
		resp = commandResponse{Error: "unknown command: " + req.Name}
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// cmdStatus returns current session info.
func (s *Server) cmdStatus() commandResponse {
	agent := s.ActiveAgent()
	convs, _ := s.memory.List()

	return commandResponse{
		Content: fmt.Sprintf("**Status**\n- Agent: `%s`\n- Grant ID: `%s`\n- Conversations: %d",
			agent.String(), s.grantID, len(convs)),
	}
}

// cmdEmail runs a quick email lookup.
func (s *Server) cmdEmail(ctx context.Context, query string) commandResponse {
	args := map[string]any{"limit": float64(5)}
	if query != "" {
		// Use as search query
		result := s.executor.Execute(ctx, ToolCall{Name: "search_emails", Args: map[string]any{
			"query": query,
			"limit": float64(5),
		}})
		return toolResultToResponse(result, "emails")
	}

	result := s.executor.Execute(ctx, ToolCall{Name: "list_emails", Args: args})
	return toolResultToResponse(result, "emails")
}

// cmdCalendar lists upcoming events.
func (s *Server) cmdCalendar(ctx context.Context, daysStr string) commandResponse {
	limit := float64(10)
	if daysStr != "" {
		if n, err := strconv.Atoi(daysStr); err == nil && n > 0 {
			limit = float64(n)
		}
	}

	result := s.executor.Execute(ctx, ToolCall{Name: "list_events", Args: map[string]any{
		"limit": limit,
	}})
	return toolResultToResponse(result, "events")
}

// cmdContacts searches contacts.
func (s *Server) cmdContacts(ctx context.Context, query string) commandResponse {
	args := map[string]any{"limit": float64(10)}
	if query != "" {
		args["query"] = query
	}

	result := s.executor.Execute(ctx, ToolCall{Name: "list_contacts", Args: args})
	return toolResultToResponse(result, "contacts")
}

// toolResultToResponse converts a ToolResult into a command response.
func toolResultToResponse(result ToolResult, label string) commandResponse {
	if result.Error != "" {
		return commandResponse{Error: result.Error}
	}

	data, err := json.MarshalIndent(result.Data, "", "  ")
	if err != nil {
		return commandResponse{Error: "failed to format results"}
	}

	return commandResponse{
		Content: fmt.Sprintf("**%s results:**\n```json\n%s\n```", label, string(data)),
	}
}
