package domain

import "time"

// Thread represents an email thread/conversation.
type Thread struct {
	ID                    string             `json:"id"`
	GrantID               string             `json:"grant_id"`
	LatestDraftOrMessage  Message            `json:"latest_draft_or_message,omitempty"`
	HasAttachments        bool               `json:"has_attachments"`
	HasDrafts             bool               `json:"has_drafts"`
	Starred               bool               `json:"starred"`
	Unread                bool               `json:"unread"`
	EarliestMessageDate   time.Time          `json:"earliest_message_date"`
	LatestMessageRecvDate time.Time          `json:"latest_message_received_date"`
	LatestMessageSentDate time.Time          `json:"latest_message_sent_date"`
	Participants          []EmailParticipant `json:"participants"`
	MessageIDs            []string           `json:"message_ids"`
	DraftIDs              []string           `json:"draft_ids"`
	FolderIDs             []string           `json:"folders"`
	Snippet               string             `json:"snippet"`
	Subject               string             `json:"subject"`
}

// Draft represents an email draft.
type Draft struct {
	ID           string             `json:"id"`
	GrantID      string             `json:"grant_id"`
	Subject      string             `json:"subject"`
	Body         string             `json:"body"`
	From         []EmailParticipant `json:"from"`
	To           []EmailParticipant `json:"to"`
	Cc           []EmailParticipant `json:"cc,omitempty"`
	Bcc          []EmailParticipant `json:"bcc,omitempty"`
	ReplyTo      []EmailParticipant `json:"reply_to,omitempty"`
	ReplyToMsgID string             `json:"reply_to_message_id,omitempty"`
	ThreadID     string             `json:"thread_id,omitempty"`
	Attachments  []Attachment       `json:"attachments,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// Folder represents an email folder/label.
type Folder struct {
	ID              string   `json:"id"`
	GrantID         string   `json:"grant_id"`
	Name            string   `json:"name"`
	SystemFolder    string   `json:"system_folder,omitempty"`
	ParentID        string   `json:"parent_id,omitempty"`
	BackgroundColor string   `json:"background_color,omitempty"`
	TextColor       string   `json:"text_color,omitempty"`
	TotalCount      int      `json:"total_count"`
	UnreadCount     int      `json:"unread_count"`
	ChildIDs        []string `json:"child_ids,omitempty"`
	Attributes      []string `json:"attributes,omitempty"`
}

// SystemFolder constants for common folder types.
const (
	FolderInbox   = "inbox"
	FolderSent    = "sent"
	FolderDrafts  = "drafts"
	FolderTrash   = "trash"
	FolderSpam    = "spam"
	FolderArchive = "archive"
	FolderAll     = "all"
)

// Attachment represents an email attachment.
type Attachment struct {
	ID          string `json:"id,omitempty"`
	GrantID     string `json:"grant_id,omitempty"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	ContentID   string `json:"content_id,omitempty"`
	IsInline    bool   `json:"is_inline,omitempty"`
	Content     []byte `json:"-"` // Binary content, not serialized to JSON
}

// SendMessageRequest represents a request to send an email.
type SendMessageRequest struct {
	Subject      string             `json:"subject"`
	Body         string             `json:"body"`
	From         []EmailParticipant `json:"from,omitempty"`
	To           []EmailParticipant `json:"to"`
	Cc           []EmailParticipant `json:"cc,omitempty"`
	Bcc          []EmailParticipant `json:"bcc,omitempty"`
	ReplyTo      []EmailParticipant `json:"reply_to,omitempty"`
	ReplyToMsgID string             `json:"reply_to_message_id,omitempty"`
	TrackingOpts *TrackingOptions   `json:"tracking_options,omitempty"`
	Attachments  []Attachment       `json:"attachments,omitempty"`
	SendAt       int64              `json:"send_at,omitempty"` // Unix timestamp for scheduled sending
	Metadata     map[string]string  `json:"metadata,omitempty"`
}

// Validate checks that SendMessageRequest has at least one recipient.
func (r SendMessageRequest) Validate() error {
	if len(r.To) == 0 && len(r.Cc) == 0 && len(r.Bcc) == 0 {
		return ErrInvalidInput
	}
	return nil
}

// ScheduledMessage represents a scheduled email.
type ScheduledMessage struct {
	ScheduleID string `json:"schedule_id"`
	Status     string `json:"status"` // pending, scheduled, sending, sent, failed, cancelled
	CloseTime  int64  `json:"close_time"`
}

// ScheduledMessageListResponse represents a list of scheduled messages.
type ScheduledMessageListResponse struct {
	Data []ScheduledMessage `json:"data"`
}

// TrackingOptions for email tracking.
type TrackingOptions struct {
	Opens bool   `json:"opens"`
	Links bool   `json:"links"`
	Label string `json:"label,omitempty"`
}

// MessageQueryParams for filtering messages.
type MessageQueryParams struct {
	Limit          int      `json:"limit,omitempty"`
	Offset         int      `json:"offset,omitempty"`
	PageToken      string   `json:"page_token,omitempty"` // Cursor for pagination
	Subject        string   `json:"subject,omitempty"`
	From           string   `json:"from,omitempty"`
	To             string   `json:"to,omitempty"`
	Cc             string   `json:"cc,omitempty"`
	Bcc            string   `json:"bcc,omitempty"`
	In             []string `json:"in,omitempty"` // Folder IDs
	Unread         *bool    `json:"unread,omitempty"`
	Starred        *bool    `json:"starred,omitempty"`
	ThreadID       string   `json:"thread_id,omitempty"`
	ReceivedBefore int64    `json:"received_before,omitempty"`
	ReceivedAfter  int64    `json:"received_after,omitempty"`
	HasAttachment  *bool    `json:"has_attachment,omitempty"`
	SearchQuery    string   `json:"q,omitempty"`             // Full-text search
	Fields         string   `json:"fields,omitempty"`        // e.g., "include_headers"
	MetadataPair   string   `json:"metadata_pair,omitempty"` // Metadata filtering (format: "key:value", only key1-key5 supported)
}

// ThreadQueryParams for filtering threads.
type ThreadQueryParams struct {
	Limit           int      `json:"limit,omitempty"`
	Offset          int      `json:"offset,omitempty"`
	PageToken       string   `json:"page_token,omitempty"` // Cursor for pagination
	Subject         string   `json:"subject,omitempty"`
	From            string   `json:"from,omitempty"`
	To              string   `json:"to,omitempty"`
	In              []string `json:"in,omitempty"`
	Unread          *bool    `json:"unread,omitempty"`
	Starred         *bool    `json:"starred,omitempty"`
	LatestMsgBefore int64    `json:"latest_message_before,omitempty"`
	LatestMsgAfter  int64    `json:"latest_message_after,omitempty"`
	HasAttachment   *bool    `json:"has_attachment,omitempty"`
	SearchQuery     string   `json:"q,omitempty"`
}

// UpdateMessageRequest for updating message properties.
type UpdateMessageRequest struct {
	Unread  *bool    `json:"unread,omitempty"`
	Starred *bool    `json:"starred,omitempty"`
	Folders []string `json:"folders,omitempty"`
}

// CreateDraftRequest for creating a new draft.
type CreateDraftRequest struct {
	Subject      string             `json:"subject"`
	Body         string             `json:"body"`
	To           []EmailParticipant `json:"to,omitempty"`
	Cc           []EmailParticipant `json:"cc,omitempty"`
	Bcc          []EmailParticipant `json:"bcc,omitempty"`
	ReplyTo      []EmailParticipant `json:"reply_to,omitempty"`
	ReplyToMsgID string             `json:"reply_to_message_id,omitempty"`
	Attachments  []Attachment       `json:"attachments,omitempty"`
	Metadata     map[string]string  `json:"metadata,omitempty"`
}

// CreateFolderRequest for creating a new folder.
type CreateFolderRequest struct {
	Name            string `json:"name"`
	ParentID        string `json:"parent_id,omitempty"`
	BackgroundColor string `json:"background_color,omitempty"`
	TextColor       string `json:"text_color,omitempty"`
}

// UpdateFolderRequest for updating a folder.
type UpdateFolderRequest struct {
	Name            string `json:"name,omitempty"`
	ParentID        string `json:"parent_id,omitempty"`
	BackgroundColor string `json:"background_color,omitempty"`
	TextColor       string `json:"text_color,omitempty"`
}

// Pagination represents pagination info in API responses.
type Pagination struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// MessageListResponse represents a paginated message list response.
type MessageListResponse struct {
	Data       []Message  `json:"data"`
	Pagination Pagination `json:"pagination,omitempty"`
}

// ThreadListResponse represents a paginated thread list response.
type ThreadListResponse struct {
	Data       []Thread   `json:"data"`
	Pagination Pagination `json:"pagination,omitempty"`
}

// FolderListResponse represents a paginated folder list response.
type FolderListResponse struct {
	Data       []Folder   `json:"data"`
	Pagination Pagination `json:"pagination,omitempty"`
}

// DraftListResponse represents a paginated draft list response.
type DraftListResponse struct {
	Data       []Draft    `json:"data"`
	Pagination Pagination `json:"pagination,omitempty"`
}

// SmartComposeRequest represents a request to generate an AI email draft.
type SmartComposeRequest struct {
	Prompt string `json:"prompt"` // AI instruction (max 1000 tokens)
}

// SmartComposeSuggestion represents an AI-generated email suggestion.
type SmartComposeSuggestion struct {
	Suggestion string `json:"suggestion"` // The generated email text
}

// TrackingData represents tracking statistics for a message.
type TrackingData struct {
	MessageID string       `json:"message_id"`
	Opens     []OpenEvent  `json:"opens,omitempty"`
	Clicks    []ClickEvent `json:"clicks,omitempty"`
	Replies   []ReplyEvent `json:"replies,omitempty"`
}

// OpenEvent represents an email open tracking event.
type OpenEvent struct {
	OpenedID  string    `json:"opened_id"`
	Timestamp time.Time `json:"timestamp"`
	IPAddress string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
}

// ClickEvent represents a link click tracking event.
type ClickEvent struct {
	ClickID   string    `json:"click_id"`
	Timestamp time.Time `json:"timestamp"`
	URL       string    `json:"url"`
	IPAddress string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	LinkIndex int       `json:"link_index"`
}

// ReplyEvent represents a reply tracking event.
type ReplyEvent struct {
	MessageID     string    `json:"message_id"`
	Timestamp     time.Time `json:"timestamp"`
	ThreadID      string    `json:"thread_id,omitempty"`
	RootMessageID string    `json:"root_message_id,omitempty"`
}
