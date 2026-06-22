//go:build !integration

package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.NotNil(t, cfg)
	assert.Equal(t, "us", cfg.Region)
	assert.Equal(t, 9007, cfg.CallbackPort)
	// API should be nil by default - region determines base URL
	assert.Nil(t, cfg.API)
}

func TestResolveBaseURL(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "defaults to US",
			cfg:  Config{Region: "us"},
			want: BaseURLUS,
		},
		{
			name: "EU region",
			cfg:  Config{Region: "eu"},
			want: BaseURLEU,
		},
		{
			name: "empty region defaults to US",
			cfg:  Config{},
			want: BaseURLUS,
		},
		{
			name: "custom base URL takes precedence over region",
			cfg: Config{
				Region: "eu",
				API:    &APIConfig{BaseURL: "https://api-staging.us.nylas.com"},
			},
			want: "https://api-staging.us.nylas.com",
		},
		{
			name: "nil API config uses region",
			cfg:  Config{Region: "eu", API: nil},
			want: BaseURLEU,
		},
		{
			name: "empty base URL uses region",
			cfg:  Config{Region: "eu", API: &APIConfig{BaseURL: ""}},
			want: BaseURLEU,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.cfg.ResolveBaseURL())
		})
	}
}

func TestResolveAPITimeout(t *testing.T) {
	t.Run("defaults to TimeoutAPI when unset", func(t *testing.T) {
		t.Setenv("NYLAS_API_TIMEOUT", "")
		assert.Equal(t, TimeoutAPI, (&Config{}).ResolveAPITimeout())
		assert.Equal(t, TimeoutAPI, (&Config{API: &APIConfig{}}).ResolveAPITimeout())
	})

	t.Run("config api.timeout overrides default", func(t *testing.T) {
		t.Setenv("NYLAS_API_TIMEOUT", "")
		cfg := &Config{API: &APIConfig{Timeout: "45s"}}
		assert.Equal(t, 45*time.Second, cfg.ResolveAPITimeout())
	})

	t.Run("env overrides config", func(t *testing.T) {
		t.Setenv("NYLAS_API_TIMEOUT", "3m")
		cfg := &Config{API: &APIConfig{Timeout: "45s"}}
		assert.Equal(t, 3*time.Minute, cfg.ResolveAPITimeout())
	})

	t.Run("invalid or non-positive values fall back to default", func(t *testing.T) {
		t.Setenv("NYLAS_API_TIMEOUT", "")
		for _, bad := range []string{"banana", "0s", "-5s"} {
			cfg := &Config{API: &APIConfig{Timeout: bad}}
			assert.Equal(t, TimeoutAPI, cfg.ResolveAPITimeout(), "value %q should fall back", bad)
		}
	})
}

func TestDefaultWorkingHours(t *testing.T) {
	schedule := DefaultWorkingHours()

	require.NotNil(t, schedule)
	assert.True(t, schedule.Enabled)
	assert.Equal(t, "09:00", schedule.Start)
	assert.Equal(t, "17:00", schedule.End)
	assert.Nil(t, schedule.Breaks)
}

func TestGetScheduleForDay(t *testing.T) {
	tests := []struct {
		name      string
		config    *WorkingHoursConfig
		weekday   string
		wantStart string
		wantEnd   string
		wantIsNil bool
	}{
		{
			name:      "nil config returns default",
			config:    nil,
			weekday:   "monday",
			wantStart: "09:00",
			wantEnd:   "17:00",
		},
		{
			name: "specific Monday schedule",
			config: &WorkingHoursConfig{
				Monday: &DaySchedule{
					Enabled: true,
					Start:   "08:00",
					End:     "16:00",
				},
			},
			weekday:   "monday",
			wantStart: "08:00",
			wantEnd:   "16:00",
		},
		{
			name: "Tuesday schedule",
			config: &WorkingHoursConfig{
				Tuesday: &DaySchedule{
					Enabled: true,
					Start:   "10:00",
					End:     "18:00",
				},
			},
			weekday:   "tuesday",
			wantStart: "10:00",
			wantEnd:   "18:00",
		},
		{
			name: "Wednesday schedule",
			config: &WorkingHoursConfig{
				Wednesday: &DaySchedule{
					Enabled: true,
					Start:   "09:30",
					End:     "17:30",
				},
			},
			weekday:   "wednesday",
			wantStart: "09:30",
			wantEnd:   "17:30",
		},
		{
			name: "Thursday schedule",
			config: &WorkingHoursConfig{
				Thursday: &DaySchedule{
					Enabled: true,
					Start:   "07:00",
					End:     "15:00",
				},
			},
			weekday:   "thursday",
			wantStart: "07:00",
			wantEnd:   "15:00",
		},
		{
			name: "Friday schedule",
			config: &WorkingHoursConfig{
				Friday: &DaySchedule{
					Enabled: true,
					Start:   "09:00",
					End:     "14:00",
				},
			},
			weekday:   "friday",
			wantStart: "09:00",
			wantEnd:   "14:00",
		},
		{
			name: "Saturday schedule",
			config: &WorkingHoursConfig{
				Saturday: &DaySchedule{
					Enabled: false,
					Start:   "",
					End:     "",
				},
			},
			weekday:   "saturday",
			wantStart: "",
			wantEnd:   "",
		},
		{
			name: "Sunday schedule",
			config: &WorkingHoursConfig{
				Sunday: &DaySchedule{
					Enabled: false,
					Start:   "",
					End:     "",
				},
			},
			weekday:   "sunday",
			wantStart: "",
			wantEnd:   "",
		},
		{
			name: "weekend schedule applies to Saturday",
			config: &WorkingHoursConfig{
				Weekend: &DaySchedule{
					Enabled: true,
					Start:   "10:00",
					End:     "14:00",
				},
			},
			weekday:   "saturday",
			wantStart: "10:00",
			wantEnd:   "14:00",
		},
		{
			name: "weekend schedule applies to Sunday",
			config: &WorkingHoursConfig{
				Weekend: &DaySchedule{
					Enabled: true,
					Start:   "10:00",
					End:     "14:00",
				},
			},
			weekday:   "sunday",
			wantStart: "10:00",
			wantEnd:   "14:00",
		},
		{
			name: "day-specific overrides weekend",
			config: &WorkingHoursConfig{
				Saturday: &DaySchedule{
					Enabled: true,
					Start:   "08:00",
					End:     "12:00",
				},
				Weekend: &DaySchedule{
					Enabled: true,
					Start:   "10:00",
					End:     "14:00",
				},
			},
			weekday:   "saturday",
			wantStart: "08:00",
			wantEnd:   "12:00",
		},
		{
			name: "default schedule used for weekday without specific config",
			config: &WorkingHoursConfig{
				Default: &DaySchedule{
					Enabled: true,
					Start:   "08:30",
					End:     "16:30",
				},
			},
			weekday:   "wednesday",
			wantStart: "08:30",
			wantEnd:   "16:30",
		},
		{
			name: "case-insensitive weekday - uppercase",
			config: &WorkingHoursConfig{
				Monday: &DaySchedule{
					Enabled: true,
					Start:   "07:00",
					End:     "15:00",
				},
			},
			weekday:   "MONDAY",
			wantStart: "07:00",
			wantEnd:   "15:00",
		},
		{
			name: "case-insensitive weekday - mixed case",
			config: &WorkingHoursConfig{
				Friday: &DaySchedule{
					Enabled: true,
					Start:   "09:00",
					End:     "13:00",
				},
			},
			weekday:   "FriDay",
			wantStart: "09:00",
			wantEnd:   "13:00",
		},
		{
			name:      "empty config returns default",
			config:    &WorkingHoursConfig{},
			weekday:   "monday",
			wantStart: "09:00",
			wantEnd:   "17:00",
		},
		{
			name: "unknown weekday returns default",
			config: &WorkingHoursConfig{
				Default: &DaySchedule{
					Enabled: true,
					Start:   "09:00",
					End:     "17:00",
				},
			},
			weekday:   "funday",
			wantStart: "09:00",
			wantEnd:   "17:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetScheduleForDay(tt.weekday)

			require.NotNil(t, result)
			assert.Equal(t, tt.wantStart, result.Start)
			assert.Equal(t, tt.wantEnd, result.End)
		})
	}
}

func TestUnixTimeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantUnix  int64
		wantErr   bool
		skipCheck bool // Skip Unix comparison for special cases
	}{
		{
			name:     "Unix timestamp integer",
			json:     `1704067200`,
			wantUnix: 1704067200,
		},
		{
			name:     "RFC3339 string",
			json:     `"2024-01-01T12:00:00Z"`,
			wantUnix: 1704110400,
		},
		{
			name:     "RFC3339 with timezone offset",
			json:     `"2024-06-15T10:30:00-05:00"`,
			wantUnix: 1718465400,
		},
		{
			name:     "zero timestamp",
			json:     `0`,
			wantUnix: 0,
		},
		{
			name:    "invalid RFC3339 string - error returned",
			json:    `"not a date"`,
			wantErr: true,
		},
		{
			name:      "null value - zero time",
			json:      `null`,
			skipCheck: true, // null results in zero value, Unix() returns special value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ut UnixTime
			err := json.Unmarshal([]byte(tt.json), &ut)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if !tt.skipCheck {
				assert.Equal(t, tt.wantUnix, ut.Unix())
			}
		})
	}
}

func TestUnixTimeUnmarshalJSON_InGrant(t *testing.T) {
	// Test UnixTime as part of Grant struct
	tests := []struct {
		name      string
		json      string
		wantUnix  int64
		skipCheck bool
	}{
		{
			name:     "grant with integer timestamps",
			json:     `{"id": "grant-1", "email": "test@example.com", "provider": "google", "created_at": 1704067200}`,
			wantUnix: 1704067200,
		},
		{
			name:     "grant with RFC3339 timestamps",
			json:     `{"id": "grant-2", "email": "test@example.com", "provider": "microsoft", "created_at": "2024-06-15T10:30:00Z"}`,
			wantUnix: 1718447400,
		},
		{
			name:      "grant with missing timestamps",
			json:      `{"id": "grant-3", "email": "test@example.com", "provider": "imap"}`,
			skipCheck: true, // Missing timestamps result in zero value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var grant Grant
			err := json.Unmarshal([]byte(tt.json), &grant)

			require.NoError(t, err)
			if !tt.skipCheck {
				assert.Equal(t, tt.wantUnix, grant.CreatedAt.Unix())
			}
		})
	}
}

func TestDayScheduleWithBreaks(t *testing.T) {
	schedule := &DaySchedule{
		Enabled: true,
		Start:   "09:00",
		End:     "17:00",
		Breaks: []BreakBlock{
			{Name: "Lunch", Start: "12:00", End: "13:00", Type: "lunch"},
			{Name: "Coffee", Start: "15:00", End: "15:15", Type: "coffee"},
		},
	}

	assert.True(t, schedule.Enabled)
	assert.Equal(t, "09:00", schedule.Start)
	assert.Equal(t, "17:00", schedule.End)
	assert.Len(t, schedule.Breaks, 2)
	assert.Equal(t, "Lunch", schedule.Breaks[0].Name)
	assert.Equal(t, "lunch", schedule.Breaks[0].Type)
	assert.Equal(t, "Coffee", schedule.Breaks[1].Name)
}
