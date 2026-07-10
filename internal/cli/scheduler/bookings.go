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
		Long: `Manage scheduler bookings (scheduled meetings).

API reference: https://developer.nylas.com/docs/reference/api/bookings/`,
	}

	cmd.AddCommand(newBookingShowCmd())
	cmd.AddCommand(newBookingConfirmCmd())
	cmd.AddCommand(newBookingRescheduleCmd())
	cmd.AddCommand(newBookingCancelCmd())

	return cmd
}

func newBookingShowCmd() *cobra.Command {
	var configurationID string
	cmd := &cobra.Command{
		Use:   "show <booking-id>",
		Short: "Show booking details",
		Long: `Show detailed information about a specific booking.

Booking endpoints are authorized by a Scheduler session token minted from the
configuration, so --configuration-id is required.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bookingID := args[0]
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				booking, err := client.GetBooking(ctx, configurationID, bookingID)
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

	addConfigurationIDFlag(cmd, &configurationID)

	return cmd
}

// addConfigurationIDFlag registers the required --configuration-id flag shared by
// all booking commands. Booking endpoints authenticate with a Scheduler session
// token minted from the configuration, so the configuration ID is mandatory.
func addConfigurationIDFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, "configuration-id", "", "Scheduler configuration ID that owns the booking (required)")
	_ = cmd.MarkFlagRequired("configuration-id")
}

func newBookingConfirmCmd() *cobra.Command {
	var (
		configurationID string
		salt            string
		reason          string // deprecated: the v3 confirm payload has no reason field
	)

	cmd := &cobra.Command{
		Use:   "confirm <booking-id>",
		Short: "Confirm a booking",
		Long: `Confirm a pending booking.

Confirming a booking requires the --salt from that booking's reference. Nylas
does not expose the salt through any read API, so it cannot be looked up from
the booking ID alone; you must take it from the booking reference, which appears
in the organizer confirmation link, the cancel/reschedule page URL, or a
Scheduler webhook payload.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bookingID := args[0]
			if salt == "" {
				return common.NewUserErrorWithSuggestions(
					"the --salt is required to confirm a booking",
					"Find it in the booking reference (it is not retrievable from the booking ID).",
					"The reference appears in the organizer confirmation link and the cancel/reschedule page URL.",
					"You can also read it from a Scheduler webhook payload.",
					"Then pass it: nylas scheduler bookings confirm <booking-id> --salt <salt>",
				)
			}
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				req := &domain.ConfirmBookingRequest{
					Salt:   salt,
					Status: "confirmed",
				}

				booking, err := client.ConfirmBooking(ctx, configurationID, bookingID, req)
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

	addConfigurationIDFlag(cmd, &configurationID)
	cmd.Flags().StringVar(&salt, "salt", "", "Salt from the booking reference, required (found in the confirmation link, cancel/reschedule URL, or a Scheduler webhook)")
	// --reason was removed from the v3 confirm payload; keep it as a deprecated
	// no-op so existing scripts degrade gracefully instead of hitting cobra's
	// "unknown flag" error on upgrade.
	cmd.Flags().StringVar(&reason, "reason", "", "")
	_ = cmd.Flags().MarkDeprecated("reason", "confirm no longer takes a reason; the flag is ignored")

	return cmd
}

func newBookingRescheduleCmd() *cobra.Command {
	var (
		configurationID string
		startTime       int64
		endTime         int64
		timezone        string
		reason          string
	)

	cmd := &cobra.Command{
		Use:   "reschedule <booking-id>",
		Short: "Reschedule a booking",
		Long: `Reschedule an existing booking to a new time.

You must provide the new start and end times as Unix timestamps.`,
		Example: `  # Reschedule to a new time (Unix timestamps)
  nylas scheduler bookings reschedule abc123 --configuration-id cfg123 --start-time 1704067200 --end-time 1704070800`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if startTime == 0 || endTime == 0 {
				return fmt.Errorf("both --start-time and --end-time are required")
			}

			if endTime <= startTime {
				return fmt.Errorf("end-time must be after start-time")
			}

			bookingID := args[0]
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				req := &domain.RescheduleBookingRequest{
					StartTime: startTime,
					EndTime:   endTime,
				}

				booking, err := client.RescheduleBooking(ctx, configurationID, bookingID, req)
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

	addConfigurationIDFlag(cmd, &configurationID)
	cmd.Flags().Int64Var(&startTime, "start-time", 0, "New start time (Unix timestamp, required)")
	cmd.Flags().Int64Var(&endTime, "end-time", 0, "New end time (Unix timestamp, required)")
	// The v3 reschedule payload (booking_update) has only start/end times.
	// --timezone and --reason are kept as deprecated no-ops so existing scripts
	// don't break on upgrade.
	cmd.Flags().StringVar(&timezone, "timezone", "", "")
	cmd.Flags().StringVar(&reason, "reason", "", "")
	_ = cmd.Flags().MarkDeprecated("timezone", "reschedule does not accept a timezone; the flag is ignored")
	_ = cmd.Flags().MarkDeprecated("reason", "reschedule does not accept a reason; the flag is ignored")

	return cmd
}

func newBookingCancelCmd() *cobra.Command {
	var (
		configurationID string
		reason          string
		yes             bool
	)

	cmd := &cobra.Command{
		Use:   "cancel <booking-id>",
		Short: "Cancel a booking",
		Long:  "Cancel a scheduled booking.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				if !common.Confirm(fmt.Sprintf("Are you sure you want to cancel booking %s?", args[0]), false) {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			bookingID := args[0]
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				if err := client.CancelBooking(ctx, configurationID, bookingID, reason); err != nil {
					return struct{}{}, common.WrapCancelError("booking", err)
				}

				_, _ = common.Green.Printf("✓ Cancelled booking: %s\n", bookingID)

				return struct{}{}, nil
			})
			return err
		},
	}

	addConfigurationIDFlag(cmd, &configurationID)
	cmd.Flags().StringVar(&reason, "reason", "", "Cancellation reason")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
