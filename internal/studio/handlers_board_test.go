package studio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	nylasmock "github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/agentgraph"
)

func newTestServer() *Server {
	return NewServer("127.0.0.1:0", nylasmock.NewMockClient())
}

func TestHandleBoard_ReturnsGraph(t *testing.T) {
	t.Parallel()

	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/board", nil)
	w := httptest.NewRecorder()

	server.handleBoard(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	var board agentgraph.Overview
	if err := json.NewDecoder(w.Body).Decode(&board); err != nil {
		t.Fatalf("decode board: %v", err)
	}
	if len(board.Accounts) != 1 {
		t.Fatalf("expected one account from mock client, got %d", len(board.Accounts))
	}
	if board.Accounts[0].WorkspaceID != "workspace-1" {
		t.Fatalf("expected workspace-1, got %q", board.Accounts[0].WorkspaceID)
	}
	if board.Accounts[0].Policy == nil || board.Accounts[0].Policy.ID != "policy-1" {
		t.Fatalf("expected resolved policy-1, got %+v", board.Accounts[0].Policy)
	}
	if board.Totals["lists"] != 1 {
		t.Fatalf("expected one list in totals, got %d", board.Totals["lists"])
	}
}

func TestHandleBoard_RejectsNonGET(t *testing.T) {
	t.Parallel()

	server := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/board", nil)
	w := httptest.NewRecorder()

	server.handleBoard(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", w.Code)
	}
}
