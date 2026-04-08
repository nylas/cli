package templatecmd

import (
	"context"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newRenderCmd() *cobra.Command {
	var opts scopeOptions
	var data string
	var dataFile string
	var strict bool

	cmd := &cobra.Command{
		Use:   "render <template-id>",
		Short: "Render a hosted template with variables",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, grantID, err := resolveScope(opts)
			if err != nil {
				return err
			}
			variables, err := common.ReadJSONStringMap(data, dataFile)
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			return withClient(ctx, func(ctx context.Context, client ports.NylasClient) error {
				result, err := client.RenderRemoteTemplate(ctx, scope, grantID, args[0], &domain.TemplateRenderRequest{
					Strict:    &strict,
					Variables: variables,
				})
				if err != nil {
					return common.WrapError(err)
				}
				if common.IsStructuredOutput(cmd) {
					return common.GetOutputWriter(cmd).Write(result)
				}
				return printRenderResult(result)
			})
		},
	}

	addScopeFlags(cmd, &opts)
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON object with template variables")
	cmd.Flags().StringVar(&dataFile, "data-file", "", "Path to a JSON file with template variables")
	cmd.Flags().BoolVar(&strict, "strict", true, "Fail when the template references missing variables")

	return cmd
}

func newRenderHTMLCmd() *cobra.Command {
	var opts scopeOptions
	var body string
	var bodyFile string
	var data string
	var dataFile string
	var strict bool
	var engine string

	cmd := &cobra.Command{
		Use:   "render-html",
		Short: "Render arbitrary template HTML with variables",
		RunE: func(cmd *cobra.Command, _ []string) error {
			scope, grantID, err := resolveScope(opts)
			if err != nil {
				return err
			}
			if err := common.ValidateRequiredFlag("--engine", engine); err != nil {
				return err
			}
			if err := validateEngine(engine); err != nil {
				return err
			}
			body, err = common.ReadStringOrFile("body", body, bodyFile, true)
			if err != nil {
				return err
			}
			variables, err := common.ReadJSONStringMap(data, dataFile)
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			return withClient(ctx, func(ctx context.Context, client ports.NylasClient) error {
				result, err := client.RenderRemoteTemplateHTML(ctx, scope, grantID, &domain.TemplateRenderHTMLRequest{
					Body:      trimBody(body),
					Engine:    engine,
					Strict:    &strict,
					Variables: variables,
				})
				if err != nil {
					return common.WrapError(err)
				}
				if common.IsStructuredOutput(cmd) {
					return common.GetOutputWriter(cmd).Write(result)
				}
				return printRenderResult(result)
			})
		},
	}

	addScopeFlags(cmd, &opts)
	cmd.Flags().StringVar(&body, "body", "", "Template HTML to render")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Path to a file containing template HTML")
	cmd.Flags().StringVar(&engine, "engine", "", "Template engine")
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON object with template variables")
	cmd.Flags().StringVar(&dataFile, "data-file", "", "Path to a JSON file with template variables")
	cmd.Flags().BoolVar(&strict, "strict", true, "Fail when the template references missing variables")

	return cmd
}
