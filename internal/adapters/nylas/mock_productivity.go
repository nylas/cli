package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) ListNotetakers(ctx context.Context, grantID string, params *domain.NotetakerQueryParams) ([]domain.Notetaker, error) {
	return []domain.Notetaker{
		{
			ID:           "notetaker-1",
			State:        domain.NotetakerStateComplete,
			MeetingLink:  "https://zoom.us/j/123456789",
			MeetingTitle: "Test Meeting",
		},
	}, nil
}

// GetNotetaker retrieves a single notetaker.
func (m *MockClient) GetNotetaker(ctx context.Context, grantID, notetakerID string) (*domain.Notetaker, error) {
	return &domain.Notetaker{
		ID:           notetakerID,
		State:        domain.NotetakerStateComplete,
		MeetingLink:  "https://zoom.us/j/123456789",
		MeetingTitle: "Test Meeting",
		MeetingInfo: &domain.MeetingInfo{
			Provider: "zoom",
		},
	}, nil
}

// CreateNotetaker creates a new notetaker.
func (m *MockClient) CreateNotetaker(ctx context.Context, grantID string, req *domain.CreateNotetakerRequest) (*domain.Notetaker, error) {
	return &domain.Notetaker{
		ID:          "new-notetaker-id",
		State:       domain.NotetakerStateScheduled,
		MeetingLink: req.MeetingLink,
		BotConfig:   req.BotConfig,
	}, nil
}

// DeleteNotetaker deletes a notetaker.
func (m *MockClient) DeleteNotetaker(ctx context.Context, grantID, notetakerID string) error {
	return nil
}

// GetNotetakerMedia retrieves notetaker media.
func (m *MockClient) GetNotetakerMedia(ctx context.Context, grantID, notetakerID string) (*domain.MediaData, error) {
	return &domain.MediaData{
		Recording: &domain.MediaFile{
			URL:         "https://storage.nylas.com/recording.mp4",
			ContentType: "video/mp4",
			Size:        1024000,
			ExpiresAt:   1700000000,
		},
		Transcript: &domain.MediaFile{
			URL:         "https://storage.nylas.com/transcript.txt",
			ContentType: "text/plain",
			Size:        4096,
			ExpiresAt:   1700000000,
		},
	}, nil
}

// ListInboundInboxes lists all inbound inboxes.
func (m *MockClient) ListInboundInboxes(ctx context.Context) ([]domain.InboundInbox, error) {
	return []domain.InboundInbox{
		{
			ID:          "inbox-1",
			Email:       "support@app.nylas.email",
			GrantStatus: "valid",
		},
		{
			ID:          "inbox-2",
			Email:       "info@app.nylas.email",
			GrantStatus: "valid",
		},
	}, nil
}

// GetInboundInbox retrieves a specific inbound inbox.
func (m *MockClient) GetInboundInbox(ctx context.Context, grantID string) (*domain.InboundInbox, error) {
	m.LastGrantID = grantID
	return &domain.InboundInbox{
		ID:          grantID,
		Email:       "support@app.nylas.email",
		GrantStatus: "valid",
	}, nil
}

// CreateInboundInbox creates a new inbound inbox.
func (m *MockClient) CreateInboundInbox(ctx context.Context, email string) (*domain.InboundInbox, error) {
	return &domain.InboundInbox{
		ID:          "new-inbox-id",
		Email:       email + "@app.nylas.email",
		GrantStatus: "valid",
	}, nil
}

// DeleteInboundInbox deletes an inbound inbox.
func (m *MockClient) DeleteInboundInbox(ctx context.Context, grantID string) error {
	m.LastGrantID = grantID
	return nil
}

// GetInboundMessages retrieves messages for an inbound inbox.
func (m *MockClient) GetInboundMessages(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.InboundMessage, error) {
	m.LastGrantID = grantID
	return []domain.InboundMessage{
		{
			ID:      "inbound-msg-1",
			GrantID: grantID,
			Subject: "New Lead Submission",
			From:    []domain.EmailParticipant{{Name: "John Doe", Email: "john@example.com"}},
			Snippet: "Hi, I'm interested in your services...",
			Unread:  true,
		},
		{
			ID:      "inbound-msg-2",
			GrantID: grantID,
			Subject: "Support Request #12345",
			From:    []domain.EmailParticipant{{Name: "Jane Smith", Email: "jane@example.com"}},
			Snippet: "I need help with my account...",
			Unread:  false,
		},
	}, nil
}

// Scheduler Mock Implementations
