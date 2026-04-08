package templatecmd

import (
	"context"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var opts scopeOptions
	var params domain.CursorListParams

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List hosted templates",
		RunE: func(cmd *cobra.Command, _ []string) error {
			scope, grantID, err := resolveScope(opts)
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			return withClient(ctx, func(ctx context.Context, client ports.NylasClient) error {
				resp, err := client.ListRemoteTemplates(ctx, scope, grantID, &params)
				if err != nil {
					return common.WrapListError("templates", err)
				}

				out := common.GetOutputWriter(cmd)
				quiet, _ := cmd.Flags().GetBool("quiet")
				format, _ := cmd.Flags().GetString("format")
				if quiet || format == "quiet" {
					return out.WriteList(resp.Data, templateColumns)
				}
				if common.IsStructuredOutput(cmd) {
					return out.Write(resp)
				}
				if err := out.WriteList(resp.Data, templateColumns); err != nil {
					return err
				}
				nextCursorNote(cmd, resp.NextCursor)
				return nil
			})
		},
	}

	addScopeFlags(cmd, &opts)
	common.AddLimitFlag(cmd, &params.Limit, 50)
	common.AddPageTokenFlag(cmd, &params.PageToken)

	return cmd
}
