package workflow

import (
	"context"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var opts scopeOptions
	var file string
	var name string
	var templateID string
	var trigger string
	var delay int
	var enabled bool
	var disabled bool
	var fromName string
	var fromEmail string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a hosted workflow",
		RunE: func(cmd *cobra.Command, _ []string) error {
			scope, grantID, err := resolveScope(opts)
			if err != nil {
				return err
			}

			req := &domain.CreateRemoteWorkflowRequest{}
			if file != "" {
				if err := common.LoadJSONFile(file, req); err != nil {
					return err
				}
			} else {
				if err := common.ValidateRequiredFlag("--name", name); err != nil {
					return err
				}
				if err := common.ValidateRequiredFlag("--template-id", templateID); err != nil {
					return err
				}
				if err := common.ValidateRequiredFlag("--trigger-event", trigger); err != nil {
					return err
				}
				if err := validateTrigger(trigger); err != nil {
					return err
				}

				req.Name = name
				req.TemplateID = templateID
				req.TriggerEvent = trigger
				req.Delay = delay

				isEnabled, err := enabledPtr(enabled, disabled)
				if err != nil {
					return err
				}
				req.IsEnabled = isEnabled
				req.From = senderFromFlags(fromName, fromEmail)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			return withClient(ctx, func(ctx context.Context, client ports.NylasClient) error {
				workflow, err := client.CreateWorkflow(ctx, scope, grantID, req)
				if err != nil {
					return common.WrapCreateError("workflow", err)
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
	cmd.Flags().StringVar(&file, "file", "", "Path to a JSON workflow definition")
	cmd.Flags().StringVar(&name, "name", "", "Workflow name")
	cmd.Flags().StringVar(&templateID, "template-id", "", "Hosted template ID")
	cmd.Flags().StringVar(&trigger, "trigger-event", "", "Workflow trigger event")
	cmd.Flags().IntVar(&delay, "delay", 0, "Delay in minutes before the workflow sends")
	cmd.Flags().BoolVar(&enabled, "enabled", false, "Create the workflow in an enabled state")
	cmd.Flags().BoolVar(&disabled, "disabled", false, "Create the workflow in a disabled state")
	cmd.Flags().StringVar(&fromName, "from-name", "", "Transactional sender display name")
	cmd.Flags().StringVar(&fromEmail, "from-email", "", "Transactional sender email")

	return cmd
}

func newUpdateCmd() *cobra.Command {
	var opts scopeOptions
	var file string
	var name string
	var templateID string
	var trigger string
	var delay int
	var setDelay bool
	var enabled bool
	var disabled bool
	var fromName string
	var fromEmail string

	cmd := &cobra.Command{
		Use:   "update <workflow-id>",
		Short: "Update a hosted workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, grantID, err := resolveScope(opts)
			if err != nil {
				return err
			}

			req := &domain.UpdateRemoteWorkflowRequest{}
			if file != "" {
				if err := common.LoadJSONFile(file, req); err != nil {
					return err
				}
			} else {
				if trigger != "" {
					if err := validateTrigger(trigger); err != nil {
						return err
					}
				}
				isEnabled, err := enabledPtr(enabled, disabled)
				if err != nil {
					return err
				}
				if err := common.ValidateAtLeastOne("workflow field", name, templateID, trigger, fromName, fromEmail); err != nil && !setDelay && isEnabled == nil {
					return err
				}

				if name != "" {
					req.Name = &name
				}
				if templateID != "" {
					req.TemplateID = &templateID
				}
				if trigger != "" {
					req.TriggerEvent = &trigger
				}
				if setDelay {
					req.Delay = &delay
				}
				req.IsEnabled = isEnabled
				if sender := senderFromFlags(fromName, fromEmail); sender != nil {
					req.From = sender
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			return withClient(ctx, func(ctx context.Context, client ports.NylasClient) error {
				workflow, err := client.UpdateWorkflow(ctx, scope, grantID, args[0], req)
				if err != nil {
					return common.WrapUpdateError("workflow", err)
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
	cmd.Flags().StringVar(&file, "file", "", "Path to a JSON workflow update definition")
	cmd.Flags().StringVar(&name, "name", "", "Updated workflow name")
	cmd.Flags().StringVar(&templateID, "template-id", "", "Updated hosted template ID")
	cmd.Flags().StringVar(&trigger, "trigger-event", "", "Updated workflow trigger event")
	cmd.Flags().IntVar(&delay, "delay", 0, "Updated workflow delay in minutes")
	cmd.Flags().BoolVar(&enabled, "enabled", false, "Enable the workflow")
	cmd.Flags().BoolVar(&disabled, "disabled", false, "Disable the workflow")
	cmd.Flags().StringVar(&fromName, "from-name", "", "Updated transactional sender display name")
	cmd.Flags().StringVar(&fromEmail, "from-email", "", "Updated transactional sender email")
	cmd.Flags().BoolVar(&setDelay, "set-delay", false, "Apply the value from --delay during update")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var opts scopeOptions
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <workflow-id>",
		Short: "Delete a hosted workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, grantID, err := resolveScope(opts)
			if err != nil {
				return err
			}
			if !yes {
				return common.NewUserError(
					"deletion requires confirmation",
					"Re-run with --yes to delete the workflow",
				)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			return withClient(ctx, func(ctx context.Context, client ports.NylasClient) error {
				if err := client.DeleteWorkflow(ctx, scope, grantID, args[0]); err != nil {
					return common.WrapDeleteError("workflow", err)
				}
				if !common.IsStructuredOutput(cmd) {
					common.PrintSuccess("Workflow deleted")
				}
				return nil
			})
		},
	}

	addScopeFlags(cmd, &opts)
	common.AddYesFlag(cmd, &yes)

	return cmd
}
