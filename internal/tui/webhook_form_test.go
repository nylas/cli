package tui

import (
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestNewWebhookFormCreate(t *testing.T) {
	app := createTestApp(t)

	form := NewWebhookForm(app, nil, nil, nil)

	if form == nil {
		t.Fatal("NewWebhookForm returned nil")
		return
	}

	if form.mode != WebhookFormCreate {
		t.Errorf("mode = %v, want WebhookFormCreate", form.mode)
	}

	// Check defaults
	if form.status != "active" {
		t.Errorf("status = %q, want 'active'", form.status)
	}

	if len(form.triggerTypes) != 0 {
		t.Errorf("triggerTypes = %v, want empty slice", form.triggerTypes)
	}
}

func TestNewWebhookFormEdit(t *testing.T) {
	app := createTestApp(t)

	webhook := &domain.Webhook{
		ID:                         "webhook-123",
		WebhookURL:                 "https://example.com/webhook",
		Description:                "Test webhook",
		TriggerTypes:               []string{domain.TriggerMessageCreated, domain.TriggerMessageUpdated},
		NotificationEmailAddresses: []string{"admin@example.com", "alerts@example.com"},
		Status:                     "inactive",
	}

	form := NewWebhookForm(app, webhook, nil, nil)

	if form == nil {
		t.Fatal("NewWebhookForm returned nil")
		return
	}

	if form.mode != WebhookFormEdit {
		t.Errorf("mode = %v, want WebhookFormEdit", form.mode)
	}

	if form.webhookURL != webhook.WebhookURL {
		t.Errorf("webhookURL = %q, want %q", form.webhookURL, webhook.WebhookURL)
	}

	if form.description != webhook.Description {
		t.Errorf("description = %q, want %q", form.description, webhook.Description)
	}

	if len(form.triggerTypes) != len(webhook.TriggerTypes) {
		t.Errorf("triggerTypes length = %d, want %d", len(form.triggerTypes), len(webhook.TriggerTypes))
	}

	if form.status != webhook.Status {
		t.Errorf("status = %q, want %q", form.status, webhook.Status)
	}

	expectedNotifyEmails := "admin@example.com, alerts@example.com"
	if form.notifyEmails != expectedNotifyEmails {
		t.Errorf("notifyEmails = %q, want %q", form.notifyEmails, expectedNotifyEmails)
	}
}

func TestWebhookFormValidation(t *testing.T) {
	app := createTestApp(t)

	tests := []struct {
		name      string
		setup     func(f *WebhookForm)
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid webhook",
			setup: func(f *WebhookForm) {
				f.webhookURL = "https://example.com/webhook"
				f.triggerTypes = []string{domain.TriggerMessageCreated}
			},
			wantError: false,
		},
		{
			name: "valid with http",
			setup: func(f *WebhookForm) {
				f.webhookURL = "http://localhost:8080/webhook"
				f.triggerTypes = []string{domain.TriggerEventCreated}
			},
			wantError: false,
		},
		{
			name: "missing webhook URL",
			setup: func(f *WebhookForm) {
				f.webhookURL = ""
				f.triggerTypes = []string{domain.TriggerMessageCreated}
			},
			wantError: true,
			errorMsg:  "Webhook URL is required",
		},
		{
			name: "whitespace only URL",
			setup: func(f *WebhookForm) {
				f.webhookURL = "   "
				f.triggerTypes = []string{domain.TriggerMessageCreated}
			},
			wantError: true,
			errorMsg:  "Webhook URL is required",
		},
		{
			name: "invalid URL protocol",
			setup: func(f *WebhookForm) {
				f.webhookURL = "ftp://example.com/webhook"
				f.triggerTypes = []string{domain.TriggerMessageCreated}
			},
			wantError: true,
			errorMsg:  "Webhook URL must start with http:// or https://",
		},
		{
			name: "missing protocol",
			setup: func(f *WebhookForm) {
				f.webhookURL = "example.com/webhook"
				f.triggerTypes = []string{domain.TriggerMessageCreated}
			},
			wantError: true,
			errorMsg:  "Webhook URL must start with http:// or https://",
		},
		{
			name: "missing trigger types",
			setup: func(f *WebhookForm) {
				f.webhookURL = "https://example.com/webhook"
				f.triggerTypes = []string{}
			},
			wantError: true,
			errorMsg:  "At least one trigger type is required",
		},
		{
			name: "nil trigger types",
			setup: func(f *WebhookForm) {
				f.webhookURL = "https://example.com/webhook"
				f.triggerTypes = nil
			},
			wantError: true,
			errorMsg:  "At least one trigger type is required",
		},
		{
			name: "valid with multiple triggers",
			setup: func(f *WebhookForm) {
				f.webhookURL = "https://example.com/webhook"
				f.triggerTypes = []string{
					domain.TriggerMessageCreated,
					domain.TriggerMessageUpdated,
					domain.TriggerEventCreated,
				}
			},
			wantError: false,
		},
		{
			name: "valid with description and emails",
			setup: func(f *WebhookForm) {
				f.webhookURL = "https://example.com/webhook"
				f.triggerTypes = []string{domain.TriggerGrantExpired}
				f.description = "Production webhook"
				f.notifyEmails = "admin@example.com, ops@example.com"
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := NewWebhookForm(app, nil, nil, nil)
			// Reset defaults
			form.webhookURL = ""
			form.triggerTypes = nil

			tt.setup(form)

			errors := form.validate()

			if tt.wantError {
				if len(errors) == 0 {
					t.Error("expected validation error, got none")
				} else {
					found := false
					for _, err := range errors {
						if err == tt.errorMsg {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error %q, got %v", tt.errorMsg, errors)
					}
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("unexpected validation errors: %v", errors)
				}
			}
		})
	}
}

func TestWebhookFormMode(t *testing.T) {
	if WebhookFormCreate != 0 {
		t.Errorf("WebhookFormCreate = %d, want 0", WebhookFormCreate)
	}
	if WebhookFormEdit != 1 {
		t.Errorf("WebhookFormEdit = %d, want 1", WebhookFormEdit)
	}
}

func TestTriggerPresets(t *testing.T) {
	// Test that all presets are defined
	expectedPresets := []string{
		"All Messages",
		"Message Tracking",
		"All Events",
		"All Contacts",
		"Grant Changes",
	}

	for _, preset := range expectedPresets {
		triggers, ok := triggerPresets[preset]
		if !ok {
			t.Errorf("triggerPresets missing preset %q", preset)
			continue
		}
		if len(triggers) == 0 {
			t.Errorf("triggerPresets[%q] is empty", preset)
		}
	}

	// Test specific presets contain expected triggers
	allMessages := triggerPresets["All Messages"]
	if len(allMessages) != 2 {
		t.Errorf("All Messages preset has %d triggers, want 2", len(allMessages))
	}

	messageTracking := triggerPresets["Message Tracking"]
	if len(messageTracking) != 3 {
		t.Errorf("Message Tracking preset has %d triggers, want 3", len(messageTracking))
	}

	allEvents := triggerPresets["All Events"]
	if len(allEvents) != 3 {
		t.Errorf("All Events preset has %d triggers, want 3", len(allEvents))
	}

	allContacts := triggerPresets["All Contacts"]
	if len(allContacts) != 3 {
		t.Errorf("All Contacts preset has %d triggers, want 3", len(allContacts))
	}

	grantChanges := triggerPresets["Grant Changes"]
	if len(grantChanges) != 4 {
		t.Errorf("Grant Changes preset has %d triggers, want 4", len(grantChanges))
	}
}

func TestWebhookFormEditWithEmptyNotificationEmails(t *testing.T) {
	app := createTestApp(t)

	webhook := &domain.Webhook{
		ID:                         "webhook-123",
		WebhookURL:                 "https://example.com/webhook",
		TriggerTypes:               []string{domain.TriggerMessageCreated},
		NotificationEmailAddresses: nil,
		Status:                     "active",
	}

	form := NewWebhookForm(app, webhook, nil, nil)

	if form.notifyEmails != "" {
		t.Errorf("notifyEmails = %q, want empty string", form.notifyEmails)
	}
}

func TestWebhookFormEditWithSingleNotificationEmail(t *testing.T) {
	app := createTestApp(t)

	webhook := &domain.Webhook{
		ID:                         "webhook-123",
		WebhookURL:                 "https://example.com/webhook",
		TriggerTypes:               []string{domain.TriggerMessageCreated},
		NotificationEmailAddresses: []string{"admin@example.com"},
		Status:                     "active",
	}

	form := NewWebhookForm(app, webhook, nil, nil)

	if form.notifyEmails != "admin@example.com" {
		t.Errorf("notifyEmails = %q, want 'admin@example.com'", form.notifyEmails)
	}
}
