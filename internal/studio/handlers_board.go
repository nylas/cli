package studio

import (
	"context"
	"net/http"

	"github.com/nylas/cli/internal/agentgraph"
)

// handleBoard returns the full agent resource graph — accounts, workspaces,
// policies, rules, lists, and health flags — in one response. The UI renders
// exclusively from this state.
func (s *Server) handleBoard(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	board, err := s.fetchBoard(ctx)
	if err != nil {
		writeUpstreamError(w, http.StatusInternalServerError, "Failed to load board", err)
		return
	}

	writeJSON(w, http.StatusOK, board)
}

// fetchBoard assembles the board state; mutation handlers reuse it so every
// write response carries fresh server truth.
func (s *Server) fetchBoard(ctx context.Context) (*agentgraph.Overview, error) {
	accounts, err := s.nylasClient.ListAgentAccounts(ctx)
	if err != nil {
		return nil, err
	}
	workspaces, err := s.nylasClient.ListWorkspaces(ctx)
	if err != nil {
		return nil, err
	}
	policies, err := s.nylasClient.ListPolicies(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := s.nylasClient.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	lists, err := s.nylasClient.ListLists(ctx)
	if err != nil {
		return nil, err
	}

	return agentgraph.Build(accounts, workspaces, policies, rules, lists), nil
}
