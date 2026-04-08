package workflow

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
		Short: "List hosted workflows",
		RunE: func(cmd *cobra.Command, _ []string) error {
			scope, grantID, err := resolveScope(opts)
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			return withClient(ctx, func(ctx context.Context, client ports.NylasClient) error {
				resp, err := client.ListWorkflows(ctx, scope, grantID, &params)
				if err != nil {
					return common.WrapListError("workflows", err)
				}

				out := common.GetOutputWriter(cmd)
				quiet, _ := cmd.Flags().GetBool("quiet")
				format, _ := cmd.Flags().GetString("format")
				if quiet || format == "quiet" {
					return out.WriteList(resp.Data, workflowColumns)
				}
				if common.IsStructuredOutput(cmd) {
					return out.Write(resp)
				}
				if err := out.WriteList(resp.Data, workflowColumns); err != nil {
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

func newShowCmd() *cobra.Command {
	var opts scopeOptions

	cmd := &cobra.Command{
		Use:   "show <workflow-id>",
		Short: "Show a hosted workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, grantID, err := resolveScope(opts)
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			return withClient(ctx, func(ctx context.Context, client ports.NylasClient) error {
				workflow, err := client.GetWorkflow(ctx, scope, grantID, args[0])
				if err != nil {
					return common.WrapError(err)
				}

				if common.IsStructuredOutput(cmd) {
					return common.GetOutputWriter(cmd).Write(workflow)
				}

				printWorkflow(workflow)
				return nil
			})
		},
	}

	addScopeFlags(cmd, &opts)
	return cmd
}
