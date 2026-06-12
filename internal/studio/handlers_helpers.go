package studio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/nylas/cli/internal/domain"
)

const maxBodyBytes = 1 << 20 // 1 MiB

func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// pathID extracts the trailing resource ID from prefix-routed paths like
// /api/workspaces/{id}. Empty when the path is the bare collection.
func pathID(r *http.Request, prefix string) string {
	id := strings.TrimPrefix(r.URL.Path, prefix)
	return strings.Trim(id, "/")
}

// respondMutation answers a successful write with the affected resource ID and
// fresh board state, so the UI always re-renders from server truth.
func (s *Server) respondMutation(ctx context.Context, w http.ResponseWriter, status int, id string) {
	board, err := s.fetchBoard(ctx)
	if err != nil {
		writeUpstreamError(w, http.StatusInternalServerError, "Change applied but board refresh failed", err)
		return
	}
	writeJSON(w, status, map[string]any{"id": id, "board": board})
}

// writeMutationError translates upstream failures: plan-cap 403s become a
// structured plan_limit error the UI can render meaningfully; everything else
// follows the generic-message discipline.
func writeMutationError(w http.ResponseWriter, msg string, err error) {
	var apiErr *domain.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error":   "plan_limit",
			"message": "Plan limit reached for this resource",
		})
		return
	}
	writeUpstreamError(w, http.StatusInternalServerError, msg, err)
}
