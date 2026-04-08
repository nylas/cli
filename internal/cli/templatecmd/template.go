package templatecmd

import (
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

type scopeOptions struct {
	scope   string
	grantID string
}

// NewTemplateCmd creates the hosted template command group.
func NewTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage hosted templates",
		Long: `Manage Nylas-hosted templates at the application or grant scope.

Use --scope app for application-level templates and --scope grant to target
templates attached to a specific grant.`,
	}

	common.AddOutputFlags(cmd)
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newRenderCmd())
	cmd.AddCommand(newRenderHTMLCmd())

	return cmd
}
