package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// DashboardView shows an overview.
type DashboardView struct {
	app   *App
	view  *tview.TextView
	name  string
	title string
}

// NewDashboardView creates a new dashboard view.
func NewDashboardView(app *App) *DashboardView {
	v := &DashboardView{
		app:   app,
		view:  tview.NewTextView(),
		name:  "dashboard",
		title: "Dashboard",
	}

	v.view.SetDynamicColors(true)
	v.view.SetBackgroundColor(app.styles.BgColor)
	v.view.SetBorderPadding(1, 1, 2, 2)

	return v
}

func (v *DashboardView) Name() string               { return v.name }
func (v *DashboardView) Title() string              { return v.title }
func (v *DashboardView) Primitive() tview.Primitive { return v.view }
func (v *DashboardView) Filter(string)              {}
func (v *DashboardView) Refresh()                   { v.Load() }

func (v *DashboardView) Hints() []Hint {
	return []Hint{
		{Key: ":", Desc: "command"},
		{Key: "?", Desc: "help"},
		{Key: "^C", Desc: "quit"},
	}
}

func (v *DashboardView) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	return event
}

func (v *DashboardView) Load() {
	v.view.Clear()

	// Use cached Hex() method
	st := v.app.styles
	title := st.Hex(st.TitleFg)
	key := st.Hex(st.MenuKeyFg)
	desc := st.Hex(st.FgColor)
	muted := st.Hex(st.BorderColor)

	resources := []struct {
		cmd  string
		name string
		desc string
	}{
		{":m", "Messages", "Email messages"},
		{":e", "Events", "Calendar events"},
		{":c", "Contacts", "Contacts"},
		{":w", "Webhooks", "Webhooks"},
		{":ws", "Server", "Webhook server (local)"},
		{":g", "Grants", "Connected accounts"},
	}

	_, _ = fmt.Fprintf(v.view, "[%s::b]Quick Navigation[-::-]\n\n", title)

	for _, r := range resources {
		_, _ = fmt.Fprintf(v.view, "  [%s]%-6s[-]  [%s]%-12s[-]  [%s::d]%s[-::-]\n",
			key, r.cmd,
			desc, r.name,
			muted, r.desc,
		)
	}

	_, _ = fmt.Fprintf(v.view, "\n[%s::d]Press : to enter command mode[-::-]", muted)
}
