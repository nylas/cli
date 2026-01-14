package nylas

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func (d *DemoClient) ListScheduledMessages(ctx context.Context, grantID string) ([]domain.ScheduledMessage, error) {
	now := time.Now()
	return []domain.ScheduledMessage{
		{
			ScheduleID: "schedule-001",
			Status:     "scheduled",
			CloseTime:  now.Add(1 * time.Hour).Unix(),
		},
		{
			ScheduleID: "schedule-002",
			Status:     "scheduled",
			CloseTime:  now.Add(24 * time.Hour).Unix(),
		},
	}, nil
}

// GetScheduledMessage returns a demo scheduled message.
func (d *DemoClient) GetScheduledMessage(ctx context.Context, grantID, scheduleID string) (*domain.ScheduledMessage, error) {
	return &domain.ScheduledMessage{
		ScheduleID: scheduleID,
		Status:     "scheduled",
		CloseTime:  time.Now().Add(1 * time.Hour).Unix(),
	}, nil
}

// CancelScheduledMessage simulates canceling a scheduled message.
func (d *DemoClient) CancelScheduledMessage(ctx context.Context, grantID, scheduleID string) error {
	return nil
}

// SmartCompose generates an AI-powered email draft based on a prompt.
func (d *DemoClient) SmartCompose(ctx context.Context, grantID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
	// Generate realistic demo response based on prompt
	suggestion := "Dear Colleague,\n\nThank you for reaching out. "
	if req != nil && req.Prompt != "" {
		suggestion += "I understand you'd like to " + req.Prompt + ". "
	}
	suggestion += "I've reviewed your request and wanted to follow up with some thoughts.\n\n"
	suggestion += "Based on our previous discussions, I believe we can move forward with this initiative. "
	suggestion += "I'll coordinate with the team and get back to you with a detailed plan by the end of the week.\n\n"
	suggestion += "Please let me know if you have any questions or need clarification on any points.\n\n"
	suggestion += "Best regards"

	return &domain.SmartComposeSuggestion{
		Suggestion: suggestion,
	}, nil
}

// SmartComposeReply generates an AI-powered reply to a specific message.
func (d *DemoClient) SmartComposeReply(ctx context.Context, grantID, messageID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
	// Generate realistic demo reply based on prompt
	suggestion := "Hi there,\n\nThank you for your message. "
	if req != nil && req.Prompt != "" {
		suggestion += req.Prompt + "\n\n"
	}
	suggestion += "I've taken a look at what you sent and wanted to respond quickly. "
	suggestion += "Your points are well-taken, and I agree with your assessment.\n\n"
	suggestion += "I'll review the details more thoroughly and provide a comprehensive response shortly. "
	suggestion += "In the meantime, please don't hesitate to reach out if you have any urgent concerns.\n\n"
	suggestion += "Thanks again for bringing this to my attention.\n\n"
	suggestion += "Best"

	return &domain.SmartComposeSuggestion{
		Suggestion: suggestion,
	}, nil
}

// ListNotetakers returns demo notetakers.
func (d *DemoClient) ListNotetakers(ctx context.Context, grantID string, params *domain.NotetakerQueryParams) ([]domain.Notetaker, error) {
	now := time.Now()
	return []domain.Notetaker{
		{
			ID:           "notetaker-001",
			State:        domain.NotetakerStateComplete,
			MeetingLink:  "https://zoom.us/j/123456789",
			MeetingTitle: "Q4 Planning Meeting",
			CreatedAt:    now.Add(-2 * time.Hour),
			UpdatedAt:    now.Add(-1 * time.Hour),
		},
		{
			ID:           "notetaker-002",
			State:        domain.NotetakerStateAttending,
			MeetingLink:  "https://meet.google.com/abc-defg-hij",
			MeetingTitle: "Weekly Standup",
			CreatedAt:    now.Add(-30 * time.Minute),
			UpdatedAt:    now.Add(-5 * time.Minute),
		},
		{
			ID:           "notetaker-003",
			State:        domain.NotetakerStateScheduled,
			MeetingLink:  "https://teams.microsoft.com/l/meetup-join/xyz",
			MeetingTitle: "Client Demo",
			JoinTime:     now.Add(2 * time.Hour),
			CreatedAt:    now.Add(-24 * time.Hour),
			UpdatedAt:    now.Add(-24 * time.Hour),
		},
	}, nil
}

// GetNotetaker returns a demo notetaker.
func (d *DemoClient) GetNotetaker(ctx context.Context, grantID, notetakerID string) (*domain.Notetaker, error) {
	notetakers, _ := d.ListNotetakers(ctx, grantID, nil)
	for _, nt := range notetakers {
		if nt.ID == notetakerID {
			return &nt, nil
		}
	}
	return &notetakers[0], nil
}

// CreateNotetaker simulates creating a notetaker.
func (d *DemoClient) CreateNotetaker(ctx context.Context, grantID string, req *domain.CreateNotetakerRequest) (*domain.Notetaker, error) {
	now := time.Now()
	nt := &domain.Notetaker{
		ID:          "new-notetaker",
		State:       domain.NotetakerStateScheduled,
		MeetingLink: req.MeetingLink,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if req.JoinTime > 0 {
		nt.JoinTime = time.Unix(req.JoinTime, 0)
	}
	if req.BotConfig != nil {
		nt.BotConfig = req.BotConfig
	}
	return nt, nil
}

// DeleteNotetaker simulates deleting a notetaker.
func (d *DemoClient) DeleteNotetaker(ctx context.Context, grantID, notetakerID string) error {
	return nil
}

// GetNotetakerMedia returns demo notetaker media.
func (d *DemoClient) GetNotetakerMedia(ctx context.Context, grantID, notetakerID string) (*domain.MediaData, error) {
	return &domain.MediaData{
		Recording: &domain.MediaFile{
			URL:         "https://storage.nylas.com/recordings/demo-recording.mp4",
			ContentType: "video/mp4",
			Size:        125829120, // 120 MB
			ExpiresAt:   time.Now().Add(24 * time.Hour).Unix(),
		},
		Transcript: &domain.MediaFile{
			URL:         "https://storage.nylas.com/transcripts/demo-transcript.json",
			ContentType: "application/json",
			Size:        51200, // 50 KB
			ExpiresAt:   time.Now().Add(24 * time.Hour).Unix(),
		},
	}, nil
}

// ListInboundInboxes returns demo inbound inboxes.
