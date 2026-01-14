package notetaker

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var (
		meetingLink string
		joinTime    string
		botName     string
		outputJSON  bool
	)

	cmd := &cobra.Command{
		Use:   "create [grant-id]",
		Short: "Create a notetaker to join a meeting",
		Long: `Create a notetaker bot that will join a video meeting to record and transcribe.

Supported meeting providers:
- Zoom
- Google Meet
- Microsoft Teams

The notetaker will join the meeting at the specified time (or immediately if not specified),
record the meeting, and generate a transcript when complete.`,
		Example: `  # Create notetaker to join immediately
  nylas notetaker create --meeting-link "https://zoom.us/j/123456789"

  # Create notetaker to join at a specific time
  nylas notetaker create --meeting-link "https://meet.google.com/abc-defg-hij" --join-time "2024-01-15 14:00"

  # Create notetaker with custom bot name
  nylas notetaker create --meeting-link "https://zoom.us/j/123" --bot-name "Meeting Recorder"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if meetingLink == "" {
				return common.NewUserError("meeting link is required", "Use --meeting-link with a Zoom, Google Meet, or Teams URL")
			}

			// Validate meeting link URL format
			u, err := url.Parse(meetingLink)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return common.NewUserError(fmt.Sprintf("invalid meeting link URL: %s", meetingLink), "must be a valid http/https URL")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			grantID, err := getGrantID(args)
			if err != nil {
				return err
			}

			req := &domain.CreateNotetakerRequest{
				MeetingLink: meetingLink,
			}

			// Parse join time if provided
			if joinTime != "" {
				parsedTime, err := parseJoinTime(joinTime)
				if err != nil {
					return common.WrapDateParseError("join time", err)
				}
				req.JoinTime = parsedTime.Unix()
			}

			// Set bot config if name provided
			if botName != "" {
				req.BotConfig = &domain.BotConfig{
					Name: botName,
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			notetaker, err := client.CreateNotetaker(ctx, grantID, req)
			if err != nil {
				return common.WrapCreateError("notetaker", err)
			}

			if outputJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(notetaker)
			}

			_, _ = common.BoldGreen.Println("✓ Notetaker created successfully!")
			fmt.Println()
			_, _ = common.Cyan.Printf("ID:    %s\n", notetaker.ID)
			fmt.Printf("State: %s\n", formatState(notetaker.State))
			fmt.Printf("Link:  %s\n", notetaker.MeetingLink)
			if !notetaker.JoinTime.IsZero() {
				fmt.Printf("Join:  %s\n", notetaker.JoinTime.Local().Format(common.DisplayWeekdayFullWithTZ))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&meetingLink, "meeting-link", "m", "", "Meeting URL (Zoom, Google Meet, or Teams)")
	cmd.Flags().StringVarP(&joinTime, "join-time", "j", "", "When to join (e.g., '2024-01-15 14:00', 'tomorrow 9am', '30m')")
	cmd.Flags().StringVar(&botName, "bot-name", "", "Custom name for the notetaker bot")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	_ = cmd.MarkFlagRequired("meeting-link") // Hardcoded flag name, won't fail

	return cmd
}

// parseJoinTime parses various time formats for join time.
func parseJoinTime(input string) (time.Time, error) {
	now := time.Now()
	input = strings.TrimSpace(input)
	lower := strings.ToLower(input)

	// Try Unix timestamp first
	if ts, err := strconv.ParseInt(input, 10, 64); err == nil && ts > 1000000000 {
		return time.Unix(ts, 0), nil
	}

	// Duration formats: 30m, 2h, 1d
	if len(input) >= 2 {
		numStr := input[:len(input)-1]
		unit := input[len(input)-1:]
		if num, err := strconv.Atoi(numStr); err == nil {
			switch unit {
			case "m":
				return now.Add(time.Duration(num) * time.Minute), nil
			case "h":
				return now.Add(time.Duration(num) * time.Hour), nil
			case "d":
				return now.AddDate(0, 0, num), nil
			}
		}
	}

	// "tomorrow" keyword
	if strings.HasPrefix(lower, "tomorrow") {
		tomorrow := now.AddDate(0, 0, 1)
		rest := strings.TrimPrefix(lower, "tomorrow")
		rest = strings.TrimSpace(rest)
		if rest == "" {
			return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, now.Location()), nil
		}
		if t, err := common.ParseTimeOfDay(rest); err == nil {
			return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), t.Hour(), t.Minute(), 0, 0, now.Location()), nil
		}
	}

	// Try standard date/time formats
	formats := []string{
		"2006-01-02 15:04",
		"2006-01-02T15:04",
		"2006-01-02 3:04pm",
		"Jan 2 15:04",
		"Jan 2 3:04pm",
		"Jan 2, 2006 15:04",
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, input, now.Location()); err == nil {
			if t.Year() == 0 {
				t = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
			}
			return t, nil
		}
	}

	// Try just time of day
	if t, err := common.ParseTimeOfDay(lower); err == nil {
		result := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
		if result.Before(now) {
			result = result.AddDate(0, 0, 1)
		}
		return result, nil
	}

	return time.Time{}, common.NewInputError(fmt.Sprintf("could not parse time format: %s", input))
}
