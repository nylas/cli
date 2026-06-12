package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

// workspaceAssigner is the slice of ports.NylasClient that a move needs.
type workspaceAssigner interface {
	AssignWorkspaceGrants(ctx context.Context, workspaceID string, req *domain.WorkspaceAssignRequest) (*domain.WorkspaceAssignResult, error)
}

func newMoveCmd() *cobra.Command {
	var workspaceID string

	cmd := &cobra.Command{
		Use:   "move [agent-id|email]",
		Short: "Move an agent account to another workspace",
		Long: `Move a Nylas agent account to another workspace.

The target workspace's policy and rules govern the account immediately. Moves
use the workspace manual-assign API (POST /v3/workspaces/{id}/manual-assign);
assigning a grant moves it even when it currently belongs to another
workspace.

Examples:
  nylas agent account move me@yourapp.nylas.email --workspace <workspace-id>
  nylas agent account move 123456 --workspace <workspace-id>

Use 'nylas workspace list' to find workspace IDs.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier, err := getRequiredAgentIdentifier(args)
			if err != nil {
				return err
			}

			client, err := common.GetNylasClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			grantID, err := resolveAgentID(ctx, client, identifier)
			cancel()
			if err != nil {
				return common.WrapGetError("agent account", err)
			}

			ctx2, cancel2 := common.CreateContext()
			account, err := client.GetAgentAccount(ctx2, grantID)
			cancel2()
			if err != nil {
				return common.WrapGetError("agent account", err)
			}

			ctx3, cancel3 := common.CreateContext()
			defer cancel3()

			err = common.RunWithSpinner("Moving agent account...", func() error {
				return moveAgentAccount(ctx3, client, grantID, workspaceID)
			})
			if err != nil {
				return common.WrapUpdateError("agent account", err)
			}

			common.PrintSuccess("Agent account %s moved to workspace %s!", account.Email, workspaceID)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "Target workspace ID (required)")
	_ = cmd.MarkFlagRequired("workspace")

	return cmd
}

// moveAgentAccount assigns the grant to the target workspace. A move is a
// single assign: the manual-assign API moves the grant out of its old
// workspace itself, and remove_grants would strand the account in no
// workspace (it does not fall back to the default).
func moveAgentAccount(ctx context.Context, client workspaceAssigner, grantID, workspaceID string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return fmt.Errorf("target workspace ID is required")
	}

	_, err := client.AssignWorkspaceGrants(ctx, workspaceID, &domain.WorkspaceAssignRequest{
		AssignGrants: []string{grantID},
	})
	return err
}
