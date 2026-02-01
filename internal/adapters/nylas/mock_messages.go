package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) GetMessages(ctx context.Context, grantID string, limit int) ([]domain.Message, error) {
	m.GetMessagesCalled = true
	m.LastGrantID = grantID
	if m.GetMessagesFunc != nil {
		return m.GetMessagesFunc(ctx, grantID, limit)
	}
	return []domain.Message{}, nil
}

// GetMessagesWithParams retrieves messages with query parameters.
func (m *MockClient) GetMessagesWithParams(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
	m.GetMessagesWithParamsCalled = true
	m.LastGrantID = grantID
	if m.GetMessagesWithParamsFunc != nil {
		return m.GetMessagesWithParamsFunc(ctx, grantID, params)
	}
	return []domain.Message{}, nil
}

// GetMessagesWithCursor retrieves messages with pagination cursor support.
func (m *MockClient) GetMessagesWithCursor(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error) {
	m.GetMessagesWithParamsCalled = true
	m.LastGrantID = grantID
	if m.GetMessagesWithParamsFunc != nil {
		msgs, err := m.GetMessagesWithParamsFunc(ctx, grantID, params)
		return &domain.MessageListResponse{Data: msgs}, err
	}
	return &domain.MessageListResponse{Data: []domain.Message{}}, nil
}

// GetMessage retrieves a single message.
func (m *MockClient) GetMessage(ctx context.Context, grantID, messageID string) (*domain.Message, error) {
	m.GetMessageCalled = true
	m.LastGrantID = grantID
	m.LastMessageID = messageID
	if m.GetMessageFunc != nil {
		return m.GetMessageFunc(ctx, grantID, messageID)
	}
	return &domain.Message{
		ID:      messageID,
		GrantID: grantID,
		Subject: "Test Message",
		From:    []domain.EmailParticipant{{Email: "sender@example.com"}},
		Body:    "Test body",
	}, nil
}

// GetMessageWithFields retrieves a message with optional field selection.
func (m *MockClient) GetMessageWithFields(ctx context.Context, grantID, messageID string, fields string) (*domain.Message, error) {
	msg, err := m.GetMessage(ctx, grantID, messageID)
	if err != nil {
		return nil, err
	}

	// Add mock MIME data if raw_mime field requested
	if fields == "raw_mime" || fields == "raw_mime," {
		msg.RawMIME = "From: mock@example.com\r\nTo: user@example.com\r\nSubject: Mock MIME Message\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nThis is a mock MIME message body."
	}

	return msg, nil
}

// SendMessage sends an email.
func (m *MockClient) SendMessage(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error) {
	m.SendMessageCalled = true
	m.LastGrantID = grantID
	if m.SendMessageFunc != nil {
		return m.SendMessageFunc(ctx, grantID, req)
	}
	return &domain.Message{
		ID:      "sent-message-id",
		GrantID: grantID,
		Subject: req.Subject,
		To:      req.To,
		Body:    req.Body,
	}, nil
}

// SendRawMessage sends a raw MIME message.
func (m *MockClient) SendRawMessage(ctx context.Context, grantID string, rawMIME []byte) (*domain.Message, error) {
	m.LastGrantID = grantID
	if m.SendRawMessageFunc != nil {
		return m.SendRawMessageFunc(ctx, grantID, rawMIME)
	}
	return &domain.Message{
		ID:      "sent-raw-message-id",
		GrantID: grantID,
		RawMIME: string(rawMIME),
	}, nil
}

// UpdateMessage updates message properties.
func (m *MockClient) UpdateMessage(ctx context.Context, grantID, messageID string, req *domain.UpdateMessageRequest) (*domain.Message, error) {
	m.UpdateMessageCalled = true
	m.LastGrantID = grantID
	m.LastMessageID = messageID
	if m.UpdateMessageFunc != nil {
		return m.UpdateMessageFunc(ctx, grantID, messageID, req)
	}
	msg := &domain.Message{
		ID:      messageID,
		GrantID: grantID,
		Subject: "Updated Message",
	}
	if req.Unread != nil {
		msg.Unread = *req.Unread
	}
	if req.Starred != nil {
		msg.Starred = *req.Starred
	}
	return msg, nil
}

// DeleteMessage deletes a message.
func (m *MockClient) DeleteMessage(ctx context.Context, grantID, messageID string) error {
	m.DeleteMessageCalled = true
	m.LastGrantID = grantID
	m.LastMessageID = messageID
	if m.DeleteMessageFunc != nil {
		return m.DeleteMessageFunc(ctx, grantID, messageID)
	}
	return nil
}

// ListScheduledMessages retrieves scheduled messages.
func (m *MockClient) ListScheduledMessages(ctx context.Context, grantID string) ([]domain.ScheduledMessage, error) {
	m.LastGrantID = grantID
	return []domain.ScheduledMessage{
		{ScheduleID: "schedule-1", Status: "pending", CloseTime: 1700000000},
		{ScheduleID: "schedule-2", Status: "scheduled", CloseTime: 1700100000},
	}, nil
}

// GetScheduledMessage retrieves a specific scheduled message.
func (m *MockClient) GetScheduledMessage(ctx context.Context, grantID, scheduleID string) (*domain.ScheduledMessage, error) {
	m.LastGrantID = grantID
	return &domain.ScheduledMessage{
		ScheduleID: scheduleID,
		Status:     "pending",
		CloseTime:  1700000000,
	}, nil
}

// CancelScheduledMessage cancels a scheduled message.
func (m *MockClient) CancelScheduledMessage(ctx context.Context, grantID, scheduleID string) error {
	m.LastGrantID = grantID
	return nil
}

// SmartCompose generates an AI-powered email draft.
func (m *MockClient) SmartCompose(ctx context.Context, grantID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
	m.LastGrantID = grantID
	return &domain.SmartComposeSuggestion{
		Suggestion: "Thank you for your email. I appreciate you reaching out and will respond to your inquiry shortly.",
	}, nil
}

// SmartComposeReply generates an AI-powered reply to a message.
func (m *MockClient) SmartComposeReply(ctx context.Context, grantID, messageID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
	m.LastGrantID = grantID
	m.LastMessageID = messageID
	return &domain.SmartComposeSuggestion{
		Suggestion: "Thank you for your message. I've reviewed your request and will follow up with the details shortly.",
	}, nil
}

// GetThreads retrieves threads.
