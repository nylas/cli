package email

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nylas/cli/internal/adapters/templates"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newTemplatesUseCmd() *cobra.Command {
	var to []string
	var cc []string
	var bcc []string
	var vars []string
	var preview bool
	var noConfirm bool
	var jsonOutput bool
	var signatureID string

	cmd := &cobra.Command{
		Use:   "use <template-id>",
		Short: "Use a template to send an email",
		Long: `Use an email template to compose and send a message.

Variables in the template ({{variable}}) are replaced with values provided
via --var flags. Missing variables will cause an error unless --preview is used.

Use --preview to see the expanded template without sending.`,
		Example: `  # Send using a template
  nylas email templates use tpl_1234567890 --to user@example.com --var name=John

  # Preview without sending
  nylas email templates use tpl_1234567890 --to user@example.com --var name=John --preview

  # Multiple recipients and variables
  nylas email templates use tpl_1234567890 \
    --to alice@example.com --to bob@example.com \
    --var name=Team --var topic="Q4 Planning"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateID := args[0]

			// Validation
			if len(to) == 0 {
				return common.NewUserError("at least one recipient is required", "Use --to to specify recipient email addresses")
			}

			store := getTemplateStore()
			ctx, cancel := common.CreateContext()
			defer cancel()

			// Get template
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

			// Parse variables
			varMap := parseVars(vars)

			// Expand variables in subject and body
			expandedSubject, missingSubject := templates.ExpandVariables(tpl.Subject, varMap)
			expandedBody, missingBody := templates.ExpandVariables(tpl.HTMLBody, varMap)

			// Check for missing variables
			missingVars := append(missingSubject, missingBody...)
			if len(missingVars) > 0 && !preview {
				return common.NewUserError(
					fmt.Sprintf("missing variable(s): %s", strings.Join(uniqueStrings(missingVars), ", ")),
					fmt.Sprintf("Provide values with: %s", formatVarFlags(missingVars)),
				)
			}

			// Preview mode
			if preview {
				printTemplatePreview(tpl.Name, expandedSubject, expandedBody, to, cc, bcc, missingVars)
				return nil
			}

			// Confirmation
			fmt.Println("\nEmail preview:")
			fmt.Printf("  Template: %s\n", tpl.Name)
			fmt.Printf("  To:       %s\n", strings.Join(to, ", "))
			if len(cc) > 0 {
				fmt.Printf("  Cc:       %s\n", strings.Join(cc, ", "))
			}
			if len(bcc) > 0 {
				fmt.Printf("  Bcc:      %s\n", strings.Join(bcc, ", "))
			}
			fmt.Printf("  Subject:  %s\n", expandedSubject)
			fmt.Printf("  Body:     %s\n", common.Truncate(expandedBody, 50))

			if !noConfirm {
				fmt.Print("\nSend this email? [y/N]: ")
				reader := bufio.NewReader(os.Stdin)
				confirm, _ := reader.ReadString('\n')
				confirm = strings.ToLower(strings.TrimSpace(confirm))
				if confirm != "y" && confirm != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			// Parse recipients
			toContacts, err := parseContacts(to)
			if err != nil {
				return common.WrapRecipientError("to", err)
			}

			// Build request
			req := &domain.SendMessageRequest{
				Subject:     expandedSubject,
				Body:        expandedBody,
				To:          toContacts,
				SignatureID: signatureID,
			}

			if len(cc) > 0 {
				ccContacts, err := parseContacts(cc)
				if err != nil {
					return common.WrapRecipientError("cc", err)
				}
				req.Cc = ccContacts
			}
			if len(bcc) > 0 {
				bccContacts, err := parseContacts(bcc)
				if err != nil {
					return common.WrapRecipientError("bcc", err)
				}
				req.Bcc = bccContacts
			}

			// Send email
			_, sendErr := common.WithClient(nil, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				if _, err := validateSignatureSelection(ctx, client, grantID, signatureID, nil); err != nil {
					return struct{}{}, err
				}

				spinner := common.NewSpinner("Sending email...")
				spinner.Start()

				msg, err := client.SendMessage(ctx, grantID, req)
				spinner.Stop()

				if err != nil {
					return struct{}{}, common.WrapSendError("email", err)
				}

				// Increment usage count (best effort)
				storeCtx, storeCancel := common.CreateContext()
				defer storeCancel()
				_ = store.IncrementUsage(storeCtx, templateID)

				if jsonOutput {
					return struct{}{}, common.PrintJSON(msg)
				}

				printSuccess("Email sent successfully! Message ID: %s", msg.ID)
				return struct{}{}, nil
			})

			return sendErr
		},
	}

	cmd.Flags().StringSliceVarP(&to, "to", "t", nil, "Recipient email addresses (required)")
	cmd.Flags().StringSliceVar(&cc, "cc", nil, "CC email addresses")
	cmd.Flags().StringSliceVar(&bcc, "bcc", nil, "BCC email addresses")
	cmd.Flags().StringSliceVar(&vars, "var", nil, "Variable values as key=value (can be repeated)")
	cmd.Flags().BoolVarP(&preview, "preview", "p", false, "Preview the expanded template without sending")
	cmd.Flags().BoolVarP(&noConfirm, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&signatureID, "signature-id", "", "Stored signature ID to append when sending")

	return cmd
}

// parseVars parses key=value pairs into a map.
func parseVars(vars []string) map[string]string {
	result := make(map[string]string)
	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

// uniqueStrings returns unique strings from a slice.
func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// formatVarFlags formats variable names as --var flags.
func formatVarFlags(vars []string) string {
	var flags []string
	for _, v := range uniqueStrings(vars) {
		flags = append(flags, fmt.Sprintf("--var %s=<value>", v))
	}
	return strings.Join(flags, " ")
}

// printTemplatePreview prints a preview of the expanded template.
func printTemplatePreview(name, subject, body string, to, cc, bcc, missing []string) {
	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))
	_, _ = common.BoldWhite.Printf("TEMPLATE PREVIEW: %s\n", name)
	fmt.Println(strings.Repeat("─", 60))

	fmt.Printf("To:      %s\n", strings.Join(to, ", "))
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

	if len(missing) > 0 {
		fmt.Println()
		_, _ = common.Yellow.Printf("Warning: Missing variables: %s\n", strings.Join(uniqueStrings(missing), ", "))
		fmt.Println("Provide values with:", formatVarFlags(missing))
	}

	fmt.Println()
	_, _ = common.Dim.Println("This is a preview. Remove --preview to send the email.")
	fmt.Println()
}
