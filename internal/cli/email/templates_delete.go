package email

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nylas/cli/internal/adapters/templates"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newTemplatesDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <template-id>",
		Short: "Delete an email template",
		Long: `Delete an email template from local storage.

By default, you'll be prompted for confirmation. Use --force to skip the prompt.`,
		Example: `  # Delete with confirmation
  nylas email templates delete tpl_1234567890

  # Delete without confirmation
  nylas email templates delete tpl_1234567890 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateID := args[0]

			store := getTemplateStore()
			ctx, cancel := common.CreateContext()
			defer cancel()

			// Get template to show what's being deleted
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

			// Confirmation
			if !force {
				fmt.Printf("Delete template %q?\n", tpl.Name)
				fmt.Printf("  ID:      %s\n", tpl.ID)
				fmt.Printf("  Subject: %s\n", tpl.Subject)
				if tpl.UsageCount > 0 {
					_, _ = common.Yellow.Printf("  Used:    %d time(s)\n", tpl.UsageCount)
				}
				fmt.Print("\nAre you sure? [y/N]: ")

				reader := bufio.NewReader(os.Stdin)
				confirm, _ := reader.ReadString('\n')
				confirm = strings.ToLower(strings.TrimSpace(confirm))
				if confirm != "y" && confirm != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			// Delete
			if err := store.Delete(ctx, templateID); err != nil {
				return common.WrapDeleteError("template", err)
			}

			common.PrintSuccess("Template %q deleted successfully", tpl.Name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}
