package email

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newTemplatesCreateCmd() *cobra.Command {
	var name string
	var subject string
	var body string
	var category string
	var interactive bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an email template",
		Long: `Create a new email template with optional variable placeholders.

Variables use {{variable}} syntax and are automatically extracted from the
subject and body. When using the template, you can provide values for these
variables with --var key=value flags.

Use --interactive for a guided creation experience.`,
		Example: `  # Create a simple template
  nylas email templates create --name "Welcome" --subject "Welcome!" --body "Hello and welcome!"

  # Create a template with variables
  nylas email templates create \
    --name "Follow-up" \
    --subject "Following up on {{topic}}" \
    --body "Hi {{name}},\n\nWanted to follow up on {{topic}}.\n\nBest,\n{{sender}}" \
    --category sales

  # Interactive mode
  nylas email templates create --interactive`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Interactive mode
			if interactive {
				reader := bufio.NewReader(os.Stdin)

				fmt.Println("Create a new email template")
				fmt.Println("Use {{variable}} syntax for placeholders")
				fmt.Println()

				if name == "" {
					fmt.Print("Template name: ")
					input, _ := reader.ReadString('\n')
					name = strings.TrimSpace(input)
				}

				if subject == "" {
					fmt.Print("Subject (supports {{variables}}): ")
					input, _ := reader.ReadString('\n')
					subject = strings.TrimSpace(input)
				}

				if body == "" {
					fmt.Println("Body (end with a line containing only '.'):")
					var bodyLines []string
					for {
						line, _ := reader.ReadString('\n')
						line = strings.TrimSuffix(line, "\n")
						if line == "." {
							break
						}
						bodyLines = append(bodyLines, line)
					}
					body = strings.Join(bodyLines, "\n")
				}

				if category == "" {
					fmt.Print("Category (optional, press Enter to skip): ")
					input, _ := reader.ReadString('\n')
					category = strings.TrimSpace(input)
				}
			}

			// Validation
			if name == "" {
				return common.NewUserError("template name is required", "Use --name to specify the template name")
			}
			if subject == "" {
				return common.NewUserError("subject is required", "Use --subject to specify the email subject")
			}
			if body == "" {
				return common.NewUserError("body is required", "Use --body to specify the email body")
			}

			// Create template
			store := getTemplateStore()
			ctx, cancel := common.CreateContext()
			defer cancel()

			tpl := &domain.EmailTemplate{
				Name:     name,
				Subject:  subject,
				HTMLBody: body,
				Category: category,
			}

			created, err := store.Create(ctx, tpl)
			if err != nil {
				return common.WrapCreateError("template", err)
			}

			// JSON output
			if common.IsStructuredOutput(cmd) {
				return common.PrintJSON(created)
			}

			// Success output
			printSuccess("Template created successfully!")
			fmt.Printf("\n  ID:        %s\n", created.ID)
			fmt.Printf("  Name:      %s\n", created.Name)
			fmt.Printf("  Subject:   %s\n", created.Subject)
			if created.Category != "" {
				fmt.Printf("  Category:  %s\n", created.Category)
			}
			if len(created.Variables) > 0 {
				fmt.Printf("  Variables: %s\n", strings.Join(created.Variables, ", "))
			}

			fmt.Println("\nUse this template with:")
			if len(created.Variables) > 0 {
				varFlags := ""
				for _, v := range created.Variables {
					varFlags += fmt.Sprintf(" --var %s=<value>", v)
				}
				fmt.Printf("  nylas email templates use %s --to recipient@example.com%s\n", created.ID, varFlags)
			} else {
				fmt.Printf("  nylas email templates use %s --to recipient@example.com\n", created.ID)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Template name (required)")
	cmd.Flags().StringVarP(&subject, "subject", "s", "", "Email subject (supports {{variables}})")
	cmd.Flags().StringVarP(&body, "body", "b", "", "Email body (supports {{variables}})")
	cmd.Flags().StringVarP(&category, "category", "c", "", "Template category (e.g., sales, support, marketing)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode")

	return cmd
}
