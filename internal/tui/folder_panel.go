package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/domain"
	"github.com/rivo/tview"
)

// FolderPanel displays a list of email folders with unread counts.
type FolderPanel struct {
	*tview.Flex
	app        *App
	list       *tview.List
	folders    []domain.Folder
	selectedID string
	onSelect   func(folder *domain.Folder)
	visible    bool
}

// NewFolderPanel creates a new folder panel.
func NewFolderPanel(app *App, onSelect func(folder *domain.Folder)) *FolderPanel {
	p := &FolderPanel{
		Flex:     tview.NewFlex(),
		app:      app,
		onSelect: onSelect,
		visible:  false,
	}

	p.init()
	return p
}

func (p *FolderPanel) init() {
	styles := p.app.styles

	p.list = tview.NewList()
	p.list.SetBackgroundColor(styles.BgColor)
	p.list.SetMainTextColor(styles.FgColor)
	p.list.SetSecondaryTextColor(styles.InfoColor)
	p.list.SetSelectedBackgroundColor(styles.FocusColor)
	p.list.SetSelectedTextColor(styles.BgColor)
	p.list.SetBorder(true)
	p.list.SetBorderColor(styles.BorderColor)
	p.list.SetTitle(" Folders ")
	p.list.SetTitleColor(styles.TitleFg)
	p.list.ShowSecondaryText(true)

	// Handle folder selection
	p.list.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		if index < len(p.folders) {
			p.selectedID = p.folders[index].ID
			if p.onSelect != nil {
				p.onSelect(&p.folders[index])
			}
		}
	})

	p.list.SetInputCapture(p.handleInput)

	p.SetDirection(tview.FlexRow)
	p.AddItem(p.list, 0, 1, true)
}

func (p *FolderPanel) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		// Close folder panel
		if p.visible {
			p.Hide()
			return nil
		}
	case tcell.KeyRune:
		switch event.Rune() {
		case 'n':
			// Create new folder
			p.showCreateFolderForm()
			return nil
		case 'd':
			// Delete selected folder
			if idx := p.list.GetCurrentItem(); idx >= 0 && idx < len(p.folders) {
				folder := &p.folders[idx]
				// Don't allow deleting system folders
				if folder.SystemFolder != "" {
					p.app.Flash(FlashWarn, "Cannot delete system folder: %s", folder.Name)
					return nil
				}
				p.deleteFolder(folder)
			}
			return nil
		case 'r':
			// Refresh folders
			p.Load()
			return nil
		}
	}
	return event
}

// Load fetches folders from the API in a background goroutine and applies
// the results on the event loop via QueueUpdateDraw. Must be called from
// the event loop; it is non-blocking.
func (p *FolderPanel) Load() {
	grantID := p.app.config.GrantID

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		folders, err := p.app.config.Client.GetFolders(ctx, grantID)
		if err != nil {
			p.app.FlashLoadError("Failed to load folders", err)
			return
		}

		p.app.QueueUpdateDraw(func() {
			if !p.app.grantStillCurrent(grantID) {
				return // grant switched while fetch was in flight; drop stale data
			}
			p.folders = folders
			p.render()
		})
	}()
}

func (p *FolderPanel) render() {
	p.list.Clear()

	// Define system folder order for sorting
	systemOrder := map[string]int{
		domain.FolderInbox:   1,
		domain.FolderDrafts:  2,
		domain.FolderSent:    3,
		domain.FolderArchive: 4,
		domain.FolderSpam:    5,
		domain.FolderTrash:   6,
		domain.FolderAll:     7,
	}

	// Sort folders: system folders first (in order), then custom folders alphabetically
	sortedFolders := make([]domain.Folder, len(p.folders))
	copy(sortedFolders, p.folders)

	// Simple bubble sort since folder count is typically small
	for i := 0; i < len(sortedFolders)-1; i++ {
		for j := 0; j < len(sortedFolders)-i-1; j++ {
			order1 := systemOrder[sortedFolders[j].SystemFolder]
			order2 := systemOrder[sortedFolders[j+1].SystemFolder]

			// System folders come first
			if order1 == 0 && order2 != 0 {
				sortedFolders[j], sortedFolders[j+1] = sortedFolders[j+1], sortedFolders[j]
			} else if order1 != 0 && order2 != 0 && order1 > order2 {
				sortedFolders[j], sortedFolders[j+1] = sortedFolders[j+1], sortedFolders[j]
			} else if order1 == 0 && order2 == 0 && sortedFolders[j].Name > sortedFolders[j+1].Name {
				sortedFolders[j], sortedFolders[j+1] = sortedFolders[j+1], sortedFolders[j]
			}
		}
	}

	p.folders = sortedFolders

	for _, folder := range p.folders {
		icon := p.getFolderIcon(folder.SystemFolder)
		name := folder.Name
		if folder.SystemFolder != "" {
			name = p.getSystemFolderName(folder.SystemFolder)
		}

		// Format: icon + name
		mainText := fmt.Sprintf("%s %s", icon, name)

		// Secondary text shows counts
		secondaryText := ""
		if folder.UnreadCount > 0 {
			secondaryText = fmt.Sprintf("  %d unread", folder.UnreadCount)
		} else if folder.TotalCount > 0 {
			secondaryText = fmt.Sprintf("  %d total", folder.TotalCount)
		}

		shortcut := rune(0) // No shortcut
		p.list.AddItem(mainText, secondaryText, shortcut, nil)
	}

	// Select the previously selected folder
	for i, folder := range p.folders {
		if folder.ID == p.selectedID {
			p.list.SetCurrentItem(i)
			break
		}
	}
}

func (p *FolderPanel) getFolderIcon(systemFolder string) string {
	switch systemFolder {
	case domain.FolderInbox:
		return "📥"
	case domain.FolderDrafts:
		return "📝"
	case domain.FolderSent:
		return "📤"
	case domain.FolderArchive:
		return "📦"
	case domain.FolderSpam:
		return "⚠️"
	case domain.FolderTrash:
		return "🗑️"
	case domain.FolderAll:
		return "📬"
	default:
		return "📁"
	}
}

func (p *FolderPanel) getSystemFolderName(systemFolder string) string {
	switch systemFolder {
	case domain.FolderInbox:
		return "Inbox"
	case domain.FolderDrafts:
		return "Drafts"
	case domain.FolderSent:
		return "Sent"
	case domain.FolderArchive:
		return "Archive"
	case domain.FolderSpam:
		return "Spam"
	case domain.FolderTrash:
		return "Trash"
	case domain.FolderAll:
		return "All Mail"
	default:
		return systemFolder
	}
}

// Show makes the folder panel visible.
func (p *FolderPanel) Show() {
	p.visible = true
}

// Hide makes the folder panel invisible.
func (p *FolderPanel) Hide() {
	p.visible = false
}

// IsVisible returns whether the panel is visible.
func (p *FolderPanel) IsVisible() bool {
	return p.visible
}

// Toggle toggles the visibility of the folder panel.
func (p *FolderPanel) Toggle() {
	p.visible = !p.visible
}

// SetSelectedFolder sets the currently selected folder by ID.
func (p *FolderPanel) SetSelectedFolder(folderID string) {
	p.selectedID = folderID
	for i, folder := range p.folders {
		if folder.ID == folderID {
			p.list.SetCurrentItem(i)
			break
		}
	}
}

// GetSelectedFolder returns the currently selected folder.
func (p *FolderPanel) GetSelectedFolder() *domain.Folder {
	if idx := p.list.GetCurrentItem(); idx >= 0 && idx < len(p.folders) {
		return &p.folders[idx]
	}
	return nil
}

// GetFolderBySystemName returns a folder by its system name.
func (p *FolderPanel) GetFolderBySystemName(systemName string) *domain.Folder {
	for i := range p.folders {
		if p.folders[i].SystemFolder == systemName {
			return &p.folders[i]
		}
	}
	return nil
}

// Focus sets focus to the list.
func (p *FolderPanel) Focus(delegate func(p tview.Primitive)) {
	delegate(p.list)
}

func (p *FolderPanel) showCreateFolderForm() {
	styles := p.app.styles

	form := tview.NewForm()
	form.SetBackgroundColor(styles.BgColor)
	form.SetFieldBackgroundColor(styles.BgColor)
	form.SetFieldTextColor(styles.FgColor)
	form.SetLabelColor(styles.TitleFg)
	form.SetButtonBackgroundColor(styles.FocusColor)
	form.SetButtonTextColor(styles.BgColor)
	form.SetBorder(true)
	form.SetBorderColor(styles.FocusColor)
	form.SetTitle(" New Folder ")
	form.SetTitleColor(styles.TitleFg)

	var folderName string
	var parentID string

	form.AddInputField("Name", "", 30, nil, func(text string) {
		folderName = text
	})

	// Create parent folder options
	parentOptions := []string{"(None)"}
	parentIDs := []string{""}
	for _, folder := range p.folders {
		if folder.SystemFolder == "" { // Only allow non-system folders as parents
			parentOptions = append(parentOptions, folder.Name)
			parentIDs = append(parentIDs, folder.ID)
		}
	}

	form.AddDropDown("Parent Folder", parentOptions, 0, func(option string, index int) {
		if index < len(parentIDs) {
			parentID = parentIDs[index]
		}
	})

	onClose := func() {
		p.app.content.Pop()
		p.app.SetFocus(p.list)
	}

	form.AddButton("Create", func() {
		if folderName == "" {
			p.app.Flash(FlashError, "Folder name is required")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req := &domain.CreateFolderRequest{
			Name:     folderName,
			ParentID: parentID,
		}

		_, err := p.app.config.Client.CreateFolder(ctx, p.app.config.GrantID, req)
		if err != nil {
			p.app.Flash(FlashError, "Failed to create folder: %v", err)
			return
		}

		p.app.Flash(FlashInfo, "Folder created: %s", folderName)
		onClose()
		p.Load() // Refresh folder list
	})

	form.AddButton("Cancel", onClose)

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			onClose()
			return nil
		}
		return event
	})

	// Center the form
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 50, 0, true).
			AddItem(nil, 0, 1, false), 0, 2, true).
		AddItem(nil, 0, 1, false)

	p.app.content.Push("folder-form", flex)
	p.app.SetFocus(form)
}

func (p *FolderPanel) deleteFolder(folder *domain.Folder) {
	p.app.ShowConfirmDialog("Delete Folder", fmt.Sprintf("Delete folder '%s'? All emails in this folder will be moved to Trash.", folder.Name), func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := p.app.config.Client.DeleteFolder(ctx, p.app.config.GrantID, folder.ID)
		if err != nil {
			p.app.Flash(FlashError, "Failed to delete folder: %v", err)
			return
		}

		p.app.Flash(FlashInfo, "Folder deleted: %s", folder.Name)
		p.Load() // Refresh folder list
	})
}
