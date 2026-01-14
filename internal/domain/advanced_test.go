package domain

import (
	"testing"
)

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
// INBOUND INBOX TESTS
// =============================================================================

func TestInboundInbox(t *testing.T) {
	t.Run("IsValid_returns_true_for_valid_status", func(t *testing.T) {
		inbox := InboundInbox{
			ID:          "inbox-001",
			Email:       "support@app.nylas.email",
			GrantStatus: "valid",
		}
		if !inbox.IsValid() {
			t.Error("Expected IsValid() to return true for valid status")
		}
	})

	t.Run("IsValid_returns_false_for_invalid_status", func(t *testing.T) {
		inbox := InboundInbox{
			ID:          "inbox-001",
			Email:       "support@app.nylas.email",
			GrantStatus: "invalid",
		}
		if inbox.IsValid() {
			t.Error("Expected IsValid() to return false for invalid status")
		}
	})

	t.Run("IsValid_returns_false_for_empty_status", func(t *testing.T) {
		inbox := InboundInbox{
			ID:          "inbox-001",
			Email:       "support@app.nylas.email",
			GrantStatus: "",
		}
		if inbox.IsValid() {
			t.Error("Expected IsValid() to return false for empty status")
		}
	})

	t.Run("IsValid_returns_false_for_other_statuses", func(t *testing.T) {
		statuses := []string{"pending", "error", "suspended", "VALID", "Valid"}
		for _, status := range statuses {
			inbox := InboundInbox{GrantStatus: status}
			if inbox.IsValid() {
				t.Errorf("Expected IsValid() to return false for status %q", status)
			}
		}
	})

	t.Run("inbox_creation", func(t *testing.T) {
		inbox := InboundInbox{
			ID:          "inbox-001",
			Email:       "support@app.nylas.email",
			GrantStatus: "valid",
		}

		if inbox.ID != "inbox-001" {
			t.Errorf("InboundInbox.ID = %q, want %q", inbox.ID, "inbox-001")
		}
		if inbox.Email != "support@app.nylas.email" {
			t.Errorf("InboundInbox.Email = %q, want %q", inbox.Email, "support@app.nylas.email")
		}
	})
}

// =============================================================================
// INBOUND WEBHOOK EVENT TESTS
// =============================================================================

func TestInboundWebhookEvent(t *testing.T) {
	t.Run("IsInboundEvent_returns_true_for_inbox_source", func(t *testing.T) {
		event := InboundWebhookEvent{
			Type:      "message.created",
			Source:    "inbox",
			GrantID:   "inbox-001",
			MessageID: "msg-001",
		}
		if !event.IsInboundEvent() {
			t.Error("Expected IsInboundEvent() to return true for source 'inbox'")
		}
	})

	t.Run("IsInboundEvent_returns_false_for_other_sources", func(t *testing.T) {
		sources := []string{"", "email", "calendar", "Inbox", "INBOX", "imap"}
		for _, source := range sources {
			event := InboundWebhookEvent{
				Type:   "message.created",
				Source: source,
			}
			if event.IsInboundEvent() {
				t.Errorf("Expected IsInboundEvent() to return false for source %q", source)
			}
		}
	})

	t.Run("event_creation_with_message", func(t *testing.T) {
		msg := &Message{
			ID:      "msg-001",
			Subject: "Test Subject",
		}
		event := InboundWebhookEvent{
			Type:      "message.created",
			Source:    "inbox",
			GrantID:   "inbox-001",
			MessageID: "msg-001",
			Message:   msg,
		}

		if event.Type != "message.created" {
			t.Errorf("InboundWebhookEvent.Type = %q, want %q", event.Type, "message.created")
		}
		if event.Message == nil {
			t.Error("InboundWebhookEvent.Message should not be nil")
		}
		if event.Message.ID != "msg-001" {
			t.Errorf("InboundWebhookEvent.Message.ID = %q, want %q", event.Message.ID, "msg-001")
		}
	})

	t.Run("event_creation_without_message", func(t *testing.T) {
		event := InboundWebhookEvent{
			Type:      "message.created",
			Source:    "inbox",
			GrantID:   "inbox-001",
			MessageID: "msg-001",
			Message:   nil,
		}

		if event.Message != nil {
			t.Error("InboundWebhookEvent.Message should be nil")
		}
		if event.MessageID != "msg-001" {
			t.Errorf("InboundWebhookEvent.MessageID = %q, want %q", event.MessageID, "msg-001")
		}
	})
}

// =============================================================================
// CREATE INBOUND INBOX REQUEST TESTS
// =============================================================================

func TestCreateInboundInboxRequest(t *testing.T) {
	t.Run("request_creation", func(t *testing.T) {
		req := CreateInboundInboxRequest{
			Email: "support",
		}

		if req.Email != "support" {
			t.Errorf("CreateInboundInboxRequest.Email = %q, want %q", req.Email, "support")
		}
	})

	t.Run("request_with_various_prefixes", func(t *testing.T) {
		prefixes := []string{"support", "sales", "info", "help-desk", "team123"}
		for _, prefix := range prefixes {
			req := CreateInboundInboxRequest{Email: prefix}
			if req.Email != prefix {
				t.Errorf("CreateInboundInboxRequest.Email = %q, want %q", req.Email, prefix)
			}
		}
	})
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
