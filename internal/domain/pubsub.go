package domain

import "time"

// PubSubChannel represents a Nylas Pub/Sub notification channel.
type PubSubChannel struct {
	ID                         string    `json:"id"`
	Description                string    `json:"description,omitempty"`
	TriggerTypes               []string  `json:"trigger_types"`
	Topic                      string    `json:"topic"`
	EncryptionKey              string    `json:"encryption_key,omitempty"`
	Status                     string    `json:"status,omitempty"`
	NotificationEmailAddresses []string  `json:"notification_email_addresses,omitempty"`
	CreatedAt                  time.Time `json:"created_at,omitempty"`
	UpdatedAt                  time.Time `json:"updated_at,omitempty"`
	Object                     string    `json:"object,omitempty"`
}

// QuietField returns the field used by quiet output mode.
func (c PubSubChannel) QuietField() string {
	return c.ID
}

// PubSubChannelListResponse contains Pub/Sub channels for an application.
type PubSubChannelListResponse struct {
	Data       []PubSubChannel `json:"data"`
	NextCursor string          `json:"next_cursor,omitempty"`
	RequestID  string          `json:"request_id,omitempty"`
}

// CreatePubSubChannelRequest creates a Pub/Sub notification channel.
type CreatePubSubChannelRequest struct {
	TriggerTypes               []string `json:"trigger_types"`
	Topic                      string   `json:"topic"`
	Description                string   `json:"description,omitempty"`
	EncryptionKey              string   `json:"encryption_key,omitempty"`
	NotificationEmailAddresses []string `json:"notification_email_addresses,omitempty"`
}

// UpdatePubSubChannelRequest updates a Pub/Sub notification channel.
type UpdatePubSubChannelRequest struct {
	TriggerTypes               []string `json:"trigger_types,omitempty"`
	Topic                      string   `json:"topic,omitempty"`
	Description                string   `json:"description,omitempty"`
	EncryptionKey              string   `json:"encryption_key,omitempty"`
	NotificationEmailAddresses []string `json:"notification_email_addresses,omitempty"`
	Status                     string   `json:"status,omitempty"`
}
