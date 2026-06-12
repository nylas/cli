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

func (v *MessagesView) showDetail(thread *domain.Thread) {
	v.currentThread = thread

	detail := tview.NewTextView()
	detail.SetDynamicColors(true)
	detail.SetBackgroundColor(v.app.styles.BgColor)
	detail.SetBorderPadding(1, 1, 2, 2)
	detail.SetScrollable(true)

	// k9s style colors - use cached Hex() method
	s := v.app.styles
	title := s.Hex(s.TitleFg)
	key := s.Hex(s.FgColor)
	value := s.Hex(s.InfoSectionFg)
	muted := s.Hex(s.BorderColor)
	hint := s.Hex(s.InfoColor)

	// Format participants
	var participants []string
	for _, p := range thread.Participants {
		participants = append(participants, p.String())
	}

	// Show loading state first
	_, _ = fmt.Fprintf(detail, "[%s::b]%s[-::-]\n", title, thread.Subject)
	_, _ = fmt.Fprintf(detail, "[%s]Participants:[-] [%s]%s[-]\n", key, value, strings.Join(participants, ", "))
	_, _ = fmt.Fprintf(detail, "[%s]Messages:[-] [%s]%d[-]\n\n", key, value, len(thread.MessageIDs))
	_, _ = fmt.Fprintf(detail, "[%s]────────────────────────────────────────[-]\n\n", muted)
	_, _ = fmt.Fprintf(detail, "[%s]Loading messages...[-]\n\n", muted)

	// Fetch all messages in the thread asynchronously
	grantID := v.app.config.GrantID
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Fetch each message in the thread
		var messages []*domain.Message
		for _, msgID := range thread.MessageIDs {
			msg, err := v.app.config.Client.GetMessage(ctx, grantID, msgID)
			if err == nil {
				messages = append(messages, msg)
			}
		}

		v.app.QueueUpdateDraw(func() {
			if !v.app.grantStillCurrent(grantID) {
				return // grant switched while fetch was in flight; drop stale data
			}
			detail.Clear()

			// Clear attachments list
			v.attachments = nil

			_, _ = fmt.Fprintf(detail, "[%s::b]%s[-::-]\n", title, thread.Subject)
			_, _ = fmt.Fprintf(detail, "[%s]Participants:[-] [%s]%s[-]\n", key, value, strings.Join(participants, ", "))
			_, _ = fmt.Fprintf(detail, "[%s]Messages:[-] [%s]%d[-]\n\n", key, value, len(thread.MessageIDs))

			if len(messages) == 0 {
				_, _ = fmt.Fprintf(detail, "[%s]────────────────────────────────────────[-]\n\n", muted)
				_, _ = fmt.Fprintf(detail, "[%s]%s[-]\n\n", value, thread.Snippet)
			} else {
				// Display all messages in chronological order
				for i, msg := range messages {
					_, _ = fmt.Fprintf(detail, "[%s]════════════════════════════════════════[-]\n", muted)

					from := ""
					if len(msg.From) > 0 {
						from = msg.From[0].String()
					}

					_, _ = fmt.Fprintf(detail, "[%s]From:[-] [%s]%s[-]\n", key, value, from)
					_, _ = fmt.Fprintf(detail, "[%s]Date:[-] [%s]%s[-]\n", key, value, msg.Date.Format(common.DisplayWeekdayComma))

					// Display attachments if any
					if len(msg.Attachments) > 0 {
						_, _ = fmt.Fprintf(detail, "[%s]Attachments:[-]", key)
						for _, att := range msg.Attachments {
							if att.IsInline {
								continue // Skip inline attachments (images in HTML)
							}
							// Track attachment with its message ID
							attachmentIdx := len(v.attachments)
							v.attachments = append(v.attachments, AttachmentInfo{
								MessageID:  msg.ID,
								Attachment: att,
							})
							sizeStr := formatFileSize(att.Size)
							_, _ = fmt.Fprintf(detail, " [%s][%d] %s (%s)[-]", hint, attachmentIdx+1, att.Filename, sizeStr)
						}
						_, _ = fmt.Fprintln(detail)
					}
					_, _ = fmt.Fprintln(detail)

					// Use full body, strip HTML for terminal display
					body := msg.Body
					if body == "" {
						body = msg.Snippet
					}
					body = stripHTMLForTUI(body)
					_, _ = fmt.Fprintf(detail, "[%s]%s[-]\n\n", value, tview.Escape(body))

					// Store the last message for reply
					if i == len(messages)-1 {
						v.currentMessage = msg
					}
				}
			}

			// Build help line based on available actions
			helpLine := fmt.Sprintf("[%s]R[-][%s::d]=reply  [-::-][%s]A[-][%s::d]=reply all  [-::-]", hint, muted, hint, muted)
			if len(v.attachments) > 0 {
				helpLine += fmt.Sprintf("[%s]D[-][%s::d]=download  [-::-]", hint, muted)
			}
			helpLine += fmt.Sprintf("[%s]Esc[-][%s::d]=back[-::-]", hint, muted)
			_, _ = fmt.Fprint(detail, helpLine)
		})
	}()

	_, _ = fmt.Fprintf(detail, "[%s]R[-][%s::d]=reply  [-::-][%s]A[-][%s::d]=reply all  [-::-][%s]Esc[-][%s::d]=back[-::-]", hint, muted, hint, muted, hint, muted)

	// Handle key events for reply actions in detail view
	detail.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			v.closeDetail()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'R':
				if v.currentMessage != nil {
					v.showCompose(ComposeModeReply, v.currentMessage)
				}
				return nil
			case 'A':
				if v.currentMessage != nil {
					v.showCompose(ComposeModeReplyAll, v.currentMessage)
				}
				return nil
			case 'D':
				if len(v.attachments) > 0 {
					v.showDownloadDialog()
				}
				return nil
			}
		}
		return event
	})

	// Push detail onto the page stack
	v.app.PushDetail("thread-detail", detail)
	v.showingDetail = true
}

func (v *MessagesView) closeDetail() {
	v.app.PopDetail()
	v.showingDetail = false
	v.currentThread = nil
	v.currentMessage = nil
	v.attachments = nil
	v.app.SetFocus(v.table)
}
