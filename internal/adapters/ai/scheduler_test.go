//go:build !integration

package ai

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAIScheduler(t *testing.T) {
	t.Parallel()

	scheduler := NewAIScheduler(nil, nil, "test-provider")

	require.NotNil(t, scheduler)
	assert.Nil(t, scheduler.router)
	assert.Nil(t, scheduler.nylasClient)
	assert.Equal(t, "test-provider", scheduler.providerName)
}

func TestNewAIScheduler_EmptyProvider(t *testing.T) {
	t.Parallel()

	scheduler := NewAIScheduler(nil, nil, "")

	require.NotNil(t, scheduler)
	assert.Equal(t, "", scheduler.providerName)
}

func TestScheduleRequest_Fields(t *testing.T) {
	t.Parallel()

	req := ScheduleRequest{
		Query:        "Schedule a meeting with John tomorrow at 2pm",
		GrantID:      "grant-123",
		UserTimezone: "America/New_York",
		MaxOptions:   3,
	}

	assert.Equal(t, "Schedule a meeting with John tomorrow at 2pm", req.Query)
	assert.Equal(t, "grant-123", req.GrantID)
	assert.Equal(t, "America/New_York", req.UserTimezone)
	assert.Equal(t, 3, req.MaxOptions)
}

func TestScheduleOption_Fields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	option := ScheduleOption{
		Rank:      1,
		Score:     95,
		StartTime: now,
		EndTime:   now.Add(time.Hour),
		Timezone:  "America/Los_Angeles",
		Reasoning: "Best overlap for all participants",
		Warnings:  []string{"Close to end of day for EU participants"},
		Participants: map[string]ParticipantTime{
			"john@example.com": {
				Email:     "john@example.com",
				Timezone:  "America/New_York",
				LocalTime: now,
				TimeDesc:  "9:00 AM - 10:00 AM EST",
				Notes:     "Morning",
			},
		},
	}

	assert.Equal(t, 1, option.Rank)
	assert.Equal(t, 95, option.Score)
	assert.Equal(t, "America/Los_Angeles", option.Timezone)
	assert.Equal(t, "Best overlap for all participants", option.Reasoning)
	assert.Len(t, option.Warnings, 1)
	assert.Contains(t, option.Participants, "john@example.com")
}

func TestParticipantTime_Fields(t *testing.T) {
	t.Parallel()

	pt := ParticipantTime{
		Email:     "alice@example.com",
		Timezone:  "Europe/London",
		LocalTime: time.Now(),
		TimeDesc:  "2:00 PM - 3:00 PM GMT",
		Notes:     "Afternoon",
	}

	assert.Equal(t, "alice@example.com", pt.Email)
	assert.Equal(t, "Europe/London", pt.Timezone)
	assert.Equal(t, "2:00 PM - 3:00 PM GMT", pt.TimeDesc)
	assert.Equal(t, "Afternoon", pt.Notes)
}

func TestScheduleResponse_Fields(t *testing.T) {
	t.Parallel()

	resp := ScheduleResponse{
		Options: []ScheduleOption{
			{Rank: 1, Score: 90},
			{Rank: 2, Score: 85},
		},
		Analysis:     "Based on availability, I found 2 optimal times.",
		ProviderUsed: "claude",
		TokensUsed:   500,
	}

	assert.Len(t, resp.Options, 2)
	assert.Equal(t, "Based on availability, I found 2 optimal times.", resp.Analysis)
	assert.Equal(t, "claude", resp.ProviderUsed)
	assert.Equal(t, 500, resp.TokensUsed)
}

func TestBuildSystemPrompt(t *testing.T) {
	t.Parallel()

	scheduler := NewAIScheduler(nil, nil, "")
	req := &ScheduleRequest{
		UserTimezone: "America/Chicago",
	}

	prompt := scheduler.buildSystemPrompt(req)

	// Check that the prompt contains key elements
	assert.Contains(t, prompt, "expert AI scheduling assistant")
	assert.Contains(t, prompt, "America/Chicago")
	assert.Contains(t, prompt, "IANA timezone IDs")
	assert.Contains(t, prompt, "DST transitions")
	assert.Contains(t, prompt, "rank (1-3)")
	assert.Contains(t, prompt, "score (0-100)")
}

func TestBuildUserQuery(t *testing.T) {
	t.Parallel()

	scheduler := NewAIScheduler(nil, nil, "")
	req := &ScheduleRequest{
		Query: "Schedule a 30-min sync with the team next Tuesday",
	}

	query := scheduler.buildUserQuery(req)

	assert.Contains(t, query, "Schedule a 30-min sync with the team next Tuesday")
	assert.Contains(t, query, "top 3 recommended meeting times")
}

func TestParseScheduleOptions_ValidJSON(t *testing.T) {
	t.Parallel()

	scheduler := NewAIScheduler(nil, nil, "")

	now := time.Now()
	testOptions := []ScheduleOption{
		{
			Rank:      1,
			Score:     90,
			StartTime: now,
			EndTime:   now.Add(time.Hour),
			Timezone:  "America/New_York",
			Reasoning: "Good for all participants",
		},
	}

	optionsJSON, err := json.Marshal(testOptions)
	require.NoError(t, err)

	content := "Here are my suggestions:\n" + string(optionsJSON) + "\nLet me know if you need changes."

	options, err := scheduler.parseScheduleOptions(content)
	require.NoError(t, err)
	require.Len(t, options, 1)
	assert.Equal(t, 1, options[0].Rank)
	assert.Equal(t, 90, options[0].Score)
}

func TestParseScheduleOptions_NoJSON(t *testing.T) {
	t.Parallel()

	scheduler := NewAIScheduler(nil, nil, "")

	content := "I suggest meeting tomorrow at 2pm. It would be a good time for everyone."

	options, err := scheduler.parseScheduleOptions(content)
	require.NoError(t, err)
	// Should return fallback options
	require.Len(t, options, 1)
	assert.Equal(t, 1, options[0].Rank)
	assert.Equal(t, 85, options[0].Score)
}

func TestParseScheduleOptions_InvalidJSON(t *testing.T) {
	t.Parallel()

	scheduler := NewAIScheduler(nil, nil, "")

	content := "Here are the options: [{invalid json}]"

	options, err := scheduler.parseScheduleOptions(content)
	require.NoError(t, err)
	// Should return fallback options
	require.Len(t, options, 1)
}

func TestCreateFallbackOptions(t *testing.T) {
	t.Parallel()

	scheduler := NewAIScheduler(nil, nil, "")

	options := scheduler.createFallbackOptions()

	require.Len(t, options, 1)
	assert.Equal(t, 1, options[0].Rank)
	assert.Equal(t, 85, options[0].Score)
	assert.Contains(t, options[0].Reasoning, "Tomorrow afternoon")
	assert.Empty(t, options[0].Warnings)

	// Verify time is tomorrow at 2 PM
	assert.Equal(t, 14, options[0].StartTime.Hour())

	// Verify duration is 30 minutes
	duration := options[0].EndTime.Sub(options[0].StartTime)
	assert.Equal(t, 30*time.Minute, duration)
}

func TestExecuteTool_FindMeetingTime(t *testing.T) {
	t.Parallel()

	client := nylas.NewMockClient()
	client.GetAvailabilityFunc = func(ctx context.Context, req *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error) {
		return &domain.AvailabilityResponse{
			Data: domain.AvailabilityData{
				TimeSlots: []domain.AvailableSlot{
					{
						StartTime: time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC).Unix(),
						EndTime:   time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC).Unix(),
						Emails:    []string{"alice@example.com"},
					},
				},
			},
		}, nil
	}
	scheduler := NewAIScheduler(nil, client, "")
	req := &ScheduleRequest{GrantID: "grant-123", UserTimezone: "UTC"}

	result, err := scheduler.executeTool(context.Background(), "findMeetingTime", map[string]any{
		"participants": []string{"alice@example.com"},
		"duration":     30,
		"dateRange": map[string]any{
			"start": "2024-01-15",
			"end":   "2024-01-15",
		},
	}, req)

	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &payload))
	assert.Equal(t, "success", payload["status"])
	slots, ok := payload["slots"].([]any)
	require.True(t, ok)
	require.Len(t, slots, 1)
}

func TestExecuteTool_CheckDST(t *testing.T) {
	t.Parallel()

	scheduler := NewAIScheduler(nil, nil, "")

	result, err := scheduler.executeTool(context.Background(), "checkDST", map[string]any{
		"time":     "2024-03-10T02:30:00",
		"timezone": "America/New_York",
	}, nil)

	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &payload))
	assert.Equal(t, "success", payload["status"])
	assert.Equal(t, "America/New_York", payload["timezone"])
	_, hasDST := payload["isDST"]
	assert.True(t, hasDST)
}

func TestExecuteTool_ValidateWorkingHours(t *testing.T) {
	t.Parallel()

	scheduler := NewAIScheduler(nil, nil, "")

	result, err := scheduler.executeTool(context.Background(), "validateWorkingHours", map[string]any{
		"time":     "2024-01-15T19:00:00",
		"timezone": "America/New_York",
	}, nil)

	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &payload))
	assert.Equal(t, "success", payload["status"])
	assert.Equal(t, false, payload["isValid"])
}

func TestExecuteTool_CreateEvent(t *testing.T) {
	t.Parallel()

	client := nylas.NewMockClient()
	client.GetCalendarsFunc = func(ctx context.Context, grantID string) ([]domain.Calendar, error) {
		return []domain.Calendar{{ID: "primary", IsPrimary: true}}, nil
	}
	client.CreateEventFunc = func(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error) {
		assert.Equal(t, "grant-123", grantID)
		assert.Equal(t, "primary", calendarID)
		assert.Equal(t, "Team Standup", req.Title)
		assert.Equal(t, "UTC", req.When.StartTimezone)
		return &domain.Event{ID: "event-123", CalendarID: calendarID, Title: req.Title}, nil
	}
	scheduler := NewAIScheduler(nil, client, "")
	req := &ScheduleRequest{GrantID: "grant-123", UserTimezone: "UTC"}

	result, err := scheduler.executeTool(context.Background(), "createEvent", map[string]any{
		"title":     "Team Standup",
		"startTime": "2024-01-15T10:00:00Z",
		"endTime":   "2024-01-15T10:30:00Z",
		"timezone":  "UTC",
	}, req)

	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &payload))
	assert.Equal(t, "success", payload["status"])
	assert.Equal(t, "event-123", payload["eventID"])
	assert.Equal(t, "Team Standup", payload["title"])
}

func TestExecuteTool_GetAvailability(t *testing.T) {
	t.Parallel()

	client := nylas.NewMockClient()
	client.GetAvailabilityFunc = func(ctx context.Context, req *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error) {
		return &domain.AvailabilityResponse{
			Data: domain.AvailabilityData{
				TimeSlots: []domain.AvailableSlot{
					{
						StartTime: time.Date(2024, 1, 15, 16, 0, 0, 0, time.UTC).Unix(),
						EndTime:   time.Date(2024, 1, 15, 16, 30, 0, 0, time.UTC).Unix(),
						Emails:    []string{"alice@example.com"},
					},
				},
			},
		}, nil
	}
	scheduler := NewAIScheduler(nil, client, "")
	req := &ScheduleRequest{GrantID: "grant-123", UserTimezone: "UTC"}

	result, err := scheduler.executeTool(context.Background(), "getAvailability", map[string]any{
		"participants": []string{"alice@example.com"},
		"startTime":    "2024-01-15T00:00:00Z",
		"endTime":      "2024-01-15T23:59:59Z",
		"duration":     30,
	}, req)

	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &payload))
	assert.Equal(t, "success", payload["status"])
	slots, ok := payload["availableSlots"].([]any)
	require.True(t, ok)
	require.Len(t, slots, 1)
}

func TestExecuteTool_GetTimezoneInfo(t *testing.T) {
	t.Parallel()

	scheduler := NewAIScheduler(nil, nil, "")

	result, err := scheduler.executeTool(context.Background(), "getTimezoneInfo", map[string]any{
		"email":    "alice@example.com",
		"timezone": "Europe/London",
	}, &ScheduleRequest{UserTimezone: "UTC"})

	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &payload))
	assert.Equal(t, "success", payload["status"])
	assert.Equal(t, "Europe/London", payload["timezone"])
	assert.Equal(t, "alice@example.com", payload["email"])
}

func TestExecuteTool_UnknownFunction(t *testing.T) {
	t.Parallel()

	scheduler := NewAIScheduler(nil, nil, "")

	result, err := scheduler.executeTool(context.Background(), "unknownFunction", nil, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown function")
	assert.Empty(t, result)
}

func TestExecuteToolCalls(t *testing.T) {
	t.Parallel()

	scheduler := NewAIScheduler(nil, nil, "")
	req := &ScheduleRequest{GrantID: "grant-123"}

	toolCalls := []struct {
		Function  string
		Arguments map[string]any
	}{
		{
			Function: "checkDST",
			Arguments: map[string]any{
				"time":     "2024-03-10T02:30:00",
				"timezone": "America/New_York",
			},
		},
		{
			Function:  "validateWorkingHours",
			Arguments: map[string]any{"time": "2024-01-15T10:00:00", "timezone": "UTC"},
		},
	}

	// Test individual tool execution
	for _, tc := range toolCalls {
		result, err := scheduler.executeTool(context.Background(), tc.Function, tc.Arguments, req)
		require.NoError(t, err)
		assert.Contains(t, result, "success")
	}
}

func TestExecuteToolCalls_WithError(t *testing.T) {
	t.Parallel()

	scheduler := NewAIScheduler(nil, nil, "")

	// Test that unknown function error is captured in result
	result, err := scheduler.executeTool(context.Background(), "invalidTool", nil, nil)
	require.Error(t, err)
	assert.Empty(t, result)
}

func TestScheduleOption_JSONSerialization(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)
	option := ScheduleOption{
		Rank:      1,
		Score:     95,
		StartTime: now,
		EndTime:   now.Add(time.Hour),
		Timezone:  "America/Los_Angeles",
		Reasoning: "Best for everyone",
		Warnings:  []string{"DST transition nearby"},
		Participants: map[string]ParticipantTime{
			"test@example.com": {
				Email:    "test@example.com",
				Timezone: "UTC",
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(option)
	require.NoError(t, err)

	// Unmarshal back
	var decoded ScheduleOption
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, option.Rank, decoded.Rank)
	assert.Equal(t, option.Score, decoded.Score)
	assert.Equal(t, option.Timezone, decoded.Timezone)
	assert.Equal(t, option.Reasoning, decoded.Reasoning)
	assert.Equal(t, option.Warnings, decoded.Warnings)
	assert.Contains(t, decoded.Participants, "test@example.com")
}

func TestScheduleResponse_JSONSerialization(t *testing.T) {
	t.Parallel()

	resp := ScheduleResponse{
		Options: []ScheduleOption{
			{Rank: 1, Score: 90},
		},
		Analysis:     "Analysis text",
		ProviderUsed: "openai",
		TokensUsed:   100,
	}

	// Marshal to JSON
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	// Unmarshal back
	var decoded ScheduleResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Options, 1)
	assert.Equal(t, resp.Analysis, decoded.Analysis)
	assert.Equal(t, resp.ProviderUsed, decoded.ProviderUsed)
	assert.Equal(t, resp.TokensUsed, decoded.TokensUsed)
}
