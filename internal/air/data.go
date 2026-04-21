package air

import "time"

// PageData contains all data needed to render the Air UI.
type PageData struct {
	// User info
	UserName   string
	UserEmail  string
	UserAvatar string

	// Auth & Config (Phase 2)
	Configured     bool
	ClientID       string
	Region         string
	HasAPIKey      bool
	DefaultGrantID string
	Provider       string
	Grants         []GrantInfo

	// Email data
	Folders       []Folder
	Emails        []Email
	SelectedEmail *Email

	// Calendar data
	Calendars []Calendar
	Events    []Event

	// Contacts data
	Contacts []Contact

	// UI state
	ActiveView    string // "email", "calendar", "contacts"
	ActiveFolder  string
	UnreadCount   int
	SyncedAt      time.Time
	AccountsCount int
}

// GrantInfo represents an authenticated account for the UI.
type GrantInfo struct {
	ID        string
	Email     string
	Provider  string
	IsDefault bool
}

// Folder represents an email folder.
type Folder struct {
	ID          string
	Name        string
	Icon        string
	Count       int
	UnreadCount int
	IsActive    bool
}

// Label represents an email label.
type Label struct {
	Name  string
	Color string
}

// Email represents an email message.
type Email struct {
	ID              string
	From            string
	FromEmail       string
	FromAvatar      string
	Subject         string
	Preview         string
	Body            string
	Time            string
	IsUnread        bool
	IsStarred       bool
	IsSelected      bool
	HasAttachment   bool
	AttachmentCount int
	Priority        string // "high", "normal", "low"
	HasAISummary    bool
	Attachments     []Attachment
}

// Attachment represents an email attachment.
type Attachment struct {
	Name string
	Size string
	Icon string
}

// Calendar represents a calendar.
type Calendar struct {
	ID        string
	Name      string
	Color     string
	IsPrimary bool
}

// Event represents a calendar event.
type Event struct {
	ID             string
	Title          string
	Description    string
	StartTime      string
	EndTime        string
	Location       string
	IsFocusTime    bool
	IsAISuggestion bool
	Attendees      []Attendee
	Tags           []string
}

// Attendee represents an event attendee.
type Attendee struct {
	Name   string
	Avatar string
	Color  string
}

// Contact represents a contact.
type Contact struct {
	ID          string
	Name        string
	Email       string
	Role        string
	Company     string
	Avatar      string
	AvatarColor string
	IsVIP       bool
	Letter      string // For alphabetical grouping
}

// buildMockPageData creates mock data for Phase 1 (static design).
func buildMockPageData() PageData {
	return PageData{
		UserName:       "Nylas",
		UserEmail:      "nylas@example.com",
		UserAvatar:     "N",
		Configured:     true,
		ClientID:       "demo-client-id",
		Region:         "us",
		HasAPIKey:      true,
		DefaultGrantID: "demo-grant-nylas",
		Provider:       "nylas",
		ActiveView:     "email",
		ActiveFolder:   "inbox",
		UnreadCount:    23,
		SyncedAt:       time.Now(),
		AccountsCount:  1,
		Grants: []GrantInfo{{
			ID:        "demo-grant-nylas",
			Email:     "nylas@example.com",
			Provider:  "nylas",
			IsDefault: true,
		}},

		Folders: []Folder{
			{ID: "inbox", Name: "Inbox", Icon: "inbox", Count: 23, UnreadCount: 23, IsActive: true},
			{ID: "starred", Name: "Starred", Icon: "star", Count: 8},
			{ID: "snoozed", Name: "Snoozed", Icon: "clock", Count: 3},
			{ID: "sent", Name: "Sent", Icon: "send"},
			{ID: "drafts", Name: "Drafts", Icon: "file", Count: 2},
			{ID: "scheduled", Name: "Scheduled", Icon: "calendar"},
			{ID: "trash", Name: "Trash", Icon: "trash"},
		},

		Emails: []Email{
			{
				ID:              "1",
				From:            "Sarah Chen",
				FromEmail:       "sarah.chen@company.com",
				FromAvatar:      "SC",
				Subject:         "Q4 Product Roadmap Review",
				Preview:         "Hi team, I've attached the updated roadmap for Q4...",
				Body:            mockEmailBody(),
				Time:            "2m",
				IsUnread:        true,
				IsStarred:       true,
				IsSelected:      true,
				HasAttachment:   true,
				AttachmentCount: 2,
				Priority:        "high",
				HasAISummary:    true,
				Attachments: []Attachment{
					{Name: "Q4_Roadmap_v2.pdf", Size: "2.4 MB", Icon: "file-text"},
					{Name: "Timeline_Changes.xlsx", Size: "156 KB", Icon: "file-spreadsheet"},
				},
			},
			{
				ID:         "2",
				From:       "GitHub",
				FromEmail:  "notifications@github.com",
				FromAvatar: "GH",
				Subject:    "[nylas/cli] PR #142: Add focus time feature",
				Preview:    "mergify[bot] merged 1 commit into main...",
				Time:       "15m",
				IsUnread:   true,
			},
			{
				ID:         "3",
				From:       "Alex Johnson",
				FromEmail:  "demo@example.com",
				FromAvatar: "AJ",
				Subject:    "Re: Meeting Tomorrow",
				Preview:    "That works for me. I'll send a calendar invite...",
				Time:       "1h",
			},
			{
				ID:         "4",
				From:       "Stripe",
				FromEmail:  "billing@stripe.com",
				FromAvatar: "ST",
				Subject:    "Your December invoice is ready",
				Preview:    "Your invoice for December 2024 is now available...",
				Time:       "3h",
				IsStarred:  true,
			},
			{
				ID:         "5",
				From:       "Design Weekly",
				FromEmail:  "newsletter@designweekly.com",
				FromAvatar: "DW",
				Subject:    "This week in design: AI tools reshaping...",
				Preview:    "The latest trends, tools, and inspiration...",
				Time:       "5h",
			},
			{
				ID:              "6",
				From:            "Michael Park",
				FromEmail:       "michael.park@company.com",
				FromAvatar:      "MP",
				Subject:         "API Integration Documentation",
				Preview:         "Here's the documentation you requested...",
				Time:            "Yesterday",
				HasAttachment:   true,
				AttachmentCount: 1,
			},
		},

		SelectedEmail: &Email{
			ID:              "1",
			From:            "Sarah Chen",
			FromEmail:       "sarah.chen@company.com",
			FromAvatar:      "SC",
			Subject:         "Q4 Product Roadmap Review",
			Body:            mockEmailBody(),
			Time:            "2m",
			IsStarred:       true,
			HasAttachment:   true,
			AttachmentCount: 2,
			Priority:        "high",
			Attachments: []Attachment{
				{Name: "Q4_Roadmap_v2.pdf", Size: "2.4 MB", Icon: "file-text"},
				{Name: "Timeline_Changes.xlsx", Size: "156 KB", Icon: "file-spreadsheet"},
			},
		},

		Calendars: []Calendar{
			{ID: "work", Name: "Work", Color: "#8b5cf6", IsPrimary: true},
			{ID: "personal", Name: "Personal", Color: "#22c55e"},
			{ID: "family", Name: "Family", Color: "#f59e0b"},
		},

		Events: []Event{
			{
				ID:          "e1",
				Title:       "Focus Time",
				Description: "Deep work block - protected by AI",
				StartTime:   "9:00 AM",
				EndTime:     "11:00 AM",
				IsFocusTime: true,
			},
			{
				ID:          "e2",
				Title:       "Team Standup",
				Description: "Weekly sync with engineering",
				StartTime:   "11:30 AM",
				EndTime:     "12:00 PM",
				Attendees: []Attendee{
					{Name: "SC", Color: "var(--gradient-1)"},
					{Name: "AJ", Color: "var(--gradient-2)"},
					{Name: "+3", Color: "var(--gradient-3)"},
				},
			},
			{
				ID:          "e3",
				Title:       "Product Review",
				Description: "Q4 roadmap discussion",
				StartTime:   "2:00 PM",
				EndTime:     "3:00 PM",
				Tags:        []string{"Zoom"},
			},
			{
				ID:             "e4",
				Title:          "Catch-up with Sarah",
				Description:    "Optimal slot across 3 timezones",
				StartTime:      "4:00 PM",
				EndTime:        "4:30 PM",
				IsAISuggestion: true,
			},
		},

		Contacts: []Contact{
			{ID: "c1", Name: "Alex Johnson", Email: "demo@example.com", Role: "Senior Engineer", Avatar: "AJ", AvatarColor: "var(--gradient-1)", IsVIP: true, Letter: "A"},
			{ID: "c2", Name: "Amanda Peters", Email: "amanda.peters@stripe.com", Role: "Product Manager at Stripe", Avatar: "AP", AvatarColor: "var(--gradient-4)", Letter: "A"},
			{ID: "c3", Name: "James Davis", Email: "james.davis@techcorp.com", Role: "CEO at TechCorp", Avatar: "JD", AvatarColor: "var(--gradient-2)", IsVIP: true, Letter: "J"},
			{ID: "c4", Name: "Michael Park", Email: "michael.park@github.com", Role: "Developer Relations at GitHub", Avatar: "MP", AvatarColor: "linear-gradient(135deg, #a8e6cf 0%, #88d8b0 100%)", Letter: "M"},
			{ID: "c5", Name: "Sarah Chen", Email: "sarah.chen@company.com", Role: "VP Engineering at Acme Inc", Avatar: "SC", AvatarColor: "var(--gradient-2)", IsVIP: true, Letter: "S"},
		},
	}
}

// mockEmailBody returns mock HTML email body content.
func mockEmailBody() string {
	return `<p>Hi team,</p>
<p>I've attached the updated roadmap for Q4. Please review the timeline changes and let me know if you have any concerns.</p>
<p><strong>Key Updates:</strong></p>
<ul>
    <li>Mobile app launch moved from Oct 30 to Nov 15</li>
    <li>API v3 delayed to Q1 2025 due to security audit</li>
    <li>New analytics dashboard added to November sprint</li>
</ul>
<p>Please review and provide feedback by end of day Friday.</p>
<p>Best,<br>Sarah</p>`
}
