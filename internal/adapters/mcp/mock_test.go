package mcp

import (
	"context"
	"io"

	"github.com/nylas/cli/internal/domain"
)

// mockNylasClient is a test double for ports.NylasClient.
// Fields ending in Func are called if non-nil; otherwise zero values are returned.
type mockNylasClient struct {
	getMessagesWithParamsFunc  func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error)
	getMessagesWithCursorFunc  func(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error)
	getMessageFunc             func(ctx context.Context, grantID, messageID string) (*domain.Message, error)
	sendMessageFunc            func(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error)
	updateMessageFunc          func(ctx context.Context, grantID, messageID string, req *domain.UpdateMessageRequest) (*domain.Message, error)
	deleteMessageFunc          func(ctx context.Context, grantID, messageID string) error
	smartComposeFunc           func(ctx context.Context, grantID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error)
	smartComposeReplyFunc      func(ctx context.Context, grantID, messageID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error)
	getDraftsFunc              func(ctx context.Context, grantID string, limit int) ([]domain.Draft, error)
	getDraftFunc               func(ctx context.Context, grantID, draftID string) (*domain.Draft, error)
	createDraftFunc            func(ctx context.Context, grantID string, req *domain.CreateDraftRequest) (*domain.Draft, error)
	updateDraftFunc            func(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error)
	sendDraftFunc              func(ctx context.Context, grantID, draftID string) (*domain.Message, error)
	deleteDraftFunc            func(ctx context.Context, grantID, draftID string) error
	getFoldersFunc             func(ctx context.Context, grantID string) ([]domain.Folder, error)
	getFolderFunc              func(ctx context.Context, grantID, folderID string) (*domain.Folder, error)
	createFolderFunc           func(ctx context.Context, grantID string, req *domain.CreateFolderRequest) (*domain.Folder, error)
	updateFolderFunc           func(ctx context.Context, grantID, folderID string, req *domain.UpdateFolderRequest) (*domain.Folder, error)
	deleteFolderFunc           func(ctx context.Context, grantID, folderID string) error
	getCalendarsFunc           func(ctx context.Context, grantID string) ([]domain.Calendar, error)
	getCalendarFunc            func(ctx context.Context, grantID, calendarID string) (*domain.Calendar, error)
	createCalendarFunc         func(ctx context.Context, grantID string, req *domain.CreateCalendarRequest) (*domain.Calendar, error)
	updateCalendarFunc         func(ctx context.Context, grantID, calendarID string, req *domain.UpdateCalendarRequest) (*domain.Calendar, error)
	deleteCalendarFunc         func(ctx context.Context, grantID, calendarID string) error
	getEventsFunc              func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error)
	getEventsWithCursorFunc    func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error)
	getEventFunc               func(ctx context.Context, grantID, calendarID, eventID string) (*domain.Event, error)
	createEventFunc            func(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error)
	updateEventFunc            func(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error)
	deleteEventFunc            func(ctx context.Context, grantID, calendarID, eventID string) error
	sendRSVPFunc               func(ctx context.Context, grantID, calendarID, eventID string, req *domain.SendRSVPRequest) error
	getFreeBusyFunc            func(ctx context.Context, grantID string, req *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error)
	getAvailabilityFunc        func(ctx context.Context, req *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error)
	getContactsFunc            func(ctx context.Context, grantID string, params *domain.ContactQueryParams) ([]domain.Contact, error)
	getContactsWithCursorFunc  func(ctx context.Context, grantID string, params *domain.ContactQueryParams) (*domain.ContactListResponse, error)
	getContactFunc             func(ctx context.Context, grantID, contactID string) (*domain.Contact, error)
	createContactFunc          func(ctx context.Context, grantID string, req *domain.CreateContactRequest) (*domain.Contact, error)
	updateContactFunc          func(ctx context.Context, grantID, contactID string, req *domain.UpdateContactRequest) (*domain.Contact, error)
	deleteContactFunc          func(ctx context.Context, grantID, contactID string) error
	getThreadsFunc             func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error)
	getThreadFunc              func(ctx context.Context, grantID, threadID string) (*domain.Thread, error)
	updateThreadFunc           func(ctx context.Context, grantID, threadID string, req *domain.UpdateMessageRequest) (*domain.Thread, error)
	deleteThreadFunc           func(ctx context.Context, grantID, threadID string) error
	listAttachmentsFunc        func(ctx context.Context, grantID, messageID string) ([]domain.Attachment, error)
	getAttachmentFunc          func(ctx context.Context, grantID, messageID, attachmentID string) (*domain.Attachment, error)
	listScheduledMessagesFunc  func(ctx context.Context, grantID string) ([]domain.ScheduledMessage, error)
	cancelScheduledMessageFunc func(ctx context.Context, grantID, scheduleID string) error
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
	if m.getMessagesWithCursorFunc != nil {
		return m.getMessagesWithCursorFunc(ctx, grantID, params)
	}
	// Fallback: delegate to getMessagesWithParamsFunc if set
	if m.getMessagesWithParamsFunc != nil {
		msgs, err := m.getMessagesWithParamsFunc(ctx, grantID, params)
		return &domain.MessageListResponse{Data: msgs}, err
	}
	return &domain.MessageListResponse{}, nil
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
	if m.updateMessageFunc != nil {
		return m.updateMessageFunc(ctx, grantID, messageID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) DeleteMessage(ctx context.Context, grantID, messageID string) error {
	if m.deleteMessageFunc != nil {
		return m.deleteMessageFunc(ctx, grantID, messageID)
	}
	return nil
}
func (m *mockNylasClient) ListScheduledMessages(ctx context.Context, grantID string) ([]domain.ScheduledMessage, error) {
	if m.listScheduledMessagesFunc != nil {
		return m.listScheduledMessagesFunc(ctx, grantID)
	}
	return nil, nil
}
func (m *mockNylasClient) GetScheduledMessage(ctx context.Context, grantID, scheduleID string) (*domain.ScheduledMessage, error) {
	return nil, nil
}
func (m *mockNylasClient) CancelScheduledMessage(ctx context.Context, grantID, scheduleID string) error {
	if m.cancelScheduledMessageFunc != nil {
		return m.cancelScheduledMessageFunc(ctx, grantID, scheduleID)
	}
	return nil
}
func (m *mockNylasClient) SmartCompose(ctx context.Context, grantID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
	if m.smartComposeFunc != nil {
		return m.smartComposeFunc(ctx, grantID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) SmartComposeReply(ctx context.Context, grantID, messageID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
	if m.smartComposeReplyFunc != nil {
		return m.smartComposeReplyFunc(ctx, grantID, messageID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) GetThreads(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error) {
	if m.getThreadsFunc != nil {
		return m.getThreadsFunc(ctx, grantID, params)
	}
	return nil, nil
}
func (m *mockNylasClient) GetThread(ctx context.Context, grantID, threadID string) (*domain.Thread, error) {
	if m.getThreadFunc != nil {
		return m.getThreadFunc(ctx, grantID, threadID)
	}
	return nil, nil
}
func (m *mockNylasClient) UpdateThread(ctx context.Context, grantID, threadID string, req *domain.UpdateMessageRequest) (*domain.Thread, error) {
	if m.updateThreadFunc != nil {
		return m.updateThreadFunc(ctx, grantID, threadID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) DeleteThread(ctx context.Context, grantID, threadID string) error {
	if m.deleteThreadFunc != nil {
		return m.deleteThreadFunc(ctx, grantID, threadID)
	}
	return nil
}
func (m *mockNylasClient) GetDrafts(ctx context.Context, grantID string, limit int) ([]domain.Draft, error) {
	if m.getDraftsFunc != nil {
		return m.getDraftsFunc(ctx, grantID, limit)
	}
	return nil, nil
}
func (m *mockNylasClient) GetDraft(ctx context.Context, grantID, draftID string) (*domain.Draft, error) {
	if m.getDraftFunc != nil {
		return m.getDraftFunc(ctx, grantID, draftID)
	}
	return nil, nil
}
func (m *mockNylasClient) CreateDraft(ctx context.Context, grantID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	if m.createDraftFunc != nil {
		return m.createDraftFunc(ctx, grantID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) UpdateDraft(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	if m.updateDraftFunc != nil {
		return m.updateDraftFunc(ctx, grantID, draftID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) DeleteDraft(ctx context.Context, grantID, draftID string) error {
	if m.deleteDraftFunc != nil {
		return m.deleteDraftFunc(ctx, grantID, draftID)
	}
	return nil
}
func (m *mockNylasClient) SendDraft(ctx context.Context, grantID, draftID string) (*domain.Message, error) {
	if m.sendDraftFunc != nil {
		return m.sendDraftFunc(ctx, grantID, draftID)
	}
	return nil, nil
}
func (m *mockNylasClient) GetFolders(ctx context.Context, grantID string) ([]domain.Folder, error) {
	if m.getFoldersFunc != nil {
		return m.getFoldersFunc(ctx, grantID)
	}
	return nil, nil
}
func (m *mockNylasClient) GetFolder(ctx context.Context, grantID, folderID string) (*domain.Folder, error) {
	if m.getFolderFunc != nil {
		return m.getFolderFunc(ctx, grantID, folderID)
	}
	return nil, nil
}
func (m *mockNylasClient) CreateFolder(ctx context.Context, grantID string, req *domain.CreateFolderRequest) (*domain.Folder, error) {
	if m.createFolderFunc != nil {
		return m.createFolderFunc(ctx, grantID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) UpdateFolder(ctx context.Context, grantID, folderID string, req *domain.UpdateFolderRequest) (*domain.Folder, error) {
	if m.updateFolderFunc != nil {
		return m.updateFolderFunc(ctx, grantID, folderID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) DeleteFolder(ctx context.Context, grantID, folderID string) error {
	if m.deleteFolderFunc != nil {
		return m.deleteFolderFunc(ctx, grantID, folderID)
	}
	return nil
}
func (m *mockNylasClient) ListAttachments(ctx context.Context, grantID, messageID string) ([]domain.Attachment, error) {
	if m.listAttachmentsFunc != nil {
		return m.listAttachmentsFunc(ctx, grantID, messageID)
	}
	return nil, nil
}
func (m *mockNylasClient) GetAttachment(ctx context.Context, grantID, messageID, attachmentID string) (*domain.Attachment, error) {
	if m.getAttachmentFunc != nil {
		return m.getAttachmentFunc(ctx, grantID, messageID, attachmentID)
	}
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
	if m.getCalendarFunc != nil {
		return m.getCalendarFunc(ctx, grantID, calendarID)
	}
	return nil, nil
}
func (m *mockNylasClient) CreateCalendar(ctx context.Context, grantID string, req *domain.CreateCalendarRequest) (*domain.Calendar, error) {
	if m.createCalendarFunc != nil {
		return m.createCalendarFunc(ctx, grantID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) UpdateCalendar(ctx context.Context, grantID, calendarID string, req *domain.UpdateCalendarRequest) (*domain.Calendar, error) {
	if m.updateCalendarFunc != nil {
		return m.updateCalendarFunc(ctx, grantID, calendarID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) DeleteCalendar(ctx context.Context, grantID, calendarID string) error {
	if m.deleteCalendarFunc != nil {
		return m.deleteCalendarFunc(ctx, grantID, calendarID)
	}
	return nil
}
func (m *mockNylasClient) GetEvents(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error) {
	if m.getEventsFunc != nil {
		return m.getEventsFunc(ctx, grantID, calendarID, params)
	}
	return nil, nil
}
func (m *mockNylasClient) GetEventsWithCursor(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error) {
	if m.getEventsWithCursorFunc != nil {
		return m.getEventsWithCursorFunc(ctx, grantID, calendarID, params)
	}
	// Fallback: delegate to getEventsFunc if set
	if m.getEventsFunc != nil {
		events, err := m.getEventsFunc(ctx, grantID, calendarID, params)
		return &domain.EventListResponse{Data: events}, err
	}
	return &domain.EventListResponse{}, nil
}
func (m *mockNylasClient) GetEvent(ctx context.Context, grantID, calendarID, eventID string) (*domain.Event, error) {
	if m.getEventFunc != nil {
		return m.getEventFunc(ctx, grantID, calendarID, eventID)
	}
	return nil, nil
}
func (m *mockNylasClient) CreateEvent(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error) {
	if m.createEventFunc != nil {
		return m.createEventFunc(ctx, grantID, calendarID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) UpdateEvent(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error) {
	if m.updateEventFunc != nil {
		return m.updateEventFunc(ctx, grantID, calendarID, eventID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) DeleteEvent(ctx context.Context, grantID, calendarID, eventID string) error {
	if m.deleteEventFunc != nil {
		return m.deleteEventFunc(ctx, grantID, calendarID, eventID)
	}
	return nil
}
func (m *mockNylasClient) SendRSVP(ctx context.Context, grantID, calendarID, eventID string, req *domain.SendRSVPRequest) error {
	if m.sendRSVPFunc != nil {
		return m.sendRSVPFunc(ctx, grantID, calendarID, eventID, req)
	}
	return nil
}
func (m *mockNylasClient) GetFreeBusy(ctx context.Context, grantID string, req *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error) {
	if m.getFreeBusyFunc != nil {
		return m.getFreeBusyFunc(ctx, grantID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) GetAvailability(ctx context.Context, req *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error) {
	if m.getAvailabilityFunc != nil {
		return m.getAvailabilityFunc(ctx, req)
	}
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
	if m.getContactsWithCursorFunc != nil {
		return m.getContactsWithCursorFunc(ctx, grantID, params)
	}
	// Fallback: delegate to getContactsFunc if set
	if m.getContactsFunc != nil {
		contacts, err := m.getContactsFunc(ctx, grantID, params)
		return &domain.ContactListResponse{Data: contacts}, err
	}
	return &domain.ContactListResponse{}, nil
}
func (m *mockNylasClient) GetContact(ctx context.Context, grantID, contactID string) (*domain.Contact, error) {
	if m.getContactFunc != nil {
		return m.getContactFunc(ctx, grantID, contactID)
	}
	return nil, nil
}
func (m *mockNylasClient) GetContactWithPicture(ctx context.Context, grantID, contactID string, includePicture bool) (*domain.Contact, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateContact(ctx context.Context, grantID string, req *domain.CreateContactRequest) (*domain.Contact, error) {
	if m.createContactFunc != nil {
		return m.createContactFunc(ctx, grantID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) UpdateContact(ctx context.Context, grantID, contactID string, req *domain.UpdateContactRequest) (*domain.Contact, error) {
	if m.updateContactFunc != nil {
		return m.updateContactFunc(ctx, grantID, contactID, req)
	}
	return nil, nil
}
func (m *mockNylasClient) DeleteContact(ctx context.Context, grantID, contactID string) error {
	if m.deleteContactFunc != nil {
		return m.deleteContactFunc(ctx, grantID, contactID)
	}
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
