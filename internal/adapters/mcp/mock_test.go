package mcp

import (
	"context"
	"io"

	"github.com/nylas/cli/internal/domain"
)

// mockNylasClient is a test double for ports.NylasClient.
// Fields ending in Func are called if non-nil; otherwise zero values are returned.
type mockNylasClient struct {
	getMessagesWithParamsFunc func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error)
	getMessageFunc            func(ctx context.Context, grantID, messageID string) (*domain.Message, error)
	sendMessageFunc           func(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error)
	getFoldersFunc            func(ctx context.Context, grantID string) ([]domain.Folder, error)
	getCalendarsFunc          func(ctx context.Context, grantID string) ([]domain.Calendar, error)
	getEventsFunc             func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error)
	getContactsFunc           func(ctx context.Context, grantID string, params *domain.ContactQueryParams) ([]domain.Contact, error)
}

// ============================================================================
// AuthClient
// ============================================================================

func (m *mockNylasClient) BuildAuthURL(provider domain.Provider, redirectURI string) string {
	return ""
}
func (m *mockNylasClient) ExchangeCode(ctx context.Context, code, redirectURI string) (*domain.Grant, error) {
	return nil, nil
}
func (m *mockNylasClient) ListGrants(ctx context.Context) ([]domain.Grant, error) {
	return nil, nil
}
func (m *mockNylasClient) GetGrant(ctx context.Context, grantID string) (*domain.Grant, error) {
	return nil, nil
}
func (m *mockNylasClient) RevokeGrant(ctx context.Context, grantID string) error { return nil }

// ============================================================================
// MessageClient
// ============================================================================

func (m *mockNylasClient) GetMessages(ctx context.Context, grantID string, limit int) ([]domain.Message, error) {
	return nil, nil
}
func (m *mockNylasClient) GetMessagesWithParams(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
	if m.getMessagesWithParamsFunc != nil {
		return m.getMessagesWithParamsFunc(ctx, grantID, params)
	}
	return nil, nil
}
func (m *mockNylasClient) GetMessagesWithCursor(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error) {
	return nil, nil
}
func (m *mockNylasClient) GetMessage(ctx context.Context, grantID, messageID string) (*domain.Message, error) {
	if m.getMessageFunc != nil {
		return m.getMessageFunc(ctx, grantID, messageID)
	}
	return nil, nil
}
func (m *mockNylasClient) GetMessageWithFields(ctx context.Context, grantID, messageID, fields string) (*domain.Message, error) {
	return nil, nil
}
func (m *mockNylasClient) SendMessage(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error) {
	if m.sendMessageFunc != nil {
		return m.sendMessageFunc(ctx, grantID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) SendRawMessage(ctx context.Context, grantID string, rawMIME []byte) (*domain.Message, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateMessage(ctx context.Context, grantID, messageID string, req *domain.UpdateMessageRequest) (*domain.Message, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteMessage(ctx context.Context, grantID, messageID string) error {
	return nil
}
func (m *mockNylasClient) ListScheduledMessages(ctx context.Context, grantID string) ([]domain.ScheduledMessage, error) {
	return nil, nil
}
func (m *mockNylasClient) GetScheduledMessage(ctx context.Context, grantID, scheduleID string) (*domain.ScheduledMessage, error) {
	return nil, nil
}
func (m *mockNylasClient) CancelScheduledMessage(ctx context.Context, grantID, scheduleID string) error {
	return nil
}
func (m *mockNylasClient) SmartCompose(ctx context.Context, grantID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
	return nil, nil
}
func (m *mockNylasClient) SmartComposeReply(ctx context.Context, grantID, messageID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
	return nil, nil
}
func (m *mockNylasClient) GetThreads(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error) {
	return nil, nil
}
func (m *mockNylasClient) GetThread(ctx context.Context, grantID, threadID string) (*domain.Thread, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateThread(ctx context.Context, grantID, threadID string, req *domain.UpdateMessageRequest) (*domain.Thread, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteThread(ctx context.Context, grantID, threadID string) error {
	return nil
}
func (m *mockNylasClient) GetDrafts(ctx context.Context, grantID string, limit int) ([]domain.Draft, error) {
	return nil, nil
}
func (m *mockNylasClient) GetDraft(ctx context.Context, grantID, draftID string) (*domain.Draft, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateDraft(ctx context.Context, grantID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateDraft(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteDraft(ctx context.Context, grantID, draftID string) error {
	return nil
}
func (m *mockNylasClient) SendDraft(ctx context.Context, grantID, draftID string) (*domain.Message, error) {
	return nil, nil
}
func (m *mockNylasClient) GetFolders(ctx context.Context, grantID string) ([]domain.Folder, error) {
	if m.getFoldersFunc != nil {
		return m.getFoldersFunc(ctx, grantID)
	}
	return nil, nil
}
func (m *mockNylasClient) GetFolder(ctx context.Context, grantID, folderID string) (*domain.Folder, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateFolder(ctx context.Context, grantID string, req *domain.CreateFolderRequest) (*domain.Folder, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateFolder(ctx context.Context, grantID, folderID string, req *domain.UpdateFolderRequest) (*domain.Folder, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteFolder(ctx context.Context, grantID, folderID string) error {
	return nil
}
func (m *mockNylasClient) ListAttachments(ctx context.Context, grantID, messageID string) ([]domain.Attachment, error) {
	return nil, nil
}
func (m *mockNylasClient) GetAttachment(ctx context.Context, grantID, messageID, attachmentID string) (*domain.Attachment, error) {
	return nil, nil
}
func (m *mockNylasClient) DownloadAttachment(ctx context.Context, grantID, messageID, attachmentID string) (io.ReadCloser, error) {
	return nil, nil
}

// ============================================================================
// CalendarClient
// ============================================================================

func (m *mockNylasClient) GetCalendars(ctx context.Context, grantID string) ([]domain.Calendar, error) {
	if m.getCalendarsFunc != nil {
		return m.getCalendarsFunc(ctx, grantID)
	}
	return nil, nil
}
func (m *mockNylasClient) GetCalendar(ctx context.Context, grantID, calendarID string) (*domain.Calendar, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateCalendar(ctx context.Context, grantID string, req *domain.CreateCalendarRequest) (*domain.Calendar, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateCalendar(ctx context.Context, grantID, calendarID string, req *domain.UpdateCalendarRequest) (*domain.Calendar, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteCalendar(ctx context.Context, grantID, calendarID string) error {
	return nil
}
func (m *mockNylasClient) GetEvents(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error) {
	if m.getEventsFunc != nil {
		return m.getEventsFunc(ctx, grantID, calendarID, params)
	}
	return nil, nil
}
func (m *mockNylasClient) GetEventsWithCursor(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error) {
	return nil, nil
}
func (m *mockNylasClient) GetEvent(ctx context.Context, grantID, calendarID, eventID string) (*domain.Event, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateEvent(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateEvent(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteEvent(ctx context.Context, grantID, calendarID, eventID string) error {
	return nil
}
func (m *mockNylasClient) SendRSVP(ctx context.Context, grantID, calendarID, eventID string, req *domain.SendRSVPRequest) error {
	return nil
}
func (m *mockNylasClient) GetFreeBusy(ctx context.Context, grantID string, req *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error) {
	return nil, nil
}
func (m *mockNylasClient) GetAvailability(ctx context.Context, req *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateVirtualCalendarGrant(ctx context.Context, email string) (*domain.VirtualCalendarGrant, error) {
	return nil, nil
}
func (m *mockNylasClient) ListVirtualCalendarGrants(ctx context.Context) ([]domain.VirtualCalendarGrant, error) {
	return nil, nil
}
func (m *mockNylasClient) GetVirtualCalendarGrant(ctx context.Context, grantID string) (*domain.VirtualCalendarGrant, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteVirtualCalendarGrant(ctx context.Context, grantID string) error {
	return nil
}
func (m *mockNylasClient) GetRecurringEventInstances(ctx context.Context, grantID, calendarID, masterEventID string, params *domain.EventQueryParams) ([]domain.Event, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateRecurringEventInstance(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteRecurringEventInstance(ctx context.Context, grantID, calendarID, eventID string) error {
	return nil
}

// ============================================================================
// ContactClient
// ============================================================================

func (m *mockNylasClient) GetContacts(ctx context.Context, grantID string, params *domain.ContactQueryParams) ([]domain.Contact, error) {
	if m.getContactsFunc != nil {
		return m.getContactsFunc(ctx, grantID, params)
	}
	return nil, nil
}
func (m *mockNylasClient) GetContactsWithCursor(ctx context.Context, grantID string, params *domain.ContactQueryParams) (*domain.ContactListResponse, error) {
	return nil, nil
}
func (m *mockNylasClient) GetContact(ctx context.Context, grantID, contactID string) (*domain.Contact, error) {
	return nil, nil
}
func (m *mockNylasClient) GetContactWithPicture(ctx context.Context, grantID, contactID string, includePicture bool) (*domain.Contact, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateContact(ctx context.Context, grantID string, req *domain.CreateContactRequest) (*domain.Contact, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateContact(ctx context.Context, grantID, contactID string, req *domain.UpdateContactRequest) (*domain.Contact, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteContact(ctx context.Context, grantID, contactID string) error {
	return nil
}
func (m *mockNylasClient) GetContactGroups(ctx context.Context, grantID string) ([]domain.ContactGroup, error) {
	return nil, nil
}
func (m *mockNylasClient) GetContactGroup(ctx context.Context, grantID, groupID string) (*domain.ContactGroup, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateContactGroup(ctx context.Context, grantID string, req *domain.CreateContactGroupRequest) (*domain.ContactGroup, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateContactGroup(ctx context.Context, grantID, groupID string, req *domain.UpdateContactGroupRequest) (*domain.ContactGroup, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteContactGroup(ctx context.Context, grantID, groupID string) error {
	return nil
}

// ============================================================================
// WebhookClient
// ============================================================================

func (m *mockNylasClient) ListWebhooks(ctx context.Context) ([]domain.Webhook, error) {
	return nil, nil
}
func (m *mockNylasClient) GetWebhook(ctx context.Context, webhookID string) (*domain.Webhook, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateWebhook(ctx context.Context, req *domain.CreateWebhookRequest) (*domain.Webhook, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateWebhook(ctx context.Context, webhookID string, req *domain.UpdateWebhookRequest) (*domain.Webhook, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteWebhook(ctx context.Context, webhookID string) error { return nil }
func (m *mockNylasClient) SendWebhookTestEvent(ctx context.Context, webhookURL string) error {
	return nil
}
func (m *mockNylasClient) GetWebhookMockPayload(ctx context.Context, triggerType string) (map[string]any, error) {
	return nil, nil
}

// ============================================================================
// NotetakerClient
// ============================================================================

func (m *mockNylasClient) ListNotetakers(ctx context.Context, grantID string, params *domain.NotetakerQueryParams) ([]domain.Notetaker, error) {
	return nil, nil
}
func (m *mockNylasClient) GetNotetaker(ctx context.Context, grantID, notetakerID string) (*domain.Notetaker, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateNotetaker(ctx context.Context, grantID string, req *domain.CreateNotetakerRequest) (*domain.Notetaker, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteNotetaker(ctx context.Context, grantID, notetakerID string) error {
	return nil
}
func (m *mockNylasClient) GetNotetakerMedia(ctx context.Context, grantID, notetakerID string) (*domain.MediaData, error) {
	return nil, nil
}

// ============================================================================
// InboundClient
// ============================================================================

func (m *mockNylasClient) ListInboundInboxes(ctx context.Context) ([]domain.InboundInbox, error) {
	return nil, nil
}
func (m *mockNylasClient) GetInboundInbox(ctx context.Context, grantID string) (*domain.InboundInbox, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateInboundInbox(ctx context.Context, email string) (*domain.InboundInbox, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteInboundInbox(ctx context.Context, grantID string) error { return nil }
func (m *mockNylasClient) GetInboundMessages(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.InboundMessage, error) {
	return nil, nil
}

// ============================================================================
// SchedulerClient
// ============================================================================

func (m *mockNylasClient) ListSchedulerConfigurations(ctx context.Context) ([]domain.SchedulerConfiguration, error) {
	return nil, nil
}
func (m *mockNylasClient) GetSchedulerConfiguration(ctx context.Context, configID string) (*domain.SchedulerConfiguration, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateSchedulerConfiguration(ctx context.Context, req *domain.CreateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateSchedulerConfiguration(ctx context.Context, configID string, req *domain.UpdateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteSchedulerConfiguration(ctx context.Context, configID string) error {
	return nil
}
func (m *mockNylasClient) CreateSchedulerSession(ctx context.Context, req *domain.CreateSchedulerSessionRequest) (*domain.SchedulerSession, error) {
	return nil, nil
}
func (m *mockNylasClient) GetSchedulerSession(ctx context.Context, sessionID string) (*domain.SchedulerSession, error) {
	return nil, nil
}
func (m *mockNylasClient) ListBookings(ctx context.Context, configID string) ([]domain.Booking, error) {
	return nil, nil
}
func (m *mockNylasClient) GetBooking(ctx context.Context, bookingID string) (*domain.Booking, error) {
	return nil, nil
}
func (m *mockNylasClient) ConfirmBooking(ctx context.Context, bookingID string, req *domain.ConfirmBookingRequest) (*domain.Booking, error) {
	return nil, nil
}
func (m *mockNylasClient) RescheduleBooking(ctx context.Context, bookingID string, req *domain.RescheduleBookingRequest) (*domain.Booking, error) {
	return nil, nil
}
func (m *mockNylasClient) CancelBooking(ctx context.Context, bookingID string, reason string) error {
	return nil
}
func (m *mockNylasClient) ListSchedulerPages(ctx context.Context) ([]domain.SchedulerPage, error) {
	return nil, nil
}
func (m *mockNylasClient) GetSchedulerPage(ctx context.Context, pageID string) (*domain.SchedulerPage, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateSchedulerPage(ctx context.Context, req *domain.CreateSchedulerPageRequest) (*domain.SchedulerPage, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateSchedulerPage(ctx context.Context, pageID string, req *domain.UpdateSchedulerPageRequest) (*domain.SchedulerPage, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteSchedulerPage(ctx context.Context, pageID string) error { return nil }

// ============================================================================
// AdminClient
// ============================================================================

func (m *mockNylasClient) ListApplications(ctx context.Context) ([]domain.Application, error) {
	return nil, nil
}
func (m *mockNylasClient) GetApplication(ctx context.Context, appID string) (*domain.Application, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateApplication(ctx context.Context, req *domain.CreateApplicationRequest) (*domain.Application, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateApplication(ctx context.Context, appID string, req *domain.UpdateApplicationRequest) (*domain.Application, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteApplication(ctx context.Context, appID string) error { return nil }
func (m *mockNylasClient) ListConnectors(ctx context.Context) ([]domain.Connector, error) {
	return nil, nil
}
func (m *mockNylasClient) GetConnector(ctx context.Context, connectorID string) (*domain.Connector, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateConnector(ctx context.Context, req *domain.CreateConnectorRequest) (*domain.Connector, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateConnector(ctx context.Context, connectorID string, req *domain.UpdateConnectorRequest) (*domain.Connector, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteConnector(ctx context.Context, connectorID string) error { return nil }
func (m *mockNylasClient) ListCredentials(ctx context.Context, connectorID string) ([]domain.ConnectorCredential, error) {
	return nil, nil
}
func (m *mockNylasClient) GetCredential(ctx context.Context, credentialID string) (*domain.ConnectorCredential, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateCredential(ctx context.Context, connectorID string, req *domain.CreateCredentialRequest) (*domain.ConnectorCredential, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateCredential(ctx context.Context, credentialID string, req *domain.UpdateCredentialRequest) (*domain.ConnectorCredential, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteCredential(ctx context.Context, credentialID string) error {
	return nil
}
func (m *mockNylasClient) ListAllGrants(ctx context.Context, params *domain.GrantsQueryParams) ([]domain.Grant, error) {
	return nil, nil
}
func (m *mockNylasClient) GetGrantStats(ctx context.Context) (*domain.GrantStats, error) {
	return nil, nil
}

// ============================================================================
// TransactionalClient
// ============================================================================

func (m *mockNylasClient) SendTransactionalMessage(ctx context.Context, domainName string, req *domain.SendMessageRequest) (*domain.Message, error) {
	return nil, nil
}

// ============================================================================
// Configuration methods
// ============================================================================

func (m *mockNylasClient) SetRegion(region string)                              {}
func (m *mockNylasClient) SetCredentials(clientID, clientSecret, apiKey string) {}
