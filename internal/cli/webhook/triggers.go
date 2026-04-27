package webhook

import (
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newTriggersCmd() *cobra.Command {
	var category string

	cmd := &cobra.Command{
		Use:     "triggers",
		Aliases: []string{"trigger-types", "events"},
		Short:   "List available webhook trigger types",
		Long: `List all available webhook trigger types.

Trigger types define which events will cause webhook notifications.
Use these values when creating or updating webhooks.`,
		Example: `  # List all trigger types
  nylas webhook triggers

  # List in JSON format
  nylas webhook triggers --format json

  # List only message-related triggers
  nylas webhook triggers --category message`,
		RunE: func(cmd *cobra.Command, args []string) error {
			categories := domain.TriggerTypeCategories()

			// Filter by category if specified
			if category != "" {
				if triggers, ok := categories[category]; ok {
					categories = map[string][]string{category: triggers}
				} else {
					validCategories := []string{}
					for c := range categories {
						validCategories = append(validCategories, c)
					}
					fmt.Printf("Invalid category: %s\n", category)
					fmt.Printf("Valid categories: %v\n", validCategories)
					return nil
				}
			}

			// "list" is a special flat format unique to this command.
			format, _ := cmd.Flags().GetString("format")
			if format == "list" {
				for _, triggers := range categories {
					for _, t := range triggers {
						fmt.Println(t)
					}
				}
				return nil
			}

			if common.IsStructuredOutput(cmd) {
				out := common.GetOutputWriter(cmd)
				return out.Write(categories)
			}

			return displayTriggerCategories(categories)
		},
	}

	cmd.Flags().StringVarP(&category, "category", "c", "", "Filter by category (grant, message, thread, event, contact, calendar, folder, notetaker)")

	return cmd
}

func displayTriggerCategories(categories map[string][]string) error {
	fmt.Println("Available Webhook Trigger Types")
	fmt.Println("================================")
	fmt.Println()

	// Display in a nice order
	categoryOrder := []string{"grant", "message", "thread", "event", "contact", "calendar", "folder", "notetaker"}
	categoryDescriptions := map[string]string{
		"grant":     "Authentication grant events",
		"message":   "Email message events",
		"thread":    "Email thread events",
		"event":     "Calendar event events",
		"contact":   "Contact events",
		"calendar":  "Calendar events",
		"folder":    "Email folder events",
		"notetaker": "Meeting notetaker events",
	}

	categoryEmojis := map[string]string{
		"grant":     "🔑",
		"message":   "📧",
		"thread":    "💬",
		"event":     "📅",
		"contact":   "👤",
		"calendar":  "📆",
		"folder":    "📁",
		"notetaker": "📝",
	}

	for _, cat := range categoryOrder {
		triggers, ok := categories[cat]
		if !ok {
			continue
		}

		emoji := categoryEmojis[cat]
		desc := categoryDescriptions[cat]

		fmt.Printf("%s %s\n", emoji, capitalize(cat))
		if desc != "" {
			fmt.Printf("   %s\n", desc)
		}
		fmt.Println()

		for _, t := range triggers {
			fmt.Printf("   • %s\n", t)
		}
		fmt.Println()
	}

	fmt.Println("Usage:")
	fmt.Println("  nylas webhook create --url <URL> --triggers message.created")
	fmt.Println("  nylas webhook create --url <URL> --triggers message.created,event.created")

	return nil
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}
