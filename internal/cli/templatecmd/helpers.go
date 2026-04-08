package templatecmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

var templateColumns = []ports.Column{
	{Header: "ID", Field: "ID", Width: -1},
	{Header: "Name", Field: "Name", Width: 28},
	{Header: "Engine", Field: "Engine", Width: 12},
	{Header: "Subject", Field: "Subject", Width: 48},
	{Header: "Updated", Field: "UpdatedAt", Width: 18},
}

func addScopeFlags(cmd *cobra.Command, opts *scopeOptions) {
	cmd.Flags().StringVar(&opts.scope, "scope", string(domain.ScopeApplication), "Template scope: app or grant")
	cmd.Flags().StringVar(&opts.grantID, "grant-id", "", "Grant ID or email for grant-scoped templates")
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

func printTemplate(template *domain.RemoteTemplate) {
	fmt.Printf("ID:         %s\n", template.ID)
	fmt.Printf("Name:       %s\n", template.Name)
	fmt.Printf("Engine:     %s\n", template.Engine)
	fmt.Printf("Subject:    %s\n", template.Subject)
	if template.GrantID != "" {
		fmt.Printf("Grant ID:   %s\n", template.GrantID)
	}
	if template.AppID != nil && *template.AppID != "" {
		fmt.Printf("App ID:     %s\n", *template.AppID)
	}
	if !template.CreatedAt.IsZero() {
		fmt.Printf("Created:    %s\n", template.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	if !template.UpdatedAt.IsZero() {
		fmt.Printf("Updated:    %s\n", template.UpdatedAt.Format("2006-01-02 15:04:05"))
	}
	if template.Object != "" {
		fmt.Printf("Object:     %s\n", template.Object)
	}
	fmt.Printf("\nBody:\n%s\n", template.Body)
}

func printRenderResult(result domain.TemplateRenderResult) error {
	if len(result) == 0 {
		fmt.Println("Render completed with an empty response.")
		return nil
	}

	if subject, ok := result["subject"].(string); ok && subject != "" {
		fmt.Printf("Subject:\n%s\n\n", subject)
	}
	if body, ok := result["body"].(string); ok && body != "" {
		fmt.Printf("Body:\n%s\n", body)
		return nil
	}
	if html, ok := result["html"].(string); ok && html != "" {
		fmt.Printf("%s\n", html)
		return nil
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func validateEngine(engine string) error {
	return common.ValidateOneOf("engine", engine, domain.TemplateEngines())
}

func nextCursorNote(cmd *cobra.Command, nextCursor string) {
	if nextCursor == "" || common.IsStructuredOutput(cmd) {
		return
	}
	fmt.Printf("\nNext page token: %s\n", nextCursor)
}

func trimBody(body string) string {
	return strings.TrimRight(body, "\n")
}
