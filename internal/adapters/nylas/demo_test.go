//go:build !integration
// +build !integration

package nylas

import (
	"context"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestDemoClient(t *testing.T) {
	client := NewDemoClient()
	ctx := context.Background()

	t.Run("returns_demo_messages", func(t *testing.T) {
		messages, err := client.GetMessages(ctx, "demo-grant", 10)
		if err != nil {
			t.Fatalf("GetMessages failed: %v", err)
		}
		if len(messages) == 0 {
			t.Error("Expected demo messages, got none")
		}
		// Check first message has expected fields
		if messages[0].Subject == "" {
			t.Error("Expected message to have subject")
		}
		if len(messages[0].From) == 0 {
			t.Error("Expected message to have From field")
		}
	})

	t.Run("returns_demo_events", func(t *testing.T) {
		events, err := client.GetEvents(ctx, "demo-grant", "primary", nil)
		if err != nil {
			t.Fatalf("GetEvents failed: %v", err)
		}
		if len(events) == 0 {
			t.Error("Expected demo events, got none")
		}
		// Check first event has expected fields
		if events[0].Title == "" {
			t.Error("Expected event to have title")
		}
	})

	t.Run("returns_demo_contacts", func(t *testing.T) {
		contacts, err := client.GetContacts(ctx, "demo-grant", nil)
		if err != nil {
			t.Fatalf("GetContacts failed: %v", err)
		}
		if len(contacts) == 0 {
			t.Error("Expected demo contacts, got none")
		}
		// Check first contact has expected fields
		if contacts[0].GivenName == "" {
			t.Error("Expected contact to have given name")
		}
	})

	t.Run("returns_demo_webhooks", func(t *testing.T) {
		webhooks, err := client.ListWebhooks(ctx)
		if err != nil {
			t.Fatalf("ListWebhooks failed: %v", err)
		}
		if len(webhooks) == 0 {
			t.Error("Expected demo webhooks, got none")
		}
		// Check first webhook has expected fields
		if webhooks[0].Description == "" {
			t.Error("Expected webhook to have description")
		}
		if webhooks[0].WebhookURL == "" {
			t.Error("Expected webhook to have URL")
		}
	})

	t.Run("returns_demo_grants", func(t *testing.T) {
		grants, err := client.ListGrants(ctx)
		if err != nil {
			t.Fatalf("ListGrants failed: %v", err)
		}
		if len(grants) == 0 {
			t.Error("Expected demo grants, got none")
		}
		// Check first grant has expected fields
		if grants[0].Email == "" {
			t.Error("Expected grant to have email")
		}
		if grants[0].Provider == "" {
			t.Error("Expected grant to have provider")
		}
	})

	t.Run("returns_demo_calendars", func(t *testing.T) {
		calendars, err := client.GetCalendars(ctx, "demo-grant")
		if err != nil {
			t.Fatalf("GetCalendars failed: %v", err)
		}
		if len(calendars) == 0 {
			t.Error("Expected demo calendars, got none")
		}
		// Check first calendar has expected fields
		if calendars[0].Name == "" {
			t.Error("Expected calendar to have name")
		}
	})

	t.Run("returns_demo_folders", func(t *testing.T) {
		folders, err := client.GetFolders(ctx, "demo-grant")
		if err != nil {
			t.Fatalf("GetFolders failed: %v", err)
		}
		if len(folders) == 0 {
			t.Error("Expected demo folders, got none")
		}
		// Check first folder has expected fields
		if folders[0].Name == "" {
			t.Error("Expected folder to have name")
		}
	})

	t.Run("simulates_send_message", func(t *testing.T) {
		msg, err := client.SendMessage(ctx, "demo-grant", nil)
		if err != nil {
			t.Fatalf("SendMessage failed: %v", err)
		}
		if msg.ID == "" {
			t.Error("Expected sent message to have ID")
		}
	})

	t.Run("no_errors_on_delete_operations", func(t *testing.T) {
		if err := client.DeleteMessage(ctx, "demo-grant", "msg-001"); err != nil {
			t.Errorf("DeleteMessage should not error: %v", err)
		}
		if err := client.DeleteContact(ctx, "demo-grant", "contact-001"); err != nil {
			t.Errorf("DeleteContact should not error: %v", err)
		}
		if err := client.DeleteEvent(ctx, "demo-grant", "primary", "event-001"); err != nil {
			t.Errorf("DeleteEvent should not error: %v", err)
		}
		if err := client.DeleteWebhook(ctx, "webhook-001"); err != nil {
			t.Errorf("DeleteWebhook should not error: %v", err)
		}
	})
}

func TestDemoClientMessages(t *testing.T) {
	client := NewDemoClient()
	ctx := context.Background()

	messages, _ := client.GetMessages(ctx, "demo-grant", 10)

	t.Run("messages_have_realistic_subjects", func(t *testing.T) {
		subjects := make(map[string]bool)
		for _, msg := range messages {
			subjects[msg.Subject] = true
		}

		// Check for some expected demo subjects
		expectedSubjects := []string{
			"Q4 Planning Meeting - Action Items",
			"[GitHub] Pull request #247: Add dark mode support",
			"Re: Lunch tomorrow?",
		}

		for _, expected := range expectedSubjects {
			if !subjects[expected] {
				t.Errorf("Expected demo message with subject %q", expected)
			}
		}
	})

	t.Run("messages_have_different_states", func(t *testing.T) {
		hasUnread := false
		hasRead := false
		hasStarred := false

		for _, msg := range messages {
			if msg.Unread {
				hasUnread = true
			} else {
				hasRead = true
			}
			if msg.Starred {
				hasStarred = true
			}
		}

		if !hasUnread {
			t.Error("Expected some unread messages in demo data")
		}
		if !hasRead {
			t.Error("Expected some read messages in demo data")
		}
		if !hasStarred {
			t.Error("Expected some starred messages in demo data")
		}
	})
}

func TestDemoClientEvents(t *testing.T) {
	client := NewDemoClient()
	ctx := context.Background()

	events, _ := client.GetEvents(ctx, "demo-grant", "primary", nil)

	t.Run("events_have_realistic_titles", func(t *testing.T) {
		titles := make(map[string]bool)
		for _, event := range events {
			titles[event.Title] = true
		}

		// Check for some expected demo events
		expectedTitles := []string{
			"Team Standup",
			"1:1 with Manager",
			"Lunch Break",
		}

		for _, expected := range expectedTitles {
			if !titles[expected] {
				t.Errorf("Expected demo event with title %q", expected)
			}
		}
	})

	t.Run("events_have_time_ranges", func(t *testing.T) {
		for _, event := range events {
			if event.When.StartTime == 0 {
				t.Errorf("Event %q should have start time", event.Title)
			}
			if event.When.EndTime == 0 {
				t.Errorf("Event %q should have end time", event.Title)
			}
			if event.When.EndTime <= event.When.StartTime {
				t.Errorf("Event %q end time should be after start time", event.Title)
			}
		}
	})
}

func TestDemoClientContacts(t *testing.T) {
	client := NewDemoClient()
	ctx := context.Background()

	contacts, _ := client.GetContacts(ctx, "demo-grant", nil)

	t.Run("contacts_have_names_and_emails", func(t *testing.T) {
		for _, contact := range contacts {
			if contact.GivenName == "" && contact.Surname == "" {
				t.Error("Contact should have a name")
			}
			if len(contact.Emails) == 0 {
				t.Errorf("Contact %s should have email", contact.GivenName)
			}
		}
	})
}

func TestDemoClientNotetakers(t *testing.T) {
	client := NewDemoClient()
	ctx := context.Background()

	t.Run("returns_demo_notetakers", func(t *testing.T) {
		notetakers, err := client.ListNotetakers(ctx, "demo-grant", nil)
		if err != nil {
			t.Fatalf("ListNotetakers failed: %v", err)
		}
		if len(notetakers) == 0 {
			t.Error("Expected demo notetakers, got none")
		}
		// Check first notetaker has expected fields
		if notetakers[0].ID == "" {
			t.Error("Expected notetaker to have ID")
		}
		if notetakers[0].State == "" {
			t.Error("Expected notetaker to have state")
		}
		if notetakers[0].MeetingLink == "" {
			t.Error("Expected notetaker to have meeting link")
		}
	})

	t.Run("notetakers_have_different_states", func(t *testing.T) {
		notetakers, _ := client.ListNotetakers(ctx, "demo-grant", nil)
		states := make(map[string]bool)
		for _, nt := range notetakers {
			states[nt.State] = true
		}

		// Check for expected states in demo data
		if !states["complete"] {
			t.Error("Expected at least one complete notetaker")
		}
		if !states["attending"] {
			t.Error("Expected at least one attending notetaker")
		}
		if !states["scheduled"] {
			t.Error("Expected at least one scheduled notetaker")
		}
	})

	t.Run("get_notetaker_by_id", func(t *testing.T) {
		notetaker, err := client.GetNotetaker(ctx, "demo-grant", "notetaker-001")
		if err != nil {
			t.Fatalf("GetNotetaker failed: %v", err)
		}
		if notetaker.ID != "notetaker-001" {
			t.Errorf("Expected notetaker ID notetaker-001, got %s", notetaker.ID)
		}
	})

	t.Run("create_notetaker", func(t *testing.T) {
		req := &domain.CreateNotetakerRequest{
			MeetingLink: "https://zoom.us/j/test123",
			JoinTime:    1234567890,
			BotConfig: &domain.BotConfig{
				Name: "TestBot",
			},
		}
		notetaker, err := client.CreateNotetaker(ctx, "demo-grant", req)
		if err != nil {
			t.Fatalf("CreateNotetaker failed: %v", err)
		}
		if notetaker.ID == "" {
			t.Error("Expected created notetaker to have ID")
		}
		if notetaker.MeetingLink != req.MeetingLink {
			t.Errorf("Expected meeting link %s, got %s", req.MeetingLink, notetaker.MeetingLink)
		}
		if notetaker.State != "scheduled" {
			t.Errorf("Expected state scheduled, got %s", notetaker.State)
		}
	})

	t.Run("delete_notetaker", func(t *testing.T) {
		if err := client.DeleteNotetaker(ctx, "demo-grant", "notetaker-001"); err != nil {
			t.Errorf("DeleteNotetaker should not error: %v", err)
		}
	})

	t.Run("get_notetaker_media", func(t *testing.T) {
		media, err := client.GetNotetakerMedia(ctx, "demo-grant", "notetaker-001")
		if err != nil {
			t.Fatalf("GetNotetakerMedia failed: %v", err)
		}
		if media.Recording == nil {
			t.Error("Expected media to have recording")
		}
		if media.Transcript == nil {
			t.Error("Expected media to have transcript")
		}
		if media.Recording.URL == "" {
			t.Error("Expected recording to have URL")
		}
		if media.Transcript.URL == "" {
			t.Error("Expected transcript to have URL")
		}
		if media.Recording.ContentType == "" {
			t.Error("Expected recording to have content type")
		}
		if media.Recording.Size == 0 {
			t.Error("Expected recording to have size")
		}
	})
}

func TestDemoClientSendMessageWithTracking(t *testing.T) {
	client := NewDemoClient()
	ctx := context.Background()

	t.Run("send_message_with_tracking", func(t *testing.T) {
		req := &domain.SendMessageRequest{
			Subject: "Test email with tracking",
			Body:    "This is a test email",
			To:      []domain.EmailParticipant{{Email: "test@example.com"}},
			TrackingOpts: &domain.TrackingOptions{
				Opens: true,
				Links: true,
				Label: "test-campaign",
			},
		}
		msg, err := client.SendMessage(ctx, "demo-grant", req)
		if err != nil {
			t.Fatalf("SendMessage with tracking failed: %v", err)
		}
		if msg.ID == "" {
			t.Error("Expected sent message to have ID")
		}
		if msg.Subject != req.Subject {
			t.Errorf("Expected subject %s, got %s", req.Subject, msg.Subject)
		}
	})

	t.Run("send_message_with_metadata", func(t *testing.T) {
		req := &domain.SendMessageRequest{
			Subject: "Test email with metadata",
			Body:    "This is a test email",
			To:      []domain.EmailParticipant{{Email: "test@example.com"}},
			Metadata: map[string]string{
				"campaign":    "q4-newsletter",
				"customer_id": "cust-123",
			},
		}
		msg, err := client.SendMessage(ctx, "demo-grant", req)
		if err != nil {
			t.Fatalf("SendMessage with metadata failed: %v", err)
		}
		if msg.ID == "" {
			t.Error("Expected sent message to have ID")
		}
	})
}
