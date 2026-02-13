package email

import (
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newTemplatesListCmd() *cobra.Command {
	var category string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List email templates",
		Long: `List all locally-stored email templates.

Use --category to filter templates by category (case-insensitive).
Use --json or --yaml for machine-readable output.`,
		Example: `  # List all templates
  nylas email templates list

  # List templates in a category
  nylas email templates list --category sales

  # Output as JSON
  nylas email templates list --json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			store := getTemplateStore()
			ctx, cancel := common.CreateContext()
			defer cancel()

			templates, err := store.List(ctx, category)
			if err != nil {
				return common.WrapListError("templates", err)
			}

			// JSON output
			if common.IsStructuredOutput(cmd) {
				return common.PrintJSON(templates)
			}

			// Table output
			if len(templates) == 0 {
				if category != "" {
					fmt.Printf("No templates found in category %q\n", category)
				} else {
					fmt.Println("No templates found.")
					fmt.Println("\nCreate a template with:")
					fmt.Println("  nylas email templates create --name \"My Template\" --subject \"Subject\" --body \"Body\"")
				}
				return nil
			}

			// Print header
			fmt.Printf("\n%s\n", common.BoldWhite.Sprint("EMAIL TEMPLATES"))
			if category != "" {
				_, _ = common.Dim.Printf("Category: %s\n", category)
			}
			_, _ = common.Dim.Printf("Storage: %s\n\n", store.Path())

			// Print templates
			for _, t := range templates {
				printTemplateSummary(t.ID, t.Name, t.Subject, t.Category, t.UsageCount, len(t.Variables))
			}

			fmt.Printf("\nTotal: %d template(s)\n", len(templates))
			return nil
		},
	}

	cmd.Flags().StringVarP(&category, "category", "c", "", "Filter by category")

	return cmd
}

// printTemplateSummary prints a single-line template summary.
func printTemplateSummary(id, name, subject, category string, usageCount, varCount int) {
	// Truncate for display
	nameDisplay := common.Truncate(name, 25)
	subjectDisplay := common.Truncate(subject, 35)

	// Format category
	categoryDisplay := "-"
	if category != "" {
		categoryDisplay = category
	}

	// Format variables
	varsDisplay := ""
	if varCount > 0 {
		varsDisplay = fmt.Sprintf(" (%d vars)", varCount)
	}

	// Format usage
	usageDisplay := ""
	if usageCount > 0 {
		usageDisplay = fmt.Sprintf(" [used %dx]", usageCount)
	}

	fmt.Printf("  %s  %-25s  %-35s  %s%s%s\n",
		common.Dim.Sprint(common.Truncate(id, 16)),
		common.BoldWhite.Sprint(nameDisplay),
		subjectDisplay,
		common.Cyan.Sprint(categoryDisplay),
		common.Dim.Sprint(varsDisplay),
		common.Green.Sprint(usageDisplay))
}
