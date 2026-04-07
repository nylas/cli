package tui

import (
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestNewFolderPanel(t *testing.T) {
	app := createTestApp(t)

	var selectedFolder *domain.Folder
	panel := NewFolderPanel(app, func(folder *domain.Folder) {
		selectedFolder = folder
	})

	if panel == nil {
		t.Fatal("NewFolderPanel returned nil")
		return
	}

	if panel.list == nil {
		t.Error("list should not be nil")
	}

	if panel.visible {
		t.Error("panel should be hidden by default")
	}

	// Verify callback is set
	if panel.onSelect == nil {
		t.Error("onSelect callback should be set")
	}

	// Test callback invocation (simulated)
	testFolder := &domain.Folder{ID: "test-123", Name: "Test"}
	panel.onSelect(testFolder)
	if selectedFolder == nil || selectedFolder.ID != "test-123" {
		t.Error("onSelect callback was not invoked correctly")
	}
}

func TestFolderPanelVisibility(t *testing.T) {
	app := createTestApp(t)

	panel := NewFolderPanel(app, nil)

	// Initially hidden
	if panel.IsVisible() {
		t.Error("panel should be hidden initially")
	}

	// Show
	panel.Show()
	if !panel.IsVisible() {
		t.Error("panel should be visible after Show()")
	}

	// Hide
	panel.Hide()
	if panel.IsVisible() {
		t.Error("panel should be hidden after Hide()")
	}

	// Toggle from hidden to visible
	panel.Toggle()
	if !panel.IsVisible() {
		t.Error("panel should be visible after Toggle() from hidden")
	}

	// Toggle from visible to hidden
	panel.Toggle()
	if panel.IsVisible() {
		t.Error("panel should be hidden after Toggle() from visible")
	}
}

func TestFolderPanelIcons(t *testing.T) {
	app := createTestApp(t)

	panel := NewFolderPanel(app, nil)

	tests := []struct {
		systemFolder string
		wantIcon     string
	}{
		{domain.FolderInbox, "📥"},
		{domain.FolderDrafts, "📝"},
		{domain.FolderSent, "📤"},
		{domain.FolderArchive, "📦"},
		{domain.FolderSpam, "⚠️"},
		{domain.FolderTrash, "🗑️"},
		{domain.FolderAll, "📬"},
		{"", "📁"}, // Custom folder
	}

	for _, tt := range tests {
		t.Run(tt.systemFolder, func(t *testing.T) {
			icon := panel.getFolderIcon(tt.systemFolder)
			if icon != tt.wantIcon {
				t.Errorf("getFolderIcon(%q) = %q, want %q", tt.systemFolder, icon, tt.wantIcon)
			}
		})
	}
}

func TestFolderPanelSystemNames(t *testing.T) {
	app := createTestApp(t)

	panel := NewFolderPanel(app, nil)

	tests := []struct {
		systemFolder string
		wantName     string
	}{
		{domain.FolderInbox, "Inbox"},
		{domain.FolderDrafts, "Drafts"},
		{domain.FolderSent, "Sent"},
		{domain.FolderArchive, "Archive"},
		{domain.FolderSpam, "Spam"},
		{domain.FolderTrash, "Trash"},
		{domain.FolderAll, "All Mail"},
		{"custom", "custom"}, // Unknown returns as-is
	}

	for _, tt := range tests {
		t.Run(tt.systemFolder, func(t *testing.T) {
			name := panel.getSystemFolderName(tt.systemFolder)
			if name != tt.wantName {
				t.Errorf("getSystemFolderName(%q) = %q, want %q", tt.systemFolder, name, tt.wantName)
			}
		})
	}
}

func TestFolderPanelSetSelectedFolder(t *testing.T) {
	app := createTestApp(t)

	panel := NewFolderPanel(app, nil)

	// Manually set folders
	panel.folders = []domain.Folder{
		{ID: "folder-1", Name: "Folder 1"},
		{ID: "folder-2", Name: "Folder 2"},
		{ID: "folder-3", Name: "Folder 3"},
	}
	panel.render()

	// Set selected folder
	panel.SetSelectedFolder("folder-2")

	if panel.selectedID != "folder-2" {
		t.Errorf("selectedID = %q, want 'folder-2'", panel.selectedID)
	}
}

func TestFolderPanelGetSelectedFolder(t *testing.T) {
	app := createTestApp(t)

	panel := NewFolderPanel(app, nil)

	// Empty folders should return nil
	if panel.GetSelectedFolder() != nil {
		t.Error("GetSelectedFolder should return nil when no folders")
	}

	// Add folders
	panel.folders = []domain.Folder{
		{ID: "folder-1", Name: "Folder 1"},
		{ID: "folder-2", Name: "Folder 2"},
	}
	panel.render()

	// First folder should be selected by default
	folder := panel.GetSelectedFolder()
	if folder == nil {
		t.Fatal("GetSelectedFolder returned nil")
		return
	}
	if folder.ID != "folder-1" {
		t.Errorf("GetSelectedFolder().ID = %q, want 'folder-1'", folder.ID)
	}
}

func TestFolderPanelGetFolderBySystemName(t *testing.T) {
	app := createTestApp(t)

	panel := NewFolderPanel(app, nil)

	panel.folders = []domain.Folder{
		{ID: "inbox-id", Name: "INBOX", SystemFolder: domain.FolderInbox},
		{ID: "sent-id", Name: "Sent", SystemFolder: domain.FolderSent},
		{ID: "custom-id", Name: "Custom", SystemFolder: ""},
	}

	// Find inbox
	inbox := panel.GetFolderBySystemName(domain.FolderInbox)
	if inbox == nil {
		t.Fatal("GetFolderBySystemName(FolderInbox) returned nil")
		return
	}
	if inbox.ID != "inbox-id" {
		t.Errorf("inbox.ID = %q, want 'inbox-id'", inbox.ID)
	}

	// Find sent
	sent := panel.GetFolderBySystemName(domain.FolderSent)
	if sent == nil {
		t.Fatal("GetFolderBySystemName(FolderSent) returned nil")
		return
	}
	if sent.ID != "sent-id" {
		t.Errorf("sent.ID = %q, want 'sent-id'", sent.ID)
	}

	// Find non-existent
	nonExistent := panel.GetFolderBySystemName(domain.FolderTrash)
	if nonExistent != nil {
		t.Errorf("GetFolderBySystemName(FolderTrash) should return nil, got %+v", nonExistent)
	}
}

func TestFolderPanelRender(t *testing.T) {
	app := createTestApp(t)

	panel := NewFolderPanel(app, nil)

	// Set up folders with different types
	panel.folders = []domain.Folder{
		{ID: "custom-id", Name: "Custom Folder", SystemFolder: "", UnreadCount: 5},
		{ID: "inbox-id", Name: "INBOX", SystemFolder: domain.FolderInbox, UnreadCount: 10},
		{ID: "sent-id", Name: "Sent", SystemFolder: domain.FolderSent, TotalCount: 100},
	}

	// Render should sort system folders first
	panel.render()

	// Check that list has 3 items
	itemCount := panel.list.GetItemCount()
	if itemCount != 3 {
		t.Errorf("list has %d items, want 3", itemCount)
	}
}

func TestMessagesViewFolderIntegration(t *testing.T) {
	app := createTestApp(t)

	view := NewMessagesView(app)

	if view == nil {
		t.Fatal("NewMessagesView returned nil")
		return
	}

	if view.folderPanel == nil {
		t.Error("folderPanel should not be nil")
	}

	if view.layout == nil {
		t.Error("layout should not be nil")
	}

	if view.showingFolders {
		t.Error("showingFolders should be false by default")
	}

	if view.currentFolder != "Inbox" {
		t.Errorf("currentFolder = %q, want 'Inbox'", view.currentFolder)
	}
}
