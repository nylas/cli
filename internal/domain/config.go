package domain

import (
	"strings"
	"time"
)

// Timeout constants for consistent behavior across the application.
// Use these instead of hardcoding timeout values.
const (
	// TimeoutAPI is the default timeout for Nylas API calls (90s).
	TimeoutAPI = 90 * time.Second

	// TimeoutMCP is the timeout for MCP proxy operations (90s).
	// Allows time for tool execution and response processing.
	TimeoutMCP = 90 * time.Second

	// TimeoutHealthCheck is the timeout for health/connectivity checks (10s).
	TimeoutHealthCheck = 10 * time.Second

	// TimeoutOAuth is the timeout for OAuth authentication flows (5m).
	// OAuth requires user interaction in browser, so needs longer timeout.
	TimeoutOAuth = 5 * time.Minute

	// TimeoutBulkOperation is the timeout for bulk operations (10m).
	TimeoutBulkOperation = 10 * time.Minute

	// TimeoutQuickCheck is the timeout for quick checks like version checking (5s).
	TimeoutQuickCheck = 5 * time.Second

	// HTTP Server timeouts
	HTTPReadHeaderTimeout = 10 * time.Second  // Time to read request headers
	HTTPReadTimeout       = 30 * time.Second  // Time to read entire request
	HTTPWriteTimeout      = 30 * time.Second  // Time to write response
	HTTPIdleTimeout       = 120 * time.Second // Keep-alive connection idle timeout
)

// Config represents the application configuration.
// Note: Secrets (API key, client_id), grants, and default_grant are stored in system keyring, not here.
type Config struct {
	Region       string `yaml:"region"`
	CallbackPort int    `yaml:"callback_port"`

	// Working hours settings
	WorkingHours *WorkingHoursConfig `yaml:"working_hours,omitempty"`
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Region:       "us",
		CallbackPort: 8080,
	}
}

// ConfigStatus represents the current configuration status.
type ConfigStatus struct {
	IsConfigured    bool   `json:"configured"`
	Region          string `json:"region"`
	ClientID        string `json:"client_id,omitempty"`
	HasAPIKey       bool   `json:"has_api_key"`
	HasClientSecret bool   `json:"has_client_secret"`
	SecretStore     string `json:"secret_store"`
	ConfigPath      string `json:"config_path"`
	GrantCount      int    `json:"grant_count"`
	DefaultGrant    string `json:"default_grant,omitempty"`
}

// WorkingHoursConfig represents working hours configuration.
type WorkingHoursConfig struct {
	Default   *DaySchedule `yaml:"default,omitempty"`
	Monday    *DaySchedule `yaml:"monday,omitempty"`
	Tuesday   *DaySchedule `yaml:"tuesday,omitempty"`
	Wednesday *DaySchedule `yaml:"wednesday,omitempty"`
	Thursday  *DaySchedule `yaml:"thursday,omitempty"`
	Friday    *DaySchedule `yaml:"friday,omitempty"`
	Saturday  *DaySchedule `yaml:"saturday,omitempty"`
	Sunday    *DaySchedule `yaml:"sunday,omitempty"`
	Weekend   *DaySchedule `yaml:"weekend,omitempty"` // Applies to Sat/Sun if specific days not set
}

// DaySchedule represents working hours for a specific day.
type DaySchedule struct {
	Enabled bool         `yaml:"enabled"`          // Whether working hours apply
	Start   string       `yaml:"start,omitempty"`  // Start time (HH:MM format)
	End     string       `yaml:"end,omitempty"`    // End time (HH:MM format)
	Breaks  []BreakBlock `yaml:"breaks,omitempty"` // Break periods (lunch, coffee, etc.)
}

// BreakBlock represents a break period within working hours.
type BreakBlock struct {
	Name  string `yaml:"name"`           // Break name (e.g., "Lunch", "Coffee Break")
	Start string `yaml:"start"`          // Start time (HH:MM format)
	End   string `yaml:"end"`            // End time (HH:MM format)
	Type  string `yaml:"type,omitempty"` // Optional type: "lunch", "coffee", "custom"
}

// Validate checks that BreakBlock has valid time format and end is after start.
func (b BreakBlock) Validate() error {
	start, err := time.Parse("15:04", b.Start)
	if err != nil {
		return ErrInvalidInput
	}
	end, err := time.Parse("15:04", b.End)
	if err != nil {
		return ErrInvalidInput
	}
	if !end.After(start) {
		return ErrInvalidInput
	}
	return nil
}

// GetScheduleForDay returns the schedule for a given weekday.
// Checks day-specific, weekend, then default in order of precedence.
// Weekday is case-insensitive (e.g., "Monday", "monday", "MONDAY" all work).
func (w *WorkingHoursConfig) GetScheduleForDay(weekday string) *DaySchedule {
	if w == nil {
		return DefaultWorkingHours()
	}

	// Normalize weekday to lowercase for case-insensitive matching
	weekday = strings.ToLower(weekday)

	// Check day-specific schedule first
	var daySchedule *DaySchedule
	switch weekday {
	case "monday":
		daySchedule = w.Monday
	case "tuesday":
		daySchedule = w.Tuesday
	case "wednesday":
		daySchedule = w.Wednesday
	case "thursday":
		daySchedule = w.Thursday
	case "friday":
		daySchedule = w.Friday
	case "saturday":
		daySchedule = w.Saturday
	case "sunday":
		daySchedule = w.Sunday
	}

	if daySchedule != nil {
		return daySchedule
	}

	// Check weekend schedule for Sat/Sun
	if (weekday == "saturday" || weekday == "sunday") && w.Weekend != nil {
		return w.Weekend
	}

	// Fall back to default
	if w.Default != nil {
		return w.Default
	}

	return DefaultWorkingHours()
}

// DefaultWorkingHours returns standard 9-5 working hours.
func DefaultWorkingHours() *DaySchedule {
	return &DaySchedule{
		Enabled: true,
		Start:   "09:00",
		End:     "17:00",
	}
}
