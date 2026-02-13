package chat

import (
	"net/http"
	"strings"

	"github.com/nylas/cli/internal/httputil"
)

// handleConversations handles listing and creating conversations.
// GET /api/conversations — list all conversations
// POST /api/conversations — create new conversation
func (s *Server) handleConversations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listConversations(w, r)
	case http.MethodPost:
		s.createConversation(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleConversationByID handles operations on a specific conversation.
// GET /api/conversations/{id} — get conversation
// DELETE /api/conversations/{id} — delete conversation
func (s *Server) handleConversationByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/conversations/")
	if id == "" {
		http.Error(w, "conversation ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getConversation(w, id)
	case http.MethodDelete:
		s.deleteConversation(w, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listConversations(w http.ResponseWriter, _ *http.Request) {
	summaries, err := s.memory.List()
	if err != nil {
		http.Error(w, "Failed to list conversations", http.StatusInternalServerError)
		return
	}

	if summaries == nil {
		summaries = []ConversationSummary{}
	}

	httputil.WriteJSON(w, http.StatusOK, summaries)
}

func (s *Server) createConversation(w http.ResponseWriter, _ *http.Request) {
	conv, err := s.memory.Create(string(s.agent.Type))
	if err != nil {
		http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]string{
		"id":    conv.ID,
		"title": conv.Title,
	})
}

func (s *Server) getConversation(w http.ResponseWriter, id string) {
	conv, err := s.memory.Get(id)
	if err != nil {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, conv)
}

func (s *Server) deleteConversation(w http.ResponseWriter, id string) {
	if err := s.memory.Delete(id); err != nil {
		http.Error(w, "Failed to delete conversation", http.StatusNotFound)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
