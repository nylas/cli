package email

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type hostedTemplateSendOptions struct {
	TemplateID       string
	TemplateScope    string
	TemplateGrantID  string
	TemplateData     string
	TemplateDataFile string
	RenderOnly       bool
	Strict           bool
}

type hostedTemplateSendResult struct {
	GrantID string
	Result  domain.TemplateRenderResult
	Subject string
	Body    string
}

func validateHostedTemplateSendOptions(opts hostedTemplateSendOptions, subject, body string) error {
	if opts.TemplateID == "" {
		switch {
		case opts.TemplateGrantID != "":
			return common.NewUserError("`--template-grant-id` requires `--template-id`", "Use --template-id to render and send a hosted template")
		case opts.TemplateData != "" || opts.TemplateDataFile != "":
			return common.NewUserError("template data requires `--template-id`", "Use --template-id with --template-data or --template-data-file")
		case opts.RenderOnly:
			return common.NewUserError("`--render-only` requires `--template-id`", "Use --template-id to preview a hosted template render")
		case opts.TemplateScope != "" && opts.TemplateScope != string(domain.ScopeApplication):
			return common.NewUserError("`--template-scope` requires `--template-id`", "Use --template-id with --template-scope app or grant")
		}
		return nil
	}

	if subject != "" || body != "" {
		return common.NewUserError(
			"`--template-id` cannot be combined with `--subject` or `--body`",
			"Use the hosted template subject/body, or remove --template-id and provide raw content",
		)
	}

	if opts.TemplateScope == "" {
		opts.TemplateScope = string(domain.ScopeApplication)
	}

	return nil
}

func renderHostedTemplateForSend(
	ctx context.Context,
	client ports.NylasClient,
	sendGrantID string,
	opts hostedTemplateSendOptions,
) (*hostedTemplateSendResult, error) {
	if opts.TemplateID == "" {
		return nil, nil
	}
	if opts.TemplateScope == "" {
		opts.TemplateScope = string(domain.ScopeApplication)
	}

	scope, err := domain.ParseRemoteScope(opts.TemplateScope)
	if err != nil {
		return nil, common.NewUserError("invalid `--template-scope` value", "Use --template-scope app or --template-scope grant")
	}

	renderGrantID := ""
	if scope == domain.ScopeGrant {
		renderGrantID, err = common.ResolveGrantIdentifier(opts.TemplateGrantID)
		if err != nil {
			return nil, err
		}
		if renderGrantID == "" {
			renderGrantID = sendGrantID
		}
		if renderGrantID == "" {
			return nil, common.NewUserError("grant-scoped templates require a grant ID", "Provide a send grant or set --template-grant-id")
		}
	}

	vars, err := common.ReadJSONStringMap(opts.TemplateData, opts.TemplateDataFile)
	if err != nil {
		return nil, err
	}

	rendered, err := client.RenderRemoteTemplate(ctx, scope, renderGrantID, opts.TemplateID, &domain.TemplateRenderRequest{
		Strict:    &opts.Strict,
		Variables: vars,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render hosted template %s: %w", opts.TemplateID, err)
	}

	subject, body, err := extractRenderedEmailContent(rendered)
	if err != nil {
		return nil, err
	}

	return &hostedTemplateSendResult{
		GrantID: renderGrantID,
		Result:  rendered,
		Subject: subject,
		Body:    body,
	}, nil
}

func hostedTemplateSendNeedsGrant(opts hostedTemplateSendOptions) (bool, error) {
	if opts.TemplateID == "" || !opts.RenderOnly {
		return true, nil
	}

	scope := domain.ScopeApplication
	if opts.TemplateScope != "" {
		parsedScope, err := domain.ParseRemoteScope(opts.TemplateScope)
		if err != nil {
			return false, common.NewUserError("invalid `--template-scope` value", "Use --template-scope app or --template-scope grant")
		}
		scope = parsedScope
	}

	if scope == domain.ScopeApplication {
		return false, nil
	}

	return strings.TrimSpace(opts.TemplateGrantID) == "", nil
}

func extractRenderedEmailContent(result domain.TemplateRenderResult) (string, string, error) {
	subject, _ := result["subject"].(string)
	body, _ := result["body"].(string)
	if body == "" {
		if html, ok := result["html"].(string); ok {
			body = html
		}
	}

	if strings.TrimSpace(subject) == "" {
		return "", "", common.NewUserError("rendered template is missing a subject", "Ensure the hosted template returns a subject")
	}
	if strings.TrimSpace(body) == "" {
		return "", "", common.NewUserError("rendered template is missing a body", "Ensure the hosted template returns a body")
	}

	return subject, body, nil
}

func printHostedTemplatePreview(templateID, subject, body string, to, cc, bcc []string) {
	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))
	_, _ = common.BoldWhite.Printf("HOSTED TEMPLATE PREVIEW: %s\n", templateID)
	fmt.Println(strings.Repeat("─", 60))

	if len(to) > 0 {
		fmt.Printf("To:      %s\n", strings.Join(to, ", "))
	}
	if len(cc) > 0 {
		fmt.Printf("Cc:      %s\n", strings.Join(cc, ", "))
	}
	if len(bcc) > 0 {
		fmt.Printf("Bcc:     %s\n", strings.Join(bcc, ", "))
	}
	fmt.Printf("Subject: %s\n", subject)

	fmt.Println()
	_, _ = common.Dim.Println("Body:")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println(body)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()
	_, _ = common.Dim.Println("This is a preview. Remove --render-only to send the email.")
	fmt.Println()
}
