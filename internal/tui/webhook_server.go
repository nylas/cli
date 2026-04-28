package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/adapters/tunnel"
	"github.com/nylas/cli/internal/adapters/webhookserver"
	"github.com/nylas/cli/internal/ports"
	"github.com/rivo/tview"
)

// WebhookServerView displays webhook server status and events.
type WebhookServerView struct {
	app           *App
	layout        *tview.Flex
	statusPanel   *tview.TextView
	eventsPanel   *tview.TextView
	name          string
	title         string
	server        *webhookserver.Server
	serverRunning bool
	events        []*ports.WebhookEvent
	maxEvents     int
	tunnelEnabled bool
	publicURL     string
	port          int
	cancelFunc    context.CancelFunc
}

// NewWebhookServerView creates a new webhook server view.
func NewWebhookServerView(app *App) *WebhookServerView {
	v := &WebhookServerView{
		app:       app,
		name:      "webhook-server",
		title:     "Webhook Server",
		maxEvents: 50,
		port:      3000,
	}

	// Create status panel
	v.statusPanel = tview.NewTextView()
	v.statusPanel.SetDynamicColors(true)
	v.statusPanel.SetBackgroundColor(app.styles.BgColor)
	v.statusPanel.SetBorder(true)
	v.statusPanel.SetBorderColor(app.styles.BorderColor)
	v.statusPanel.SetTitle(" Server Status ")
	v.statusPanel.SetTitleColor(app.styles.TitleFg)
	v.statusPanel.SetBorderPadding(0, 0, 1, 1)

	// Create events panel
	v.eventsPanel = tview.NewTextView()
	v.eventsPanel.SetDynamicColors(true)
	v.eventsPanel.SetBackgroundColor(app.styles.BgColor)
	v.eventsPanel.SetBorder(true)
	v.eventsPanel.SetBorderColor(app.styles.BorderColor)
	v.eventsPanel.SetTitle(" Webhook Events ")
	v.eventsPanel.SetTitleColor(app.styles.TitleFg)
	v.eventsPanel.SetBorderPadding(0, 0, 1, 1)
	v.eventsPanel.SetScrollable(true)

	// Create layout: Status (top) | Events (bottom)
	v.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.statusPanel, 10, 0, false).
		AddItem(v.eventsPanel, 0, 1, true)

	return v
}

func (v *WebhookServerView) Name() string               { return v.name }
func (v *WebhookServerView) Title() string              { return v.title }
func (v *WebhookServerView) Primitive() tview.Primitive { return v.layout }
func (v *WebhookServerView) Filter(string)              {}

func (v *WebhookServerView) Hints() []Hint {
	if v.serverRunning {
		return []Hint{
			{Key: "s", Desc: "stop server"},
			{Key: "c", Desc: "clear events"},
			{Key: "t", Desc: "toggle tunnel"},
			{Key: "r", Desc: "refresh"},
		}
	}
	return []Hint{
		{Key: "s", Desc: "start server"},
		{Key: "t", Desc: "toggle tunnel"},
		{Key: "p", Desc: "set port"},
		{Key: "r", Desc: "refresh"},
	}
}

func (v *WebhookServerView) Load() {
	v.renderStatus()
	v.renderEvents()
}

func (v *WebhookServerView) Refresh() {
	v.renderStatus()
	v.renderEvents()
}

func (v *WebhookServerView) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		// Stop server if running before leaving
		if v.serverRunning {
			v.stopServer()
		}
		return event

	case tcell.KeyRune:
		switch event.Rune() {
		case 's':
			// Toggle server
			if v.serverRunning {
				v.stopServer()
			} else {
				v.startServer()
			}
			return nil
		case 'c':
			// Clear events
			v.events = nil
			v.renderEvents()
			return nil
		case 't':
			// Toggle tunnel
			if !v.serverRunning {
				v.tunnelEnabled = !v.tunnelEnabled
				v.renderStatus()
				if v.tunnelEnabled && !tunnel.IsCloudflaredInstalled() {
					v.app.Flash(FlashWarn, "cloudflared not installed. Install with: brew install cloudflared")
					v.tunnelEnabled = false
					v.renderStatus()
				}
			} else {
				v.app.Flash(FlashWarn, "Stop server first to toggle tunnel")
			}
			return nil
		case 'p':
			// Change port (only when not running)
			if !v.serverRunning {
				v.showPortPrompt()
			} else {
				v.app.Flash(FlashWarn, "Stop server first to change port")
			}
			return nil
		}
	}

	return event
}

func (v *WebhookServerView) startServer() {
	if v.serverRunning {
		return
	}

	// Create server config
	config := ports.WebhookServerConfig{
		Port:           v.port,
		Path:           "/webhook",
		TunnelProvider: "",
	}

	if v.tunnelEnabled {
		config.TunnelProvider = "cloudflared"
	}

	// Create webhook server
	v.server = webhookserver.NewServer(config)

	// Set up tunnel if enabled
	if v.tunnelEnabled {
		localURL := webhookserver.LocalBaseURL(v.port)
		t := tunnel.NewCloudflaredTunnel(localURL)
		v.server.SetTunnel(t)
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	v.cancelFunc = cancel

	// Set up event handler
	v.server.OnEvent(func(event *ports.WebhookEvent) {
		v.events = append([]*ports.WebhookEvent{event}, v.events...)
		if len(v.events) > v.maxEvents {
			v.events = v.events[:v.maxEvents]
		}
		v.app.QueueUpdateDraw(func() {
			v.renderEvents()
			v.renderStatus()
		})
	})

	// Start server in goroutine
	go func() {
		if err := v.server.Start(ctx); err != nil {
			v.app.QueueUpdateDraw(func() {
				v.app.Flash(FlashError, "Failed to start server: %v", err)
				v.serverRunning = false
				v.renderStatus()
			})
			return
		}

		v.app.QueueUpdateDraw(func() {
			v.serverRunning = true
			stats := v.server.GetStats()
			v.publicURL = stats.PublicURL
			v.renderStatus()
			v.app.Flash(FlashInfo, "Webhook server started on port %d", v.port)
		})
	}()
}

func (v *WebhookServerView) stopServer() {
	if !v.serverRunning || v.server == nil {
		return
	}

	if v.cancelFunc != nil {
		v.cancelFunc()
	}

	if err := v.server.Stop(); err != nil {
		v.app.Flash(FlashError, "Error stopping server: %v", err)
	}

	v.serverRunning = false
	v.publicURL = ""
	v.server = nil
	v.renderStatus()
	v.app.Flash(FlashInfo, "Webhook server stopped")
}

func (v *WebhookServerView) renderStatus() {
	v.statusPanel.Clear()

	// Use cached hex colors for better performance
	s := v.app.styles
	title := s.Hex(s.TitleFg)
	info := s.Hex(s.InfoColor)
	success := s.Hex(s.SuccessColor)
	warn := s.Hex(s.WarnColor)
	muted := s.Hex(s.BorderColor)
	value := s.Hex(s.FgColor)

	// Server status
	statusColor := warn
	statusText := "Stopped"
	if v.serverRunning {
		statusColor = success
		statusText = "Running"
	}

	_, _ = fmt.Fprintf(v.statusPanel, "\n  [%s::b]Server Status:[-::-] [%s]%s[-]\n\n", title, statusColor, statusText)

	// Port
	_, _ = fmt.Fprintf(v.statusPanel, "  [%s]Port:[-]    [%s]%d[-]\n", muted, value, v.port)

	// URLs
	localURL := webhookserver.LocalBaseURL(v.port) + "/webhook"
	_, _ = fmt.Fprintf(v.statusPanel, "  [%s]Local:[-]   [%s]%s[-]\n", muted, value, localURL)

	// Tunnel status
	tunnelStatus := "Disabled"
	tunnelColor := muted
	if v.tunnelEnabled {
		tunnelStatus = "Enabled"
		tunnelColor = info
		if v.serverRunning && v.publicURL != "" {
			tunnelStatus = "Connected"
			tunnelColor = success
			_, _ = fmt.Fprintf(v.statusPanel, "  [%s]Public:[-]  [%s]%s[-]\n", muted, success, v.publicURL)
		}
	}
	_, _ = fmt.Fprintf(v.statusPanel, "  [%s]Tunnel:[-]  [%s]%s[-]\n", muted, tunnelColor, tunnelStatus)

	// Event count
	if v.serverRunning && v.server != nil {
		stats := v.server.GetStats()
		_, _ = fmt.Fprintf(v.statusPanel, "  [%s]Events:[-]  [%s]%d[-]\n", muted, value, stats.EventsReceived)
	}
}

func (v *WebhookServerView) renderEvents() {
	v.eventsPanel.Clear()

	// Use cached hex colors for better performance
	s := v.app.styles
	muted := s.Hex(s.BorderColor)

	if len(v.events) == 0 {
		_, _ = fmt.Fprintf(v.eventsPanel, "\n  [%s]No webhook events received yet.[-]\n", muted)
		if v.serverRunning {
			_, _ = fmt.Fprintf(v.eventsPanel, "\n  [%s]Waiting for incoming webhooks...[-]\n", muted)
		} else {
			_, _ = fmt.Fprintf(v.eventsPanel, "\n  [%s]Press 's' to start the server.[-]\n", muted)
		}
		return
	}

	title := s.Hex(s.TitleFg)
	info := s.Hex(s.InfoColor)
	success := s.Hex(s.SuccessColor)
	warn := s.Hex(s.WarnColor)
	errColor := s.Hex(s.ErrorColor)
	value := s.Hex(s.FgColor)

	for i, event := range v.events {
		timestamp := event.ReceivedAt.Format("15:04:05")

		// Determine event type color
		typeColor := warn
		switch {
		case strings.Contains(event.Type, "created"):
			typeColor = success
		case strings.Contains(event.Type, "deleted"):
			typeColor = errColor
		case strings.Contains(event.Type, "updated"):
			typeColor = info
		}

		// Verification status
		verifyIcon := ""
		if event.Signature != "" {
			if event.Verified {
				verifyIcon = fmt.Sprintf(" [%s]✓[-]", success)
			} else {
				verifyIcon = fmt.Sprintf(" [%s]✗[-]", errColor)
			}
		}

		_, _ = fmt.Fprintf(v.eventsPanel, "  [%s][%s][-] [%s::b]%s[-::-]%s\n",
			muted, timestamp,
			typeColor, event.Type,
			verifyIcon,
		)

		// Show additional details
		if event.ID != "" {
			_, _ = fmt.Fprintf(v.eventsPanel, "    [%s]ID:[-] [%s]%s[-]\n", muted, value, truncateStr(event.ID, 50))
		}
		if event.GrantID != "" {
			_, _ = fmt.Fprintf(v.eventsPanel, "    [%s]Grant:[-] [%s]%s[-]\n", muted, value, event.GrantID)
		}

		// Extract and show key fields from body
		if event.Body != nil {
			if data, ok := event.Body["data"].(map[string]any); ok {
				if obj, ok := data["object"].(map[string]any); ok {
					if subject, ok := obj["subject"].(string); ok {
						_, _ = fmt.Fprintf(v.eventsPanel, "    [%s]Subject:[-] [%s]%s[-]\n", muted, title, truncateStr(subject, 50))
					}
					if eventTitle, ok := obj["title"].(string); ok {
						_, _ = fmt.Fprintf(v.eventsPanel, "    [%s]Title:[-] [%s]%s[-]\n", muted, title, truncateStr(eventTitle, 50))
					}
				}
			}
		}

		// Separator between events
		if i < len(v.events)-1 {
			_, _ = fmt.Fprintf(v.eventsPanel, "  [%s]────────────────────────────────[-]\n", muted)
		}
	}
}

func (v *WebhookServerView) showPortPrompt() {
	// Create a simple modal for port input
	modal := tview.NewInputField()
	modal.SetLabel("Port: ")
	modal.SetFieldWidth(10)
	modal.SetText(fmt.Sprintf("%d", v.port))
	modal.SetBackgroundColor(v.app.styles.BgColor)
	modal.SetFieldBackgroundColor(v.app.styles.TableSelectBg)
	modal.SetFieldTextColor(v.app.styles.FgColor)
	modal.SetLabelColor(v.app.styles.TitleFg)
	modal.SetAcceptanceFunc(tview.InputFieldInteger)

	modal.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			var port int
			if _, err := fmt.Sscanf(modal.GetText(), "%d", &port); err == nil && port > 0 && port < 65536 {
				v.port = port
				v.app.Flash(FlashInfo, "Port set to %d", port)
			} else {
				v.app.Flash(FlashError, "Invalid port number")
			}
		}
		v.app.PopDetail()
		v.renderStatus()
	})

	// Wrap in a frame
	frame := tview.NewFrame(modal)
	frame.SetBorder(true)
	frame.SetBorderColor(v.app.styles.FocusColor)
	frame.SetTitle(" Set Port ")
	frame.SetTitleColor(v.app.styles.TitleFg)
	frame.SetBackgroundColor(v.app.styles.BgColor)

	// Create centered layout
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(frame, 30, 0, true).
			AddItem(nil, 0, 1, false), 5, 0, true).
		AddItem(nil, 0, 1, false)
	flex.SetBackgroundColor(v.app.styles.BgColor)

	v.app.PushDetail("port-input", flex)
	v.app.SetFocus(modal)
}

// truncateStr truncates a string to maxLen characters.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
