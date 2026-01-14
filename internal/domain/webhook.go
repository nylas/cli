package domain

import "time"

// Webhook represents a Nylas webhook subscription.
type Webhook struct {
	ID                         string    `json:"id"`
	Description                string    `json:"description,omitempty"`
	TriggerTypes               []string  `json:"trigger_types"`
	WebhookURL                 string    `json:"webhook_url"`
	WebhookSecret              string    `json:"webhook_secret,omitempty"`
	Status                     string    `json:"status"` // active, inactive, failing
	NotificationEmailAddresses []string  `json:"notification_email_addresses,omitempty"`
	StatusUpdatedAt            time.Time `json:"status_updated_at,omitempty"`
	CreatedAt                  time.Time `json:"created_at,omitempty"`
	UpdatedAt                  time.Time `json:"updated_at,omitempty"`
}

// CreateWebhookRequest for creating a new webhook.
type CreateWebhookRequest struct {
	TriggerTypes               []string `json:"trigger_types"`
	WebhookURL                 string   `json:"webhook_url"`
	Description                string   `json:"description,omitempty"`
	NotificationEmailAddresses []string `json:"notification_email_addresses,omitempty"`
}

// UpdateWebhookRequest for updating a webhook.
type UpdateWebhookRequest struct {
	TriggerTypes               []string `json:"trigger_types,omitempty"`
	WebhookURL                 string   `json:"webhook_url,omitempty"`
	Description                string   `json:"description,omitempty"`
	NotificationEmailAddresses []string `json:"notification_email_addresses,omitempty"`
	Status                     string   `json:"status,omitempty"` // active, inactive
}

// WebhookTestRequest for sending a test webhook event.
type WebhookTestRequest struct {
	WebhookURL string `json:"webhook_url"`
}

// WebhookMockPayloadRequest for getting a mock payload.
type WebhookMockPayloadRequest struct {
	TriggerType string `json:"trigger_type"`
}

// WebhookListResponse represents a paginated webhook list.
type WebhookListResponse struct {
	Data       []Webhook  `json:"data"`
	Pagination Pagination `json:"pagination,omitempty"`
}

// Common webhook trigger types.
const (
	// Grant triggers
	TriggerGrantCreated          = "grant.created"
	TriggerGrantDeleted          = "grant.deleted"
	TriggerGrantExpired          = "grant.expired"
	TriggerGrantUpdated          = "grant.updated"
	TriggerGrantIMAPSyncComplete = "grant.imap_sync_completed"

	// Message triggers
	TriggerMessageCreated         = "message.created"
	TriggerMessageUpdated         = "message.updated"
	TriggerMessageOpened          = "message.opened"
	TriggerMessageBounceDetected  = "message.bounce_detected"
	TriggerMessageSendSuccess     = "message.send_success"
	TriggerMessageSendFailed      = "message.send_failed"
	TriggerMessageOpenedTruncated = "message.opened.truncated"
	TriggerMessageLinkClicked     = "message.link_clicked"

	// Thread triggers
	TriggerThreadReplied = "thread.replied"

	// Event triggers
	TriggerEventCreated = "event.created"
	TriggerEventUpdated = "event.updated"
	TriggerEventDeleted = "event.deleted"

	// Contact triggers
	TriggerContactCreated = "contact.created"
	TriggerContactUpdated = "contact.updated"
	TriggerContactDeleted = "contact.deleted"

	// Calendar triggers
	TriggerCalendarCreated = "calendar.created"
	TriggerCalendarUpdated = "calendar.updated"
	TriggerCalendarDeleted = "calendar.deleted"

	// Folder triggers
	TriggerFolderCreated = "folder.created"
	TriggerFolderUpdated = "folder.updated"
	TriggerFolderDeleted = "folder.deleted"

	// Notetaker triggers
	TriggerNotetakerMedia = "notetaker.media"
)

// AllTriggerTypes returns all available trigger types.
func AllTriggerTypes() []string {
	return []string{
		// Grant
		TriggerGrantCreated,
		TriggerGrantDeleted,
		TriggerGrantExpired,
		TriggerGrantUpdated,
		TriggerGrantIMAPSyncComplete,
		// Message
		TriggerMessageCreated,
		TriggerMessageUpdated,
		TriggerMessageOpened,
		TriggerMessageBounceDetected,
		TriggerMessageSendSuccess,
		TriggerMessageSendFailed,
		TriggerMessageOpenedTruncated,
		TriggerMessageLinkClicked,
		// Thread
		TriggerThreadReplied,
		// Event
		TriggerEventCreated,
		TriggerEventUpdated,
		TriggerEventDeleted,
		// Contact
		TriggerContactCreated,
		TriggerContactUpdated,
		TriggerContactDeleted,
		// Calendar
		TriggerCalendarCreated,
		TriggerCalendarUpdated,
		TriggerCalendarDeleted,
		// Folder
		TriggerFolderCreated,
		TriggerFolderUpdated,
		TriggerFolderDeleted,
		// Notetaker
		TriggerNotetakerMedia,
	}
}

// TriggerTypeCategories returns trigger types grouped by category.
func TriggerTypeCategories() map[string][]string {
	return map[string][]string{
		"grant": {
			TriggerGrantCreated,
			TriggerGrantDeleted,
			TriggerGrantExpired,
			TriggerGrantUpdated,
			TriggerGrantIMAPSyncComplete,
		},
		"message": {
			TriggerMessageCreated,
			TriggerMessageUpdated,
			TriggerMessageOpened,
			TriggerMessageBounceDetected,
			TriggerMessageSendSuccess,
			TriggerMessageSendFailed,
			TriggerMessageOpenedTruncated,
			TriggerMessageLinkClicked,
		},
		"thread": {
			TriggerThreadReplied,
		},
		"event": {
			TriggerEventCreated,
			TriggerEventUpdated,
			TriggerEventDeleted,
		},
		"contact": {
			TriggerContactCreated,
			TriggerContactUpdated,
			TriggerContactDeleted,
		},
		"calendar": {
			TriggerCalendarCreated,
			TriggerCalendarUpdated,
			TriggerCalendarDeleted,
		},
		"folder": {
			TriggerFolderCreated,
			TriggerFolderUpdated,
			TriggerFolderDeleted,
		},
		"notetaker": {
			TriggerNotetakerMedia,
		},
	}
}
