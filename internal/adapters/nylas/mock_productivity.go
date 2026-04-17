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

// Scheduler Mock Implementations
