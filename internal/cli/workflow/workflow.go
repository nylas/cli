package workflow

import (
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

type scopeOptions struct {
	scope   string
	grantID string
}

// NewWorkflowCmd creates the hosted workflow command group.
func NewWorkflowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Manage hosted workflows",
		Long: `Manage Nylas-hosted workflows at the application or grant scope.

Workflows connect booking events to hosted templates.

API reference: https://developer.nylas.com/docs/reference/api/application-level-workflows/`,
	}

	common.AddOutputFlags(cmd)
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}
