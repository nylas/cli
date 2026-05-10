package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/domain"
	"github.com/rivo/tview"
)

// ContactsView displays contacts.
type ContactsView struct {
	*BaseTableView
	contacts []domain.Contact
}

// NewContactsView creates a new contacts view.
func NewContactsView(app *App) *ContactsView {
	v := &ContactsView{
		BaseTableView: newBaseTableView(app, "contacts", "Contacts"),
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
		{Title: "NAME", Width: 30},
		{Title: "EMAIL", Expand: true},
		{Title: "COMPANY", Width: 25},
	})

	return v
}

func (v *ContactsView) Load() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	contacts, err := v.app.config.Client.GetContacts(ctx, v.app.config.GrantID, nil)
	if err != nil {
		v.app.FlashLoadError("Failed to load contacts", err)
		return
	}
	v.contacts = contacts
	v.render()
}

func (v *ContactsView) Refresh() { v.Load() }

func (v *ContactsView) render() {
	var data [][]string
	var meta []RowMeta

	for _, c := range v.contacts {
		email := ""
		if len(c.Emails) > 0 {
			email = c.Emails[0].Email
		}

		name := c.GivenName
		if c.Surname != "" {
			name += " " + c.Surname
		}

		data = append(data, []string{
			"",
			name,
			email,
			c.CompanyName,
		})
		meta = append(meta, RowMeta{ID: c.ID, Data: &c})
	}

	v.table.SetData(data, meta)
}

// HandleKey handles keyboard input for contacts view.
func (v *ContactsView) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEnter:
		// View contact detail
		if idx, _ := v.table.GetSelection(); idx > 0 && idx-1 < len(v.contacts) {
			v.showContactDetail(&v.contacts[idx-1])
		}
		return nil

	case tcell.KeyRune:
		switch event.Rune() {
		case 'n': // New contact
			v.app.ShowContactForm(nil, func(contact *domain.Contact) {
				v.Refresh()
			})
			return nil

		case 'e': // Edit selected contact
			if idx, _ := v.table.GetSelection(); idx > 0 && idx-1 < len(v.contacts) {
				contact := v.contacts[idx-1]
				v.app.ShowContactForm(&contact, func(updatedContact *domain.Contact) {
					v.Refresh()
				})
			}
			return nil

		case 'd': // Delete selected contact
			if idx, _ := v.table.GetSelection(); idx > 0 && idx-1 < len(v.contacts) {
				contact := v.contacts[idx-1]
				v.app.DeleteContact(&contact, func() {
					v.Refresh()
				})
			}
			return nil
		}
	}

	return event
}

func (v *ContactsView) showContactDetail(contact *domain.Contact) {
	detail := tview.NewTextView()
	detail.SetDynamicColors(true)
	detail.SetBackgroundColor(v.app.styles.BgColor)
	detail.SetBorder(true)
	detail.SetBorderColor(v.app.styles.FocusColor)
	detail.SetTitle(fmt.Sprintf(" %s ", contact.DisplayName()))
	detail.SetTitleColor(v.app.styles.TitleFg)
	detail.SetBorderPadding(1, 1, 2, 2)
	detail.SetScrollable(true)

	// Use cached Hex() method
	s := v.app.styles
	info := s.Hex(s.InfoColor)
	value := s.Hex(s.InfoSectionFg)
	muted := s.Hex(s.BorderColor)

	// Name
	if contact.GivenName != "" || contact.Surname != "" {
		_, _ = fmt.Fprintf(detail, "[%s::b]Name[-::-]\n", info)
		if contact.GivenName != "" {
			_, _ = fmt.Fprintf(detail, "[%s]%s[-]", value, contact.GivenName)
		}
		if contact.Surname != "" {
			if contact.GivenName != "" {
				_, _ = fmt.Fprintf(detail, "[%s] %s[-]", value, contact.Surname)
			} else {
				_, _ = fmt.Fprintf(detail, "[%s]%s[-]", value, contact.Surname)
			}
		}
		_, _ = fmt.Fprintln(detail)
	}

	// Emails
	if len(contact.Emails) > 0 {
		_, _ = fmt.Fprintf(detail, "[%s::b]Email[-::-]\n", info)
		for _, e := range contact.Emails {
			typeStr := e.Type
			if typeStr == "" {
				typeStr = "other"
			}
			_, _ = fmt.Fprintf(detail, "[%s]%s[-] [%s](%s)[-]\n", value, e.Email, muted, typeStr)
		}
		_, _ = fmt.Fprintln(detail)
	}

	// Phone numbers
	if len(contact.PhoneNumbers) > 0 {
		_, _ = fmt.Fprintf(detail, "[%s::b]Phone[-::-]\n", info)
		for _, p := range contact.PhoneNumbers {
			typeStr := p.Type
			if typeStr == "" {
				typeStr = "other"
			}
			_, _ = fmt.Fprintf(detail, "[%s]%s[-] [%s](%s)[-]\n", value, p.Number, muted, typeStr)
		}
		_, _ = fmt.Fprintln(detail)
	}

	// Company
	if contact.CompanyName != "" || contact.JobTitle != "" {
		_, _ = fmt.Fprintf(detail, "[%s::b]Work[-::-]\n", info)
		if contact.JobTitle != "" {
			_, _ = fmt.Fprintf(detail, "[%s]%s[-]\n", value, contact.JobTitle)
		}
		if contact.CompanyName != "" {
			_, _ = fmt.Fprintf(detail, "[%s]%s[-]\n", value, contact.CompanyName)
		}
		_, _ = fmt.Fprintln(detail)
	}

	// Notes
	if contact.Notes != "" {
		_, _ = fmt.Fprintf(detail, "[%s::b]Notes[-::-]\n", info)
		_, _ = fmt.Fprintf(detail, "[%s]%s[-]\n\n", value, contact.Notes)
	}

	_, _ = fmt.Fprintf(detail, "\n[%s::d]Press Esc to go back, 'e' to edit, 'd' to delete[-::-]", muted)

	// Handle keyboard
	contactCopy := contact
	detail.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			v.app.PopDetail()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'e':
				v.app.PopDetail()
				v.app.ShowContactForm(contactCopy, func(updatedContact *domain.Contact) {
					v.Refresh()
				})
				return nil
			case 'd':
				v.app.PopDetail()
				v.app.DeleteContact(contactCopy, func() {
					v.Refresh()
				})
				return nil
			}
		}
		return event
	})

	v.app.PushDetail("contact-detail", detail)
	v.app.SetFocus(detail)
}
