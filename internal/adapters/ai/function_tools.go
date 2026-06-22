package ai

import (
	"github.com/nylas/cli/internal/domain"
)

// GetSchedulingTools returns the function calling tools available for AI scheduling.
func GetSchedulingTools() []domain.Tool {
	return []domain.Tool{
		{
			Name:        "findMeetingTime",
			Description: "Find optimal meeting times across multiple timezones. Returns ranked time slots with scores based on timezone overlap, working hours, and participant availability.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"participants": map[string]any{
						"type":        "array",
						"description": "Array of participant email addresses",
						"items": map[string]any{
							"type": "string",
						},
					},
					"duration": map[string]any{
						"type":        "integer",
						"description": "Meeting duration in minutes",
					},
					"dateRange": map[string]any{
						"type":        "object",
						"description": "Date range to search for meeting times",
						"properties": map[string]any{
							"start": map[string]any{
								"type":        "string",
								"description": "Start date (YYYY-MM-DD)",
							},
							"end": map[string]any{
								"type":        "string",
								"description": "End date (YYYY-MM-DD)",
							},
						},
						"required": []string{"start", "end"},
					},
					"workingHoursOnly": map[string]any{
						"type":        "boolean",
						"description": "Only consider working hours (9 AM - 5 PM)",
						"default":     true,
					},
				},
				"required": []string{"participants", "duration", "dateRange"},
			},
		},
		{
			Name:        "checkDST",
			Description: "Check if a specific time falls during a DST transition. Returns warnings for ambiguous times (fall back) or invalid times (spring forward).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"time": map[string]any{
						"type":        "string",
						"description": "ISO 8601 datetime (e.g., 2025-03-09T02:30:00-08:00)",
					},
					"timezone": map[string]any{
						"type":        "string",
						"description": "IANA timezone ID (e.g., America/Los_Angeles)",
					},
				},
				"required": []string{"time", "timezone"},
			},
		},
		{
			Name:        "validateWorkingHours",
			Description: "Check if a proposed meeting time falls within working hours for all participants. Returns validation status and any violations.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"time": map[string]any{
						"type":        "string",
						"description": "ISO 8601 datetime",
					},
					"timezone": map[string]any{
						"type":        "string",
						"description": "IANA timezone ID",
					},
					"workStart": map[string]any{
						"type":        "string",
						"description": "Working hours start time (HH:MM format, e.g., 09:00)",
						"default":     "09:00",
					},
					"workEnd": map[string]any{
						"type":        "string",
						"description": "Working hours end time (HH:MM format, e.g., 17:00)",
						"default":     "17:00",
					},
				},
				"required": []string{"time", "timezone"},
			},
		},
		{
			Name:        "createEvent",
			Description: "Create a calendar event with the specified details. This actually creates the event in the user's calendar.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{
						"type":        "string",
						"description": "Event title/subject",
					},
					"startTime": map[string]any{
						"type":        "string",
						"description": "Event start time (ISO 8601 datetime)",
					},
					"endTime": map[string]any{
						"type":        "string",
						"description": "Event end time (ISO 8601 datetime)",
					},
					"participants": map[string]any{
						"type":        "array",
						"description": "Array of participant email addresses",
						"items": map[string]any{
							"type": "string",
						},
					},
					"timezone": map[string]any{
						"type":        "string",
						"description": "IANA timezone ID for the event",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "Optional event description/notes",
					},
				},
				"required": []string{"title", "startTime", "endTime", "timezone"},
			},
		},
		{
			Name:        "getAvailability",
			Description: "Get free/busy information for participants within a specified time range. Returns available time slots.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"participants": map[string]any{
						"type":        "array",
						"description": "Array of participant email addresses",
						"items": map[string]any{
							"type": "string",
						},
					},
					"startTime": map[string]any{
						"type":        "string",
						"description": "Start of availability check period (ISO 8601 datetime)",
					},
					"endTime": map[string]any{
						"type":        "string",
						"description": "End of availability check period (ISO 8601 datetime)",
					},
				},
				"required": []string{"participants", "startTime", "endTime"},
			},
		},
		{
			Name:        "getTimezoneInfo",
			Description: "Get timezone information for a participant, including current offset, DST status, and typical working hours.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"email": map[string]any{
						"type":        "string",
						"description": "Participant email address",
					},
					"timezone": map[string]any{
						"type":        "string",
						"description": "IANA timezone ID (optional, will auto-detect if not provided)",
					},
				},
				"required": []string{"email"},
			},
		},
	}
}
