package nylas

import (
	"context"
	"io"

	"github.com/nylas/cli/internal/domain"
)

type MockClient struct {
	// State
	Region       string
	ClientID     string
	ClientSecret string
	APIKey       string

	// Call tracking
	ExchangeCodeCalled          bool
	ListGrantsCalled            bool
	GetGrantCalled              bool
	RevokeGrantCalled           bool
	GetMessagesCalled           bool
	GetMessagesWithParamsCalled bool
	GetMessageCalled            bool
	SendMessageCalled           bool
	UpdateMessageCalled         bool
	DeleteMessageCalled         bool
	GetThreadsCalled            bool
	GetThreadCalled             bool
	UpdateThreadCalled          bool
	DeleteThreadCalled          bool
	GetDraftsCalled             bool
	GetDraftCalled              bool
	CreateDraftCalled           bool
	UpdateDraftCalled           bool
	DeleteDraftCalled           bool
	SendDraftCalled             bool
	GetFoldersCalled            bool
	GetFolderCalled             bool
	CreateFolderCalled          bool
	UpdateFolderCalled          bool
	DeleteFolderCalled          bool
	ListAttachmentsCalled       bool
	GetAttachmentCalled         bool
	DownloadAttachmentCalled    bool
	LastGrantID                 string
	LastMessageID               string
	LastThreadID                string
	LastDraftID                 string
	LastFolderID                string
	LastAttachmentID            string

	// Custom functions
	ExchangeCodeFunc          func(ctx context.Context, code, redirectURI string) (*domain.Grant, error)
	ListGrantsFunc            func(ctx context.Context) ([]domain.Grant, error)
	GetGrantFunc              func(ctx context.Context, grantID string) (*domain.Grant, error)
	RevokeGrantFunc           func(ctx context.Context, grantID string) error
	GetMessagesFunc           func(ctx context.Context, grantID string, limit int) ([]domain.Message, error)
	GetMessagesWithParamsFunc func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error)
	GetMessageFunc            func(ctx context.Context, grantID, messageID string) (*domain.Message, error)
	SendMessageFunc           func(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error)
	UpdateMessageFunc         func(ctx context.Context, grantID, messageID string, req *domain.UpdateMessageRequest) (*domain.Message, error)
	DeleteMessageFunc         func(ctx context.Context, grantID, messageID string) error
	GetThreadsFunc            func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error)
	GetThreadFunc             func(ctx context.Context, grantID, threadID string) (*domain.Thread, error)
	UpdateThreadFunc          func(ctx context.Context, grantID, threadID string, req *domain.UpdateMessageRequest) (*domain.Thread, error)
	DeleteThreadFunc          func(ctx context.Context, grantID, threadID string) error
	GetDraftsFunc             func(ctx context.Context, grantID string, limit int) ([]domain.Draft, error)
	GetDraftFunc              func(ctx context.Context, grantID, draftID string) (*domain.Draft, error)
	CreateDraftFunc           func(ctx context.Context, grantID string, req *domain.CreateDraftRequest) (*domain.Draft, error)
	UpdateDraftFunc           func(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error)
	DeleteDraftFunc           func(ctx context.Context, grantID, draftID string) error
	SendDraftFunc             func(ctx context.Context, grantID, draftID string) (*domain.Message, error)
	GetFoldersFunc            func(ctx context.Context, grantID string) ([]domain.Folder, error)
	GetFolderFunc             func(ctx context.Context, grantID, folderID string) (*domain.Folder, error)
	CreateFolderFunc          func(ctx context.Context, grantID string, req *domain.CreateFolderRequest) (*domain.Folder, error)
	UpdateFolderFunc          func(ctx context.Context, grantID, folderID string, req *domain.UpdateFolderRequest) (*domain.Folder, error)
	DeleteFolderFunc          func(ctx context.Context, grantID, folderID string) error
	ListAttachmentsFunc       func(ctx context.Context, grantID, messageID string) ([]domain.Attachment, error)
	GetAttachmentFunc         func(ctx context.Context, grantID, messageID, attachmentID string) (*domain.Attachment, error)
	DownloadAttachmentFunc    func(ctx context.Context, grantID, messageID, attachmentID string) (io.ReadCloser, error)

	// Calendar functions
	GetCalendarsFunc func(ctx context.Context, grantID string) ([]domain.Calendar, error)
	GetEventsFunc    func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error)
	GetEventFunc     func(ctx context.Context, grantID, calendarID, eventID string) (*domain.Event, error)
	CreateEventFunc  func(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error)
	UpdateEventFunc  func(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error)
	DeleteEventFunc  func(ctx context.Context, grantID, calendarID, eventID string) error
}

// NewMockClient creates a new MockClient.
func NewMockClient() *MockClient {
	return &MockClient{}
}

// SetRegion sets the API region.
func (m *MockClient) SetRegion(region string) {
	m.Region = region
}

// SetCredentials sets the API credentials.
func (m *MockClient) SetCredentials(clientID, clientSecret, apiKey string) {
	m.ClientID = clientID
	m.ClientSecret = clientSecret
	m.APIKey = apiKey
}

// BuildAuthURL returns a mock auth URL.
func (m *MockClient) BuildAuthURL(provider domain.Provider, redirectURI string) string {
	return "https://mock.nylas.com/auth"
}
