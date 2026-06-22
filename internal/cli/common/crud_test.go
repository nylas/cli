//go:build !integration

package common

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withStdin replaces os.Stdin with a pipe containing input for the duration
// of the test. Used to drive the interactive Confirm path.
func withStdin(t *testing.T, input string) {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, err = w.WriteString(input)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = r.Close()
	})
}

// TestRunDelete_Confirmation verifies the destructive-delete gate after the
// consolidation onto common.Confirm: the default answer is NO (empty input or
// no terminal must cancel), only an explicit yes or --force may delete.
func TestRunDelete_Confirmation(t *testing.T) {
	tests := []struct {
		name       string
		force      bool
		quiet      bool   // quiet mode: Confirm returns the default without prompting
		stdin      string // interactive input when not quiet
		wantDelete bool
	}{
		{name: "force skips confirmation and deletes", force: true, quiet: true, wantDelete: true},
		{name: "no input cancels (default no)", quiet: true, wantDelete: false},
		{name: "interactive empty line cancels", stdin: "\n", wantDelete: false},
		{name: "interactive n cancels", stdin: "n\n", wantDelete: false},
		{name: "interactive y deletes", stdin: "y\n", wantDelete: true},
		{name: "interactive yes deletes", stdin: "yes\n", wantDelete: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetQuiet(tt.quiet)
			if !tt.quiet {
				withStdin(t, tt.stdin)
			}

			deleted := false
			err := RunDelete(DeleteConfig{
				ResourceName: "contact",
				ResourceID:   "contact-123",
				GrantID:      "grant-123",
				Force:        tt.force,
				DeleteFunc: func(ctx context.Context, grantID, resourceID string) error {
					deleted = true
					assert.Equal(t, "grant-123", grantID)
					assert.Equal(t, "contact-123", resourceID)
					return nil
				},
			})

			require.NoError(t, err)
			assert.Equal(t, tt.wantDelete, deleted,
				"delete invocation mismatch: cancelled confirmations must never delete")
		})
	}
}

func TestRunDelete_WrapsDeleteError(t *testing.T) {
	SetQuiet(true)

	deleteErr := errors.New("backend exploded")
	err := RunDelete(DeleteConfig{
		ResourceName: "contact",
		ResourceID:   "contact-123",
		GrantID:      "grant-123",
		Force:        true,
		DeleteFunc: func(ctx context.Context, grantID, resourceID string) error {
			return deleteErr
		},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, deleteErr)
}

// TestNewDeleteCommand_Confirmation exercises the full cobra command path
// (no-grant variant) for cancel-on-default and --force skip.
func TestNewDeleteCommand_Confirmation(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		quiet      bool
		stdin      string
		wantDelete bool
	}{
		{name: "default cancels without input", args: []string{"webhook-1"}, quiet: true, wantDelete: false},
		{name: "force flag skips confirmation", args: []string{"webhook-1", "--force"}, quiet: true, wantDelete: true},
		{name: "interactive y deletes", args: []string{"webhook-1"}, stdin: "y\n", wantDelete: true},
		{name: "interactive empty line cancels", args: []string{"webhook-1"}, stdin: "\n", wantDelete: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetQuiet(tt.quiet)
			if !tt.quiet {
				withStdin(t, tt.stdin)
			}

			deleted := false
			cmd := NewDeleteCommand(DeleteCommandConfig{
				Use:          "delete <webhook-id>",
				Short:        "Delete a webhook",
				ResourceName: "webhook",
				GetClient:    func() (ports.NylasClient, error) { return nil, nil },
				DeleteFuncNoGrant: func(ctx context.Context, resourceID string) error {
					deleted = true
					assert.Equal(t, "webhook-1", resourceID)
					return nil
				},
			})
			cmd.SetArgs(tt.args)

			require.NoError(t, cmd.Execute())
			assert.Equal(t, tt.wantDelete, deleted,
				"delete invocation mismatch: cancelled confirmations must never delete")
		})
	}
}

// TestNewDeleteCommand_GrantPathForce verifies the grant-scoped delete path
// resolves the grant ID from the second positional argument and honors
// --force.
func TestNewDeleteCommand_GrantPathForce(t *testing.T) {
	SetQuiet(true)

	deleted := false
	cmd := NewDeleteCommand(DeleteCommandConfig{
		Use:          "delete <contact-id> [grant-id]",
		Short:        "Delete a contact",
		ResourceName: "contact",
		GetClient:    func() (ports.NylasClient, error) { return nil, nil },
		DeleteFunc: func(ctx context.Context, grantID, resourceID string) error {
			deleted = true
			assert.Equal(t, "grant-456", grantID)
			assert.Equal(t, "contact-1", resourceID)
			return nil
		},
	})
	cmd.SetArgs([]string{"contact-1", "grant-456", "--force"})

	require.NoError(t, cmd.Execute())
	assert.True(t, deleted)
}
