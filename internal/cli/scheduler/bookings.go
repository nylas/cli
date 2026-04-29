package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newBookingsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bookings",
		Aliases: []string{"booking"},
		Short:   "Manage scheduler bookings",
		Long:    "Manage scheduler bookings (scheduled meetings).",
	}

	cmd.AddCommand(newBookingListCmd())
	cmd.AddCommand(newBookingShowCmd())
	cmd.AddCommand(newBookingConfirmCmd())
	cmd.AddCommand(newBookingRescheduleCmd())
	cmd.AddCommand(newBookingCancelCmd())

	return cmd
}

func newBookingListCmd() *cobra.Command {
	var (
		configID string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List scheduler bookings",
		Long:    "List all scheduler bookings, optionally filtered by configuration.",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				bookings, err := client.ListBookings(ctx, configID)
				if err != nil {
					return struct{}{}, common.WrapListError("bookings", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(bookings)
				}

				if len(bookings) == 0 {
					common.PrintEmptyState("bookings")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d booking(s):\n\n", len(bookings))

				table := common.NewTable("TITLE", "ID", "START TIME", "STATUS")
				for _, b := range bookings {
					startTime := b.StartTime.Format("2006-01-02 15:04")
					table.AddRow(common.Cyan.Sprint(b.Title), b.BookingID, startTime, common.ColorSprint(b.Status))
				}
				table.Render()

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&configID, "config-id", "", "Filter by configuration ID")

	return cmd
}

func newBookingShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <booking-id>",
		Short: "Show booking details",
		Long:  "Show detailed information about a specific booking.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bookingID := args[0]
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				booking, err := client.GetBooking(ctx, bookingID)
				if err != nil {
					return struct{}{}, common.WrapGetError("booking", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(booking)
				}

				_, _ = common.Bold.Printf("Booking: %s\n", booking.Title)
				fmt.Printf("  ID: %s\n", common.Cyan.Sprint(booking.BookingID))
				fmt.Printf("  Status: %s\n", common.ColorSprint(booking.Status))
				fmt.Printf("  Start: %s\n", booking.StartTime.Format(time.RFC1123))
				fmt.Printf("  End: %s\n", booking.EndTime.Format(time.RFC1123))

				if booking.EventID != "" {
					fmt.Printf("  Event ID: %s\n", booking.EventID)
				}

				if len(booking.Participants) > 0 {
					fmt.Printf("\nParticipants (%d):\n", len(booking.Participants))
					for i, p := range booking.Participants {
						fmt.Printf("  %d. %s <%s>", i+1, p.Name, p.Email)
						if p.Status == "yes" {
							fmt.Printf(" %s", common.Green.Sprint("✓"))
						}
						fmt.Println()
					}
				}

				if booking.Conferencing != nil && booking.Conferencing.URL != "" {
					fmt.Printf("\nConferencing:\n")
					fmt.Printf("  URL: %s\n", common.Cyan.Sprint(booking.Conferencing.URL))
					if booking.Conferencing.MeetingCode != "" {
						fmt.Printf("  Meeting Code: %s\n", booking.Conferencing.MeetingCode)
					}
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}

func newBookingConfirmCmd() *cobra.Command {
	var (
		reason string
	)

	cmd := &cobra.Command{
		Use:   "confirm <booking-id>",
		Short: "Confirm a booking",
		Long:  "Confirm a pending booking.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bookingID := args[0]
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				req := &domain.ConfirmBookingRequest{
					Status: "confirmed",
					Reason: reason,
				}

				booking, err := client.ConfirmBooking(ctx, bookingID, req)
				if err != nil {
					return struct{}{}, common.WrapUpdateError("booking", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, common.PrintJSON(booking)
				}

				_, _ = common.Green.Printf("✓ Confirmed booking: %s\n", booking.BookingID)
				fmt.Printf("  Status: %s\n", booking.Status)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&reason, "reason", "", "Reason for confirmation")

	return cmd
}

func newBookingRescheduleCmd() *cobra.Command {
	var (
		startTime int64
		endTime   int64
		timezone  string
		reason    string
	)

	cmd := &cobra.Command{
		Use:   "reschedule <booking-id>",
		Short: "Reschedule a booking",
		Long: `Reschedule an existing booking to a new time.

You must provide the new start and end times as Unix timestamps.`,
		Example: `  # Reschedule to a new time
  nylas scheduler bookings reschedule abc123 --start-time 1704067200 --end-time 1704070800

  # Reschedule with timezone
  nylas scheduler bookings reschedule abc123 --start-time 1704067200 --end-time 1704070800 --timezone "America/New_York"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if startTime == 0 || endTime == 0 {
				return fmt.Errorf("both --start-time and --end-time are required")
			}

			if endTime <= startTime {
				return fmt.Errorf("end-time must be after start-time")
			}

			bookingID := args[0]
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				req := &domain.RescheduleBookingRequest{
					StartTime: startTime,
					EndTime:   endTime,
					Timezone:  timezone,
					Reason:    reason,
				}

				booking, err := client.RescheduleBooking(ctx, bookingID, req)
				if err != nil {
					return struct{}{}, common.WrapUpdateError("booking", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, common.PrintJSON(booking)
				}

				_, _ = common.Green.Printf("✓ Rescheduled booking: %s\n", booking.BookingID)
				fmt.Printf("  New start: %s\n", booking.StartTime.Format(time.RFC1123))
				fmt.Printf("  New end: %s\n", booking.EndTime.Format(time.RFC1123))

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().Int64Var(&startTime, "start-time", 0, "New start time (Unix timestamp, required)")
	cmd.Flags().Int64Var(&endTime, "end-time", 0, "New end time (Unix timestamp, required)")
	cmd.Flags().StringVar(&timezone, "timezone", "", "Timezone for the booking (e.g., America/New_York)")
	cmd.Flags().StringVar(&reason, "reason", "", "Reason for rescheduling")

	return cmd
}

func newBookingCancelCmd() *cobra.Command {
	var (
		reason string
		yes    bool
	)

	cmd := &cobra.Command{
		Use:   "cancel <booking-id>",
		Short: "Cancel a booking",
		Long:  "Cancel a scheduled booking.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Printf("Are you sure you want to cancel booking %s? (y/N): ", args[0])
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			bookingID := args[0]
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				if err := client.CancelBooking(ctx, bookingID, reason); err != nil {
					return struct{}{}, common.WrapCancelError("booking", err)
				}

				_, _ = common.Green.Printf("✓ Cancelled booking: %s\n", bookingID)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&reason, "reason", "", "Cancellation reason")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
