package nylas

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// GetThreads returns demo threads.
func (d *DemoClient) GetThreads(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error) {
	return d.getDemoThreads(), nil
}

func (d *DemoClient) getDemoThreads() []domain.Thread {
	now := time.Now()
	return []domain.Thread{
		{
			ID:                    "thread-001",
			Subject:               "Q4 Planning Meeting - Action Items",
			Unread:                true,
			Starred:               true,
			Snippet:               "Hi team, here are the action items from today's planning meeting...",
			LatestMessageRecvDate: now.Add(-15 * time.Minute),
			EarliestMessageDate:   now.Add(-2 * time.Hour),
			MessageIDs:            []string{"msg-001"},
			Participants: []domain.EmailParticipant{
				{Name: "Sarah Chen", Email: "sarah.chen@company.com"},
				{Name: "Demo User", Email: "demo@example.com"},
			},
			HasAttachments: true,
		},
		{
			ID:                    "thread-002",
			Subject:               "[GitHub] Pull request #247: Add dark mode support",
			Unread:                true,
			Starred:               false,
			Snippet:               "alex-dev requested your review on: Add dark mode support for the dashboard...",
			LatestMessageRecvDate: now.Add(-2 * time.Hour),
			EarliestMessageDate:   now.Add(-3 * time.Hour),
			MessageIDs:            []string{"msg-002", "msg-002b"},
			Participants: []domain.EmailParticipant{
				{Name: "GitHub", Email: "noreply@github.com"},
			},
		},
		{
			ID:                    "thread-003",
			Subject:               "Your AWS bill for December 2024",
			Unread:                false,
			Starred:               false,
			Snippet:               "Your AWS charges for December 2024 are $127.43. View your detailed bill...",
			LatestMessageRecvDate: now.Add(-5 * time.Hour),
			EarliestMessageDate:   now.Add(-5 * time.Hour),
			MessageIDs:            []string{"msg-003"},
			Participants: []domain.EmailParticipant{
				{Name: "Amazon Web Services", Email: "billing@aws.amazon.com"},
			},
		},
		{
			ID:                    "thread-004",
			Subject:               "Re: Lunch tomorrow?",
			Unread:                false,
			Starred:               true,
			Snippet:               "Sounds great! How about that new Italian place on 5th? I heard they have...",
			LatestMessageRecvDate: now.Add(-1 * 24 * time.Hour),
			EarliestMessageDate:   now.Add(-2 * 24 * time.Hour),
			MessageIDs:            []string{"msg-004", "msg-004b", "msg-004c"},
			Participants: []domain.EmailParticipant{
				{Name: "Mike Johnson", Email: "mike.j@gmail.com"},
				{Name: "Demo User", Email: "demo@example.com"},
			},
		},
		{
			ID:                    "thread-005",
			Subject:               "Weekly Newsletter: Top Tech Stories",
			Unread:                false,
			Starred:               false,
			Snippet:               "This week's top stories: AI breakthroughs, startup funding rounds, and more...",
			LatestMessageRecvDate: now.Add(-1*24*time.Hour - 3*time.Hour),
			EarliestMessageDate:   now.Add(-1*24*time.Hour - 3*time.Hour),
			MessageIDs:            []string{"msg-005"},
			Participants: []domain.EmailParticipant{
				{Name: "TechCrunch", Email: "newsletter@techcrunch.com"},
			},
		},
		{
			ID:                    "thread-006",
			Subject:               "Your package has been delivered",
			Unread:                false,
			Starred:               false,
			Snippet:               "Your package was delivered. It was handed directly to a resident...",
			LatestMessageRecvDate: now.Add(-2 * 24 * time.Hour),
			EarliestMessageDate:   now.Add(-3 * 24 * time.Hour),
			MessageIDs:            []string{"msg-006", "msg-006b"},
			Participants: []domain.EmailParticipant{
				{Name: "Amazon", Email: "ship-confirm@amazon.com"},
			},
		},
		{
			ID:                    "thread-007",
			Subject:               "Invitation: Team Standup @ Daily 9:00 AM",
			Unread:                false,
			Starred:               false,
			Snippet:               "You've been invited to a recurring event: Team Standup...",
			LatestMessageRecvDate: now.Add(-3 * 24 * time.Hour),
			EarliestMessageDate:   now.Add(-3 * 24 * time.Hour),
			MessageIDs:            []string{"msg-007"},
			Participants: []domain.EmailParticipant{
				{Name: "Google Calendar", Email: "calendar-notification@google.com"},
			},
		},
		{
			ID:                    "thread-008",
			Subject:               "Your Spotify Wrapped 2024 is here!",
			Unread:                false,
			Starred:               false,
			Snippet:               "See your year in music. You listened to 47,832 minutes of music this year...",
			LatestMessageRecvDate: now.Add(-5 * 24 * time.Hour),
			EarliestMessageDate:   now.Add(-5 * 24 * time.Hour),
			MessageIDs:            []string{"msg-008"},
			Participants: []domain.EmailParticipant{
				{Name: "Spotify", Email: "no-reply@spotify.com"},
			},
		},
	}
}

// GetThread returns a demo thread.
func (d *DemoClient) GetThread(ctx context.Context, grantID, threadID string) (*domain.Thread, error) {
	threads := d.getDemoThreads()
	for _, t := range threads {
		if t.ID == threadID {
			return &t, nil
		}
	}
	return &threads[0], nil
}

// UpdateThread simulates updating a thread.
func (d *DemoClient) UpdateThread(ctx context.Context, grantID, threadID string, req *domain.UpdateMessageRequest) (*domain.Thread, error) {
	thread := &domain.Thread{ID: threadID, Subject: "Updated Thread"}
	if req.Unread != nil {
		thread.Unread = *req.Unread
	}
	if req.Starred != nil {
		thread.Starred = *req.Starred
	}
	return thread, nil
}

// DeleteThread simulates deleting a thread.
func (d *DemoClient) DeleteThread(ctx context.Context, grantID, threadID string) error {
	return nil
}
