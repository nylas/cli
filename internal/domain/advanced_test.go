package domain

import (
	"testing"
)

func TestOTPResult(t *testing.T) {
	t.Run("otp_result_creation", func(t *testing.T) {
		result := OTPResult{
			Code:      "123456",
			From:      "service@example.com",
			Subject:   "Your verification code",
			MessageID: "msg-123",
		}

		if result.Code != "123456" {
			t.Errorf("OTPResult.Code = %q, want %q", result.Code, "123456")
		}
		if result.From != "service@example.com" {
			t.Errorf("OTPResult.From = %q, want %q", result.From, "service@example.com")
		}
	})
}

// TestErrors tests error definitions.
func TestErrors(t *testing.T) {
	errors := []error{
		ErrNotConfigured,
		ErrAuthFailed,
		ErrAuthTimeout,
		ErrInvalidProvider,
		ErrGrantNotFound,
		ErrNoDefaultGrant,
		ErrInvalidGrant,
		ErrTokenExpired,
		ErrAPIError,
		ErrNetworkError,
		ErrSecretNotFound,
		ErrSecretStoreFailed,
		ErrConfigNotFound,
		ErrConfigInvalid,
		ErrOTPNotFound,
		ErrAccountNotFound,
		ErrNoMessages,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("Expected non-nil error")
		}
		if err.Error() == "" {
			t.Error("Error message should not be empty")
		}
	}
}

// =============================================================================
// WEBHOOK TRIGGER TYPES TESTS
// =============================================================================

// TestWebhookTriggerTypes tests the webhook trigger type functions.
func TestWebhookTriggerTypes(t *testing.T) {
	t.Run("AllTriggerTypes_returns_expected_count", func(t *testing.T) {
		triggers := AllTriggerTypes()
		// Should have at least 25 trigger types now
		if len(triggers) < 25 {
			t.Errorf("AllTriggerTypes() returned %d triggers, expected at least 25", len(triggers))
		}
	})

	t.Run("AllTriggerTypes_contains_grant_triggers", func(t *testing.T) {
		triggers := AllTriggerTypes()
		expected := []string{
			TriggerGrantCreated,
			TriggerGrantDeleted,
			TriggerGrantExpired,
			TriggerGrantUpdated,
			TriggerGrantIMAPSyncComplete,
		}
		for _, e := range expected {
			found := false
			for _, t := range triggers {
				if t == e {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("AllTriggerTypes() missing expected trigger: %s", e)
			}
		}
	})

	t.Run("AllTriggerTypes_contains_message_triggers", func(t *testing.T) {
		triggers := AllTriggerTypes()
		expected := []string{
			TriggerMessageCreated,
			TriggerMessageUpdated,
			TriggerMessageOpened,
			TriggerMessageBounceDetected,
			TriggerMessageSendSuccess,
			TriggerMessageSendFailed,
			TriggerMessageOpenedTruncated,
			TriggerMessageLinkClicked,
		}
		for _, e := range expected {
			found := false
			for _, t := range triggers {
				if t == e {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("AllTriggerTypes() missing expected trigger: %s", e)
			}
		}
	})

	t.Run("AllTriggerTypes_contains_event_triggers", func(t *testing.T) {
		triggers := AllTriggerTypes()
		expected := []string{
			TriggerEventCreated,
			TriggerEventUpdated,
			TriggerEventDeleted,
		}
		for _, e := range expected {
			found := false
			for _, t := range triggers {
				if t == e {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("AllTriggerTypes() missing expected trigger: %s", e)
			}
		}
	})

	t.Run("AllTriggerTypes_contains_contact_triggers", func(t *testing.T) {
		triggers := AllTriggerTypes()
		expected := []string{
			TriggerContactCreated,
			TriggerContactUpdated,
			TriggerContactDeleted,
		}
		for _, e := range expected {
			found := false
			for _, t := range triggers {
				if t == e {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("AllTriggerTypes() missing expected trigger: %s", e)
			}
		}
	})

	t.Run("AllTriggerTypes_contains_calendar_triggers", func(t *testing.T) {
		triggers := AllTriggerTypes()
		expected := []string{
			TriggerCalendarCreated,
			TriggerCalendarUpdated,
			TriggerCalendarDeleted,
		}
		for _, e := range expected {
			found := false
			for _, t := range triggers {
				if t == e {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("AllTriggerTypes() missing expected trigger: %s", e)
			}
		}
	})

	t.Run("AllTriggerTypes_contains_folder_triggers", func(t *testing.T) {
		triggers := AllTriggerTypes()
		expected := []string{
			TriggerFolderCreated,
			TriggerFolderUpdated,
			TriggerFolderDeleted,
		}
		for _, e := range expected {
			found := false
			for _, t := range triggers {
				if t == e {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("AllTriggerTypes() missing expected trigger: %s", e)
			}
		}
	})

	t.Run("AllTriggerTypes_contains_notetaker_trigger", func(t *testing.T) {
		triggers := AllTriggerTypes()
		found := false
		for _, t := range triggers {
			if t == TriggerNotetakerMedia {
				found = true
				break
			}
		}
		if !found {
			t.Error("AllTriggerTypes() missing expected trigger: notetaker.media")
		}
	})

	t.Run("AllTriggerTypes_contains_thread_trigger", func(t *testing.T) {
		triggers := AllTriggerTypes()
		found := false
		for _, t := range triggers {
			if t == TriggerThreadReplied {
				found = true
				break
			}
		}
		if !found {
			t.Error("AllTriggerTypes() missing expected trigger: thread.replied")
		}
	})

	t.Run("TriggerTypeCategories_has_all_categories", func(t *testing.T) {
		categories := TriggerTypeCategories()
		expectedCategories := []string{
			"grant",
			"message",
			"thread",
			"event",
			"contact",
			"calendar",
			"folder",
			"notetaker",
		}
		for _, cat := range expectedCategories {
			if _, ok := categories[cat]; !ok {
				t.Errorf("TriggerTypeCategories() missing category: %s", cat)
			}
		}
	})

	t.Run("TriggerTypeCategories_grant_has_expected_triggers", func(t *testing.T) {
		categories := TriggerTypeCategories()
		grantTriggers := categories["grant"]
		if len(grantTriggers) != 5 {
			t.Errorf("Expected 5 grant triggers, got %d", len(grantTriggers))
		}
	})

	t.Run("TriggerTypeCategories_message_has_expected_triggers", func(t *testing.T) {
		categories := TriggerTypeCategories()
		messageTriggers := categories["message"]
		if len(messageTriggers) != 8 {
			t.Errorf("Expected 8 message triggers, got %d", len(messageTriggers))
		}
	})

	t.Run("TriggerTypeCategories_notetaker_has_expected_triggers", func(t *testing.T) {
		categories := TriggerTypeCategories()
		notetakerTriggers := categories["notetaker"]
		if len(notetakerTriggers) != 1 {
			t.Errorf("Expected 1 notetaker trigger, got %d", len(notetakerTriggers))
		}
		if notetakerTriggers[0] != TriggerNotetakerMedia {
			t.Errorf("Expected notetaker.media trigger, got %s", notetakerTriggers[0])
		}
	})

	t.Run("TriggerConstants_have_correct_values", func(t *testing.T) {
		tests := []struct {
			constant string
			expected string
		}{
			{TriggerGrantCreated, "grant.created"},
			{TriggerGrantDeleted, "grant.deleted"},
			{TriggerGrantExpired, "grant.expired"},
			{TriggerGrantUpdated, "grant.updated"},
			{TriggerGrantIMAPSyncComplete, "grant.imap_sync_completed"},
			{TriggerMessageCreated, "message.created"},
			{TriggerMessageUpdated, "message.updated"},
			{TriggerMessageOpened, "message.opened"},
			{TriggerMessageBounceDetected, "message.bounce_detected"},
			{TriggerMessageSendSuccess, "message.send_success"},
			{TriggerMessageSendFailed, "message.send_failed"},
			{TriggerMessageOpenedTruncated, "message.opened.truncated"},
			{TriggerMessageLinkClicked, "message.link_clicked"},
			{TriggerThreadReplied, "thread.replied"},
			{TriggerEventCreated, "event.created"},
			{TriggerEventUpdated, "event.updated"},
			{TriggerEventDeleted, "event.deleted"},
			{TriggerContactCreated, "contact.created"},
			{TriggerContactUpdated, "contact.updated"},
			{TriggerContactDeleted, "contact.deleted"},
			{TriggerCalendarCreated, "calendar.created"},
			{TriggerCalendarUpdated, "calendar.updated"},
			{TriggerCalendarDeleted, "calendar.deleted"},
			{TriggerFolderCreated, "folder.created"},
			{TriggerFolderUpdated, "folder.updated"},
			{TriggerFolderDeleted, "folder.deleted"},
			{TriggerNotetakerMedia, "notetaker.media"},
		}

		for _, tt := range tests {
			if tt.constant != tt.expected {
				t.Errorf("Trigger constant = %q, want %q", tt.constant, tt.expected)
			}
		}
	})
}
