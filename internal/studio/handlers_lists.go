package studio

import (
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/nylas/cli/internal/domain"
)

func (s *Server) routeLists(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "/api/lists")
	switch {
	case r.Method == http.MethodPost && id == "":
		s.handleListCreate(w, r)
	case r.Method == http.MethodPatch && id != "":
		s.handleListPatch(w, r, id)
	case r.Method == http.MethodDelete && id != "":
		s.handleListDelete(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// routeListItems handles /api/lists/{id}/items.
func (s *Server) routeListItems(w http.ResponseWriter, r *http.Request) {
	path := pathID(r, "/api/lists")
	listID := strings.TrimSuffix(path, "/items")
	if listID == path || strings.TrimSpace(listID) == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if r.Method == http.MethodGet {
		ctx, cancel := s.withTimeout(r)
		defer cancel()
		items, err := s.nylasClient.GetListItems(ctx, listID)
		if err != nil {
			writeMutationError(w, "Failed to load list items", err)
			return
		}
		if items == nil {
			items = []string{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
		return
	}

	var body struct {
		Items []string `json:"items"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if len(body.Items) == 0 {
		writeError(w, http.StatusBadRequest, "items are required")
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	var err error
	switch r.Method {
	case http.MethodPost:
		_, err = s.nylasClient.AddListItems(ctx, listID, body.Items)
	case http.MethodDelete:
		_, err = s.nylasClient.RemoveListItems(ctx, listID, body.Items)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err != nil {
		writeMutationError(w, "Failed to modify list items", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusOK, listID)
}

func (s *Server) handleListCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string   `json:"name"`
		Type        string   `json:"type"`
		Description string   `json:"description"`
		Items       []string `json:"items"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Type = strings.TrimSpace(body.Type)
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "list name is required")
		return
	}
	if !slices.Contains(domain.AgentListTypes, body.Type) {
		writeError(w, http.StatusBadRequest, "list type must be one of: "+strings.Join(domain.AgentListTypes, ", "))
		return
	}

	payload := map[string]any{"name": body.Name, "type": body.Type}
	if desc := strings.TrimSpace(body.Description); desc != "" {
		payload["description"] = desc
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	list, err := s.nylasClient.CreateList(ctx, payload)
	if err != nil {
		writeMutationError(w, "Failed to create list", err)
		return
	}

	if len(body.Items) > 0 {
		if _, err := s.nylasClient.AddListItems(ctx, list.ID, body.Items); err != nil {
			// Don't leave a partial list behind: it would consume a plan slot
			// while holding none of the intended items.
			if cleanupErr := s.nylasClient.DeleteList(ctx, list.ID); cleanupErr != nil {
				slog.Error("studio: cleanup of partially seeded list failed", "list_id", list.ID, "err", cleanupErr)
			}
			writeMutationError(w, "Failed to seed list items", err)
			return
		}
	}

	s.respondMutation(ctx, w, http.StatusCreated, list.ID)
}

func (s *Server) handleListPatch(w http.ResponseWriter, r *http.Request, id string) {
	var payload map[string]any
	if !decodeBody(w, r, &payload) {
		return
	}
	// The list type is immutable in the API; reject it client-side too.
	delete(payload, "type")
	if len(payload) == 0 {
		writeError(w, http.StatusBadRequest, "nothing to update")
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	if _, err := s.nylasClient.UpdateList(ctx, id, payload); err != nil {
		writeMutationError(w, "Failed to update list", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusOK, id)
}

func (s *Server) handleListDelete(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := s.withTimeout(r)
	defer cancel()

	if err := s.nylasClient.DeleteList(ctx, id); err != nil {
		writeMutationError(w, "Failed to delete list", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusOK, id)
}
