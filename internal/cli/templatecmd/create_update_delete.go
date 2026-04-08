package templatecmd

import (
	"context"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var opts scopeOptions
	var name string
	var subject string
	var body string
	var bodyFile string
	var engine string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a hosted template",
		RunE: func(cmd *cobra.Command, _ []string) error {
			scope, grantID, err := resolveScope(opts)
			if err != nil {
				return err
			}
			if err := common.ValidateRequiredFlag("--name", name); err != nil {
				return err
			}
			if err := common.ValidateRequiredFlag("--subject", subject); err != nil {
				return err
			}
			body, err = common.ReadStringOrFile("body", body, bodyFile, true)
			if err != nil {
				return err
			}
			if engine != "" {
				if err := validateEngine(engine); err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			req := &domain.CreateRemoteTemplateRequest{
				Name:    name,
				Subject: subject,
				Body:    trimBody(body),
				Engine:  engine,
			}

			return withClient(ctx, func(ctx context.Context, client ports.NylasClient) error {
				template, err := client.CreateRemoteTemplate(ctx, scope, grantID, req)
				if err != nil {
					return common.WrapCreateError("template", err)
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
	cmd.Flags().StringVar(&name, "name", "", "Template name")
	cmd.Flags().StringVar(&subject, "subject", "", "Template subject")
	cmd.Flags().StringVar(&body, "body", "", "Template body HTML")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Read template body HTML from a file")
	cmd.Flags().StringVar(&engine, "engine", "", "Template engine")

	return cmd
}

func newUpdateCmd() *cobra.Command {
	var opts scopeOptions
	var name string
	var subject string
	var body string
	var bodyFile string
	var engine string

	cmd := &cobra.Command{
		Use:   "update <template-id>",
		Short: "Update a hosted template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, grantID, err := resolveScope(opts)
			if err != nil {
				return err
			}
			body, err = common.ReadStringOrFile("body", body, bodyFile, false)
			if err != nil {
				return err
			}
			if err := common.ValidateAtLeastOne("template field", name, subject, body, engine); err != nil {
				return err
			}
			if engine != "" {
				if err := validateEngine(engine); err != nil {
					return err
				}
			}

			req := &domain.UpdateRemoteTemplateRequest{}
			if name != "" {
				req.Name = &name
			}
			if subject != "" {
				req.Subject = &subject
			}
			if body != "" {
				body = trimBody(body)
				req.Body = &body
			}
			if engine != "" {
				req.Engine = &engine
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			return withClient(ctx, func(ctx context.Context, client ports.NylasClient) error {
				template, err := client.UpdateRemoteTemplate(ctx, scope, grantID, args[0], req)
				if err != nil {
					return common.WrapUpdateError("template", err)
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
	cmd.Flags().StringVar(&name, "name", "", "Updated template name")
	cmd.Flags().StringVar(&subject, "subject", "", "Updated template subject")
	cmd.Flags().StringVar(&body, "body", "", "Updated template body HTML")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Read updated template body HTML from a file")
	cmd.Flags().StringVar(&engine, "engine", "", "Updated template engine")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var opts scopeOptions
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <template-id>",
		Short: "Delete a hosted template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, grantID, err := resolveScope(opts)
			if err != nil {
				return err
			}

			if !yes {
				return common.NewUserError(
					"deletion requires confirmation",
					"Re-run with --yes to delete the template",
				)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			return withClient(ctx, func(ctx context.Context, client ports.NylasClient) error {
				if err := client.DeleteRemoteTemplate(ctx, scope, grantID, args[0]); err != nil {
					return common.WrapDeleteError("template", err)
				}
				if !common.IsStructuredOutput(cmd) {
					common.PrintSuccess("Template deleted")
				}
				return nil
			})
		},
	}

	addScopeFlags(cmd, &opts)
	common.AddYesFlag(cmd, &yes)

	return cmd
}
