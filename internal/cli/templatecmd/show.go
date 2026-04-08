package templatecmd

import (
	"context"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	var opts scopeOptions

	cmd := &cobra.Command{
		Use:   "show <template-id>",
		Short: "Show a hosted template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, grantID, err := resolveScope(opts)
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			return withClient(ctx, func(ctx context.Context, client ports.NylasClient) error {
				template, err := client.GetRemoteTemplate(ctx, scope, grantID, args[0])
				if err != nil {
					return common.WrapError(err)
				}

				if common.IsStructuredOutput(cmd) {
					return common.GetOutputWriter(cmd).Write(template)
				}

				printTemplate(template)
				return nil
			})
		},
	}

	addScopeFlags(cmd, &opts)
	return cmd
}
