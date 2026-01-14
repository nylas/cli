package demo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

// Note: fatih/color import needed for *color.Color type in statusColor variables

func newDemoConfigUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update [config-id]",
		Short: "Update a scheduler configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			configID := "config-demo-123"
			if len(args) > 0 {
				configID = args[0]
			}

			fmt.Println()
			_, _ = common.Green.Printf("✓ Configuration %s would be updated (demo mode)\n", configID)

			return nil
		},
	}
}

func newDemoConfigDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [config-id]",
		Short: "Delete a scheduler configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			configID := "config-demo-123"
			if len(args) > 0 {
				configID = args[0]
			}

			fmt.Println()
			_, _ = common.Green.Printf("✓ Configuration %s would be deleted (demo mode)\n", configID)

			return nil
		},
	}
}

// ============================================================================
// SESSIONS COMMANDS
// ============================================================================

// newDemoSchedulerSessionsCmd creates the sessions subcommand group.
func newDemoSchedulerSessionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "Manage scheduler sessions",
		Long:  "Demo commands for managing scheduler sessions.",
	}

	cmd.AddCommand(newDemoSessionsListCmd())
	cmd.AddCommand(newDemoSessionsShowCmd())
	cmd.AddCommand(newDemoSessionsCreateCmd())
	cmd.AddCommand(newDemoSessionsDeleteCmd())

	return cmd
}

func newDemoSessionsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List scheduler sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println()
			fmt.Println(common.Dim.Sprint("📆 Demo Mode - Scheduler Sessions"))
			fmt.Println()

			sessions := []struct {
				id        string
				config    string
				status    string
				createdAt string
			}{
				{"session-001", "30-Minute Meeting", "active", "5 minutes ago"},
				{"session-002", "Quick Chat", "completed", "2 hours ago"},
				{"session-003", "Team Sync", "expired", "1 day ago"},
			}

			for _, s := range sessions {
				var statusColor *color.Color
				var statusIcon string
				switch s.status {
				case "active":
					statusColor = common.Green
					statusIcon = "●"
				case "completed":
					statusColor = common.Cyan
					statusIcon = "✓"
				default:
					statusColor = common.Dim
					statusIcon = "○"
				}

				fmt.Printf("  %s %s\n", statusColor.Sprint(statusIcon), common.BoldWhite.Sprint(s.config))
				fmt.Printf("    Status:  %s\n", statusColor.Sprint(s.status))
				fmt.Printf("    Created: %s\n", s.createdAt)
				_, _ = common.Dim.Printf("    ID:      %s\n", s.id)
				fmt.Println()
			}

			return nil
		},
	}
}

func newDemoSessionsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [session-id]",
		Short: "Show session details",
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := "session-demo-001"
			if len(args) > 0 {
				sessionID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📆 Demo Mode - Session Details"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			_, _ = common.BoldWhite.Println("Session: 30-Minute Meeting")
			fmt.Printf("  ID:            %s\n", sessionID)
			fmt.Printf("  Status:        %s\n", common.Green.Sprint("active"))
			fmt.Printf("  Configuration: config-demo-001\n")
			fmt.Printf("  Created:       5 minutes ago\n")
			fmt.Printf("  Expires:       in 25 minutes\n")
			fmt.Printf("  Booking URL:   https://schedule.nylas.com/s/%s\n", sessionID)
			fmt.Println(strings.Repeat("─", 50))

			return nil
		},
	}
}

func newDemoSessionsCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create [config-id]",
		Short: "Create a scheduler session",
		RunE: func(cmd *cobra.Command, args []string) error {
			configID := "config-demo-001"
			if len(args) > 0 {
				configID = args[0]
			}

			sessionID := fmt.Sprintf("session-demo-%d", time.Now().Unix())

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📆 Demo Mode - Create Session"))
			fmt.Println()
			fmt.Printf("Configuration: %s\n", configID)
			fmt.Println()
			_, _ = common.Green.Println("✓ Session would be created (demo mode)")
			_, _ = common.Dim.Printf("  Session ID:  %s\n", sessionID)
			_, _ = common.Dim.Printf("  Booking URL: https://schedule.nylas.com/s/%s\n", sessionID)

			return nil
		},
	}
}

func newDemoSessionsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [session-id]",
		Short: "Delete a scheduler session",
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := "session-demo-123"
			if len(args) > 0 {
				sessionID = args[0]
			}

			fmt.Println()
			_, _ = common.Green.Printf("✓ Session %s would be deleted (demo mode)\n", sessionID)

			return nil
		},
	}
}

// ============================================================================
// BOOKINGS COMMANDS
// ============================================================================

// newDemoSchedulerBookingsCmd lists sample bookings.
func newDemoSchedulerBookingsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bookings",
		Short: "Manage scheduler bookings",
		Long:  "Demo commands for managing scheduler bookings.",
	}

	cmd.AddCommand(newDemoBookingsListCmd())
	cmd.AddCommand(newDemoBookingsShowCmd())
	cmd.AddCommand(newDemoBookingsCancelCmd())
	cmd.AddCommand(newDemoBookingsRescheduleCmd())

	return cmd
}

func newDemoBookingsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List bookings",
		Example: `  # List sample bookings
  nylas demo scheduler bookings list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			bookings, err := client.ListBookings(ctx, "config-demo-1")
			if err != nil {
				return common.WrapListError("bookings", err)
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📆 Demo Mode - Scheduler Bookings"))
			fmt.Println()

			if len(bookings) == 0 {
				fmt.Println("No bookings found.")
			} else {
				fmt.Printf("Found %d bookings:\n\n", len(bookings))

				for _, booking := range bookings {
					statusColor := common.Green
					statusIcon := "✓"
					switch booking.Status {
					case "cancelled":
						statusColor = common.Red
						statusIcon = "✗"
					case "pending":
						statusColor = common.Yellow
						statusIcon = "○"
					}

					fmt.Printf("  %s %s\n", statusColor.Sprint(statusIcon), common.BoldWhite.Sprint(booking.Title))
					fmt.Printf("    Status: %s\n", statusColor.Sprint(booking.Status))
					_, _ = common.Dim.Printf("    ID: %s\n", booking.BookingID)
					fmt.Println()
				}
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("To manage your own bookings: nylas auth login"))

			return nil
		},
	}
}

func newDemoBookingsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [booking-id]",
		Short: "Show booking details",
		RunE: func(cmd *cobra.Command, args []string) error {
			bookingID := "booking-demo-001"
			if len(args) > 0 {
				bookingID = args[0]
			}

			now := time.Now().Add(24 * time.Hour)

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📆 Demo Mode - Booking Details"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			_, _ = common.BoldWhite.Println("30-Minute Meeting with John Doe")
			fmt.Printf("  ID:        %s\n", bookingID)
			fmt.Printf("  Status:    %s\n", common.Green.Sprint("confirmed"))
			fmt.Printf("  Date:      %s\n", now.Format("Monday, January 2, 2006"))
			fmt.Printf("  Time:      10:00 AM - 10:30 AM\n")
			fmt.Printf("  Timezone:  America/New_York\n")
			fmt.Println()
			fmt.Println("Attendee:")
			fmt.Printf("  Name:  John Doe\n")
			fmt.Printf("  Email: john.doe@example.com\n")
			fmt.Println()
			fmt.Println("Location:")
			fmt.Printf("  Zoom Meeting\n")
			_, _ = common.Dim.Printf("  https://zoom.us/j/123456789\n")
			fmt.Println(strings.Repeat("─", 50))

			return nil
		},
	}
}

func newDemoBookingsCancelCmd() *cobra.Command {
	var reason string

	cmd := &cobra.Command{
		Use:   "cancel [booking-id]",
		Short: "Cancel a booking",
		RunE: func(cmd *cobra.Command, args []string) error {
			bookingID := "booking-demo-123"
			if len(args) > 0 {
				bookingID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📆 Demo Mode - Cancel Booking"))
			fmt.Println()
			fmt.Printf("Booking ID: %s\n", bookingID)
			if reason != "" {
				fmt.Printf("Reason: %s\n", reason)
			}
			fmt.Println()
			_, _ = common.Green.Println("✓ Booking would be cancelled (demo mode)")
			fmt.Println("  Notification would be sent to attendees")

			return nil
		},
	}

	cmd.Flags().StringVar(&reason, "reason", "", "Cancellation reason")

	return cmd
}

func newDemoBookingsRescheduleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reschedule [booking-id]",
		Short: "Reschedule a booking",
		RunE: func(cmd *cobra.Command, args []string) error {
			bookingID := "booking-demo-123"
			if len(args) > 0 {
				bookingID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📆 Demo Mode - Reschedule Booking"))
			fmt.Println()
			fmt.Printf("Booking ID: %s\n", bookingID)
			fmt.Println()
			fmt.Println("Available times:")
			now := time.Now()
			fmt.Printf("  1. %s at 11:00 AM\n", now.AddDate(0, 0, 1).Format("Mon, Jan 2"))
			fmt.Printf("  2. %s at 2:00 PM\n", now.AddDate(0, 0, 1).Format("Mon, Jan 2"))
			fmt.Printf("  3. %s at 10:00 AM\n", now.AddDate(0, 0, 2).Format("Mon, Jan 2"))
			fmt.Println()
			_, _ = common.Green.Println("✓ Booking would be rescheduled (demo mode)")

			return nil
		},
	}
}

// ============================================================================
// PAGES COMMANDS
// ============================================================================

// newDemoSchedulerPagesCmd lists sample scheduler pages.
func newDemoSchedulerPagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pages",
		Short: "Manage scheduler pages",
		Long:  "Demo commands for managing scheduler pages.",
	}

	cmd.AddCommand(newDemoPagesListCmd())
	cmd.AddCommand(newDemoPagesShowCmd())
	cmd.AddCommand(newDemoPagesCreateCmd())
	cmd.AddCommand(newDemoPagesDeleteCmd())

	return cmd
}

func newDemoPagesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List scheduler pages",
		Example: `  # List sample scheduler pages
  nylas demo scheduler pages list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			pages, err := client.ListSchedulerPages(ctx)
			if err != nil {
				return common.WrapListError("scheduler pages", err)
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📆 Demo Mode - Scheduler Pages"))
			fmt.Println()
			fmt.Printf("Found %d pages:\n\n", len(pages))

			for _, page := range pages {
				fmt.Printf("  %s %s\n", "📄", common.BoldWhite.Sprint(page.Name))
				fmt.Printf("    Slug: %s\n", page.Slug)
				fmt.Printf("    URL:  https://schedule.nylas.com/%s\n", page.Slug)
				fmt.Println()
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("To create your own pages: nylas auth login"))

			return nil
		},
	}
}

func newDemoPagesShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [page-id]",
		Short: "Show page details",
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := "page-demo-001"
			if len(args) > 0 {
				pageID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📆 Demo Mode - Page Details"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			_, _ = common.BoldWhite.Println("Meeting Booking Page")
			fmt.Printf("  ID:          %s\n", pageID)
			fmt.Printf("  Slug:        book-a-meeting\n")
			fmt.Printf("  URL:         https://schedule.nylas.com/book-a-meeting\n")
			fmt.Printf("  Theme:       Light\n")
			fmt.Printf("  Brand color: #0066CC\n")
			fmt.Println()
			fmt.Println("Available meeting types:")
			fmt.Printf("  • 15-Minute Quick Chat\n")
			fmt.Printf("  • 30-Minute Meeting\n")
			fmt.Printf("  • 60-Minute Deep Dive\n")
			fmt.Println(strings.Repeat("─", 50))

			return nil
		},
	}
}

func newDemoPagesCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create a scheduler page",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println()
			_, _ = common.Green.Println("✓ Scheduler page would be created (demo mode)")
			_, _ = common.Dim.Printf("  Page ID: page-demo-%d\n", time.Now().Unix())

			return nil
		},
	}
}

func newDemoPagesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [page-id]",
		Short: "Delete a scheduler page",
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := "page-demo-123"
			if len(args) > 0 {
				pageID = args[0]
			}

			fmt.Println()
			_, _ = common.Green.Printf("✓ Page %s would be deleted (demo mode)\n", pageID)

			return nil
		},
	}
}
