// users.go provides user listing commands for Slack workspaces.

package slack

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
)

// newUsersCmd creates the users command for managing workspace users.
func newUsersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "users",
		Aliases: []string{"user", "members"},
		Short:   "Manage workspace users",
		Long:    `Commands for listing and managing Slack workspace users.`,
	}

	cmd.AddCommand(newUserListCmd())
	cmd.AddCommand(newUserGetCmd())

	return cmd
}

// newUserListCmd creates the list subcommand for listing workspace users.
func newUserListCmd() *cobra.Command {
	var (
		limit  int
		showID bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List workspace users",
		Long: `List members of your Slack workspace.

Examples:
  # List users
  nylas slack users list

  # Show user IDs
  nylas slack users list --id

  # Limit results
  nylas slack users list --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getSlackClientOrError()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			resp, err := client.ListUsers(ctx, limit, "")
			if err != nil {
				return common.WrapListError("users", err)
			}

			if len(resp.Users) == 0 {
				common.PrintEmptyState("users")
				return nil
			}

			// Handle structured output (JSON/YAML/quiet)
			if common.IsStructuredOutput(cmd) {
				out := common.GetOutputWriter(cmd)
				return out.Write(resp.Users)
			}

			for _, u := range resp.Users {
				name := u.BestDisplayName()
				fmt.Print(common.Cyan.Sprint(name))

				if u.Name != "" && u.Name != name {
					_, _ = common.Dim.Printf(" (@%s)", u.Name)
				}

				if showID {
					_, _ = common.Dim.Printf(" [%s]", u.ID)
				}

				if u.IsBot {
					_, _ = common.Yellow.Print(" [bot]")
				}
				if u.IsAdmin {
					_, _ = common.Yellow.Print(" [admin]")
				}

				fmt.Println()

				if u.Status != "" {
					_, _ = common.Dim.Printf("  %s\n", u.Status)
				}
			}

			if resp.NextCursor != "" {
				_, _ = common.Dim.Printf("\n(more users available)\n")
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "Maximum number of users to return")
	cmd.Flags().BoolVar(&showID, "id", false, "Show user IDs")

	return cmd
}

// newUserGetCmd creates the get subcommand for retrieving a single user's details.
func newUserGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <user-id>",
		Short: "Get detailed user information",
		Long: `Get detailed information about a Slack user including email and profile.

Examples:
  # Get user by ID
  nylas slack users get UXXXXXXXX

  # Get user by username (searches for match)
  nylas slack users get @username`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getSlackClientOrError()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			userArg := args[0]

			// If it looks like a username (@name or just name), search for the user
			var userID string
			if !isUserID(userArg) {
				foundID, err := findUserIDByName(ctx, client, userArg)
				if err != nil {
					return err
				}
				userID = foundID
			} else {
				userID = userArg
			}

			user, err := client.GetUser(ctx, userID)
			if err != nil {
				return common.WrapGetError("user", err)
			}

			// Handle structured output (JSON/YAML/quiet)
			if common.IsStructuredOutput(cmd) {
				out := common.GetOutputWriter(cmd)
				return out.Write(user)
			}

			// Display user info (table format)
			fmt.Println()
			fmt.Println(common.Cyan.Sprint(user.BestDisplayName()))

			if user.Title != "" {
				printField("Title", user.Title)
			}
			if user.Name != "" {
				printField("Username", "@"+user.Name)
			}
			if user.RealName != "" && user.RealName != user.DisplayName {
				printField("Real Name", user.RealName)
			}
			printField("User ID", user.ID)

			if user.Email != "" {
				printField("Email", user.Email)
			} else {
				printField("Email", common.Dim.Sprint("(not available - requires users:read.email scope)"))
			}

			if user.Phone != "" {
				printField("Phone", user.Phone)
			}

			// Status with emoji
			if user.Status != "" {
				status := user.Status
				if user.StatusEmoji != "" {
					status = user.StatusEmoji + " " + status
				}
				printField("Status", status)
			} else if user.StatusEmoji != "" {
				printField("Status", user.StatusEmoji)
			}

			if user.Timezone != "" {
				printField("Timezone", user.Timezone)
			}

			// Custom profile fields (Department, Location, etc.)
			if len(user.CustomFields) > 0 {
				for label, value := range user.CustomFields {
					printField(label, value)
				}
			}

			if user.Avatar != "" {
				printField("Avatar", user.Avatar)
			}

			// Flags
			var flags []string
			if user.IsAdmin {
				flags = append(flags, "admin")
			}
			if user.IsBot {
				flags = append(flags, "bot")
			}
			if len(flags) > 0 {
				printField("Flags", common.Yellow.Sprint(flags))
			}

			fmt.Println()
			return nil
		},
	}

	return cmd
}

// isUserID checks if the string looks like a Slack user ID (starts with U).
func isUserID(s string) bool {
	return len(s) > 1 && s[0] == 'U'
}

// findUserIDByName searches for a user by username and returns their ID.
func findUserIDByName(ctx context.Context, client ports.SlackClient, name string) (string, error) {
	// Strip @ prefix if present
	name = strings.TrimPrefix(name, "@")

	resp, err := client.ListUsers(ctx, 500, "")
	if err != nil {
		return "", common.WrapListError("users", err)
	}

	for _, u := range resp.Users {
		if strings.EqualFold(u.Name, name) ||
			strings.EqualFold(u.DisplayName, name) ||
			strings.EqualFold(u.RealName, name) {
			return u.ID, nil
		}
	}

	return "", common.NewUserError(
		fmt.Sprintf("user '%s' not found", name),
		"Use 'nylas slack users list' to see available users",
	)
}

// printField prints a labeled field value.
func printField(label, value string) {
	_, _ = common.Dim.Printf("  %-12s", label+":")
	fmt.Println(" " + value)
}
