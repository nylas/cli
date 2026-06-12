package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/domain"
	"github.com/rivo/tview"
)

// AttachmentInfo holds attachment metadata for download reference.
type AttachmentInfo struct {
	MessageID  string
	Attachment domain.Attachment
}

// MessagesView displays email threads (conversations).
type MessagesView struct {
	*BaseTableView
	threads         []domain.Thread
	showingDetail   bool
	currentThread   *domain.Thread
	currentMessage  *domain.Message  // For reply functionality
	attachments     []AttachmentInfo // All attachments in current thread
	folderPanel     *FolderPanel
	currentFolderID string
	currentFolder   string // Display name for current folder
	showingFolders  bool
	layout          *tview.Flex // Main layout with optional folder panel
}

// NewMessagesView creates a new messages view.
func NewMessagesView(app *App) *MessagesView {
	v := &MessagesView{
		BaseTableView:   newBaseTableView(app, "messages", "Inbox"),
		currentFolder:   "Inbox",
		currentFolderID: "", // Will use INBOX by default in Load()
	}

	// Create folder panel with callback for folder selection
	v.folderPanel = NewFolderPanel(app, func(folder *domain.Folder) {
		v.currentFolderID = folder.ID
		v.currentFolder = folder.Name
		if folder.SystemFolder != "" {
			v.currentFolder = v.folderPanel.getSystemFolderName(folder.SystemFolder)
		}
		v.title = v.currentFolder
		v.showingFolders = false
		v.updateLayout()
		app.SetFocus(v.table)
		v.Load()
	})

	// Create layout
	v.layout = tview.NewFlex()
	v.updateLayout()

	v.hints = []Hint{
		{Key: "enter", Desc: "view"},
		{Key: "n", Desc: "compose"},
		{Key: "R", Desc: "reply"},
		{Key: "s", Desc: "star"},
		{Key: "u", Desc: "unread"},
		{Key: "F", Desc: "folders"},
		{Key: "r", Desc: "refresh"},
	}

	v.table.SetColumns([]Column{
		{Title: "", Width: 3},
		{Title: "FROM", Width: 25},
		{Title: "SUBJECT", Expand: true},
		{Title: "#", Width: 3},
		{Title: "DATE", Width: 12},
	})

	// Set up double-click to open thread
	v.table.SetOnDoubleClick(func(meta *RowMeta) {
		if thread, ok := meta.Data.(*domain.Thread); ok {
			v.showDetail(thread)
		}
	})

	return v
}

// effectiveFolderID returns the folder used for thread queries; an empty
// selection means INBOX.
func (v *MessagesView) effectiveFolderID() string {
	if v.currentFolderID == "" {
		return "INBOX"
	}
	return v.currentFolderID
}

// Load fetches threads in a background goroutine and applies the results on
// the event loop via QueueUpdateDraw. Must be called from the event loop;
// it is non-blocking.
func (v *MessagesView) Load() {
	// Load folders if not already loaded
	if len(v.folderPanel.folders) == 0 {
		v.folderPanel.Load()
	}

	// Snapshot the folder the request is for; an empty selection means INBOX
	requestedFolderID := v.effectiveFolderID()
	folderFilter := []string{requestedFolderID}
	grantID := v.app.config.GrantID

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		params := &domain.ThreadQueryParams{
			Limit: 50,
			In:    folderFilter,
		}
		threads, err := v.app.config.Client.GetThreads(ctx, grantID, params)
		if err != nil {
			v.app.FlashLoadError("Failed to load threads", err)
			return
		}
		v.app.QueueUpdateDraw(func() {
			if !v.app.grantStillCurrent(grantID) {
				return // grant switched while fetch was in flight; drop stale data
			}
			if v.effectiveFolderID() != requestedFolderID {
				return // folder switched while fetch was in flight; drop stale data
			}
			v.threads = threads
			v.render()
		})
	}()
}

// updateLayout rebuilds the layout based on folder panel visibility.
func (v *MessagesView) updateLayout() {
	v.layout.Clear()
	if v.showingFolders {
		v.layout.AddItem(v.folderPanel, 30, 0, true)
		v.layout.AddItem(v.table, 0, 1, false)
	} else {
		v.layout.AddItem(v.table, 0, 1, true)
	}
}

// Primitive returns the root primitive for this view.
func (v *MessagesView) Primitive() tview.Primitive {
	return v.layout
}

func (v *MessagesView) Refresh() {
	v.Load()
}

func (v *MessagesView) render() {
	var data [][]string
	var meta []RowMeta

	// Parse search query if filter is set
	var searchQuery *SearchQuery
	if v.filter != "" {
		searchQuery = ParseSearchQuery(v.filter)
	}

	for _, thread := range v.threads {
		// Apply search filter
		if searchQuery != nil && !searchQuery.MatchesThread(&thread) {
			continue
		}

		// Get the primary participant (first one, typically the sender)
		from := ""
		if len(thread.Participants) > 0 {
			from = thread.Participants[0].Name
			if from == "" {
				from = thread.Participants[0].Email
			}
		}

		date := formatDate(thread.LatestMessageRecvDate)
		msgCount := fmt.Sprintf("%d", len(thread.MessageIDs))

		data = append(data, []string{
			"", // Status column
			from,
			thread.Subject,
			msgCount,
			date,
		})

		// Create a copy of thread for the closure
		t := thread
		meta = append(meta, RowMeta{
			ID:      thread.ID,
			Data:    &t,
			Unread:  thread.Unread,
			Starred: thread.Starred,
		})
	}

	v.table.SetData(data, meta)
}

func (v *MessagesView) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	// If folder panel is showing, delegate to it
	if v.showingFolders {
		switch event.Key() {
		case tcell.KeyEscape:
			v.showingFolders = false
			v.updateLayout()
			v.app.SetFocus(v.table)
			return nil
		case tcell.KeyTab:
			// Tab switches between folder panel and message list
			v.app.SetFocus(v.table)
			return nil
		default:
			// Let folder panel handle other keys
			return v.folderPanel.handleInput(event)
		}
	}

	switch event.Key() {
	case tcell.KeyEscape:
		// If showing detail, close it and return nil to indicate we handled it
		if v.showingDetail {
			v.closeDetail()
			return nil
		}
		// Otherwise, let app handle the Escape
		return event

	case tcell.KeyTab:
		// Tab toggles folder panel when not showing detail
		if !v.showingDetail && v.showingFolders {
			v.app.SetFocus(v.folderPanel)
			return nil
		}
		return event

	case tcell.KeyEnter:
		// View thread detail
		if meta := v.table.SelectedMeta(); meta != nil {
			if thread, ok := meta.Data.(*domain.Thread); ok {
				v.showDetail(thread)
			}
		}
		return nil

	case tcell.KeyRune:
		switch event.Rune() {
		case 'n':
			// New compose
			v.showCompose(ComposeModeNew, nil)
			return nil
		case 'R':
			// Reply to selected or current message
			if v.showingDetail && v.currentMessage != nil {
				v.showCompose(ComposeModeReply, v.currentMessage)
			}
			return nil
		case 'A':
			// Reply All to selected or current message
			if v.showingDetail && v.currentMessage != nil {
				v.showCompose(ComposeModeReplyAll, v.currentMessage)
			}
			return nil
		case 's':
			v.toggleStar()
			return nil
		case 'u':
			v.markUnread()
			return nil
		case 'F':
			// Toggle folder panel
			v.showingFolders = !v.showingFolders
			v.updateLayout()
			if v.showingFolders {
				v.app.SetFocus(v.folderPanel)
			} else {
				v.app.SetFocus(v.table)
			}
			return nil
		}
	}

	return event
}
