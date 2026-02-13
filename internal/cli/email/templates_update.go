package email

import (
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/adapters/templates"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newTemplatesUpdateCmd() *cobra.Command {
	var name string
	var subject string
	var body string
	var category string

	cmd := &cobra.Command{
		Use:   "update <template-id>",
		Short: "Update an email template",
		Long: `Update an existing email template.

Only the fields you specify will be updated. Variables are automatically
re-extracted from the updated subject and body.`,
		Example: `  # Update the name
  nylas email templates update tpl_1234567890 --name "New Name"

  # Update subject and body
  nylas email templates update tpl_1234567890 \
    --subject "Updated: {{topic}}" \
    --body "Hi {{name}}, this is updated content."

  # Change category
  nylas email templates update tpl_1234567890 --category marketing`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateID := args[0]

			// Check if any update flags were provided
			if name == "" && subject == "" && body == "" && category == "" {
				return common.NewUserError(
					"no updates specified",
					"Use --name, --subject, --body, or --category to specify what to update",
				)
			}

			store := getTemplateStore()
			ctx, cancel := common.CreateContext()
			defer cancel()

			// Get existing template
			tpl, err := store.Get(ctx, templateID)
			if err != nil {
				if err == templates.ErrTemplateNotFound {
					return common.NewUserError(
						fmt.Sprintf("template %q not found", templateID),
						"Use 'nylas email templates list' to see available templates",
					)
				}
				return common.WrapGetError("template", err)
			}

			// Apply updates
			if name != "" {
				tpl.Name = name
			}
			if subject != "" {
				tpl.Subject = subject
			}
			if body != "" {
				tpl.HTMLBody = body
			}
			if category != "" {
				tpl.Category = category
			}

			// Update template
			updated, err := store.Update(ctx, tpl)
			if err != nil {
				return common.WrapUpdateError("template", err)
			}

			// JSON output
			if common.IsStructuredOutput(cmd) {
				return common.PrintJSON(updated)
			}

			// Success output
			printSuccess("Template updated successfully!")
			fmt.Printf("\n  ID:        %s\n", updated.ID)
			fmt.Printf("  Name:      %s\n", updated.Name)
			fmt.Printf("  Subject:   %s\n", updated.Subject)
			if updated.Category != "" {
				fmt.Printf("  Category:  %s\n", updated.Category)
			}
			if len(updated.Variables) > 0 {
				fmt.Printf("  Variables: %s\n", strings.Join(updated.Variables, ", "))
			}
			fmt.Printf("  Updated:   %s\n", updated.UpdatedAt.Format(common.DisplayDateTime))
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "New template name")
	cmd.Flags().StringVarP(&subject, "subject", "s", "", "New email subject")
	cmd.Flags().StringVarP(&body, "body", "b", "", "New email body")
	cmd.Flags().StringVarP(&category, "category", "c", "", "New template category")

	return cmd
}
