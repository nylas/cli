package air

import (
	"io"
	"net/http"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/httputil"
)

// Grant represents an authenticated account for API responses.
type Grant struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Provider string `json:"provider"`
}

// grantFromDomain converts a domain.GrantInfo to a Grant for API responses.
func grantFromDomain(g domain.GrantInfo) Grant {
	return Grant{
		ID:       g.ID,
		Email:    g.Email,
		Provider: string(g.Provider),
	}
}

// ConfigStatusResponse represents the config status API response.
type ConfigStatusResponse struct {
	Configured   bool   `json:"configured"`
	Region       string `json:"region"`
	ClientID     string `json:"client_id,omitempty"`
	HasAPIKey    bool   `json:"has_api_key"`
	GrantCount   int    `json:"grant_count"`
	DefaultGrant string `json:"default_grant,omitempty"`
}

// GrantsResponse represents the grants list API response.
type GrantsResponse struct {
	Grants       []Grant `json:"grants"`
	DefaultGrant string  `json:"default_grant"`
}

// SetDefaultGrantRequest represents the request to set default grant.
type SetDefaultGrantRequest struct {
	GrantID string `json:"grant_id"`
}

// SetDefaultGrantResponse represents the response for setting default grant.
type SetDefaultGrantResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// FolderResponse represents a folder in API responses.
type FolderResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SystemFolder string `json:"system_folder,omitempty"`
	TotalCount   int    `json:"total_count"`
	UnreadCount  int    `json:"unread_count"`
}

// FoldersResponse represents the folders list API response.
type FoldersResponse struct {
	Folders []FolderResponse `json:"folders"`
}

// PoliciesResponse represents the policy list API response.
type PoliciesResponse struct {
	Policies []domain.Policy `json:"policies"`
}

// RulesResponse represents the rule list API response.
type RulesResponse struct {
	Rules []domain.Rule `json:"rules"`
}

// EmailParticipantResponse represents an email participant.
type EmailParticipantResponse struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// AttachmentResponse represents an email attachment.
type AttachmentResponse struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

// EmailResponse represents an email in API responses.
type EmailResponse struct {
	ID          string                     `json:"id"`
	ThreadID    string                     `json:"thread_id,omitempty"`
	Subject     string                     `json:"subject"`
	Snippet     string                     `json:"snippet"`
	Body        string                     `json:"body,omitempty"`
	From        []EmailParticipantResponse `json:"from"`
	To          []EmailParticipantResponse `json:"to,omitempty"`
	Cc          []EmailParticipantResponse `json:"cc,omitempty"`
	Date        int64                      `json:"date"` // Unix timestamp
	Unread      bool                       `json:"unread"`
	Starred     bool                       `json:"starred"`
	Folders     []string                   `json:"folders,omitempty"`
	Attachments []AttachmentResponse       `json:"attachments,omitempty"`
}

// EmailsResponse represents the emails list API response.
type EmailsResponse struct {
	Emails     []EmailResponse `json:"emails"`
	NextCursor string          `json:"next_cursor,omitempty"`
	HasMore    bool            `json:"has_more"`
}

// UpdateEmailRequest represents a request to update an email.
type UpdateEmailRequest struct {
	Unread  *bool    `json:"unread,omitempty"`
	Starred *bool    `json:"starred,omitempty"`
	Folders []string `json:"folders,omitempty"`
}

// UpdateEmailResponse represents the response for updating an email.
type UpdateEmailResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// DraftRequest represents a request to create or update a draft.
type DraftRequest struct {
	To           []EmailParticipantResponse `json:"to"`
	Cc           []EmailParticipantResponse `json:"cc,omitempty"`
	Bcc          []EmailParticipantResponse `json:"bcc,omitempty"`
	Subject      string                     `json:"subject"`
	Body         string                     `json:"body"`
	ReplyToMsgID string                     `json:"reply_to_message_id,omitempty"`
}

// DraftResponse represents a draft in API responses.
type DraftResponse struct {
	ID      string                     `json:"id"`
	Subject string                     `json:"subject"`
	Body    string                     `json:"body,omitempty"`
	To      []EmailParticipantResponse `json:"to,omitempty"`
	Cc      []EmailParticipantResponse `json:"cc,omitempty"`
	Bcc     []EmailParticipantResponse `json:"bcc,omitempty"`
	Date    int64                      `json:"date"`
}

// DraftsResponse represents the drafts list API response.
type DraftsResponse struct {
	Drafts []DraftResponse `json:"drafts"`
}

// SendMessageRequest represents a request to send a message directly.
type SendMessageRequest struct {
	To           []EmailParticipantResponse `json:"to"`
	Cc           []EmailParticipantResponse `json:"cc,omitempty"`
	Bcc          []EmailParticipantResponse `json:"bcc,omitempty"`
	Subject      string                     `json:"subject"`
	Body         string                     `json:"body"`
	ReplyToMsgID string                     `json:"reply_to_message_id,omitempty"`
}

// SendMessageResponse represents the response for sending a message.
type SendMessageResponse struct {
	Success   bool   `json:"success"`
	MessageID string `json:"message_id,omitempty"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
}

// CalendarResponse represents a calendar in API responses.
type CalendarResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	IsPrimary   bool   `json:"is_primary"`
	ReadOnly    bool   `json:"read_only"`
	HexColor    string `json:"hex_color,omitempty"`
}

// CalendarsResponse represents the calendars list API response.
type CalendarsResponse struct {
	Calendars []CalendarResponse `json:"calendars"`
}

// EventParticipantResponse represents an event participant.
type EventParticipantResponse struct {
	Name   string `json:"name,omitempty"`
	Email  string `json:"email"`
	Status string `json:"status,omitempty"`
}

// ConferencingResponse represents video conferencing details.
type ConferencingResponse struct {
	Provider string `json:"provider,omitempty"`
	URL      string `json:"url,omitempty"`
}

// EventResponse represents an event in API responses.
type EventResponse struct {
	ID           string                     `json:"id"`
	CalendarID   string                     `json:"calendar_id"`
	Title        string                     `json:"title"`
	Description  string                     `json:"description,omitempty"`
	Location     string                     `json:"location,omitempty"`
	StartTime    int64                      `json:"start_time"`
	EndTime      int64                      `json:"end_time"`
	Timezone     string                     `json:"timezone,omitempty"`
	IsAllDay     bool                       `json:"is_all_day"`
	Status       string                     `json:"status,omitempty"`
	Busy         bool                       `json:"busy"`
	Participants []EventParticipantResponse `json:"participants,omitempty"`
	Conferencing *ConferencingResponse      `json:"conferencing,omitempty"`
	HtmlLink     string                     `json:"html_link,omitempty"`
}

// EventsResponse represents the events list API response.
type EventsResponse struct {
	Events     []EventResponse `json:"events"`
	NextCursor string          `json:"next_cursor,omitempty"`
	HasMore    bool            `json:"has_more"`
}

// CreateEventRequest represents a request to create an event.
type CreateEventRequest struct {
	CalendarID   string                     `json:"calendar_id"`
	Title        string                     `json:"title"`
	Description  string                     `json:"description,omitempty"`
	Location     string                     `json:"location,omitempty"`
	StartTime    int64                      `json:"start_time"`
	EndTime      int64                      `json:"end_time"`
	Timezone     string                     `json:"timezone,omitempty"`
	IsAllDay     bool                       `json:"is_all_day"`
	Busy         bool                       `json:"busy"`
	Participants []EventParticipantResponse `json:"participants,omitempty"`
}

// UpdateEventRequest represents a request to update an event.
type UpdateEventRequest struct {
	Title        *string                    `json:"title,omitempty"`
	Description  *string                    `json:"description,omitempty"`
	Location     *string                    `json:"location,omitempty"`
	StartTime    *int64                     `json:"start_time,omitempty"`
	EndTime      *int64                     `json:"end_time,omitempty"`
	Timezone     *string                    `json:"timezone,omitempty"`
	IsAllDay     *bool                      `json:"is_all_day,omitempty"`
	Busy         *bool                      `json:"busy,omitempty"`
	Participants []EventParticipantResponse `json:"participants,omitempty"`
}

// EventActionResponse represents the response for event actions (create/update/delete).
type EventActionResponse struct {
	Success bool           `json:"success"`
	Event   *EventResponse `json:"event,omitempty"`
	Message string         `json:"message,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// ====================================
// CONTACTS TYPES
// ====================================

// ContactEmailResponse represents a contact email in API responses.
type ContactEmailResponse struct {
	Email string `json:"email"`
	Type  string `json:"type,omitempty"`
}

// ContactPhoneResponse represents a contact phone number in API responses.
type ContactPhoneResponse struct {
	Number string `json:"number"`
	Type   string `json:"type,omitempty"`
}

// ContactAddressResponse represents a contact address in API responses.
type ContactAddressResponse struct {
	Type          string `json:"type,omitempty"`
	StreetAddress string `json:"street_address,omitempty"`
	City          string `json:"city,omitempty"`
	State         string `json:"state,omitempty"`
	PostalCode    string `json:"postal_code,omitempty"`
	Country       string `json:"country,omitempty"`
}

// ContactResponse represents a contact in API responses.
type ContactResponse struct {
	ID           string                   `json:"id"`
	GivenName    string                   `json:"given_name,omitempty"`
	Surname      string                   `json:"surname,omitempty"`
	DisplayName  string                   `json:"display_name"`
	Nickname     string                   `json:"nickname,omitempty"`
	CompanyName  string                   `json:"company_name,omitempty"`
	JobTitle     string                   `json:"job_title,omitempty"`
	Birthday     string                   `json:"birthday,omitempty"`
	Notes        string                   `json:"notes,omitempty"`
	PictureURL   string                   `json:"picture_url,omitempty"`
	Emails       []ContactEmailResponse   `json:"emails,omitempty"`
	PhoneNumbers []ContactPhoneResponse   `json:"phone_numbers,omitempty"`
	Addresses    []ContactAddressResponse `json:"addresses,omitempty"`
	Source       string                   `json:"source,omitempty"`
}

// ContactsResponse represents the contacts list API response.
type ContactsResponse struct {
	Contacts   []ContactResponse `json:"contacts"`
	NextCursor string            `json:"next_cursor,omitempty"`
	HasMore    bool              `json:"has_more"`
}

// CreateContactRequest represents a request to create a contact.
type CreateContactRequest struct {
	GivenName    string                   `json:"given_name,omitempty"`
	Surname      string                   `json:"surname,omitempty"`
	Nickname     string                   `json:"nickname,omitempty"`
	CompanyName  string                   `json:"company_name,omitempty"`
	JobTitle     string                   `json:"job_title,omitempty"`
	Birthday     string                   `json:"birthday,omitempty"`
	Notes        string                   `json:"notes,omitempty"`
	Emails       []ContactEmailResponse   `json:"emails,omitempty"`
	PhoneNumbers []ContactPhoneResponse   `json:"phone_numbers,omitempty"`
	Addresses    []ContactAddressResponse `json:"addresses,omitempty"`
}

// UpdateContactRequest represents a request to update a contact.
type UpdateContactRequest struct {
	GivenName    *string                  `json:"given_name,omitempty"`
	Surname      *string                  `json:"surname,omitempty"`
	Nickname     *string                  `json:"nickname,omitempty"`
	CompanyName  *string                  `json:"company_name,omitempty"`
	JobTitle     *string                  `json:"job_title,omitempty"`
	Birthday     *string                  `json:"birthday,omitempty"`
	Notes        *string                  `json:"notes,omitempty"`
	Emails       []ContactEmailResponse   `json:"emails,omitempty"`
	PhoneNumbers []ContactPhoneResponse   `json:"phone_numbers,omitempty"`
	Addresses    []ContactAddressResponse `json:"addresses,omitempty"`
}

// ContactActionResponse represents the response for contact actions (create/update/delete).
type ContactActionResponse struct {
	Success bool             `json:"success"`
	Contact *ContactResponse `json:"contact,omitempty"`
	Message string           `json:"message,omitempty"`
	Error   string           `json:"error,omitempty"`
}

// ContactGroupResponse represents a contact group in API responses.
type ContactGroupResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
}

// ContactGroupsResponse represents the contact groups list API response.
type ContactGroupsResponse struct {
	Groups []ContactGroupResponse `json:"groups"`
}

// limitedBody wraps a request body with a size limit.
func limitedBody(w http.ResponseWriter, r *http.Request) io.ReadCloser {
	return httputil.LimitedBody(w, r, httputil.MaxRequestBodySize)
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data any) {
	httputil.WriteJSON(w, status, data)
}
