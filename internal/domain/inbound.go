package domain

// InboundInbox represents a Nylas Inbound inbox (a grant with provider=inbox).
// Inbound inboxes receive emails at managed addresses without OAuth.
type InboundInbox struct {
	ID          string   `json:"id"`           // Grant ID
	Email       string   `json:"email"`        // Full email address (e.g., info@app.nylas.email)
	GrantStatus string   `json:"grant_status"` // Status of the inbox
	CreatedAt   UnixTime `json:"created_at"`
	UpdatedAt   UnixTime `json:"updated_at"`
}

// IsValid returns true if the inbound inbox is in a valid state.
func (i *InboundInbox) IsValid() bool {
	return i.GrantStatus == "valid"
}

// CreateInboundInboxRequest represents a request to create a new inbound inbox.
type CreateInboundInboxRequest struct {
	// Email is the local part of the email address (before @).
	// The full address will be: {email}@{your-app}.nylas.email
	Email string `json:"email"`
}

// InboundMessage represents an email received at an inbound inbox.
// This is an alias for Message but provides semantic clarity.
type InboundMessage = Message

// InboundWebhookEvent contains metadata specific to inbound webhook events.
type InboundWebhookEvent struct {
	Type      string          `json:"type"`       // e.g., "message.created"
	Source    string          `json:"source"`     // "inbox" for inbound emails
	GrantID   string          `json:"grant_id"`   // The inbound inbox grant ID
	MessageID string          `json:"message_id"` // The message ID
	Message   *InboundMessage `json:"message"`    // The message object (if included)
}

// IsInboundEvent returns true if the event is from an inbound inbox.
func (e *InboundWebhookEvent) IsInboundEvent() bool {
	return e.Source == "inbox"
}
