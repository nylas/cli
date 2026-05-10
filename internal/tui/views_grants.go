package tui

import (
	"context"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/domain"
)

// GrantsView displays grants.
type GrantsView struct {
	*BaseTableView
	grants []domain.Grant
}

// NewGrantsView creates a new grants view.
func NewGrantsView(app *App) *GrantsView {
	v := &GrantsView{
		BaseTableView: newBaseTableView(app, "grants", "Grants"),
	}

	// Different hints based on whether switching is available
	if app.CanSwitchGrant() {
		v.hints = []Hint{
			{Key: "enter", Desc: "switch"},
			{Key: "r", Desc: "refresh"},
		}
	} else {
		v.hints = []Hint{
			{Key: "r", Desc: "refresh"},
		}
	}

	v.table.SetColumns([]Column{
		{Title: "", Width: 3},
		{Title: "EMAIL", Width: 35},
		{Title: "PROVIDER", Width: 15},
		{Title: "GRANT ID", Expand: true},
	})

	return v
}

func (v *GrantsView) Load() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	grants, err := v.app.config.Client.ListGrants(ctx)
	if err != nil {
		v.app.FlashLoadError("Failed to load grants", err)
		return
	}
	v.grants = grants
	v.render()
}

func (v *GrantsView) Refresh() { v.Load() }

func (v *GrantsView) render() {
	var data [][]string
	var meta []RowMeta

	currentGrantID := v.app.config.GrantID

	for _, g := range v.grants {
		// Mark current/default grant with ★
		marker := ""
		if g.ID == currentGrantID {
			marker = "★"
		}

		data = append(data, []string{
			marker,
			g.Email,
			string(g.Provider),
			g.ID,
		})
		meta = append(meta, RowMeta{ID: g.ID, Data: &g})
	}

	v.table.SetData(data, meta)
}

// HandleKey handles key events for the grants view.
func (v *GrantsView) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEnter:
		// Switch to selected grant
		if !v.app.CanSwitchGrant() {
			v.app.Flash(FlashWarn, "Grant switching not available in demo mode")
			return nil
		}

		meta := v.table.SelectedMeta()
		if meta == nil || meta.Data == nil {
			return nil
		}

		grant, ok := meta.Data.(*domain.Grant)
		if !ok {
			return nil
		}

		// Check if already the current grant
		if grant.ID == v.app.config.GrantID {
			v.app.Flash(FlashInfo, "Already using this grant")
			return nil
		}

		// Switch to the selected grant
		if err := v.app.SwitchGrant(grant.ID, grant.Email, string(grant.Provider)); err != nil {
			v.app.Flash(FlashError, "Failed to switch: %v", err)
			return nil
		}

		v.app.Flash(FlashInfo, "Switched to %s", grant.Email)
		v.render() // Re-render to update the marker
		return nil

	case tcell.KeyEscape:
		return event
	}

	return event
}
