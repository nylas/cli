package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nylas/cli/internal/httputil"
)

// chatRequest is the request body for POST /api/chat.
type chatRequest struct {
	Message        string `json:"message"`
	ConversationID string `json:"conversation_id"`
	Agent          string `json:"agent,omitempty"` // optional: switch agent for this message
}

// maxToolIterations is the maximum number of tool call rounds per message.
const maxToolIterations = 5

// handleChat processes a chat message via SSE streaming.
// POST /api/chat
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req chatRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	// Switch agent if requested
	if req.Agent != "" {
		if !s.SetAgent(AgentType(req.Agent)) {
			http.Error(w, "unknown agent: "+req.Agent, http.StatusBadRequest)
			return
		}
	}

	agent := s.ActiveAgent()

	// Load or create conversation
	var conv *Conversation
	var err error

	if req.ConversationID != "" {
		conv, err = s.memory.Get(req.ConversationID)
		if err != nil {
			http.Error(w, "Conversation not found", http.StatusNotFound)
			return
		}
	} else {
		conv, err = s.memory.Create(string(agent.Type))
		if err != nil {
			http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
			return
		}
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Save user message
	_ = s.memory.AddMessage(conv.ID, Message{Role: "user", Content: req.Message})

	// Reload conversation with the new message
	conv, _ = s.memory.Get(conv.ID)

	// Check if compaction needed
	if s.context.NeedsCompaction(conv) {
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		_ = s.context.Compact(ctx, conv)
		cancel()
		conv, _ = s.memory.Get(conv.ID) // reload after compaction
	}

	// Send thinking event
	sendSSE(w, flusher, "thinking", map[string]string{"agent": string(agent.Type)})

	// Build prompt and run agent loop
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	prompt := s.context.BuildPrompt(conv, req.Message)
	var finalResponse string

	for i := range maxToolIterations {
		_ = i

		response, err := agent.Run(ctx, prompt)
		if err != nil {
			sendSSE(w, flusher, "error", map[string]string{"error": err.Error()})
			return
		}

		// Parse tool calls from response
		toolCalls, textResponse := ParseToolCalls(response)

		if len(toolCalls) == 0 {
			// No tool calls - this is the final response
			finalResponse = textResponse
			break
		}

		// Execute tool calls and collect results
		for _, call := range toolCalls {
			sendSSE(w, flusher, "tool_call", map[string]any{
				"name": call.Name,
				"args": call.Args,
			})

			result := s.executor.Execute(ctx, call)

			sendSSE(w, flusher, "tool_result", map[string]any{
				"name":  result.Name,
				"data":  result.Data,
				"error": result.Error,
			})

			// Save tool interactions
			callJSON, _ := json.Marshal(call)
			_ = s.memory.AddMessage(conv.ID, Message{
				Role:    "tool_call",
				Name:    call.Name,
				Content: string(callJSON),
			})

			resultJSON, _ := json.Marshal(result)
			_ = s.memory.AddMessage(conv.ID, Message{
				Role:    "tool_result",
				Name:    result.Name,
				Content: string(resultJSON),
			})

			// Append tool result to prompt (no trailing "Assistant:" yet)
			prompt += "\n" + FormatToolResult(result)
		}

		// Add single "Assistant:" after all tool results for next iteration
		prompt += "\n\nNow provide your final answer to the user based on the tool results above.\n\nAssistant: "
	}

	if finalResponse == "" {
		finalResponse = "I wasn't able to generate a response. Please try again."
	}

	// Save assistant response
	_ = s.memory.AddMessage(conv.ID, Message{Role: "assistant", Content: finalResponse})

	// Send the message
	sendSSE(w, flusher, "message", map[string]string{"content": finalResponse})

	// Auto-generate title after first exchange
	if conv.Title == "New conversation" {
		go s.generateTitle(conv.ID, req.Message, finalResponse)
	}

	// Send done event
	sendSSE(w, flusher, "done", map[string]string{
		"conversation_id": conv.ID,
		"title":           conv.Title,
	})
}

// generateTitle asks the agent to generate a short title for the conversation.
func (s *Server) generateTitle(convID, userMsg, assistantMsg string) {
	agent := s.ActiveAgent()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := fmt.Sprintf(
		"Generate a very short title (3-6 words) for this conversation. "+
			"Reply with ONLY the title, nothing else.\n\nUser: %s\nAssistant: %s",
		userMsg, assistantMsg,
	)

	title, err := agent.Run(ctx, prompt)
	if err != nil || title == "" {
		return
	}

	// Clean up the title
	if len(title) > 60 {
		title = title[:60]
	}

	_ = s.memory.UpdateTitle(convID, title)
}

// handleConfig returns or updates agent configuration.
// GET /api/config — returns current agent and available agents
// PUT /api/config — switches the active agent
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getConfig(w)
	case http.MethodPut:
		s.switchAgent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getConfig(w http.ResponseWriter) {
	agent := s.ActiveAgent()

	available := make([]string, len(s.agents))
	for i, a := range s.agents {
		available[i] = string(a.Type)
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"agent":     string(agent.Type),
		"available": available,
	})
}

func (s *Server) switchAgent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Agent string `json:"agent"`
	}
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Agent == "" {
		http.Error(w, "agent is required", http.StatusBadRequest)
		return
	}

	if !s.SetAgent(AgentType(req.Agent)) {
		http.Error(w, "unknown agent: "+req.Agent, http.StatusBadRequest)
		return
	}

	s.getConfig(w)
}

// handleHealth returns server health status.
// GET /api/health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// sendSSE writes a Server-Sent Event to the response.
func sendSSE(w http.ResponseWriter, flusher http.Flusher, event string, data any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
	flusher.Flush()
}
