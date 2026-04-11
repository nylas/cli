package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/domain"
)

// DraftsView displays email drafts.
type DraftsView struct {
	*BaseTableView
	drafts       []domain.Draft
	showingDraft bool
	currentDraft *domain.Draft
}

// NewDraftsView creates a new drafts view.
func NewDraftsView(app *App) *DraftsView {
	v := &DraftsView{
		BaseTableView: newBaseTableView(app, "drafts", "Drafts"),
	}

	v.hints = []Hint{
		{Key: "enter", Desc: "edit"},
		{Key: "n", Desc: "new"},
		{Key: "s", Desc: "send"},
		{Key: "d", Desc: "delete"},
		{Key: "r", Desc: "refresh"},
	}

	v.table.SetColumns([]Column{
		{Title: "TO", Width: 30},
		{Title: "SUBJECT", Expand: true},
		{Title: "UPDATED", Width: 18},
	})

	// Set up double-click to edit draft
	v.table.SetOnDoubleClick(func(meta *RowMeta) {
		if draft, ok := meta.Data.(*domain.Draft); ok {
			v.editDraft(draft)
		}
	})

	return v
}

func (v *DraftsView) Load() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	drafts, err := v.app.config.Client.GetDrafts(ctx, v.app.config.GrantID, 50)
	if err != nil {
		v.app.Flash(FlashError, "Failed to load drafts: %v", err)
		return
	}
	v.drafts = drafts
	v.render()
}

func (v *DraftsView) Refresh() {
	v.Load()
}

func (v *DraftsView) render() {
	var data [][]string
	var meta []RowMeta

	// Parse search query if filter is set
	var searchQuery *SearchQuery
	if v.filter != "" {
		searchQuery = ParseSearchQuery(v.filter)
	}

	for _, draft := range v.drafts {
		// Apply search filter
		if searchQuery != nil && !searchQuery.MatchesDraft(&draft) {
			continue
		}

		// Get primary recipient
		toStr := ""
		if len(draft.To) > 0 {
			toStr = draft.To[0].Name
			if toStr == "" {
				toStr = draft.To[0].Email
			}
		}
		if toStr == "" {
			toStr = "(no recipient)"
		}

		subject := draft.Subject
		if subject == "" {
			subject = "(no subject)"
		}

		updated := draft.UpdatedAt.Local().Format("Jan 2, 3:04 PM")

		data = append(data, []string{
			toStr,
			subject,
			updated,
		})

		// Create a copy for closure
		d := draft
		meta = append(meta, RowMeta{
			ID:   draft.ID,
			Data: &d,
		})
	}

	v.table.SetData(data, meta)
}

func (v *DraftsView) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		if v.showingDraft {
			v.closeDraftView()
			return nil
		}
		return event

	case tcell.KeyEnter:
		// Edit draft
		if meta := v.table.SelectedMeta(); meta != nil {
			if draft, ok := meta.Data.(*domain.Draft); ok {
				v.editDraft(draft)
			}
		}
		return nil

	case tcell.KeyRune:
		switch event.Rune() {
		case 'n':
			// New draft
			v.createNewDraft()
			return nil
		case 's':
			// Send draft
			if meta := v.table.SelectedMeta(); meta != nil {
				if draft, ok := meta.Data.(*domain.Draft); ok {
					v.sendDraft(draft)
				}
			}
			return nil
		case 'd':
			// Delete draft
			if meta := v.table.SelectedMeta(); meta != nil {
				if draft, ok := meta.Data.(*domain.Draft); ok {
					v.deleteDraft(draft)
				}
			}
			return nil
		}
	}

	return event
}

func (v *DraftsView) editDraft(draft *domain.Draft) {
	v.currentDraft = draft
	v.showingDraft = true

	compose := NewComposeViewForDraft(v.app, draft)
	compose.SetOnSent(func() {
		v.closeDraftView()
		v.Load() // Refresh list - draft should be gone after sending
	})
	compose.SetOnCancel(func() {
		v.closeDraftView()
	})
	compose.SetOnSave(func() {
		v.closeDraftView()
		v.Load() // Refresh to show updated draft
	})

	v.app.content.Push("compose-draft", compose)
	v.app.SetFocus(compose)
}

func (v *DraftsView) createNewDraft() {
	compose := NewComposeView(v.app, ComposeModeNew, nil)
	compose.SetOnSent(func() {
		v.closeDraftView()
		v.Load()
	})
	compose.SetOnCancel(func() {
		v.closeDraftView()
	})

	v.app.content.Push("compose-new-draft", compose)
	v.app.SetFocus(compose)
}

func (v *DraftsView) sendDraft(draft *domain.Draft) {
	v.app.ShowConfirmDialog("Send Draft", fmt.Sprintf("Send this draft to %s?", getDraftRecipients(draft)), func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := v.app.config.Client.SendDraft(ctx, v.app.config.GrantID, draft.ID, nil)
		if err != nil {
			v.app.Flash(FlashError, "Failed to send draft: %v", err)
			return
		}

		v.app.Flash(FlashInfo, "Draft sent successfully!")
		v.Load()
	})
}

func (v *DraftsView) deleteDraft(draft *domain.Draft) {
	subject := draft.Subject
	if subject == "" {
		subject = "(no subject)"
	}
	v.app.ShowConfirmDialog("Delete Draft", fmt.Sprintf("Delete draft '%s'?", subject), func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := v.app.config.Client.DeleteDraft(ctx, v.app.config.GrantID, draft.ID)
		if err != nil {
			v.app.Flash(FlashError, "Failed to delete draft: %v", err)
			return
		}

		v.app.Flash(FlashInfo, "Draft deleted")
		v.Load()
	})
}

func (v *DraftsView) closeDraftView() {
	v.showingDraft = false
	v.currentDraft = nil
	v.app.content.Pop()
	v.app.SetFocus(v.table)
}

func getDraftRecipients(draft *domain.Draft) string {
	if len(draft.To) == 0 {
		return "(no recipient)"
	}
	var recipients []string
	for _, to := range draft.To {
		if to.Name != "" {
			recipients = append(recipients, to.Name)
		} else {
			recipients = append(recipients, to.Email)
		}
	}
	return strings.Join(recipients, ", ")
}
