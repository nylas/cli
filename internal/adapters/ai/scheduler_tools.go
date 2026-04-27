package ai

import (
	"context"
	"fmt"
	"time"

	tzutil "github.com/nylas/cli/internal/adapters/utilities/timezone"
	"github.com/nylas/cli/internal/domain"
)

func (s *AIScheduler) runFindMeetingTime(ctx context.Context, args map[string]any, req *ScheduleRequest) (string, error) {
	if s.nylasClient == nil {
		return "", fmt.Errorf("nylas client not configured")
	}

	participants, err := participantEmailsArg(args, "participants")
	if err != nil {
		return "", err
	}
	if len(participants) == 0 {
		return "", fmt.Errorf("participants are required")
	}

	durationMinutes, err := intArg(args, "duration", 30)
	if err != nil {
		return "", fmt.Errorf("invalid duration: %w", err)
	}
	if durationMinutes <= 0 {
		return "", fmt.Errorf("duration must be greater than zero")
	}

	loc, err := requestLocation(req)
	if err != nil {
		return "", err
	}
	searchStart, searchEnd, err := dateRangeArgs(args["dateRange"], loc)
	if err != nil {
		return "", err
	}

	availabilityReq := buildAvailabilityRequest(participants, searchStart, searchEnd, durationMinutes)
	result, err := s.nylasClient.GetAvailability(ctx, availabilityReq)
	if err != nil {
		return "", fmt.Errorf("failed to get availability: %w", err)
	}

	slots := rankAvailableSlots(result.Data.TimeSlots, loc)
	payload := map[string]any{
		"status":  "success",
		"message": fmt.Sprintf("Found %d available time slots", len(slots)),
		"slots":   slots,
	}
	return marshalToolResult(payload)
}

func (s *AIScheduler) runCheckDST(ctx context.Context, args map[string]any) (string, error) {
	timezoneID, err := stringArg(args, "timezone", "")
	if err != nil {
		return "", err
	}
	if timezoneID == "" {
		return "", fmt.Errorf("timezone is required")
	}

	loc, err := time.LoadLocation(timezoneID)
	if err != nil {
		return "", fmt.Errorf("invalid timezone %q: %w", timezoneID, err)
	}

	inputTime, err := timeArg(args, "time", loc)
	if err != nil {
		return "", err
	}

	service := tzutil.NewService()
	info, err := service.GetTimeZoneInfo(ctx, timezoneID, inputTime)
	if err != nil {
		return "", err
	}

	warning, err := service.CheckDSTWarning(ctx, inputTime, timezoneID, 7)
	if err != nil {
		return "", err
	}

	payload := map[string]any{
		"status":   "success",
		"time":     inputTime.Format(time.RFC3339),
		"timezone": timezoneID,
		"isDST":    info.IsDST,
		"warning":  "",
	}
	if warning != nil {
		payload["warning"] = warning.Warning
		payload["severity"] = warning.Severity
		payload["isNearTransition"] = warning.IsNearTransition
		payload["inTransitionGap"] = warning.InTransitionGap
		payload["inDuplicateHour"] = warning.InDuplicateHour
	}

	return marshalToolResult(payload)
}

func (s *AIScheduler) runValidateWorkingHours(_ context.Context, args map[string]any) (string, error) {
	timezoneID, err := stringArg(args, "timezone", "")
	if err != nil {
		return "", err
	}
	if timezoneID == "" {
		return "", fmt.Errorf("timezone is required")
	}

	loc, err := time.LoadLocation(timezoneID)
	if err != nil {
		return "", fmt.Errorf("invalid timezone %q: %w", timezoneID, err)
	}

	inputTime, err := timeArg(args, "time", loc)
	if err != nil {
		return "", err
	}

	workStart, err := stringArg(args, "workStart", "09:00")
	if err != nil {
		return "", err
	}
	workEnd, err := stringArg(args, "workEnd", "17:00")
	if err != nil {
		return "", err
	}

	startMinutes, err := clockMinutes(workStart)
	if err != nil {
		return "", fmt.Errorf("invalid workStart: %w", err)
	}
	endMinutes, err := clockMinutes(workEnd)
	if err != nil {
		return "", fmt.Errorf("invalid workEnd: %w", err)
	}

	localTime := inputTime.In(loc)
	currentMinutes := localTime.Hour()*60 + localTime.Minute()
	isValid := currentMinutes >= startMinutes && currentMinutes < endMinutes

	violations := make([]string, 0)
	if !isValid {
		violations = append(violations,
			fmt.Sprintf("%s falls outside working hours %s-%s in %s",
				localTime.Format("3:04 PM"),
				workStart,
				workEnd,
				timezoneID,
			),
		)
	}

	return marshalToolResult(map[string]any{
		"status":       "success",
		"isValid":      isValid,
		"violations":   violations,
		"localTime":    localTime.Format(time.RFC3339),
		"workingHours": fmt.Sprintf("%s-%s", workStart, workEnd),
	})
}

func (s *AIScheduler) runCreateEvent(ctx context.Context, args map[string]any, req *ScheduleRequest) (string, error) {
	if s.nylasClient == nil {
		return "", fmt.Errorf("nylas client not configured")
	}
	if req == nil || req.GrantID == "" {
		return "", fmt.Errorf("grant ID is required to create an event")
	}

	title, err := stringArg(args, "title", "")
	if err != nil {
		return "", err
	}
	if title == "" {
		return "", fmt.Errorf("title is required")
	}

	timezoneID, err := stringArg(args, "timezone", req.UserTimezone)
	if err != nil {
		return "", err
	}
	if timezoneID == "" {
		timezoneID = "UTC"
	}

	loc, err := time.LoadLocation(timezoneID)
	if err != nil {
		return "", fmt.Errorf("invalid timezone %q: %w", timezoneID, err)
	}

	startTime, err := timeArg(args, "startTime", loc)
	if err != nil {
		return "", err
	}
	endTime, err := timeArg(args, "endTime", loc)
	if err != nil {
		return "", err
	}
	if !endTime.After(startTime) {
		return "", fmt.Errorf("end time must be after start time")
	}

	description, err := stringArg(args, "description", "")
	if err != nil {
		return "", err
	}

	participants, err := participantEmailsArg(args, "participants")
	if err != nil {
		return "", err
	}

	calendarID, err := s.defaultWritableCalendarID(ctx, req.GrantID)
	if err != nil {
		return "", err
	}

	createReq := &domain.CreateEventRequest{
		Title:       title,
		Description: description,
		Busy:        true,
		When: domain.EventWhen{
			StartTime:     startTime.Unix(),
			EndTime:       endTime.Unix(),
			StartTimezone: timezoneID,
			EndTimezone:   timezoneID,
			Object:        "timespan",
		},
	}

	for _, email := range participants {
		createReq.Participants = append(createReq.Participants, domain.Participant{
			Person: domain.Person{Email: email},
		})
	}

	event, err := s.nylasClient.CreateEvent(ctx, req.GrantID, calendarID, createReq)
	if err != nil {
		return "", fmt.Errorf("failed to create event: %w", err)
	}

	return marshalToolResult(map[string]any{
		"status":     "success",
		"eventID":    event.ID,
		"calendarID": calendarID,
		"title":      event.Title,
		"startTime":  startTime.Format(time.RFC3339),
		"endTime":    endTime.Format(time.RFC3339),
		"timezone":   timezoneID,
	})
}

func (s *AIScheduler) runGetAvailability(ctx context.Context, args map[string]any, req *ScheduleRequest) (string, error) {
	if s.nylasClient == nil {
		return "", fmt.Errorf("nylas client not configured")
	}

	participants, err := participantEmailsArg(args, "participants")
	if err != nil {
		return "", err
	}
	if len(participants) == 0 {
		return "", fmt.Errorf("participants are required")
	}

	loc, err := requestLocation(req)
	if err != nil {
		return "", err
	}
	startTime, err := timeArg(args, "startTime", loc)
	if err != nil {
		return "", err
	}
	endTime, err := timeArg(args, "endTime", loc)
	if err != nil {
		return "", err
	}
	if !endTime.After(startTime) {
		return "", fmt.Errorf("end time must be after start time")
	}

	durationMinutes, err := intArg(args, "duration", 30)
	if err != nil {
		return "", fmt.Errorf("invalid duration: %w", err)
	}

	availabilityReq := buildAvailabilityRequest(participants, startTime, endTime, durationMinutes)
	result, err := s.nylasClient.GetAvailability(ctx, availabilityReq)
	if err != nil {
		return "", fmt.Errorf("failed to get availability: %w", err)
	}

	availableSlots := make([]map[string]any, 0, len(result.Data.TimeSlots))
	for _, slot := range result.Data.TimeSlots {
		availableSlots = append(availableSlots, map[string]any{
			"start":  time.Unix(slot.StartTime, 0).UTC().Format(time.RFC3339),
			"end":    time.Unix(slot.EndTime, 0).UTC().Format(time.RFC3339),
			"emails": slot.Emails,
		})
	}

	return marshalToolResult(map[string]any{
		"status":         "success",
		"availableSlots": availableSlots,
		"count":          len(availableSlots),
	})
}

func (s *AIScheduler) runGetTimezoneInfo(ctx context.Context, args map[string]any, req *ScheduleRequest) (string, error) {
	email, err := stringArg(args, "email", "")
	if err != nil {
		return "", err
	}
	if email == "" {
		return "", fmt.Errorf("email is required")
	}

	timezoneID, err := stringArg(args, "timezone", "")
	if err != nil {
		return "", err
	}
	source := "explicit"
	if timezoneID == "" && req != nil && req.UserTimezone != "" {
		timezoneID = req.UserTimezone
		source = "request"
	}
	if timezoneID == "" {
		timezoneID = "UTC"
		source = "fallback"
	}

	service := tzutil.NewService()
	info, err := service.GetTimeZoneInfo(ctx, timezoneID, time.Now())
	if err != nil {
		return "", err
	}

	payload := map[string]any{
		"status":   "success",
		"email":    email,
		"timezone": timezoneID,
		"offset":   formatOffset(info.Offset),
		"isDST":    info.IsDST,
		"source":   source,
	}
	if source == "fallback" {
		payload["warning"] = "timezone not provided; using UTC fallback"
	}

	return marshalToolResult(payload)
}
