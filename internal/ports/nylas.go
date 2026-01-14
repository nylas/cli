package ports

import (
	"context"
	"io"

	"github.com/nylas/cli/internal/domain"
)

// NylasClient defines the interface for interacting with the Nylas API.
type NylasClient interface {
	// ================================
	// AUTH OPERATIONS
	// ================================

	BuildAuthURL(provider domain.Provider, redirectURI string) string
	ExchangeCode(ctx context.Context, code, redirectURI string) (*domain.Grant, error)
	ListGrants(ctx context.Context) ([]domain.Grant, error)
	GetGrant(ctx context.Context, grantID string) (*domain.Grant, error)
	RevokeGrant(ctx context.Context, grantID string) error

	// ================================
	// MESSAGE OPERATIONS
	// ================================

	GetMessages(ctx context.Context, grantID string, limit int) ([]domain.Message, error)
	GetMessagesWithParams(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error)
	GetMessagesWithCursor(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error)
	GetMessage(ctx context.Context, grantID, messageID string) (*domain.Message, error)
	SendMessage(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error)
	UpdateMessage(ctx context.Context, grantID, messageID string, req *domain.UpdateMessageRequest) (*domain.Message, error)
	DeleteMessage(ctx context.Context, grantID, messageID string) error

	// ================================
	// SCHEDULED MESSAGE OPERATIONS
	// ================================

	ListScheduledMessages(ctx context.Context, grantID string) ([]domain.ScheduledMessage, error)
	GetScheduledMessage(ctx context.Context, grantID, scheduleID string) (*domain.ScheduledMessage, error)
	CancelScheduledMessage(ctx context.Context, grantID, scheduleID string) error

	// ================================
	// SMART COMPOSE OPERATIONS
	// ================================

	SmartCompose(ctx context.Context, grantID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error)
	SmartComposeReply(ctx context.Context, grantID, messageID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error)

	// ================================
	// THREAD OPERATIONS
	// ================================

	GetThreads(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error)
	GetThread(ctx context.Context, grantID, threadID string) (*domain.Thread, error)
	UpdateThread(ctx context.Context, grantID, threadID string, req *domain.UpdateMessageRequest) (*domain.Thread, error)
	DeleteThread(ctx context.Context, grantID, threadID string) error

	// ================================
	// DRAFT OPERATIONS
	// ================================

	GetDrafts(ctx context.Context, grantID string, limit int) ([]domain.Draft, error)
	GetDraft(ctx context.Context, grantID, draftID string) (*domain.Draft, error)
	CreateDraft(ctx context.Context, grantID string, req *domain.CreateDraftRequest) (*domain.Draft, error)
	UpdateDraft(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error)
	DeleteDraft(ctx context.Context, grantID, draftID string) error
	SendDraft(ctx context.Context, grantID, draftID string) (*domain.Message, error)

	// ================================
	// FOLDER OPERATIONS
	// ================================

	GetFolders(ctx context.Context, grantID string) ([]domain.Folder, error)
	GetFolder(ctx context.Context, grantID, folderID string) (*domain.Folder, error)
	CreateFolder(ctx context.Context, grantID string, req *domain.CreateFolderRequest) (*domain.Folder, error)
	UpdateFolder(ctx context.Context, grantID, folderID string, req *domain.UpdateFolderRequest) (*domain.Folder, error)
	DeleteFolder(ctx context.Context, grantID, folderID string) error

	// ================================
	// ATTACHMENT OPERATIONS
	// ================================

	ListAttachments(ctx context.Context, grantID, messageID string) ([]domain.Attachment, error)
	GetAttachment(ctx context.Context, grantID, messageID, attachmentID string) (*domain.Attachment, error)
	DownloadAttachment(ctx context.Context, grantID, messageID, attachmentID string) (io.ReadCloser, error)

	// ================================
	// CALENDAR OPERATIONS
	// ================================

	GetCalendars(ctx context.Context, grantID string) ([]domain.Calendar, error)
	GetCalendar(ctx context.Context, grantID, calendarID string) (*domain.Calendar, error)
	CreateCalendar(ctx context.Context, grantID string, req *domain.CreateCalendarRequest) (*domain.Calendar, error)
	UpdateCalendar(ctx context.Context, grantID, calendarID string, req *domain.UpdateCalendarRequest) (*domain.Calendar, error)
	DeleteCalendar(ctx context.Context, grantID, calendarID string) error

	// ================================
	// EVENT OPERATIONS
	// ================================

	GetEvents(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error)
	GetEventsWithCursor(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error)
	GetEvent(ctx context.Context, grantID, calendarID, eventID string) (*domain.Event, error)
	CreateEvent(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error)
	UpdateEvent(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error)
	DeleteEvent(ctx context.Context, grantID, calendarID, eventID string) error
	SendRSVP(ctx context.Context, grantID, calendarID, eventID string, req *domain.SendRSVPRequest) error

	// ================================
	// AVAILABILITY OPERATIONS
	// ================================

	GetFreeBusy(ctx context.Context, grantID string, req *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error)
	GetAvailability(ctx context.Context, req *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error)

	// ================================
	// VIRTUAL CALENDAR OPERATIONS
	// ================================

	CreateVirtualCalendarGrant(ctx context.Context, email string) (*domain.VirtualCalendarGrant, error)
	ListVirtualCalendarGrants(ctx context.Context) ([]domain.VirtualCalendarGrant, error)
	GetVirtualCalendarGrant(ctx context.Context, grantID string) (*domain.VirtualCalendarGrant, error)
	DeleteVirtualCalendarGrant(ctx context.Context, grantID string) error

	// ================================
	// RECURRING EVENT OPERATIONS
	// ================================

	GetRecurringEventInstances(ctx context.Context, grantID, calendarID, masterEventID string, params *domain.EventQueryParams) ([]domain.Event, error)
	UpdateRecurringEventInstance(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error)
	DeleteRecurringEventInstance(ctx context.Context, grantID, calendarID, eventID string) error

	// ================================
	// CONTACT OPERATIONS
	// ================================

	GetContacts(ctx context.Context, grantID string, params *domain.ContactQueryParams) ([]domain.Contact, error)
	GetContactsWithCursor(ctx context.Context, grantID string, params *domain.ContactQueryParams) (*domain.ContactListResponse, error)
	GetContact(ctx context.Context, grantID, contactID string) (*domain.Contact, error)
	GetContactWithPicture(ctx context.Context, grantID, contactID string, includePicture bool) (*domain.Contact, error)
	CreateContact(ctx context.Context, grantID string, req *domain.CreateContactRequest) (*domain.Contact, error)
	UpdateContact(ctx context.Context, grantID, contactID string, req *domain.UpdateContactRequest) (*domain.Contact, error)
	DeleteContact(ctx context.Context, grantID, contactID string) error
	GetContactGroups(ctx context.Context, grantID string) ([]domain.ContactGroup, error)
	GetContactGroup(ctx context.Context, grantID, groupID string) (*domain.ContactGroup, error)
	CreateContactGroup(ctx context.Context, grantID string, req *domain.CreateContactGroupRequest) (*domain.ContactGroup, error)
	UpdateContactGroup(ctx context.Context, grantID, groupID string, req *domain.UpdateContactGroupRequest) (*domain.ContactGroup, error)
	DeleteContactGroup(ctx context.Context, grantID, groupID string) error

	// ================================
	// WEBHOOK OPERATIONS (Admin)
	// ================================

	ListWebhooks(ctx context.Context) ([]domain.Webhook, error)
	GetWebhook(ctx context.Context, webhookID string) (*domain.Webhook, error)
	CreateWebhook(ctx context.Context, req *domain.CreateWebhookRequest) (*domain.Webhook, error)
	UpdateWebhook(ctx context.Context, webhookID string, req *domain.UpdateWebhookRequest) (*domain.Webhook, error)
	DeleteWebhook(ctx context.Context, webhookID string) error
	SendWebhookTestEvent(ctx context.Context, webhookURL string) error
	GetWebhookMockPayload(ctx context.Context, triggerType string) (map[string]any, error)

	// ================================
	// NOTETAKER OPERATIONS
	// ================================

	ListNotetakers(ctx context.Context, grantID string, params *domain.NotetakerQueryParams) ([]domain.Notetaker, error)
	GetNotetaker(ctx context.Context, grantID, notetakerID string) (*domain.Notetaker, error)
	CreateNotetaker(ctx context.Context, grantID string, req *domain.CreateNotetakerRequest) (*domain.Notetaker, error)
	DeleteNotetaker(ctx context.Context, grantID, notetakerID string) error
	GetNotetakerMedia(ctx context.Context, grantID, notetakerID string) (*domain.MediaData, error)

	// ================================
	// INBOUND OPERATIONS
	// ================================

	ListInboundInboxes(ctx context.Context) ([]domain.InboundInbox, error)
	GetInboundInbox(ctx context.Context, grantID string) (*domain.InboundInbox, error)
	CreateInboundInbox(ctx context.Context, email string) (*domain.InboundInbox, error)
	DeleteInboundInbox(ctx context.Context, grantID string) error
	GetInboundMessages(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.InboundMessage, error)

	// ================================
	// SCHEDULER OPERATIONS
	// ================================

	// Configurations
	ListSchedulerConfigurations(ctx context.Context) ([]domain.SchedulerConfiguration, error)
	GetSchedulerConfiguration(ctx context.Context, configID string) (*domain.SchedulerConfiguration, error)
	CreateSchedulerConfiguration(ctx context.Context, req *domain.CreateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error)
	UpdateSchedulerConfiguration(ctx context.Context, configID string, req *domain.UpdateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error)
	DeleteSchedulerConfiguration(ctx context.Context, configID string) error

	// Sessions
	CreateSchedulerSession(ctx context.Context, req *domain.CreateSchedulerSessionRequest) (*domain.SchedulerSession, error)
	GetSchedulerSession(ctx context.Context, sessionID string) (*domain.SchedulerSession, error)

	// Bookings
	ListBookings(ctx context.Context, configID string) ([]domain.Booking, error)
	GetBooking(ctx context.Context, bookingID string) (*domain.Booking, error)
	ConfirmBooking(ctx context.Context, bookingID string, req *domain.ConfirmBookingRequest) (*domain.Booking, error)
	RescheduleBooking(ctx context.Context, bookingID string, req *domain.RescheduleBookingRequest) (*domain.Booking, error)
	CancelBooking(ctx context.Context, bookingID string, reason string) error

	// Scheduler Pages
	ListSchedulerPages(ctx context.Context) ([]domain.SchedulerPage, error)
	GetSchedulerPage(ctx context.Context, pageID string) (*domain.SchedulerPage, error)
	CreateSchedulerPage(ctx context.Context, req *domain.CreateSchedulerPageRequest) (*domain.SchedulerPage, error)
	UpdateSchedulerPage(ctx context.Context, pageID string, req *domain.UpdateSchedulerPageRequest) (*domain.SchedulerPage, error)
	DeleteSchedulerPage(ctx context.Context, pageID string) error

	// ================================
	// ADMIN OPERATIONS
	// ================================

	// Applications
	ListApplications(ctx context.Context) ([]domain.Application, error)
	GetApplication(ctx context.Context, appID string) (*domain.Application, error)
	CreateApplication(ctx context.Context, req *domain.CreateApplicationRequest) (*domain.Application, error)
	UpdateApplication(ctx context.Context, appID string, req *domain.UpdateApplicationRequest) (*domain.Application, error)
	DeleteApplication(ctx context.Context, appID string) error

	// Connectors
	ListConnectors(ctx context.Context) ([]domain.Connector, error)
	GetConnector(ctx context.Context, connectorID string) (*domain.Connector, error)
	CreateConnector(ctx context.Context, req *domain.CreateConnectorRequest) (*domain.Connector, error)
	UpdateConnector(ctx context.Context, connectorID string, req *domain.UpdateConnectorRequest) (*domain.Connector, error)
	DeleteConnector(ctx context.Context, connectorID string) error

	// Credentials
	ListCredentials(ctx context.Context, connectorID string) ([]domain.ConnectorCredential, error)
	GetCredential(ctx context.Context, credentialID string) (*domain.ConnectorCredential, error)
	CreateCredential(ctx context.Context, connectorID string, req *domain.CreateCredentialRequest) (*domain.ConnectorCredential, error)
	UpdateCredential(ctx context.Context, credentialID string, req *domain.UpdateCredentialRequest) (*domain.ConnectorCredential, error)
	DeleteCredential(ctx context.Context, credentialID string) error

	// Admin Grant Operations
	ListAllGrants(ctx context.Context, params *domain.GrantsQueryParams) ([]domain.Grant, error)
	GetGrantStats(ctx context.Context) (*domain.GrantStats, error)

	// ================================
	// CONFIGURATION
	// ================================

	SetRegion(region string)
	SetCredentials(clientID, clientSecret, apiKey string)
}

// ============================================================================
// OTHER INTERFACES
// ============================================================================

// OAuthServer defines the interface for the OAuth callback server.
type OAuthServer interface {
	// Start starts the server on the configured port.
	Start() error

	// Stop stops the server.
	Stop() error

	// WaitForCallback waits for the OAuth callback and returns the auth code.
	WaitForCallback(ctx context.Context) (string, error)

	// GetRedirectURI returns the redirect URI for OAuth.
	GetRedirectURI() string
}

// Browser defines the interface for opening URLs in the browser.
type Browser interface {
	// Open opens a URL in the default browser.
	Open(url string) error
}

// GrantStore defines the interface for storing grant information.
type GrantStore interface {
	// SaveGrant saves grant info to storage.
	SaveGrant(info domain.GrantInfo) error

	// GetGrant retrieves grant info by ID.
	GetGrant(grantID string) (*domain.GrantInfo, error)

	// GetGrantByEmail retrieves grant info by email.
	GetGrantByEmail(email string) (*domain.GrantInfo, error)

	// ListGrants returns all stored grants.
	ListGrants() ([]domain.GrantInfo, error)

	// DeleteGrant removes a grant from storage.
	DeleteGrant(grantID string) error

	// SetDefaultGrant sets the default grant ID.
	SetDefaultGrant(grantID string) error

	// GetDefaultGrant returns the default grant ID.
	GetDefaultGrant() (string, error)

	// ClearGrants removes all grants from storage.
	ClearGrants() error
}
