//go:build !integration

package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSchedulingTools(t *testing.T) {
	t.Parallel()

	tools := GetSchedulingTools()

	require.NotEmpty(t, tools)
	assert.Len(t, tools, 6, "Should have 6 scheduling tools")

	// Verify expected tool names
	expectedNames := map[string]bool{
		"findMeetingTime":      false,
		"checkDST":             false,
		"validateWorkingHours": false,
		"createEvent":          false,
		"getAvailability":      false,
		"getTimezoneInfo":      false,
	}

	for _, tool := range tools {
		assert.NotEmpty(t, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.Parameters)

		if _, exists := expectedNames[tool.Name]; exists {
			expectedNames[tool.Name] = true
		}
	}

	// Verify all expected tools were found
	for name, found := range expectedNames {
		assert.True(t, found, "Expected tool %s not found", name)
	}
}

func TestGetSchedulingTools_FindMeetingTime(t *testing.T) {
	t.Parallel()

	tools := GetSchedulingTools()

	var findMeetingTime *struct {
		Name        string
		Description string
		Parameters  map[string]any
	}

	for i := range tools {
		if tools[i].Name == "findMeetingTime" {
			findMeetingTime = &struct {
				Name        string
				Description string
				Parameters  map[string]any
			}{
				Name:        tools[i].Name,
				Description: tools[i].Description,
				Parameters:  tools[i].Parameters,
			}
			break
		}
	}

	require.NotNil(t, findMeetingTime)

	// Check description
	assert.Contains(t, findMeetingTime.Description, "optimal meeting times")
	assert.Contains(t, findMeetingTime.Description, "timezones")

	// Check parameters structure
	params := findMeetingTime.Parameters
	assert.Equal(t, "object", params["type"])

	properties, ok := params["properties"].(map[string]any)
	require.True(t, ok)

	// Check participants parameter
	participants, ok := properties["participants"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "array", participants["type"])

	// Check duration parameter
	duration, ok := properties["duration"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "integer", duration["type"])

	// Check required fields
	required, ok := params["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "participants")
	assert.Contains(t, required, "duration")
	assert.Contains(t, required, "dateRange")
}

func TestGetSchedulingTools_CheckDST(t *testing.T) {
	t.Parallel()

	tools := GetSchedulingTools()

	var checkDST *struct {
		Name        string
		Description string
		Parameters  map[string]any
	}

	for i := range tools {
		if tools[i].Name == "checkDST" {
			checkDST = &struct {
				Name        string
				Description string
				Parameters  map[string]any
			}{
				Name:        tools[i].Name,
				Description: tools[i].Description,
				Parameters:  tools[i].Parameters,
			}
			break
		}
	}

	require.NotNil(t, checkDST)

	// Check description mentions DST
	assert.Contains(t, checkDST.Description, "DST")

	// Check parameters
	params := checkDST.Parameters
	properties, ok := params["properties"].(map[string]any)
	require.True(t, ok)

	// Check time parameter
	timeParam, ok := properties["time"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", timeParam["type"])

	// Check timezone parameter
	tzParam, ok := properties["timezone"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", tzParam["type"])
	assert.Contains(t, tzParam["description"].(string), "IANA")
}

func TestGetSchedulingTools_ValidateWorkingHours(t *testing.T) {
	t.Parallel()

	tools := GetSchedulingTools()

	var validateWorkingHours *struct {
		Name        string
		Description string
		Parameters  map[string]any
	}

	for i := range tools {
		if tools[i].Name == "validateWorkingHours" {
			validateWorkingHours = &struct {
				Name        string
				Description string
				Parameters  map[string]any
			}{
				Name:        tools[i].Name,
				Description: tools[i].Description,
				Parameters:  tools[i].Parameters,
			}
			break
		}
	}

	require.NotNil(t, validateWorkingHours)

	// Check description
	assert.Contains(t, validateWorkingHours.Description, "working hours")

	// Check parameters include workStart and workEnd defaults
	params := validateWorkingHours.Parameters
	properties, ok := params["properties"].(map[string]any)
	require.True(t, ok)

	workStart, ok := properties["workStart"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "09:00", workStart["default"])

	workEnd, ok := properties["workEnd"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "17:00", workEnd["default"])
}

func TestGetSchedulingTools_CreateEvent(t *testing.T) {
	t.Parallel()

	tools := GetSchedulingTools()

	var createEvent *struct {
		Name        string
		Description string
		Parameters  map[string]any
	}

	for i := range tools {
		if tools[i].Name == "createEvent" {
			createEvent = &struct {
				Name        string
				Description string
				Parameters  map[string]any
			}{
				Name:        tools[i].Name,
				Description: tools[i].Description,
				Parameters:  tools[i].Parameters,
			}
			break
		}
	}

	require.NotNil(t, createEvent)

	// Check description
	assert.Contains(t, createEvent.Description, "calendar event")

	// Check required fields
	required, ok := createEvent.Parameters["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "title")
	assert.Contains(t, required, "startTime")
	assert.Contains(t, required, "endTime")
	assert.Contains(t, required, "timezone")
}

func TestGetSchedulingTools_GetAvailability(t *testing.T) {
	t.Parallel()

	tools := GetSchedulingTools()

	var getAvailability *struct {
		Name        string
		Description string
		Parameters  map[string]any
	}

	for i := range tools {
		if tools[i].Name == "getAvailability" {
			getAvailability = &struct {
				Name        string
				Description string
				Parameters  map[string]any
			}{
				Name:        tools[i].Name,
				Description: tools[i].Description,
				Parameters:  tools[i].Parameters,
			}
			break
		}
	}

	require.NotNil(t, getAvailability)

	// Check description
	assert.Contains(t, getAvailability.Description, "free/busy")
}

func TestGetSchedulingTools_GetTimezoneInfo(t *testing.T) {
	t.Parallel()

	tools := GetSchedulingTools()

	var getTimezoneInfo *struct {
		Name        string
		Description string
		Parameters  map[string]any
	}

	for i := range tools {
		if tools[i].Name == "getTimezoneInfo" {
			getTimezoneInfo = &struct {
				Name        string
				Description string
				Parameters  map[string]any
			}{
				Name:        tools[i].Name,
				Description: tools[i].Description,
				Parameters:  tools[i].Parameters,
			}
			break
		}
	}

	require.NotNil(t, getTimezoneInfo)

	// Check description
	assert.Contains(t, getTimezoneInfo.Description, "timezone information")

	// Check required only includes email
	required, ok := getTimezoneInfo.Parameters["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "email")
	assert.Len(t, required, 1)
}

func TestToolParameterTypes(t *testing.T) {
	t.Parallel()

	tools := GetSchedulingTools()

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			// All parameters should have type "object"
			assert.Equal(t, "object", tool.Parameters["type"])

			// All should have properties
			properties, ok := tool.Parameters["properties"].(map[string]any)
			assert.True(t, ok, "Tool %s should have properties", tool.Name)
			assert.NotEmpty(t, properties, "Tool %s should have at least one property", tool.Name)

			// All should have required array
			_, ok = tool.Parameters["required"].([]string)
			assert.True(t, ok, "Tool %s should have required array", tool.Name)
		})
	}
}
