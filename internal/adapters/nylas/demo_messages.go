package nylas

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// GetMessages returns demo messages.
func (d *DemoClient) GetMessages(ctx context.Context, grantID string, limit int) ([]domain.Message, error) {
	return d.getDemoMessages(), nil
}

// GetMessagesWithParams returns demo messages.
func (d *DemoClient) GetMessagesWithParams(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
	return d.getDemoMessages(), nil
}

// GetMessagesWithCursor returns demo messages with pagination.
func (d *DemoClient) GetMessagesWithCursor(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error) {
	return &domain.MessageListResponse{
		Data: d.getDemoMessages(),
	}, nil
}

func (d *DemoClient) getDemoMessages() []domain.Message {
	now := time.Now()
	return []domain.Message{
		{
			ID:       "msg-001",
			Subject:  "Q4 Planning Meeting - Action Items",
			From:     []domain.EmailParticipant{{Name: "Sarah Chen", Email: "sarah.chen@company.com"}},
			To:       []domain.EmailParticipant{{Name: "Demo User", Email: "demo@example.com"}},
			Date:     now.Add(-15 * time.Minute),
			Unread:   true,
			Starred:  true,
			Snippet:  "Hi team, here are the action items from today's planning meeting...",
			Body:     "Hi team,\n\nHere are the action items from today's planning meeting:\n\n1. Review Q4 roadmap by Friday\n2. Submit budget proposals\n3. Schedule 1:1s with new team members\n\nBest,\nSarah",
			ThreadID: "thread-001",
		},
		{
			ID:       "msg-002",
			Subject:  "[GitHub] Pull request #247: Add dark mode support",
			From:     []domain.EmailParticipant{{Name: "GitHub", Email: "noreply@github.com"}},
			To:       []domain.EmailParticipant{{Name: "Demo User", Email: "demo@example.com"}},
			Date:     now.Add(-2 * time.Hour),
			Unread:   true,
			Starred:  false,
			Snippet:  "alex-dev requested your review on: Add dark mode support for the dashboard...",
			Body:     "alex-dev requested your review on:\n\nAdd dark mode support for the dashboard\n\n+156 -23 lines changed\n\nView pull request: https://github.com/example/repo/pull/247",
			ThreadID: "thread-002",
		},
		{
			ID:       "msg-003",
			Subject:  "Your AWS bill for December 2024",
			From:     []domain.EmailParticipant{{Name: "Amazon Web Services", Email: "billing@aws.amazon.com"}},
			To:       []domain.EmailParticipant{{Name: "Demo User", Email: "demo@example.com"}},
			Date:     now.Add(-5 * time.Hour),
			Unread:   false,
			Starred:  false,
			Snippet:  "Your AWS charges for December 2024 are $127.43. View your detailed bill...",
			Body:     "Hello,\n\nYour AWS charges for December 2024 are $127.43.\n\nView your detailed bill in the AWS Billing Console.\n\nThank you for using Amazon Web Services.",
			ThreadID: "thread-003",
		},
		{
			ID:       "msg-004",
			Subject:  "Re: Lunch tomorrow?",
			From:     []domain.EmailParticipant{{Name: "Mike Johnson", Email: "mike.j@gmail.com"}},
			To:       []domain.EmailParticipant{{Name: "Demo User", Email: "demo@example.com"}},
			Date:     now.Add(-1 * 24 * time.Hour),
			Unread:   false,
			Starred:  true,
			Snippet:  "Sounds great! How about that new Italian place on 5th? I heard they have...",
			Body:     "Sounds great! How about that new Italian place on 5th? I heard they have amazing pasta.\n\nLet's meet at 12:30?\n\n- Mike",
			ThreadID: "thread-004",
		},
		{
			ID:       "msg-005",
			Subject:  "Weekly Newsletter: Top Tech Stories",
			From:     []domain.EmailParticipant{{Name: "TechCrunch", Email: "newsletter@techcrunch.com"}},
			To:       []domain.EmailParticipant{{Name: "Demo User", Email: "demo@example.com"}},
			Date:     now.Add(-1*24*time.Hour - 3*time.Hour),
			Unread:   false,
			Starred:  false,
			Snippet:  "This week's top stories: AI breakthroughs, startup funding rounds, and more...",
			Body:     "This week's top stories:\n\n1. OpenAI announces new model\n2. Startup raises $50M Series B\n3. Apple's latest patent reveals AR glasses plans\n\nRead more at techcrunch.com",
			ThreadID: "thread-005",
		},
		{
			ID:       "msg-006",
			Subject:  "Your package has been delivered",
			From:     []domain.EmailParticipant{{Name: "Amazon", Email: "ship-confirm@amazon.com"}},
			To:       []domain.EmailParticipant{{Name: "Demo User", Email: "demo@example.com"}},
			Date:     now.Add(-2 * 24 * time.Hour),
			Unread:   false,
			Starred:  false,
			Snippet:  "Your package was delivered. It was handed directly to a resident...",
			Body:     "Your package was delivered.\n\nDelivered: December 15, 2024, 2:34 PM\nLeft with: Resident\n\nTrack your package at amazon.com/orders",
			ThreadID: "thread-006",
		},
		{
			ID:       "msg-007",
			Subject:  "Invitation: Team Standup @ Daily 9:00 AM",
			From:     []domain.EmailParticipant{{Name: "Google Calendar", Email: "calendar-notification@google.com"}},
			To:       []domain.EmailParticipant{{Name: "Demo User", Email: "demo@example.com"}},
			Date:     now.Add(-3 * 24 * time.Hour),
			Unread:   false,
			Starred:  false,
			Snippet:  "You've been invited to a recurring event: Team Standup...",
			Body:     "You've been invited to a recurring event.\n\nTeam Standup\nDaily at 9:00 AM - 9:15 AM\n\nJoin with Google Meet: meet.google.com/abc-defg-hij",
			ThreadID: "thread-007",
		},
		{
			ID:       "msg-008",
			Subject:  "Your Spotify Wrapped 2024 is here!",
			From:     []domain.EmailParticipant{{Name: "Spotify", Email: "no-reply@spotify.com"}},
			To:       []domain.EmailParticipant{{Name: "Demo User", Email: "demo@example.com"}},
			Date:     now.Add(-5 * 24 * time.Hour),
			Unread:   false,
			Starred:  false,
			Snippet:  "See your year in music. You listened to 47,832 minutes of music this year...",
			Body:     "Your 2024 Wrapped is here!\n\nYou listened to 47,832 minutes of music this year.\nYour top genre: Electronic\nYour top artist: Daft Punk\n\nSee your full Wrapped at spotify.com/wrapped",
			ThreadID: "thread-008",
		},
	}
}

// GetMessage returns a demo message.
func (d *DemoClient) GetMessage(ctx context.Context, grantID, messageID string) (*domain.Message, error) {
	messages := d.getDemoMessages()
	for _, msg := range messages {
		if msg.ID == messageID {
			return &msg, nil
		}
	}
	return &messages[0], nil
}

// GetMessageWithFields retrieves a demo message with optional field selection.
func (d *DemoClient) GetMessageWithFields(ctx context.Context, grantID, messageID string, fields string) (*domain.Message, error) {
	msg, err := d.GetMessage(ctx, grantID, messageID)
	if err != nil {
		return nil, err
	}

	// Add demo MIME data if raw_mime field requested
	if fields == "raw_mime" {
		msg.RawMIME = "From: demo@example.com\r\nTo: user@example.com\r\nSubject: Demo MIME Message\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nDate: " + msg.Date.Format(time.RFC1123Z) + "\r\n\r\n" + msg.Body
	}

	return msg, nil
}

// SendMessage simulates sending a message.
func (d *DemoClient) SendMessage(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error) {
	msg := &domain.Message{
		ID:   "sent-demo-msg",
		Date: time.Now(),
	}
	if req != nil {
		msg.Subject = req.Subject
		msg.To = req.To
		msg.Body = req.Body
	}
	return msg, nil
}

// SendRawMessage simulates sending a raw MIME message.
func (d *DemoClient) SendRawMessage(ctx context.Context, grantID string, rawMIME []byte) (*domain.Message, error) {
	return &domain.Message{
		ID:      "sent-raw-demo-msg",
		Date:    time.Now(),
		RawMIME: string(rawMIME),
	}, nil
}

// UpdateMessage simulates updating a message.
func (d *DemoClient) UpdateMessage(ctx context.Context, grantID, messageID string, req *domain.UpdateMessageRequest) (*domain.Message, error) {
	msg := &domain.Message{ID: messageID, Subject: "Updated Message"}
	if req.Unread != nil {
		msg.Unread = *req.Unread
	}
	if req.Starred != nil {
		msg.Starred = *req.Starred
	}
	return msg, nil
}

// DeleteMessage simulates deleting a message.
func (d *DemoClient) DeleteMessage(ctx context.Context, grantID, messageID string) error {
	return nil
}
