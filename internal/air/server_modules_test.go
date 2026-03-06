package air

import (
	"html/template"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// =============================================================================
// Converter Tests (server_converters.go)
// =============================================================================

func TestDomainMessageToCached(t *testing.T) {
	t.Parallel()

	msg := &domain.Message{
		ID:       "msg123",
		ThreadID: "thread456",
		Subject:  "Test Subject",
		Snippet:  "This is a snippet",
		From: []domain.EmailParticipant{
			{Name: "John Doe", Email: "john@example.com"},
		},
		To: []domain.EmailParticipant{
			{Name: "Jane Smith", Email: "jane@example.com"},
		},
		Cc: []domain.EmailParticipant{
			{Name: "Bob Johnson", Email: "bob@example.com"},
		},
		Bcc:     []domain.EmailParticipant{},
		Folders: []string{"folder1"},
		Date:    time.Unix(1234567890, 0),
		Unread:  true,
		Starred: false,
		Attachments: []domain.Attachment{
			{ID: "att1", Filename: "test.pdf"},
		},
		Body: "<p>Email body</p>",
	}

	cached := domainMessageToCached(msg)

	if cached.ID != "msg123" {
		t.Errorf("expected ID msg123, got %s", cached.ID)
	}
	if cached.ThreadID != "thread456" {
		t.Errorf("expected ThreadID thread456, got %s", cached.ThreadID)
	}
	if cached.FolderID != "folder1" {
		t.Errorf("expected FolderID folder1, got %s", cached.FolderID)
	}
	if cached.Subject != "Test Subject" {
		t.Errorf("expected Subject 'Test Subject', got %s", cached.Subject)
	}
	if cached.FromName != "John Doe" {
		t.Errorf("expected FromName 'John Doe', got %s", cached.FromName)
	}
	if cached.FromEmail != "john@example.com" {
		t.Errorf("expected FromEmail 'john@example.com', got %s", cached.FromEmail)
	}
	if !cached.Unread {
		t.Error("expected Unread to be true")
	}
	if cached.Starred {
		t.Error("expected Starred to be false")
	}
	if !cached.HasAttachments {
		t.Error("expected HasAttachments to be true")
	}
	if len(cached.To) != 1 {
		t.Errorf("expected 1 To recipient, got %d", len(cached.To))
	}
	if len(cached.CC) != 1 {
		t.Errorf("expected 1 CC recipient, got %d", len(cached.CC))
	}
}

func TestDomainEventToCached(t *testing.T) {
	t.Parallel()

	evt := &domain.Event{
		ID:          "evt123",
		Title:       "Team Meeting",
		Description: "Weekly sync",
		Location:    "Conference Room A",
		When: domain.EventWhen{
			Object:    "timespan",
			StartTime: 1234567890,
			EndTime:   1234571490,
		},
		Status: "confirmed",
		Busy:   true,
		Participants: []domain.Participant{
			{Person: domain.Person{Name: "Alice", Email: "alice@example.com"}},
			{Person: domain.Person{Name: "Bob", Email: "bob@example.com"}},
		},
	}

	cached := domainEventToCached(evt, "cal123")

	if cached.ID != "evt123" {
		t.Errorf("expected ID evt123, got %s", cached.ID)
	}
	if cached.CalendarID != "cal123" {
		t.Errorf("expected CalendarID cal123, got %s", cached.CalendarID)
	}
	if cached.Title != "Team Meeting" {
		t.Errorf("expected Title 'Team Meeting', got %s", cached.Title)
	}
	if cached.Location != "Conference Room A" {
		t.Errorf("expected Location 'Conference Room A', got %s", cached.Location)
	}
	if cached.AllDay {
		t.Error("expected AllDay to be false for timespan")
	}
	if !cached.Busy {
		t.Error("expected Busy to be true")
	}
	if len(cached.Participants) != 2 {
		t.Errorf("expected 2 participants, got %d", len(cached.Participants))
	}
}

func TestDomainEventToCached_AllDay(t *testing.T) {
	t.Parallel()

	evt := &domain.Event{
		ID:    "evt456",
		Title: "All Day Event",
		When: domain.EventWhen{
			Object:    "date",
			StartTime: 1234567890,
			EndTime:   1234567890,
		},
	}

	cached := domainEventToCached(evt, "cal123")

	if !cached.AllDay {
		t.Error("expected AllDay to be true for date object")
	}
}

func TestDomainContactToCached(t *testing.T) {
	t.Parallel()

	contact := &domain.Contact{
		ID:        "contact123",
		GivenName: "John",
		Surname:   "Doe",
		Emails: []domain.ContactEmail{
			{Email: "john@example.com", Type: "work"},
		},
		PhoneNumbers: []domain.ContactPhone{
			{Number: "+1234567890", Type: "mobile"},
		},
		CompanyName: "Acme Inc",
		JobTitle:    "Software Engineer",
		Notes:       "Important contact",
	}

	cached := domainContactToCached(contact)

	if cached.ID != "contact123" {
		t.Errorf("expected ID contact123, got %s", cached.ID)
	}
	if cached.Email != "john@example.com" {
		t.Errorf("expected Email 'john@example.com', got %s", cached.Email)
	}
	if cached.GivenName != "John" {
		t.Errorf("expected GivenName 'John', got %s", cached.GivenName)
	}
	if cached.Surname != "Doe" {
		t.Errorf("expected Surname 'Doe', got %s", cached.Surname)
	}
	if cached.DisplayName != "John Doe" {
		t.Errorf("expected DisplayName 'John Doe', got %s", cached.DisplayName)
	}
	if cached.Phone != "+1234567890" {
		t.Errorf("expected Phone '+1234567890', got %s", cached.Phone)
	}
	if cached.Company != "Acme Inc" {
		t.Errorf("expected Company 'Acme Inc', got %s", cached.Company)
	}
	if cached.JobTitle != "Software Engineer" {
		t.Errorf("expected JobTitle 'Software Engineer', got %s", cached.JobTitle)
	}
}

func TestParticipantsToStrings(t *testing.T) {
	t.Parallel()

	participants := []domain.EmailParticipant{
		{Name: "John Doe", Email: "john@example.com"},
		{Name: "", Email: "noreply@example.com"},
		{Name: "Jane Smith", Email: "jane@example.com"},
	}

	result := participantsToStrings(participants)

	if len(result) != 3 {
		t.Fatalf("expected 3 participants, got %d", len(result))
	}

	if result[0] != "John Doe <john@example.com>" {
		t.Errorf("expected 'John Doe <john@example.com>', got %s", result[0])
	}
	if result[1] != "noreply@example.com" {
		t.Errorf("expected 'noreply@example.com', got %s", result[1])
	}
	if result[2] != "Jane Smith <jane@example.com>" {
		t.Errorf("expected 'Jane Smith <jane@example.com>', got %s", result[2])
	}
}

func TestEventParticipantsToStrings(t *testing.T) {
	t.Parallel()

	participants := []domain.Participant{
		{Person: domain.Person{Name: "Alice", Email: "alice@example.com"}},
		{Person: domain.Person{Name: "", Email: "calendar@example.com"}},
	}

	result := eventParticipantsToStrings(participants)

	if len(result) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(result))
	}

	if result[0] != "Alice <alice@example.com>" {
		t.Errorf("expected 'Alice <alice@example.com>', got %s", result[0])
	}
	if result[1] != "calendar@example.com" {
		t.Errorf("expected 'calendar@example.com', got %s", result[1])
	}
}

// =============================================================================
// Lifecycle Tests (server_lifecycle.go)
// =============================================================================

func TestNewServer(t *testing.T) {
	t.Parallel()

	server := NewServer(":8080")

	if server == nil {
		t.Fatal("expected non-nil server")
	}

	if server.demoMode {
		t.Error("expected demoMode to be false")
	}

	if server.addr != ":8080" {
		t.Errorf("expected addr :8080, got %s", server.addr)
	}

	// Server should have initialized components
	if server.configSvc == nil {
		t.Error("expected configSvc to be initialized")
	}

	if server.configStore == nil {
		t.Error("expected configStore to be initialized")
	}

	if server.secretStore == nil {
		t.Error("expected secretStore to be initialized")
	}

	if server.grantStore == nil {
		t.Error("expected grantStore to be initialized")
	}

	// Check initial online status
	if !server.IsOnline() {
		t.Error("expected initial online status to be true")
	}
}

func TestServer_IsOnline_SetOnline(t *testing.T) {
	t.Parallel()

	server := NewServer(":8080")

	// Initial state
	if !server.IsOnline() {
		t.Error("expected initial online status to be true")
	}

	// Set offline
	server.SetOnline(false)
	if server.IsOnline() {
		t.Error("expected online status to be false after SetOnline(false)")
	}

	// Set back online
	server.SetOnline(true)
	if !server.IsOnline() {
		t.Error("expected online status to be true after SetOnline(true)")
	}
}

// =============================================================================
// Store Accessor Tests (server_stores.go)
// =============================================================================

func TestServer_GetStores_NoCacheManager(t *testing.T) {
	t.Parallel()

	server := &Server{
		cacheManager: nil,
	}

	// All store getters should return error when cache manager is nil
	_, err := server.getEmailStore("test@example.com")
	if err == nil {
		t.Error("expected error from getEmailStore when cacheManager is nil")
	}

	_, err = server.getEventStore("test@example.com")
	if err == nil {
		t.Error("expected error from getEventStore when cacheManager is nil")
	}

	_, err = server.getContactStore("test@example.com")
	if err == nil {
		t.Error("expected error from getContactStore when cacheManager is nil")
	}

	_, err = server.getFolderStore("test@example.com")
	if err == nil {
		t.Error("expected error from getFolderStore when cacheManager is nil")
	}

	_, err = server.getSyncStore("test@example.com")
	if err == nil {
		t.Error("expected error from getSyncStore when cacheManager is nil")
	}
}

// =============================================================================
// Template Utility Tests (server_template.go)
// =============================================================================

func TestLoadTemplates(t *testing.T) {
	t.Parallel()

	tmpl, err := loadTemplates()
	if err != nil {
		t.Fatalf("expected no error loading templates, got %v", err)
	}

	if tmpl == nil {
		t.Fatal("expected non-nil template")
	}

	// Verify base template exists
	baseTemplate := tmpl.Lookup("base")
	if baseTemplate == nil {
		t.Error("expected 'base' template to exist")
	}
}

func TestTemplateFuncs_SafeHTML(t *testing.T) {
	t.Parallel()

	safeHTMLFunc, exists := templateFuncs["safeHTML"]
	if !exists {
		t.Fatal("expected safeHTML function to exist in templateFuncs")
	}

	// The safeHTML function returns template.HTML, not any
	result := safeHTMLFunc.(func(string) template.HTML)("<p>Test</p>")
	if string(result) != "<p>Test</p>" {
		t.Errorf("expected '<p>Test</p>', got %s", result)
	}
}

// =============================================================================
// BuildPageData Tests (server_template.go)
// =============================================================================

func TestBuildPageData_NonDemoMode(t *testing.T) {
	t.Parallel()

	server := NewServer(":8080")
	data := server.buildPageData()

	// In non-demo mode with no grants, should have nil data arrays
	// (JavaScript will load real data)
	if data.Emails != nil {
		t.Error("expected nil Emails in non-demo mode")
	}

	if data.Events != nil {
		t.Error("expected nil Events in non-demo mode")
	}

	if data.Calendars != nil {
		t.Error("expected nil Calendars in non-demo mode")
	}

	if data.Contacts != nil {
		t.Error("expected nil Contacts in non-demo mode")
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestDomainMessageToCached_EmptyParticipants(t *testing.T) {
	t.Parallel()

	msg := &domain.Message{
		ID:      "msg123",
		Subject: "Test",
		From:    []domain.EmailParticipant{},
		To:      []domain.EmailParticipant{},
		Folders: []string{},
	}

	cached := domainMessageToCached(msg)

	if cached.FromName != "" {
		t.Errorf("expected empty FromName, got %s", cached.FromName)
	}
	if cached.FromEmail != "" {
		t.Errorf("expected empty FromEmail, got %s", cached.FromEmail)
	}
	if cached.FolderID != "" {
		t.Errorf("expected empty FolderID, got %s", cached.FolderID)
	}
	if len(cached.To) != 0 {
		t.Errorf("expected 0 To recipients, got %d", len(cached.To))
	}
}

func TestDomainContactToCached_EmptyFields(t *testing.T) {
	t.Parallel()

	contact := &domain.Contact{
		ID:           "contact123",
		GivenName:    "John",
		Surname:      "Doe",
		Emails:       []domain.ContactEmail{},
		PhoneNumbers: []domain.ContactPhone{},
	}

	cached := domainContactToCached(contact)

	if cached.Email != "" {
		t.Errorf("expected empty Email, got %s", cached.Email)
	}
	if cached.Phone != "" {
		t.Errorf("expected empty Phone, got %s", cached.Phone)
	}
	if cached.Company != "" {
		t.Errorf("expected empty Company, got %s", cached.Company)
	}
	if cached.JobTitle != "" {
		t.Errorf("expected empty JobTitle, got %s", cached.JobTitle)
	}
}

// =============================================================================
// Timestamp Tests
// =============================================================================

func TestDomainMessageToCached_Timestamp(t *testing.T) {
	t.Parallel()

	msg := &domain.Message{
		ID:      "msg123",
		Subject: "Test",
		From:    []domain.EmailParticipant{{Email: "test@example.com"}},
	}

	before := time.Now()
	cached := domainMessageToCached(msg)
	after := time.Now()

	if cached.CachedAt.Before(before) || cached.CachedAt.After(after) {
		t.Error("CachedAt timestamp should be set to current time")
	}
}

func TestDomainEventToCached_Timestamp(t *testing.T) {
	t.Parallel()

	evt := &domain.Event{
		ID:    "evt123",
		Title: "Test Event",
		When: domain.EventWhen{
			Object:    "timespan",
			StartTime: 1234567890,
			EndTime:   1234571490,
		},
	}

	before := time.Now()
	cached := domainEventToCached(evt, "cal123")
	after := time.Now()

	if cached.CachedAt.Before(before) || cached.CachedAt.After(after) {
		t.Error("CachedAt timestamp should be set to current time")
	}

	// Verify time conversion
	expectedStart := time.Unix(1234567890, 0)
	if !cached.StartTime.Equal(expectedStart) {
		t.Errorf("expected StartTime %v, got %v", expectedStart, cached.StartTime)
	}

	expectedEnd := time.Unix(1234571490, 0)
	if !cached.EndTime.Equal(expectedEnd) {
		t.Errorf("expected EndTime %v, got %v", expectedEnd, cached.EndTime)
	}
}
