package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
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

	searchStart, searchEnd, err := dateRangeArgs(args["dateRange"], requestLocation(req))
	if err != nil {
		return "", err
	}

	availabilityReq := buildAvailabilityRequest(participants, searchStart, searchEnd, durationMinutes)
	result, err := s.nylasClient.GetAvailability(ctx, availabilityReq)
	if err != nil {
		return "", fmt.Errorf("failed to get availability: %w", err)
	}

	slots := rankAvailableSlots(result.Data.TimeSlots, requestLocation(req))
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

func (s *AIScheduler) runValidateWorkingHours(ctx context.Context, args map[string]any) (string, error) {
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

	loc := requestLocation(req)
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

func buildAvailabilityRequest(participants []string, startTime, endTime time.Time, durationMinutes int) *domain.AvailabilityRequest {
	availParticipants := make([]domain.AvailabilityParticipant, 0, len(participants))
	for _, email := range participants {
		availParticipants = append(availParticipants, domain.AvailabilityParticipant{
			Email: email,
		})
	}

	return &domain.AvailabilityRequest{
		StartTime:       startTime.Unix(),
		EndTime:         endTime.Unix(),
		DurationMinutes: durationMinutes,
		Participants:    availParticipants,
		IntervalMinutes: 30,
	}
}

func rankAvailableSlots(slots []domain.AvailableSlot, loc *time.Location) []map[string]any {
	type rankedSlot struct {
		slot  domain.AvailableSlot
		score int
	}

	ranked := make([]rankedSlot, 0, len(slots))
	for _, slot := range slots {
		start := time.Unix(slot.StartTime, 0).In(loc)
		ranked = append(ranked, rankedSlot{
			slot:  slot,
			score: localTimeScore(start),
		})
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].slot.StartTime < ranked[j].slot.StartTime
		}
		return ranked[i].score > ranked[j].score
	})

	limit := len(ranked)
	if limit > 10 {
		limit = 10
	}

	result := make([]map[string]any, 0, limit)
	for _, entry := range ranked[:limit] {
		result = append(result, map[string]any{
			"start":    time.Unix(entry.slot.StartTime, 0).UTC().Format(time.RFC3339),
			"end":      time.Unix(entry.slot.EndTime, 0).UTC().Format(time.RFC3339),
			"score":    entry.score,
			"emails":   entry.slot.Emails,
			"timezone": loc.String(),
		})
	}

	return result
}

func localTimeScore(start time.Time) int {
	localHour := float64(start.Hour()) + float64(start.Minute())/60
	distanceFromIdeal := math.Abs(localHour - 13)

	score := 100 - int(distanceFromIdeal*8)
	switch start.Weekday() {
	case time.Tuesday, time.Wednesday, time.Thursday:
		score += 5
	case time.Saturday, time.Sunday:
		score -= 25
	}

	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func (s *AIScheduler) defaultWritableCalendarID(ctx context.Context, grantID string) (string, error) {
	calendars, err := s.nylasClient.GetCalendars(ctx, grantID)
	if err != nil {
		return "", fmt.Errorf("failed to list calendars: %w", err)
	}
	if len(calendars) == 0 {
		return "", fmt.Errorf("no calendars available")
	}

	for _, cal := range calendars {
		if cal.IsPrimary && !cal.ReadOnly {
			return cal.ID, nil
		}
	}
	for _, cal := range calendars {
		if !cal.ReadOnly {
			return cal.ID, nil
		}
	}

	return "", fmt.Errorf("no writable calendar available")
}

func marshalToolResult(payload map[string]any) (string, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(bytes), nil
}

func requestLocation(req *ScheduleRequest) *time.Location {
	if req == nil || req.UserTimezone == "" {
		return time.UTC
	}

	loc, err := time.LoadLocation(req.UserTimezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

func dateRangeArgs(value any, loc *time.Location) (time.Time, time.Time, error) {
	rangeArgs, ok := value.(map[string]any)
	if !ok {
		return time.Time{}, time.Time{}, fmt.Errorf("dateRange must be an object")
	}

	startValue, ok := rangeArgs["start"]
	if !ok {
		return time.Time{}, time.Time{}, fmt.Errorf("dateRange.start is required")
	}
	endValue, ok := rangeArgs["end"]
	if !ok {
		return time.Time{}, time.Time{}, fmt.Errorf("dateRange.end is required")
	}

	startDate, err := time.ParseInLocation("2006-01-02", fmt.Sprint(startValue), loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid dateRange.start: %w", err)
	}
	endDate, err := time.ParseInLocation("2006-01-02", fmt.Sprint(endValue), loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid dateRange.end: %w", err)
	}

	endOfDay := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, loc)
	return startDate, endOfDay, nil
}

func participantEmailsArg(args map[string]any, key string) ([]string, error) {
	value, ok := args[key]
	if !ok {
		return nil, nil
	}

	switch typed := value.(type) {
	case []string:
		return cleanStrings(typed), nil
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			value, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%s entries must be strings", key)
			}
			if strings.TrimSpace(value) != "" {
				result = append(result, strings.TrimSpace(value))
			}
		}
		return result, nil
	case string:
		return cleanStrings(strings.Split(typed, ",")), nil
	default:
		return nil, fmt.Errorf("%s must be a string array", key)
	}
}

func stringArg(args map[string]any, key, fallback string) (string, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return fallback, nil
	}

	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return fallback, nil
		}
		return strings.TrimSpace(typed), nil
	default:
		return "", fmt.Errorf("%s must be a string", key)
	}
}

func intArg(args map[string]any, key string, fallback int) (int, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return fallback, nil
	}

	switch typed := value.(type) {
	case int:
		return typed, nil
	case int64:
		return int(typed), nil
	case float64:
		return int(typed), nil
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer", key)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("%s must be an integer", key)
	}
}

func timeArg(args map[string]any, key string, loc *time.Location) (time.Time, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return time.Time{}, fmt.Errorf("%s is required", key)
	}

	raw, ok := value.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("%s must be a string", key)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("%s is required", key)
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
	}

	for _, layout := range layouts {
		var (
			parsed time.Time
			err    error
		)

		if layout == time.RFC3339 {
			parsed, err = time.Parse(layout, raw)
		} else {
			parsed, err = time.ParseInLocation(layout, raw, loc)
		}
		if err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid %s: %q", key, raw)
}

func clockMinutes(value string) (int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("expected HH:MM")
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hour")
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid minute")
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, fmt.Errorf("hour must be 0-23 and minute must be 0-59")
	}

	return hour*60 + minute, nil
}

func formatOffset(seconds int) string {
	sign := "+"
	if seconds < 0 {
		sign = "-"
		seconds = -seconds
	}

	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	return fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
}

func cleanStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, strings.TrimSpace(value))
		}
	}
	return result
}
