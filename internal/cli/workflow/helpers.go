package workflow

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

var workflowColumns = []ports.Column{
	{Header: "ID", Field: "ID", Width: -1},
	{Header: "Name", Field: "Name", Width: 30},
	{Header: "Trigger", Field: "TriggerEvent", Width: 22},
	{Header: "Template", Field: "TemplateID", Width: -1},
	{Header: "Delay", Field: "Delay", Width: 8},
	{Header: "Enabled", Field: "IsEnabled", Width: 8},
}

func addScopeFlags(cmd *cobra.Command, opts *scopeOptions) {
	cmd.Flags().StringVar(&opts.scope, "scope", string(domain.ScopeApplication), "Workflow scope: app or grant")
	cmd.Flags().StringVar(&opts.grantID, "grant-id", "", "Grant ID or email for grant-scoped workflows")
}

func resolveScope(opts scopeOptions) (domain.RemoteScope, string, error) {
	scope, err := domain.ParseRemoteScope(opts.scope)
	if err != nil {
		return "", "", common.NewUserError("invalid --scope value", "Use --scope app or --scope grant")
	}

	grantID, err := common.ResolveScopeGrantID(scope, opts.grantID)
	if err != nil {
		return "", "", err
	}

	return scope, grantID, nil
}

func withClient(
	ctx context.Context,
	fn func(context.Context, ports.NylasClient) error,
) error {
	client, err := common.GetNylasClient()
	if err != nil {
		return err
	}
	return fn(ctx, client)
}

func printWorkflow(workflow *domain.RemoteWorkflow) {
	fmt.Printf("ID:           %s\n", workflow.ID)
	fmt.Printf("Name:         %s\n", workflow.Name)
	fmt.Printf("Trigger:      %s\n", workflow.TriggerEvent)
	fmt.Printf("Template ID:  %s\n", workflow.TemplateID)
	fmt.Printf("Delay:        %d minute(s)\n", workflow.Delay)
	fmt.Printf("Enabled:      %t\n", workflow.IsEnabled)
	if workflow.From != nil {
		fmt.Printf("Sender Name:  %s\n", workflow.From.Name)
		fmt.Printf("Sender Email: %s\n", workflow.From.Email)
	}
	if !workflow.DateCreated.IsZero() {
		fmt.Printf("Created:      %s\n", workflow.DateCreated.Format("2006-01-02 15:04:05"))
	}
}

func validateTrigger(trigger string) error {
	return common.ValidateOneOf("trigger_event", trigger, domain.WorkflowTriggerEvents())
}

func nextCursorNote(cmd *cobra.Command, nextCursor string) {
	if nextCursor == "" || common.IsStructuredOutput(cmd) {
		return
	}
	fmt.Printf("\nNext page token: %s\n", nextCursor)
}

func senderFromFlags(name, email string) *domain.WorkflowSender {
	if name == "" && email == "" {
		return nil
	}
	return &domain.WorkflowSender{Name: name, Email: email}
}

func enabledPtr(enabled, disabled bool) (*bool, error) {
	if enabled && disabled {
		return nil, common.NewUserError("cannot use --enabled and --disabled together", "Choose only one enable flag")
	}
	if enabled {
		value := true
		return &value, nil
	}
	if disabled {
		value := false
		return &value, nil
	}
	return nil, nil
}
