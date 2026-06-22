package email

import (
	"github.com/nylas/cli/internal/adapters/templates"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

// newTemplatesCmd creates the templates command group.
func newTemplatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Manage email templates",
		Long: `Manage locally-stored email templates for composing messages.

Templates are stored in ~/.config/nylas/templates.json and support variable
substitution using {{variable}} syntax. Variables are automatically extracted
from the subject and body when templates are created or updated.

Templates are local to your machine and are not synced with Nylas.`,
		Example: `  # List all templates
  nylas email templates list

  # Create a template with variables
  nylas email templates create --name "Welcome" --subject "Welcome {{name}}!" --body "Hello {{name}}, welcome to {{company}}!"

  # Use a template to send an email
  nylas email templates use <template-id> --to user@example.com --var name=John --var company=Acme

  # Preview before sending
  nylas email templates use <template-id> --to user@example.com --var name=John --preview`,
	}

	// Add output flags for JSON/YAML support
	common.AddOutputFlags(cmd)

	// Add subcommands
	cmd.AddCommand(newTemplatesListCmd())
	cmd.AddCommand(newTemplatesShowCmd())
	cmd.AddCommand(newTemplatesCreateCmd())
	cmd.AddCommand(newTemplatesUpdateCmd())
	cmd.AddCommand(newTemplatesDeleteCmd())
	cmd.AddCommand(newTemplatesUseCmd())

	return cmd
}

// getTemplateStore creates a template store instance.
func getTemplateStore() *templates.FileStore {
	return templates.NewDefaultFileStore()
}
