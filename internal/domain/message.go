package domain

import "time"

// Message represents an email message from Nylas.
type Message struct {
	ID          string             `json:"id"`
	GrantID     string             `json:"grant_id"`
	ThreadID    string             `json:"thread_id,omitempty"`
	Subject     string             `json:"subject"`
	From        []EmailParticipant `json:"from"`
	To          []EmailParticipant `json:"to,omitempty"`
	Cc          []EmailParticipant `json:"cc,omitempty"`
	Bcc         []EmailParticipant `json:"bcc,omitempty"`
	ReplyTo     []EmailParticipant `json:"reply_to,omitempty"`
	Body        string             `json:"body"`
	Snippet     string             `json:"snippet"`
	Date        time.Time          `json:"date"`
	Unread      bool               `json:"unread"`
	Starred     bool               `json:"starred"`
	Folders     []string           `json:"folders,omitempty"`
	Attachments []Attachment       `json:"attachments,omitempty"`
	Headers     []Header           `json:"headers,omitempty"`
	Metadata    map[string]string  `json:"metadata,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	Object      string             `json:"object,omitempty"`
}

// Header represents an email header.
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// EmailParticipant represents an email participant (sender/recipient).
// This is an alias for Person, which provides String() and DisplayName() methods.
type EmailParticipant = Person
