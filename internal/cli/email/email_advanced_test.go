package email

import (
	"fmt"
	"testing"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestThreadsSearchCommand(t *testing.T) {
	cmd := newThreadsSearchCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "search [grant-id]", cmd.Use)
	})

	t.Run("has_limit_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("limit")
		assert.NotNil(t, flag)
		assert.Equal(t, "20", flag.DefValue)
	})

	t.Run("has_from_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("from")
		assert.NotNil(t, flag)
	})

	t.Run("has_to_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("to")
		assert.NotNil(t, flag)
	})

	t.Run("has_unread_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("unread")
		assert.NotNil(t, flag)
	})

	t.Run("has_starred_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("starred")
		assert.NotNil(t, flag)
	})

	t.Run("has_in_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("in")
		assert.NotNil(t, flag)
	})

	t.Run("has_long_description_with_examples", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "Examples")
	})
}

func TestDraftsCommand(t *testing.T) {
	cmd := newDraftsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "drafts", cmd.Use)
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expectedCmds := []string{"list", "create", "show", "send", "delete"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("formatTimeAgo", func(t *testing.T) {
		tests := []struct {
			duration time.Duration
			expected string
		}{
			{30 * time.Second, "just now"},
			{1 * time.Minute, "1 minute ago"},
			{5 * time.Minute, "5 minutes ago"},
			{1 * time.Hour, "1 hour ago"},
			{5 * time.Hour, "5 hours ago"},
			{24 * time.Hour, "1 day ago"},
			{48 * time.Hour, "2 days ago"},
		}

		for _, tt := range tests {
			past := time.Now().Add(-tt.duration)
			got := common.FormatTimeAgo(past)
			assert.Equal(t, tt.expected, got)
		}
	})

	t.Run("truncate", func(t *testing.T) {
		tests := []struct {
			input    string
			maxLen   int
			expected string
		}{
			{"hello", 10, "hello"},
			{"hello world", 8, "hello..."},
			{"short", 5, "short"},
		}

		for _, tt := range tests {
			got := common.Truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, got)
		}
	})

	t.Run("formatParticipant", func(t *testing.T) {
		tests := []struct {
			contact  domain.EmailParticipant
			expected string
		}{
			{domain.EmailParticipant{Name: "John", Email: "john@example.com"}, "John"},
			{domain.EmailParticipant{Name: "", Email: "jane@example.com"}, "jane@example.com"},
		}

		for _, tt := range tests {
			got := common.FormatParticipant(tt.contact)
			assert.Equal(t, tt.expected, got)
		}
	})

	t.Run("formatParticipants", func(t *testing.T) {
		contacts := []domain.EmailParticipant{
			{Name: "John", Email: "john@example.com"},
			{Name: "", Email: "jane@example.com"},
		}
		got := common.FormatParticipants(contacts)
		assert.Equal(t, "John, jane@example.com", got)
	})

	t.Run("stripHTML", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"<p>Hello</p>", "Hello"},
			{"<html><body><h1>Title</h1><p>Content</p></body></html>", "Title\n\nContent"},
			{"Plain text", "Plain text"},
			{"Line 1<br>Line 2", "Line 1\nLine 2"},
			{"<div>Block 1</div><div>Block 2</div>", "Block 1\n\nBlock 2"},
			{"Text with &nbsp; entities &amp; &lt;tags&gt;", "Text with \u00a0 entities & <tags>"},
			{"<style>body{color:red}</style>Content", "Content"},
		}

		for _, tt := range tests {
			got := common.StripHTML(tt.input)
			assert.Equal(t, tt.expected, got)
		}
	})

	t.Run("formatSize", func(t *testing.T) {
		tests := []struct {
			bytes    int64
			expected string
		}{
			{500, "500 B"},
			{1024, "1.0 KB"},
			{1536, "1.5 KB"},
			{1048576, "1.0 MB"},
		}

		for _, tt := range tests {
			got := common.FormatSize(tt.bytes)
			assert.Equal(t, tt.expected, got)
		}
	})

	t.Run("parseEmails", func(t *testing.T) {
		tests := []struct {
			input    string
			expected []string
		}{
			{"a@b.com, c@d.com", []string{"a@b.com", "c@d.com"}},
			{"single@test.com", []string{"single@test.com"}},
			{"", nil},
			{"  spaced@test.com  ,  other@test.com  ", []string{"spaced@test.com", "other@test.com"}},
		}

		for _, tt := range tests {
			got := parseEmails(tt.input)
			assert.Equal(t, tt.expected, got)
		}
	})

	t.Run("parseContacts", func(t *testing.T) {
		tests := []struct {
			input    []string
			expected []domain.EmailParticipant
		}{
			{
				[]string{"test@example.com"},
				[]domain.EmailParticipant{{Email: "test@example.com"}},
			},
			{
				[]string{"John Doe <john@example.com>"},
				[]domain.EmailParticipant{{Name: "John Doe", Email: "john@example.com"}},
			},
			{
				[]string{"plain@test.com", "Named <named@test.com>"},
				[]domain.EmailParticipant{
					{Email: "plain@test.com"},
					{Name: "Named", Email: "named@test.com"},
				},
			},
		}

		for _, tt := range tests {
			got, err := parseContacts(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		}

		// Test invalid email
		_, err := parseContacts([]string{"invalid-email"})
		assert.Error(t, err)

		// Test empty email
		_, err = parseContacts([]string{""})
		assert.Error(t, err)
	})

	t.Run("parseDate", func(t *testing.T) {
		date, err := parseDate("2024-01-15")
		assert.NoError(t, err)
		assert.Equal(t, 2024, date.Year())
		assert.Equal(t, time.January, date.Month())
		assert.Equal(t, 15, date.Day())

		_, err = parseDate("invalid")
		assert.Error(t, err)
	})
}

func TestSendCommandScheduleFlags(t *testing.T) {
	cmd := newSendCmd()

	t.Run("has_schedule_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("schedule")
		assert.NotNil(t, flag)
	})

	t.Run("has_yes_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("yes")
		assert.NotNil(t, flag)
	})

	t.Run("has_yes_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("y")
		assert.NotNil(t, flag)
		assert.Equal(t, "yes", flag.Name)
	})
}

func TestParseScheduleTime(t *testing.T) {
	t.Run("parses_unix_timestamp", func(t *testing.T) {
		// A valid Unix timestamp in the future
		futureTs := time.Now().Add(24 * time.Hour).Unix()
		result, err := parseScheduleTime(fmt.Sprintf("%d", futureTs))
		assert.NoError(t, err)
		assert.True(t, result.After(time.Now()))
	})

	t.Run("rejects_past_unix_timestamp", func(t *testing.T) {
		// A Unix timestamp in the past (Jan 15, 2024)
		_, err := parseScheduleTime("1705320600")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "in the past")
	})

	t.Run("parses_duration_minutes", func(t *testing.T) {
		now := time.Now()
		result, err := parseScheduleTime("30m")
		assert.NoError(t, err)
		// Should be approximately 30 minutes from now
		diff := result.Sub(now)
		assert.True(t, diff >= 29*time.Minute && diff <= 31*time.Minute)
	})

	t.Run("parses_duration_hours", func(t *testing.T) {
		now := time.Now()
		result, err := parseScheduleTime("2h")
		assert.NoError(t, err)
		// Should be approximately 2 hours from now
		diff := result.Sub(now)
		assert.True(t, diff >= 119*time.Minute && diff <= 121*time.Minute)
	})

	t.Run("parses_duration_days", func(t *testing.T) {
		now := time.Now()
		result, err := parseScheduleTime("1d")
		assert.NoError(t, err)
		// Should be approximately 1 day from now
		diff := result.Sub(now)
		assert.True(t, diff >= 23*time.Hour && diff <= 25*time.Hour)
	})

	t.Run("parses_tomorrow", func(t *testing.T) {
		result, err := parseScheduleTime("tomorrow")
		assert.NoError(t, err)
		expected := time.Now().AddDate(0, 0, 1)
		assert.Equal(t, expected.Day(), result.Day())
		assert.Equal(t, 9, result.Hour()) // Default to 9am
	})

	t.Run("parses_tomorrow_with_time", func(t *testing.T) {
		result, err := parseScheduleTime("tomorrow 2pm")
		assert.NoError(t, err)
		expected := time.Now().AddDate(0, 0, 1)
		assert.Equal(t, expected.Day(), result.Day())
		assert.Equal(t, 14, result.Hour())
	})

	t.Run("parses_datetime_format", func(t *testing.T) {
		// Use a future date to avoid "in the past" error
		futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02") + " 14:30"
		result, err := parseScheduleTime(futureDate)
		assert.NoError(t, err)
		assert.Equal(t, 14, result.Hour())
		assert.Equal(t, 30, result.Minute())
	})

	t.Run("returns_error_for_invalid_format", func(t *testing.T) {
		_, err := parseScheduleTime("invalid time format xyz")
		assert.Error(t, err)
	})

	t.Run("returns_error_for_today_without_time", func(t *testing.T) {
		_, err := parseScheduleTime("today")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "specify a time")
	})
}

func TestParseTimeOfDay(t *testing.T) {
	t.Run("parses_24_hour_format", func(t *testing.T) {
		result, err := common.ParseTimeOfDay("14:30")
		assert.NoError(t, err)
		assert.Equal(t, 14, result.Hour())
		assert.Equal(t, 30, result.Minute())
	})

	t.Run("parses_12_hour_format_pm", func(t *testing.T) {
		result, err := common.ParseTimeOfDay("2:30pm")
		assert.NoError(t, err)
		assert.Equal(t, 14, result.Hour())
		assert.Equal(t, 30, result.Minute())
	})

	t.Run("parses_12_hour_format_am", func(t *testing.T) {
		result, err := common.ParseTimeOfDay("9am")
		assert.NoError(t, err)
		assert.Equal(t, 9, result.Hour())
	})

	t.Run("parses_12_hour_format_with_space", func(t *testing.T) {
		result, err := common.ParseTimeOfDay("3 pm")
		assert.NoError(t, err)
		assert.Equal(t, 15, result.Hour())
	})

	t.Run("returns_error_for_invalid_format", func(t *testing.T) {
		_, err := common.ParseTimeOfDay("invalid")
		assert.Error(t, err)
	})
}

func TestEmailCommandHelp(t *testing.T) {
	cmd := NewEmailCmd()
	stdout, _, err := executeCommand(cmd, "--help")

	assert.NoError(t, err)

	expectedStrings := []string{
		"email",
		"list",
		"read",
		"send",
		"search",
		"folders",
		"threads",
		"drafts",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, stdout, expected, "Help output should contain %q", expected)
	}
}

func TestEmailSendHelp(t *testing.T) {
	cmd := NewEmailCmd()
	stdout, _, err := executeCommand(cmd, "send", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "send")
	assert.Contains(t, stdout, "--to")
	assert.Contains(t, stdout, "--subject")
	assert.Contains(t, stdout, "--body")
	assert.Contains(t, stdout, "--schedule")
	assert.Contains(t, stdout, "--yes")
}

func TestSendCommandTrackingFlags(t *testing.T) {
	cmd := newSendCmd()

	t.Run("has_track_opens_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("track-opens")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_track_links_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("track-links")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_track_label_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("track-label")
		assert.NotNil(t, flag)
	})

	t.Run("has_metadata_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("metadata")
		assert.NotNil(t, flag)
	})
}

func TestEmailSendHelpTrackingFlags(t *testing.T) {
	cmd := NewEmailCmd()
	stdout, _, err := executeCommand(cmd, "send", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "--track-opens")
	assert.Contains(t, stdout, "--track-links")
	assert.Contains(t, stdout, "--track-label")
	assert.Contains(t, stdout, "--metadata")
}
