package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/rivo/tview"
)

// WebhooksView displays webhooks.
type WebhooksView struct {
	*BaseTableView
	webhooks []domain.Webhook
}

// NewWebhooksView creates a new webhooks view.
func NewWebhooksView(app *App) *WebhooksView {
	v := &WebhooksView{
		BaseTableView: newBaseTableView(app, "webhooks", "Webhooks"),
	}

	v.hints = []Hint{
		{Key: "enter", Desc: "view"},
		{Key: "n", Desc: "new"},
		{Key: "e", Desc: "edit"},
		{Key: "d", Desc: "delete"},
		{Key: "r", Desc: "refresh"},
	}

	v.table.SetColumns([]Column{
		{Title: "", Width: 3},
		{Title: "TRIGGERS", Width: 30},
		{Title: "URL", Expand: true},
		{Title: "STATUS", Width: 12},
	})

	return v
}

// Load fetches webhooks in a background goroutine and applies the results on
// the event loop via QueueUpdateDraw. Must be called from the event loop;
// it is non-blocking.
func (v *WebhooksView) Load() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		webhooks, err := v.app.config.Client.ListWebhooks(ctx)
		if err != nil {
			v.app.FlashLoadError("Failed to load webhooks", err)
			return
		}
		v.app.QueueUpdateDraw(func() {
			v.webhooks = webhooks
			v.render()
		})
	}()
}

func (v *WebhooksView) Refresh() { v.Load() }

func (v *WebhooksView) render() {
	var data [][]string
	var meta []RowMeta

	for _, wh := range v.webhooks {
		triggers := strings.Join(wh.TriggerTypes, ", ")
		data = append(data, []string{
			"",
			triggers,
			wh.WebhookURL,
			wh.Status,
		})
		meta = append(meta, RowMeta{
			ID:    wh.ID,
			Data:  &wh,
			Error: wh.Status != "active",
		})
	}

	v.table.SetData(data, meta)
}

// HandleKey handles keyboard input for webhooks view.
func (v *WebhooksView) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEnter:
		// View webhook detail
		if idx, _ := v.table.GetSelection(); idx > 0 && idx-1 < len(v.webhooks) {
			v.showWebhookDetail(&v.webhooks[idx-1])
		}
		return nil

	case tcell.KeyRune:
		switch event.Rune() {
		case 'n': // New webhook
			v.app.ShowWebhookForm(nil, func(webhook *domain.Webhook) {
				v.Refresh()
			})
			return nil

		case 'e': // Edit selected webhook
			if idx, _ := v.table.GetSelection(); idx > 0 && idx-1 < len(v.webhooks) {
				webhook := v.webhooks[idx-1]
				v.app.ShowWebhookForm(&webhook, func(updatedWebhook *domain.Webhook) {
					v.Refresh()
				})
			}
			return nil

		case 'd': // Delete selected webhook
			if idx, _ := v.table.GetSelection(); idx > 0 && idx-1 < len(v.webhooks) {
				webhook := v.webhooks[idx-1]
				v.app.DeleteWebhook(&webhook, func() {
					v.Refresh()
				})
			}
			return nil
		}
	}

	return event
}

func (v *WebhooksView) showWebhookDetail(webhook *domain.Webhook) {
	detail := tview.NewTextView()
	detail.SetDynamicColors(true)
	detail.SetBackgroundColor(v.app.styles.BgColor)
	detail.SetBorder(true)
	detail.SetBorderColor(v.app.styles.FocusColor)

	titleStr := webhook.Description
	if titleStr == "" {
		titleStr = webhook.ID
	}
	detail.SetTitle(fmt.Sprintf(" Webhook: %s ", titleStr))
	detail.SetTitleColor(v.app.styles.TitleFg)
	detail.SetBorderPadding(1, 1, 2, 2)
	detail.SetScrollable(true)

	// Use cached Hex() method
	st := v.app.styles
	info := st.Hex(st.InfoColor)
	value := st.Hex(st.InfoSectionFg)
	muted := st.Hex(st.BorderColor)
	success := st.Hex(st.SuccessColor)
	errColor := st.Hex(st.ErrorColor)

	// URL
	_, _ = fmt.Fprintf(detail, "[%s::b]Webhook URL[-::-]\n", info)
	_, _ = fmt.Fprintf(detail, "[%s]%s[-]\n\n", value, webhook.WebhookURL)

	// Status
	_, _ = fmt.Fprintf(detail, "[%s::b]Status[-::-]\n", info)
	statusColor := success
	if webhook.Status != "active" {
		statusColor = errColor
	}
	_, _ = fmt.Fprintf(detail, "[%s]%s[-]\n\n", statusColor, webhook.Status)

	// Description
	if webhook.Description != "" {
		_, _ = fmt.Fprintf(detail, "[%s::b]Description[-::-]\n", info)
		_, _ = fmt.Fprintf(detail, "[%s]%s[-]\n\n", value, webhook.Description)
	}

	// Trigger types
	_, _ = fmt.Fprintf(detail, "[%s::b]Trigger Types[-::-]\n", info)
	for _, trigger := range webhook.TriggerTypes {
		_, _ = fmt.Fprintf(detail, "[%s]  • %s[-]\n", value, trigger)
	}
	_, _ = fmt.Fprintln(detail)

	// Notification emails
	if len(webhook.NotificationEmailAddresses) > 0 {
		_, _ = fmt.Fprintf(detail, "[%s::b]Notification Emails[-::-]\n", info)
		for _, email := range webhook.NotificationEmailAddresses {
			_, _ = fmt.Fprintf(detail, "[%s]  • %s[-]\n", value, email)
		}
		_, _ = fmt.Fprintln(detail)
	}

	// Dates
	if !webhook.CreatedAt.IsZero() {
		_, _ = fmt.Fprintf(detail, "[%s]Created:[-] [%s]%s[-]\n", muted, value, webhook.CreatedAt.Format(common.DisplayDateTime))
	}
	if !webhook.UpdatedAt.IsZero() {
		_, _ = fmt.Fprintf(detail, "[%s]Updated:[-] [%s]%s[-]\n", muted, value, webhook.UpdatedAt.Format(common.DisplayDateTime))
	}

	_, _ = fmt.Fprintf(detail, "\n\n[%s::d]Press Esc to go back, 'e' to edit, 'd' to delete[-::-]", muted)

	// Handle keyboard
	webhookCopy := webhook
	detail.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			v.app.PopDetail()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'e':
				v.app.PopDetail()
				v.app.ShowWebhookForm(webhookCopy, func(updatedWebhook *domain.Webhook) {
					v.Refresh()
				})
				return nil
			case 'd':
				v.app.PopDetail()
				v.app.DeleteWebhook(webhookCopy, func() {
					v.Refresh()
				})
				return nil
			}
		}
		return event
	})

	v.app.PushDetail("webhook-detail", detail)
	v.app.SetFocus(detail)
}
