package agent

import (
	"context"
	"errors"
	"testing"

	nylasmock "github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubWorkspaceAssigner struct {
	gotWorkspaceID string
	gotReq         *domain.WorkspaceAssignRequest
	err            error
}

func (s *stubWorkspaceAssigner) AssignWorkspaceGrants(ctx context.Context, workspaceID string, req *domain.WorkspaceAssignRequest) (*domain.WorkspaceAssignResult, error) {
	s.gotWorkspaceID = workspaceID
	s.gotReq = req
	if s.err != nil {
		return nil, s.err
	}
	return &domain.WorkspaceAssignResult{WorkspaceID: workspaceID, GrantsAssigned: req.AssignGrants}, nil
}

// A move must be a single assign: the manual-assign API moves a grant out of
// its old workspace itself, and remove_grants would strand the account in no
// workspace.
func TestMoveAgentAccount_AssignOnly(t *testing.T) {
	stub := &stubWorkspaceAssigner{}

	err := moveAgentAccount(context.Background(), stub, "grant-1", "ws-target")

	require.NoError(t, err)
	assert.Equal(t, "ws-target", stub.gotWorkspaceID)
	require.NotNil(t, stub.gotReq)
	assert.Equal(t, []string{"grant-1"}, stub.gotReq.AssignGrants)
	assert.Empty(t, stub.gotReq.RemoveGrants, "a move must never remove grants")
}

func TestMoveAgentAccount_RequiresWorkspace(t *testing.T) {
	stub := &stubWorkspaceAssigner{}

	err := moveAgentAccount(context.Background(), stub, "grant-1", "  ")

	assert.Error(t, err)
	assert.Nil(t, stub.gotReq, "no API call should be made without a workspace ID")
}

func TestMoveAgentAccount_PropagatesUpstreamError(t *testing.T) {
	stub := &stubWorkspaceAssigner{err: errors.New("upstream rejected")}

	err := moveAgentAccount(context.Background(), stub, "grant-1", "ws-target")

	assert.ErrorContains(t, err, "upstream rejected")
}

func TestMoveCmd_Definition(t *testing.T) {
	cmd := newMoveCmd()

	assert.Equal(t, "move [agent-id|email]", cmd.Use)
	flag := cmd.Flags().Lookup("workspace")
	require.NotNil(t, flag, "move must define a --workspace flag")
}

// Move accepts either form of identifier; resolveAgentID is the helper every
// account command (get/update/delete/move) relies on for that.
func TestResolveAgentID(t *testing.T) {
	client := nylasmock.NewMockClient()

	t.Run("grant ID passes through without lookup", func(t *testing.T) {
		setupAgentIdentifierTestEnv(t)

		id, err := resolveAgentID(context.Background(), client, "agent-123")

		require.NoError(t, err)
		assert.Equal(t, "agent-123", id)
	})

	t.Run("email resolves to the matching grant ID", func(t *testing.T) {
		setupAgentIdentifierTestEnv(t)

		id, err := resolveAgentID(context.Background(), client, "agent@example.com")

		require.NoError(t, err)
		assert.Equal(t, "agent-1", id)
	})

	t.Run("unknown email reports a not-found error", func(t *testing.T) {
		setupAgentIdentifierTestEnv(t)

		_, err := resolveAgentID(context.Background(), client, "missing@example.com")

		assert.ErrorContains(t, err, "agent account not found")
	})
}
