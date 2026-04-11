package scheduler

import (
	"fmt"
	"io"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

// configFlags holds all the extended configuration flags for create/update commands.
type configFlags struct {
	// Availability
	interval           int
	roundTo            int
	availabilityMethod string
	bufferBefore       int
	bufferAfter        int

	// Event booking
	timezone             string
	bookingType          string
	conferencingProvider string
	disableEmails        bool
	reminderMinutes      []int

	// Scheduler settings
	minBookingNotice      int
	minCancellationNotice int
	confirmationMethod    string
	availableDaysInFuture int
	cancellationPolicy    string

	// File input
	file string
}

// registerConfigFlags adds all extended configuration flags to a command.
func registerConfigFlags(cmd *cobra.Command, f *configFlags) {
	// Availability
	cmd.Flags().IntVar(&f.interval, "interval", 0, "Slot interval in minutes")
	cmd.Flags().IntVar(&f.roundTo, "round-to", 0, "Round start times to nearest N minutes")
	cmd.Flags().StringVar(&f.availabilityMethod, "availability-method", "", "Availability method (max-fairness, max-availability)")
	cmd.Flags().IntVar(&f.bufferBefore, "buffer-before", 0, "Buffer minutes before meetings")
	cmd.Flags().IntVar(&f.bufferAfter, "buffer-after", 0, "Buffer minutes after meetings")

	// Event booking
	cmd.Flags().StringVar(&f.timezone, "timezone", "", "Event timezone (e.g., America/New_York)")
	cmd.Flags().StringVar(&f.bookingType, "booking-type", "", "Booking type (booking, organizer-confirmation)")
	cmd.Flags().StringVar(&f.conferencingProvider, "conferencing-provider", "", "Conferencing provider (Google Meet, Zoom, Microsoft Teams)")
	cmd.Flags().BoolVar(&f.disableEmails, "disable-emails", false, "Disable email notifications")
	cmd.Flags().IntSliceVar(&f.reminderMinutes, "reminder-minutes", nil, "Reminder minutes (comma-separated, e.g., 10,60)")

	// Scheduler settings
	cmd.Flags().IntVar(&f.minBookingNotice, "min-booking-notice", 0, "Minimum minutes before a booking can be made")
	cmd.Flags().IntVar(&f.minCancellationNotice, "min-cancellation-notice", 0, "Minimum minutes before cancellation allowed")
	cmd.Flags().StringVar(&f.confirmationMethod, "confirmation-method", "", "Confirmation method (automatic, manual)")
	cmd.Flags().IntVar(&f.availableDaysInFuture, "available-days-in-future", 0, "How many days out bookings are available")
	cmd.Flags().StringVar(&f.cancellationPolicy, "cancellation-policy", "", "Cancellation policy text")

	// File input
	cmd.Flags().StringVar(&f.file, "file", "", "Path to JSON config file (flags override file values)")
}

// validateConfigFlags validates enum flag values.
func validateConfigFlags(f *configFlags) error {
	if f.availabilityMethod != "" {
		if err := common.ValidateOneOf("availability-method", f.availabilityMethod,
			[]string{"max-fairness", "max-availability"}); err != nil {
			return err
		}
	}
	if f.bookingType != "" {
		if err := common.ValidateOneOf("booking-type", f.bookingType,
			[]string{"booking", "organizer-confirmation"}); err != nil {
			return err
		}
	}
	if f.confirmationMethod != "" {
		if err := common.ValidateOneOf("confirmation-method", f.confirmationMethod,
			[]string{"automatic", "manual"}); err != nil {
			return err
		}
	}
	if f.conferencingProvider != "" {
		if err := common.ValidateOneOf("conferencing-provider", f.conferencingProvider,
			[]string{"Google Meet", "Zoom", "Microsoft Teams"}); err != nil {
			return err
		}
	}
	return nil
}

// buildCreateRequest constructs a CreateSchedulerConfigurationRequest from file and/or flags.
func buildCreateRequest(
	cmd *cobra.Command,
	f *configFlags,
	name string,
	participants []string,
	duration int,
	title, description, location string,
) (*domain.CreateSchedulerConfigurationRequest, error) {
	req := &domain.CreateSchedulerConfigurationRequest{}

	if f.file != "" {
		if err := common.LoadJSONFile(f.file, req); err != nil {
			return nil, err
		}
	}

	if f.file == "" || name != "" {
		if name != "" {
			req.Name = name
		}
	}
	if f.file == "" || len(participants) > 0 {
		if len(participants) > 0 {
			req.Participants = buildParticipants(participants)
		}
	}
	if f.file == "" || cmd.Flags().Changed("duration") {
		req.Availability.DurationMinutes = duration
	}
	if f.file == "" || title != "" {
		if title != "" {
			req.EventBooking.Title = title
		}
	}
	if f.file == "" || description != "" {
		if description != "" {
			req.EventBooking.Description = description
		}
	}
	if f.file == "" || location != "" {
		if location != "" {
			req.EventBooking.Location = location
		}
	}

	applyAvailabilityFlags(cmd, f, &req.Availability)
	applyEventBookingFlags(cmd, f, &req.EventBooking)
	applySchedulerFlags(cmd, f, &req.Scheduler)

	return req, nil
}

func buildParticipants(emails []string) []domain.ConfigurationParticipant {
	var participants []domain.ConfigurationParticipant
	for i, email := range emails {
		participants = append(participants, domain.ConfigurationParticipant{
			Email:       email,
			IsOrganizer: i == 0,
		})
	}
	return participants
}

func validateCreateRequest(req *domain.CreateSchedulerConfigurationRequest) error {
	if req.Name == "" {
		return common.ValidateRequiredFlag("--name", "")
	}
	if len(req.Participants) == 0 {
		return common.NewUserError("at least one participant is required", "Use --participants or include participants in --file")
	}
	if req.EventBooking.Title == "" {
		return common.ValidateRequiredFlag("--title", "")
	}
	return nil
}

func applyAvailabilityFlags(cmd *cobra.Command, f *configFlags, avail *domain.AvailabilityRules) {
	if cmd.Flags().Changed("interval") {
		avail.IntervalMinutes = f.interval
	}
	if cmd.Flags().Changed("round-to") {
		avail.RoundTo = f.roundTo
	}
	if cmd.Flags().Changed("availability-method") {
		avail.AvailabilityMethod = f.availabilityMethod
	}
	if cmd.Flags().Changed("buffer-before") || cmd.Flags().Changed("buffer-after") {
		if avail.Buffer == nil {
			avail.Buffer = &domain.AvailabilityBuffer{}
		}
		if cmd.Flags().Changed("buffer-before") {
			avail.Buffer.Before = f.bufferBefore
		}
		if cmd.Flags().Changed("buffer-after") {
			avail.Buffer.After = f.bufferAfter
		}
	}
}

func applyEventBookingFlags(cmd *cobra.Command, f *configFlags, booking *domain.EventBooking) {
	if cmd.Flags().Changed("timezone") {
		booking.Timezone = f.timezone
	}
	if cmd.Flags().Changed("booking-type") {
		booking.BookingType = f.bookingType
	}
	if cmd.Flags().Changed("disable-emails") {
		booking.DisableEmails = f.disableEmails
	}
	if cmd.Flags().Changed("reminder-minutes") {
		booking.ReminderMinutes = f.reminderMinutes
	}
	if cmd.Flags().Changed("conferencing-provider") {
		booking.Conferencing = &domain.ConferencingSettings{
			Provider:   f.conferencingProvider,
			Autocreate: true,
		}
	}
}

func applySchedulerFlags(cmd *cobra.Command, f *configFlags, sched *domain.SchedulerSettings) {
	if cmd.Flags().Changed("min-booking-notice") {
		sched.MinBookingNotice = f.minBookingNotice
	}
	if cmd.Flags().Changed("min-cancellation-notice") {
		sched.MinCancellationNotice = f.minCancellationNotice
	}
	if cmd.Flags().Changed("confirmation-method") {
		sched.ConfirmationMethod = f.confirmationMethod
	}
	if cmd.Flags().Changed("available-days-in-future") {
		sched.AvailableDaysInFuture = f.availableDaysInFuture
	}
	if cmd.Flags().Changed("cancellation-policy") {
		sched.CancellationPolicy = f.cancellationPolicy
	}
}

func buildUpdateRequest(
	cmd *cobra.Command,
	f *configFlags,
	name string,
	duration int,
	title, description string,
) (*domain.UpdateSchedulerConfigurationRequest, error) {
	req := &domain.UpdateSchedulerConfigurationRequest{}

	if f.file != "" {
		if err := common.LoadJSONFile(f.file, req); err != nil {
			return nil, err
		}
	}

	if name != "" {
		req.Name = &name
	}
	if cmd.Flags().Changed("duration") {
		if req.Availability == nil {
			req.Availability = &domain.AvailabilityRules{}
		}
		req.Availability.DurationMinutes = duration
	}
	if cmd.Flags().Changed("title") || cmd.Flags().Changed("description") {
		if req.EventBooking == nil {
			req.EventBooking = &domain.EventBooking{}
		}
		if cmd.Flags().Changed("title") {
			req.EventBooking.Title = title
		}
		if cmd.Flags().Changed("description") {
			req.EventBooking.Description = description
		}
	}

	if hasAvailabilityFlags(cmd) {
		if req.Availability == nil {
			req.Availability = &domain.AvailabilityRules{}
		}
		applyAvailabilityFlags(cmd, f, req.Availability)
	}
	if hasEventBookingFlags(cmd) {
		if req.EventBooking == nil {
			req.EventBooking = &domain.EventBooking{}
		}
		applyEventBookingFlags(cmd, f, req.EventBooking)
	}
	if hasSchedulerFlags(cmd) {
		if req.Scheduler == nil {
			req.Scheduler = &domain.SchedulerSettings{}
		}
		applySchedulerFlags(cmd, f, req.Scheduler)
	}

	return req, nil
}

func hasUpdateRequestChanges(req *domain.UpdateSchedulerConfigurationRequest) bool {
	return req.Name != nil ||
		req.Slug != nil ||
		req.RequiresSessionAuth != nil ||
		req.Participants != nil ||
		req.Availability != nil ||
		req.EventBooking != nil ||
		req.Scheduler != nil ||
		req.AppearanceSettings != nil
}

func validateUpdateRequest(req *domain.UpdateSchedulerConfigurationRequest) error {
	if hasUpdateRequestChanges(req) {
		return nil
	}
	return common.NewUserError(
		"No update fields provided",
		"Specify at least one field to update with flags or --file",
	)
}

func hasAvailabilityFlags(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("interval") ||
		cmd.Flags().Changed("round-to") ||
		cmd.Flags().Changed("availability-method") ||
		cmd.Flags().Changed("buffer-before") ||
		cmd.Flags().Changed("buffer-after")
}

func hasEventBookingFlags(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("timezone") ||
		cmd.Flags().Changed("booking-type") ||
		cmd.Flags().Changed("conferencing-provider") ||
		cmd.Flags().Changed("disable-emails") ||
		cmd.Flags().Changed("reminder-minutes")
}

func hasSchedulerFlags(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("min-booking-notice") ||
		cmd.Flags().Changed("min-cancellation-notice") ||
		cmd.Flags().Changed("confirmation-method") ||
		cmd.Flags().Changed("available-days-in-future") ||
		cmd.Flags().Changed("cancellation-policy")
}

// formatConfigDetails writes a human-readable configuration summary to w.
func formatConfigDetails(w io.Writer, config *domain.SchedulerConfiguration) {
	_, _ = common.Bold.Fprintf(w, "Configuration: %s\n", config.Name)
	_, _ = fmt.Fprintf(w, "  ID: %s\n", common.Cyan.Sprint(config.ID))
	if config.Slug != "" {
		_, _ = fmt.Fprintf(w, "  Slug: %s\n", common.Green.Sprint(config.Slug))
	}
	_, _ = fmt.Fprintf(w, "  Duration: %d minutes\n", config.Availability.DurationMinutes)
	if config.Availability.IntervalMinutes > 0 {
		_, _ = fmt.Fprintf(w, "  Interval: %d minutes\n", config.Availability.IntervalMinutes)
	}
	if config.Availability.RoundTo > 0 {
		_, _ = fmt.Fprintf(w, "  Round To: %d minutes\n", config.Availability.RoundTo)
	}
	if config.Availability.AvailabilityMethod != "" {
		_, _ = fmt.Fprintf(w, "  Availability Method: %s\n", config.Availability.AvailabilityMethod)
	}
	if config.Availability.Buffer != nil && (config.Availability.Buffer.Before > 0 || config.Availability.Buffer.After > 0) {
		_, _ = fmt.Fprintf(w, "  Buffer: %d min before, %d min after\n",
			config.Availability.Buffer.Before, config.Availability.Buffer.After)
	}

	if len(config.Participants) > 0 {
		_, _ = fmt.Fprintf(w, "\nParticipants (%d):\n", len(config.Participants))
		for i, p := range config.Participants {
			_, _ = fmt.Fprintf(w, "  %d. %s <%s>", i+1, p.Name, p.Email)
			if p.IsOrganizer {
				_, _ = fmt.Fprintf(w, " %s", common.Green.Sprint("(Organizer)"))
			}
			_, _ = fmt.Fprintln(w)
		}
	}

	_, _ = fmt.Fprintf(w, "\nEvent Booking:\n")
	_, _ = fmt.Fprintf(w, "  Title: %s\n", config.EventBooking.Title)
	if config.EventBooking.Description != "" {
		_, _ = fmt.Fprintf(w, "  Description: %s\n", config.EventBooking.Description)
	}
	if config.EventBooking.Location != "" {
		_, _ = fmt.Fprintf(w, "  Location: %s\n", config.EventBooking.Location)
	}
	if config.EventBooking.Timezone != "" {
		_, _ = fmt.Fprintf(w, "  Timezone: %s\n", config.EventBooking.Timezone)
	}
	if config.EventBooking.BookingType != "" {
		_, _ = fmt.Fprintf(w, "  Booking Type: %s\n", config.EventBooking.BookingType)
	}
	if config.EventBooking.Conferencing != nil {
		label := config.EventBooking.Conferencing.Provider
		if config.EventBooking.Conferencing.Autocreate {
			label += " (autocreate)"
		}
		_, _ = fmt.Fprintf(w, "  Conferencing: %s\n", label)
	}
	if config.EventBooking.DisableEmails {
		_, _ = fmt.Fprintf(w, "  Emails: disabled\n")
	}
	if len(config.EventBooking.ReminderMinutes) > 0 {
		parts := make([]string, len(config.EventBooking.ReminderMinutes))
		for i, m := range config.EventBooking.ReminderMinutes {
			parts[i] = fmt.Sprintf("%d", m)
		}
		_, _ = fmt.Fprintf(w, "  Reminders: %s minutes\n", strings.Join(parts, ", "))
	}

	s := config.Scheduler
	if s.AvailableDaysInFuture > 0 || s.MinBookingNotice > 0 || s.MinCancellationNotice > 0 ||
		s.ConfirmationMethod != "" || s.CancellationPolicy != "" {
		_, _ = fmt.Fprintf(w, "\nScheduler Settings:\n")
		if s.AvailableDaysInFuture > 0 {
			_, _ = fmt.Fprintf(w, "  Available Days: %d\n", s.AvailableDaysInFuture)
		}
		if s.MinBookingNotice > 0 {
			_, _ = fmt.Fprintf(w, "  Min Booking Notice: %d minutes\n", s.MinBookingNotice)
		}
		if s.MinCancellationNotice > 0 {
			_, _ = fmt.Fprintf(w, "  Min Cancellation Notice: %d minutes\n", s.MinCancellationNotice)
		}
		if s.ConfirmationMethod != "" {
			_, _ = fmt.Fprintf(w, "  Confirmation: %s\n", s.ConfirmationMethod)
		}
		if s.CancellationPolicy != "" {
			_, _ = fmt.Fprintf(w, "  Cancellation Policy: %s\n", s.CancellationPolicy)
		}
	}

	if a := config.AppearanceSettings; a != nil {
		if a.CompanyName != "" || a.Color != "" || a.SubmitText != "" || a.ThankYouMessage != "" {
			_, _ = fmt.Fprintf(w, "\nAppearance:\n")
			if a.CompanyName != "" {
				_, _ = fmt.Fprintf(w, "  Company: %s\n", a.CompanyName)
			}
			if a.Color != "" {
				_, _ = fmt.Fprintf(w, "  Color: %s\n", a.Color)
			}
			if a.SubmitText != "" {
				_, _ = fmt.Fprintf(w, "  Submit Text: %s\n", a.SubmitText)
			}
			if a.ThankYouMessage != "" {
				_, _ = fmt.Fprintf(w, "  Thank You: %s\n", a.ThankYouMessage)
			}
		}
	}
}
