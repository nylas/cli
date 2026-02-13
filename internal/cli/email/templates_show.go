package email

import (
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/adapters/templates"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newTemplatesShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <template-id>",
		Short: "Show template details",
		Long: `Display detailed information about an email template.

Shows the template's name, subject, body, variables, and usage statistics.`,
		Example: `  # Show template details
  nylas email templates show tpl_1234567890

  # Output as JSON
  nylas email templates show tpl_1234567890 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateID := args[0]

			store := getTemplateStore()
			ctx, cancel := common.CreateContext()
			defer cancel()

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

			// JSON output
			if common.IsStructuredOutput(cmd) {
				return common.PrintJSON(tpl)
			}

			// Detailed output
			fmt.Println()
			fmt.Println(strings.Repeat("─", 60))
			_, _ = common.BoldWhite.Printf("Template: %s\n", tpl.Name)
			fmt.Println(strings.Repeat("─", 60))

			fmt.Printf("ID:         %s\n", tpl.ID)
			fmt.Printf("Name:       %s\n", tpl.Name)

			if tpl.Category != "" {
				fmt.Printf("Category:   %s\n", common.Cyan.Sprint(tpl.Category))
			}

			fmt.Printf("Created:    %s\n", tpl.CreatedAt.Format(common.DisplayDateTime))
			fmt.Printf("Updated:    %s\n", tpl.UpdatedAt.Format(common.DisplayDateTime))
			fmt.Printf("Used:       %d time(s)\n", tpl.UsageCount)

			if len(tpl.Variables) > 0 {
				fmt.Printf("Variables:  %s\n", common.Yellow.Sprint(strings.Join(tpl.Variables, ", ")))
			}

			fmt.Println()
			_, _ = common.Dim.Println("Subject:")
			fmt.Printf("  %s\n", tpl.Subject)

			fmt.Println()
			_, _ = common.Dim.Println("Body:")
			// Indent body lines
			bodyLines := strings.Split(tpl.HTMLBody, "\n")
			for _, line := range bodyLines {
				fmt.Printf("  %s\n", line)
			}

			// Usage hint
			fmt.Println()
			fmt.Println(strings.Repeat("─", 60))
			_, _ = common.Dim.Println("Usage:")
			if len(tpl.Variables) > 0 {
				varFlags := ""
				for _, v := range tpl.Variables {
					varFlags += fmt.Sprintf(" --var %s=<value>", v)
				}
				fmt.Printf("  nylas email templates use %s --to <email>%s\n", tpl.ID, varFlags)
			} else {
				fmt.Printf("  nylas email templates use %s --to <email>\n", tpl.ID)
			}
			fmt.Println()

			return nil
		},
	}

	return cmd
}
