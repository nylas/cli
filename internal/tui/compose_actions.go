package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/domain"
)

func (c *ComposeView) handleKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		c.cancel()
		return nil

	case tcell.KeyCtrlS:
		c.send()
		return nil

	case tcell.KeyCtrlD:
		c.saveDraft()
		return nil

	case tcell.KeyTab:
		// Cycle focus between fields
		c.cycleFocus()
		return nil

	case tcell.KeyBacktab:
		// Cycle focus backwards
		c.cycleFocusBackward()
		return nil
	}

	return event
}

func (c *ComposeView) cycleFocus() {
	focused := c.app.GetFocus()

	switch focused {
	case c.toInput:
		c.app.SetFocus(c.ccInput)
	case c.ccInput:
		c.app.SetFocus(c.subjectInput)
	case c.subjectInput:
		c.app.SetFocus(c.bodyInput)
	case c.bodyInput:
		c.app.SetFocus(c.toInput)
	default:
		c.app.SetFocus(c.toInput)
	}
}

func (c *ComposeView) cycleFocusBackward() {
	focused := c.app.GetFocus()

	switch focused {
	case c.toInput:
		c.app.SetFocus(c.bodyInput)
	case c.ccInput:
		c.app.SetFocus(c.toInput)
	case c.subjectInput:
		c.app.SetFocus(c.ccInput)
	case c.bodyInput:
		c.app.SetFocus(c.subjectInput)
	default:
		c.app.SetFocus(c.toInput)
	}
}

// SetOnSent sets the callback for when an email is sent successfully.
func (c *ComposeView) SetOnSent(handler func()) {
	c.onSent = handler
}

// SetOnCancel sets the callback for when compose is cancelled.
func (c *ComposeView) SetOnCancel(handler func()) {
	c.onCancel = handler
}

// SetOnSave sets the callback for when a draft is saved.
func (c *ComposeView) SetOnSave(handler func()) {
	c.onSave = handler
}

func (c *ComposeView) send() {
	// Validate fields
	to := strings.TrimSpace(c.toInput.GetText())
	if to == "" {
		c.app.Flash(FlashError, "To field is required")
		return
	}

	subject := strings.TrimSpace(c.subjectInput.GetText())
	body := c.bodyInput.GetText()

	// Convert plain text body to HTML for proper rendering in email clients
	htmlBody := convertToHTML(body)

	// Parse recipients
	toRecipients := parseRecipients(to)
	if len(toRecipients) == 0 {
		c.app.Flash(FlashError, "Invalid recipient email")
		return
	}

	// Parse CC
	var ccRecipients []domain.EmailParticipant
	cc := strings.TrimSpace(c.ccInput.GetText())
	if cc != "" {
		ccRecipients = parseRecipients(cc)
	}

	// Send asynchronously
	c.app.Flash(FlashInfo, "Sending message...")

	go func() {
		// Email send operations should complete within 30 seconds
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var err error

		if c.mode == ComposeModeDraft && c.draft != nil {
			// For draft mode, first update the draft then send it
			updateReq := &domain.CreateDraftRequest{
				To:      toRecipients,
				Cc:      ccRecipients,
				Subject: subject,
				Body:    htmlBody,
			}
			_, err = c.app.config.Client.UpdateDraft(ctx, c.app.config.GrantID, c.draft.ID, updateReq)
			if err != nil {
				c.app.QueueUpdateDraw(func() {
					c.app.Flash(FlashError, "Failed to update draft: %v", err)
				})
				return
			}
			// Then send the draft
			_, err = c.app.config.Client.SendDraft(ctx, c.app.config.GrantID, c.draft.ID, nil)
		} else {
			// Normal send flow
			req := &domain.SendMessageRequest{
				To:      toRecipients,
				Cc:      ccRecipients,
				Subject: subject,
				Body:    htmlBody,
			}

			// If replying, include the reply-to message ID for threading
			if c.replyToMsg != nil && (c.mode == ComposeModeReply || c.mode == ComposeModeReplyAll) {
				req.ReplyToMsgID = c.replyToMsg.ID
			}

			_, err = c.app.config.Client.SendMessage(ctx, c.app.config.GrantID, req)
		}

		if err != nil {
			c.app.QueueUpdateDraw(func() {
				c.app.Flash(FlashError, "Failed to send: %v", err)
			})
			return
		}

		c.app.QueueUpdateDraw(func() {
			c.app.Flash(FlashInfo, "Message sent successfully!")
			if c.onSent != nil {
				c.onSent()
			}
		})
	}()
}

func (c *ComposeView) cancel() {
	if c.onCancel != nil {
		c.onCancel()
	}
}

// parseRecipients parses a comma-separated list of email addresses.
func parseRecipients(input string) []domain.EmailParticipant {
	var recipients []domain.EmailParticipant

	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse "Name <email>" format
		var name, email string
		if idx := strings.Index(part, "<"); idx != -1 {
			name = strings.TrimSpace(part[:idx])
			end := strings.Index(part, ">")
			if end > idx {
				email = strings.TrimSpace(part[idx+1 : end])
			}
		} else {
			email = part
		}

		if email != "" && strings.Contains(email, "@") {
			recipients = append(recipients, domain.EmailParticipant{
				Name:  name,
				Email: email,
			})
		}
	}

	return recipients
}

// stripHTMLForQuote removes HTML tags for quoting in replies.
func stripHTMLForQuote(s string) string {
	return stripHTMLForTUI(s)
}

// convertToHTML converts plain text to HTML for proper email rendering.
func convertToHTML(text string) string {
	// Escape HTML special characters
	escaped := strings.ReplaceAll(text, "&", "&amp;")
	escaped = strings.ReplaceAll(escaped, "<", "&lt;")
	escaped = strings.ReplaceAll(escaped, ">", "&gt;")

	// Convert newlines to <br> tags
	escaped = strings.ReplaceAll(escaped, "\n", "<br>\n")

	// Wrap in basic HTML structure
	return fmt.Sprintf(`<div style="font-family: Arial, sans-serif; font-size: 14px;">%s</div>`, escaped)
}

// prefillFromDraft populates the form with draft content.
func (c *ComposeView) prefillFromDraft() {
	if c.draft == nil {
		return
	}

	// Set To field
	if len(c.draft.To) > 0 {
		c.toInput.SetText(formatParticipants(c.draft.To))
	}

	// Set Cc field
	if len(c.draft.Cc) > 0 {
		c.ccInput.SetText(formatParticipants(c.draft.Cc))
	}

	// Set Subject
	c.subjectInput.SetText(c.draft.Subject)

	// Set Body - strip HTML for editing
	body := c.draft.Body
	body = stripHTMLForTUI(body)
	c.bodyInput.SetText(body, false)
}

// saveDraft saves the current compose as a draft.
func (c *ComposeView) saveDraft() {
	subject := strings.TrimSpace(c.subjectInput.GetText())
	body := c.bodyInput.GetText()
	htmlBody := convertToHTML(body)

	// Parse recipients
	to := strings.TrimSpace(c.toInput.GetText())
	var toRecipients []domain.EmailParticipant
	if to != "" {
		toRecipients = parseRecipients(to)
	}

	// Parse CC
	var ccRecipients []domain.EmailParticipant
	cc := strings.TrimSpace(c.ccInput.GetText())
	if cc != "" {
		ccRecipients = parseRecipients(cc)
	}

	// Build request
	req := &domain.CreateDraftRequest{
		To:      toRecipients,
		Cc:      ccRecipients,
		Subject: subject,
		Body:    htmlBody,
	}

	// If replying, include the reply-to message ID
	if c.replyToMsg != nil {
		req.ReplyToMsgID = c.replyToMsg.ID
	}

	c.app.Flash(FlashInfo, "Saving draft...")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var err error
		if c.mode == ComposeModeDraft && c.draft != nil {
			// Update existing draft
			_, err = c.app.config.Client.UpdateDraft(ctx, c.app.config.GrantID, c.draft.ID, req)
		} else {
			// Create new draft
			_, err = c.app.config.Client.CreateDraft(ctx, c.app.config.GrantID, req)
		}

		if err != nil {
			c.app.QueueUpdateDraw(func() {
				c.app.Flash(FlashError, "Failed to save draft: %v", err)
			})
			return
		}

		c.app.QueueUpdateDraw(func() {
			c.app.Flash(FlashInfo, "Draft saved!")
			if c.onSave != nil {
				c.onSave()
			}
		})
	}()
}
