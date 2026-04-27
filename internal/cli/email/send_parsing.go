package email

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// parseEmails parses a comma-separated list of emails.
func parseEmails(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// parseContacts converts email strings to EmailParticipant objects with validation.
func parseContacts(emails []string) ([]domain.EmailParticipant, error) {
	contacts := make([]domain.EmailParticipant, len(emails))
	for i, email := range emails {
		email = strings.TrimSpace(email)
		if email == "" {
			return nil, common.NewInputError("email address cannot be empty")
		}

		// Try parsing as RFC 5322 address (handles "Name <email>" format)
		addr, err := mail.ParseAddress(email)
		if err == nil {
			contacts[i] = domain.EmailParticipant{Name: addr.Name, Email: addr.Address}
		} else {
			// Check if it's a plain email without angle brackets
			if !strings.Contains(email, "@") {
				return nil, common.NewInputError(fmt.Sprintf("invalid email address: %s", email))
			}
			// Basic validation for plain email
			if strings.Count(email, "@") != 1 {
				return nil, common.NewInputError(fmt.Sprintf("invalid email address: %s", email))
			}
			contacts[i] = domain.EmailParticipant{Email: email}
		}
	}
	return contacts, nil
}

// errScheduleInPast is the user-facing wrapper of common.ErrScheduleInPast.
// It wraps the sentinel via %w so callers can still match against
// common.ErrScheduleInPast with errors.Is even after this layer adds the
// "specify a future time" hint to the message.
var errScheduleInPast = fmt.Errorf("%w: specify a future time", common.ErrScheduleInPast)

// parseScheduleTime parses various time formats for scheduling. Past times
// are rejected with errScheduleInPast.
func parseScheduleTime(input string) (time.Time, error) {
	t, err := common.ParseHumanTime(input, common.ParseHumanTimeOpts{
		RejectPast:                 true,
		RollPastBareTimeToTomorrow: true,
	})
	if errors.Is(err, common.ErrScheduleInPast) {
		return time.Time{}, errScheduleInPast
	}
	return t, err
}
