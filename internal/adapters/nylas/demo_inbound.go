package nylas

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func (d *DemoClient) ListInboundInboxes(ctx context.Context) ([]domain.InboundInbox, error) {
	now := time.Now()
	return []domain.InboundInbox{
		{
			ID:          "inbox-demo-001",
			Email:       "support@demo-app.nylas.email",
			GrantStatus: "valid",
			CreatedAt:   domain.UnixTime{Time: now.Add(-30 * 24 * time.Hour)},
			UpdatedAt:   domain.UnixTime{Time: now.Add(-1 * time.Hour)},
		},
		{
			ID:          "inbox-demo-002",
			Email:       "sales@demo-app.nylas.email",
			GrantStatus: "valid",
			CreatedAt:   domain.UnixTime{Time: now.Add(-14 * 24 * time.Hour)},
			UpdatedAt:   domain.UnixTime{Time: now.Add(-2 * time.Hour)},
		},
		{
			ID:          "inbox-demo-003",
			Email:       "info@demo-app.nylas.email",
			GrantStatus: "valid",
			CreatedAt:   domain.UnixTime{Time: now.Add(-7 * 24 * time.Hour)},
			UpdatedAt:   domain.UnixTime{Time: now.Add(-30 * time.Minute)},
		},
	}, nil
}

// GetInboundInbox returns a demo inbound inbox.
func (d *DemoClient) GetInboundInbox(ctx context.Context, grantID string) (*domain.InboundInbox, error) {
	inboxes, _ := d.ListInboundInboxes(ctx)
	for _, inbox := range inboxes {
		if inbox.ID == grantID {
			return &inbox, nil
		}
	}
	return &inboxes[0], nil
}

// CreateInboundInbox simulates creating an inbound inbox.
func (d *DemoClient) CreateInboundInbox(ctx context.Context, email string) (*domain.InboundInbox, error) {
	now := time.Now()
	return &domain.InboundInbox{
		ID:          "inbox-new",
		Email:       email + "@demo-app.nylas.email",
		GrantStatus: "valid",
		CreatedAt:   domain.UnixTime{Time: now},
		UpdatedAt:   domain.UnixTime{Time: now},
	}, nil
}

// DeleteInboundInbox simulates deleting an inbound inbox.
func (d *DemoClient) DeleteInboundInbox(ctx context.Context, grantID string) error {
	return nil
}

// GetInboundMessages returns demo inbound messages.
func (d *DemoClient) GetInboundMessages(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.InboundMessage, error) {
	now := time.Now()
	return []domain.InboundMessage{
		{
			ID:       "inbound-001",
			GrantID:  grantID,
			Subject:  "New Lead: Enterprise Plan Inquiry",
			From:     []domain.EmailParticipant{{Name: "John Smith", Email: "john.smith@bigcorp.com"}},
			To:       []domain.EmailParticipant{{Name: "Sales", Email: "sales@demo-app.nylas.email"}},
			Date:     now.Add(-10 * time.Minute),
			Unread:   true,
			Starred:  true,
			Snippet:  "Hi, I'm interested in learning more about your enterprise plan...",
			Body:     "Hi,\n\nI'm interested in learning more about your enterprise plan. Our company has about 500 employees and we're looking for a solution that can scale with our growth.\n\nCan we schedule a call this week?\n\nBest,\nJohn Smith\nVP of Engineering\nBigCorp Inc.",
			ThreadID: "inbound-thread-001",
		},
		{
			ID:       "inbound-002",
			GrantID:  grantID,
			Subject:  "Support Request: Integration Help",
			From:     []domain.EmailParticipant{{Name: "Sarah Johnson", Email: "sarah@startup.io"}},
			To:       []domain.EmailParticipant{{Name: "Support", Email: "support@demo-app.nylas.email"}},
			Date:     now.Add(-1 * time.Hour),
			Unread:   true,
			Starred:  false,
			Snippet:  "We're having trouble connecting our calendar integration...",
			Body:     "Hello,\n\nWe're having trouble connecting our calendar integration. The OAuth flow completes but we're not seeing any events sync.\n\nCan you help troubleshoot?\n\nThanks,\nSarah",
			ThreadID: "inbound-thread-002",
		},
		{
			ID:       "inbound-003",
			GrantID:  grantID,
			Subject:  "Partnership Opportunity",
			From:     []domain.EmailParticipant{{Name: "Mike Chen", Email: "mike@partner-company.com"}},
			To:       []domain.EmailParticipant{{Name: "Info", Email: "info@demo-app.nylas.email"}},
			Date:     now.Add(-3 * time.Hour),
			Unread:   false,
			Starred:  true,
			Snippet:  "We're a SaaS company looking for email integration partners...",
			Body:     "Hi there,\n\nWe're a SaaS company serving the healthcare industry and we're looking for email integration partners.\n\nWould love to explore a potential partnership.\n\nBest,\nMike Chen\nBusiness Development",
			ThreadID: "inbound-thread-003",
		},
		{
			ID:       "inbound-004",
			GrantID:  grantID,
			Subject:  "Billing Question",
			From:     []domain.EmailParticipant{{Name: "Lisa Park", Email: "lisa@customer.com"}},
			To:       []domain.EmailParticipant{{Name: "Support", Email: "support@demo-app.nylas.email"}},
			Date:     now.Add(-1 * 24 * time.Hour),
			Unread:   false,
			Starred:  false,
			Snippet:  "I have a question about my latest invoice...",
			Body:     "Hi,\n\nI have a question about my latest invoice. It seems like I was charged for 15 seats but we only have 10 active users.\n\nCan you look into this?\n\nThanks,\nLisa",
			ThreadID: "inbound-thread-004",
		},
		{
			ID:       "inbound-005",
			GrantID:  grantID,
			Subject:  "Feature Request: Dark Mode",
			From:     []domain.EmailParticipant{{Name: "Alex Rivera", Email: "alex@user.org"}},
			To:       []domain.EmailParticipant{{Name: "Info", Email: "info@demo-app.nylas.email"}},
			Date:     now.Add(-2 * 24 * time.Hour),
			Unread:   false,
			Starred:  false,
			Snippet:  "Would love to see dark mode support in the dashboard...",
			Body:     "Hello,\n\nI'm a happy user of your product but I work late hours and would really appreciate dark mode support.\n\nIs this on your roadmap?\n\nThanks,\nAlex",
			ThreadID: "inbound-thread-005",
		},
	}, nil
}
