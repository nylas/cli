package air

import (
	"testing"
	"time"
)

func TestBuildMockPageData(t *testing.T) {
	t.Parallel()

	data := buildMockPageData()

	// User info
	if data.UserName == "" {
		t.Error("expected non-empty UserName")
	}
	if data.UserEmail == "" {
		t.Error("expected non-empty UserEmail")
	}
	if data.UserAvatar == "" {
		t.Error("expected non-empty UserAvatar")
	}
	if !data.Configured {
		t.Error("expected mock page data to be configured")
	}
	if data.Provider != "nylas" {
		t.Errorf("expected provider 'nylas', got %s", data.Provider)
	}
	if data.DefaultGrantID == "" {
		t.Error("expected non-empty DefaultGrantID")
	}
	if len(data.Grants) != 1 {
		t.Fatalf("expected exactly 1 mock grant, got %d", len(data.Grants))
	}
	if !data.Grants[0].IsDefault {
		t.Error("expected mock grant to be the default")
	}

	// UI state
	if data.ActiveView != "email" {
		t.Errorf("expected ActiveView 'email', got %s", data.ActiveView)
	}
	if data.ActiveFolder != "inbox" {
		t.Errorf("expected ActiveFolder 'inbox', got %s", data.ActiveFolder)
	}

	// Folders
	if len(data.Folders) == 0 {
		t.Error("expected non-empty Folders")
	}

	// Check inbox folder exists and is active
	hasInbox := false
	for _, f := range data.Folders {
		if f.ID == "inbox" {
			hasInbox = true
			if !f.IsActive {
				t.Error("expected inbox to be active")
			}
			break
		}
	}
	if !hasInbox {
		t.Error("expected inbox folder")
	}

	// Emails
	if len(data.Emails) == 0 {
		t.Error("expected non-empty Emails")
	}

	// Check first email has expected fields
	firstEmail := data.Emails[0]
	if firstEmail.ID == "" {
		t.Error("expected email to have ID")
	}
	if firstEmail.Subject == "" {
		t.Error("expected email to have Subject")
	}
	if firstEmail.From == "" {
		t.Error("expected email to have From")
	}

	// Selected email
	if data.SelectedEmail == nil {
		t.Error("expected non-nil SelectedEmail")
	}

	// Calendars
	if len(data.Calendars) == 0 {
		t.Error("expected non-empty Calendars")
	}

	// Check for primary calendar
	hasPrimary := false
	for _, c := range data.Calendars {
		if c.IsPrimary {
			hasPrimary = true
			break
		}
	}
	if !hasPrimary {
		t.Error("expected a primary calendar")
	}

	// Events
	if len(data.Events) == 0 {
		t.Error("expected non-empty Events")
	}

	// Contacts
	if len(data.Contacts) == 0 {
		t.Error("expected non-empty Contacts")
	}

	// Check for VIP contacts
	hasVIP := false
	for _, c := range data.Contacts {
		if c.IsVIP {
			hasVIP = true
			break
		}
	}
	if !hasVIP {
		t.Error("expected at least one VIP contact")
	}
}

func TestMockEmailBody(t *testing.T) {
	t.Parallel()

	body := mockEmailBody()

	if body == "" {
		t.Error("expected non-empty email body")
	}

	// Check for HTML content
	if len(body) < 100 {
		t.Error("expected substantial email body content")
	}
}

func TestDemoGrants(t *testing.T) {
	t.Parallel()

	grants := demoGrants()

	if len(grants) != 3 {
		t.Errorf("expected 3 grants, got %d", len(grants))
	}

	// Check first grant
	if grants[0].ID != "demo-grant-001" {
		t.Errorf("expected ID 'demo-grant-001', got %s", grants[0].ID)
	}

	if grants[0].Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got %s", grants[0].Email)
	}

	if grants[0].Provider != "google" {
		t.Errorf("expected provider 'google', got %s", grants[0].Provider)
	}

	// Check different providers
	providers := make(map[string]bool)
	for _, g := range grants {
		providers[g.Provider] = true
	}
	if !providers["google"] {
		t.Error("expected a google provider")
	}
	if !providers["microsoft"] {
		t.Error("expected a microsoft provider")
	}
}

func TestDemoDefaultGrant(t *testing.T) {
	t.Parallel()

	defaultGrant := demoDefaultGrant()

	if defaultGrant != "demo-grant-001" {
		t.Errorf("expected 'demo-grant-001', got %s", defaultGrant)
	}
}

func TestDemoFolders(t *testing.T) {
	t.Parallel()

	folders := demoFolders()

	if len(folders) == 0 {
		t.Error("expected non-empty folders")
	}

	// Check for standard folders
	folderMap := make(map[string]bool)
	for _, f := range folders {
		folderMap[f.SystemFolder] = true
	}

	expectedFolders := []string{"inbox", "sent", "drafts", "trash", "spam"}
	for _, expected := range expectedFolders {
		if !folderMap[expected] {
			t.Errorf("expected folder with system_folder '%s'", expected)
		}
	}

	// Check inbox has unread count
	for _, f := range folders {
		if f.SystemFolder == "inbox" {
			if f.UnreadCount == 0 {
				t.Error("expected inbox to have unread count")
			}
			break
		}
	}
}

func TestDemoEmails(t *testing.T) {
	t.Parallel()

	emails := demoEmails()

	if len(emails) == 0 {
		t.Error("expected non-empty emails")
	}

	// Check first email
	first := emails[0]
	if first.ID == "" {
		t.Error("expected email to have ID")
	}
	if first.Subject == "" {
		t.Error("expected email to have Subject")
	}
	if len(first.From) == 0 {
		t.Error("expected email to have From")
	}
	if first.Date == 0 {
		t.Error("expected email to have Date")
	}

	// Check for variety (unread, starred, attachments)
	hasUnread := false
	hasStarred := false
	hasAttachment := false

	for _, e := range emails {
		if e.Unread {
			hasUnread = true
		}
		if e.Starred {
			hasStarred = true
		}
		if len(e.Attachments) > 0 {
			hasAttachment = true
		}
	}

	if !hasUnread {
		t.Error("expected at least one unread email")
	}
	if !hasStarred {
		t.Error("expected at least one starred email")
	}
	if !hasAttachment {
		t.Error("expected at least one email with attachments")
	}
}

func TestDemoDrafts(t *testing.T) {
	t.Parallel()

	drafts := demoDrafts()

	if len(drafts) == 0 {
		t.Error("expected non-empty drafts")
	}

	// Check first draft
	first := drafts[0]
	if first.ID == "" {
		t.Error("expected draft to have ID")
	}
	if first.Subject == "" {
		t.Error("expected draft to have Subject")
	}
	if first.Date == 0 {
		t.Error("expected draft to have Date")
	}
	if len(first.To) == 0 {
		t.Error("expected draft to have recipients")
	}
}

func TestDemoCalendars(t *testing.T) {
	t.Parallel()

	calendars := demoCalendars()

	if len(calendars) == 0 {
		t.Error("expected non-empty calendars")
	}

	// Check for primary calendar
	hasPrimary := false
	for _, c := range calendars {
		if c.IsPrimary {
			hasPrimary = true
			if c.ID == "" {
				t.Error("expected primary calendar to have ID")
			}
			if c.Name == "" {
				t.Error("expected primary calendar to have Name")
			}
			break
		}
	}

	if !hasPrimary {
		t.Error("expected a primary calendar")
	}

	// Check for read-only calendar
	hasReadOnly := false
	for _, c := range calendars {
		if c.ReadOnly {
			hasReadOnly = true
			break
		}
	}
	if !hasReadOnly {
		t.Error("expected at least one read-only calendar")
	}

	// Check all calendars have colors
	for _, c := range calendars {
		if c.HexColor == "" {
			t.Errorf("expected calendar %s to have HexColor", c.ID)
		}
	}
}

func TestDemoEvents(t *testing.T) {
	t.Parallel()

	events := demoEvents()

	if len(events) == 0 {
		t.Error("expected non-empty events")
	}

	// Check first event
	first := events[0]
	if first.ID == "" {
		t.Error("expected event to have ID")
	}
	if first.Title == "" {
		t.Error("expected event to have Title")
	}
	if first.CalendarID == "" {
		t.Error("expected event to have CalendarID")
	}
	if first.StartTime == 0 {
		t.Error("expected event to have StartTime")
	}
	if first.EndTime == 0 {
		t.Error("expected event to have EndTime")
	}

	// Check for variety (all-day, with conferencing, with participants)
	hasAllDay := false
	hasConferencing := false
	hasParticipants := false

	for _, e := range events {
		if e.IsAllDay {
			hasAllDay = true
		}
		if e.Conferencing != nil {
			hasConferencing = true
		}
		if len(e.Participants) > 0 {
			hasParticipants = true
		}
	}

	if !hasAllDay {
		t.Error("expected at least one all-day event")
	}
	if !hasConferencing {
		t.Error("expected at least one event with conferencing")
	}
	if !hasParticipants {
		t.Error("expected at least one event with participants")
	}
}

func TestPageDataTypes(t *testing.T) {
	t.Parallel()

	// Test that PageData struct can be created with all fields
	data := PageData{
		UserName:       "Test User",
		UserEmail:      "test@example.com",
		UserAvatar:     "TU",
		Configured:     true,
		ClientID:       "client-123",
		Region:         "us",
		HasAPIKey:      true,
		DefaultGrantID: "grant-123",
		Provider:       "google",
		Grants: []GrantInfo{
			{ID: "g1", Email: "a@b.com", Provider: "google", IsDefault: true},
		},
		Folders: []Folder{
			{ID: "inbox", Name: "Inbox", Icon: "inbox", Count: 10, UnreadCount: 5, IsActive: true},
		},
		Emails: []Email{
			{ID: "e1", From: "Test", Subject: "Hello"},
		},
		Calendars: []Calendar{
			{ID: "c1", Name: "Work", Color: "#fff", IsPrimary: true},
		},
		Events: []Event{
			{ID: "ev1", Title: "Meeting", StartTime: "10:00 AM", EndTime: "11:00 AM"},
		},
		Contacts: []Contact{
			{ID: "co1", Name: "John", Email: "john@example.com"},
		},
		ActiveView:    "email",
		ActiveFolder:  "inbox",
		UnreadCount:   5,
		SyncedAt:      time.Now(),
		AccountsCount: 1,
	}

	if data.UserName != "Test User" {
		t.Error("PageData field assignment failed")
	}
}

func TestFolderType(t *testing.T) {
	t.Parallel()

	folder := Folder{
		ID:          "inbox",
		Name:        "Inbox",
		Icon:        "inbox",
		Count:       100,
		UnreadCount: 25,
		IsActive:    true,
	}

	if folder.ID != "inbox" {
		t.Error("expected ID 'inbox'")
	}
	if folder.UnreadCount != 25 {
		t.Error("expected UnreadCount 25")
	}
}

func TestEmailType(t *testing.T) {
	t.Parallel()

	email := Email{
		ID:              "e1",
		From:            "Sender Name",
		FromEmail:       "sender@example.com",
		FromAvatar:      "SN",
		Subject:         "Test Subject",
		Preview:         "Preview text...",
		Body:            "<p>Full body</p>",
		Time:            "2h",
		IsUnread:        true,
		IsStarred:       false,
		IsSelected:      true,
		HasAttachment:   true,
		AttachmentCount: 2,
		Priority:        "high",
		HasAISummary:    true,
		Attachments: []Attachment{
			{Name: "file.pdf", Size: "1 MB", Icon: "file"},
		},
	}

	if email.Subject != "Test Subject" {
		t.Error("expected Subject 'Test Subject'")
	}
	if email.AttachmentCount != 2 {
		t.Error("expected AttachmentCount 2")
	}
	if len(email.Attachments) != 1 {
		t.Error("expected 1 attachment")
	}
}

func TestEventType(t *testing.T) {
	t.Parallel()

	event := Event{
		ID:             "ev1",
		Title:          "Team Meeting",
		Description:    "Weekly sync",
		StartTime:      "10:00 AM",
		EndTime:        "11:00 AM",
		Location:       "Room A",
		IsFocusTime:    false,
		IsAISuggestion: true,
		Attendees: []Attendee{
			{Name: "JD", Avatar: "JD", Color: "#fff"},
		},
		Tags: []string{"meeting", "weekly"},
	}

	if event.Title != "Team Meeting" {
		t.Error("expected Title 'Team Meeting'")
	}
	if !event.IsAISuggestion {
		t.Error("expected IsAISuggestion to be true")
	}
	if len(event.Attendees) != 1 {
		t.Error("expected 1 attendee")
	}
	if len(event.Tags) != 2 {
		t.Error("expected 2 tags")
	}
}

func TestContactType(t *testing.T) {
	t.Parallel()

	contact := Contact{
		ID:          "c1",
		Name:        "John Doe",
		Email:       "john@example.com",
		Role:        "Engineer",
		Company:     "Acme Inc",
		Avatar:      "JD",
		AvatarColor: "#8b5cf6",
		IsVIP:       true,
		Letter:      "J",
	}

	if contact.Name != "John Doe" {
		t.Error("expected Name 'John Doe'")
	}
	if !contact.IsVIP {
		t.Error("expected IsVIP to be true")
	}
	if contact.Letter != "J" {
		t.Error("expected Letter 'J'")
	}
}
