package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Dialog displays a modal dialog for confirmations or messages.
type Dialog struct {
	*tview.Flex
	app       *App
	modal     *tview.Modal
	onConfirm func()
	onCancel  func()
}

// NewConfirmDialog creates a confirmation dialog with Yes/No buttons.
func NewConfirmDialog(app *App, title, message string, onConfirm, onCancel func()) *Dialog {
	d := &Dialog{
		Flex:      tview.NewFlex(),
		app:       app,
		onConfirm: onConfirm,
		onCancel:  onCancel,
	}

	styles := app.styles

	d.modal = tview.NewModal()
	d.modal.SetText(message)
	d.modal.SetTitle(fmt.Sprintf(" %s ", title))
	d.modal.SetTitleColor(styles.TitleFg)
	d.modal.SetBackgroundColor(styles.BgColor)
	d.modal.SetTextColor(styles.FgColor)
	d.modal.SetButtonBackgroundColor(styles.FocusColor)
	d.modal.SetButtonTextColor(styles.BgColor)
	d.modal.SetBorder(true)
	d.modal.SetBorderColor(styles.FocusColor)

	d.modal.AddButtons([]string{"Yes", "No"})
	d.modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonLabel == "Yes" && d.onConfirm != nil {
			d.onConfirm()
		} else if d.onCancel != nil {
			d.onCancel()
		}
	})

	// Set up keyboard shortcuts
	d.modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if d.onCancel != nil {
				d.onCancel()
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'y', 'Y':
				if d.onConfirm != nil {
					d.onConfirm()
				}
				return nil
			case 'n', 'N':
				if d.onCancel != nil {
					d.onCancel()
				}
				return nil
			}
		}
		return event
	})

	// Center the modal
	d.SetDirection(tview.FlexRow)
	d.AddItem(nil, 0, 1, false)
	d.AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(d.modal, 50, 0, true).
		AddItem(nil, 0, 1, false), 10, 0, true)
	d.AddItem(nil, 0, 1, false)

	return d
}

// NewInfoDialog creates an informational dialog with OK button.
func NewInfoDialog(app *App, title, message string, onClose func()) *Dialog {
	d := &Dialog{
		Flex:     tview.NewFlex(),
		app:      app,
		onCancel: onClose,
	}

	styles := app.styles

	d.modal = tview.NewModal()
	d.modal.SetText(message)
	d.modal.SetTitle(fmt.Sprintf(" %s ", title))
	d.modal.SetTitleColor(styles.TitleFg)
	d.modal.SetBackgroundColor(styles.BgColor)
	d.modal.SetTextColor(styles.FgColor)
	d.modal.SetButtonBackgroundColor(styles.FocusColor)
	d.modal.SetButtonTextColor(styles.BgColor)
	d.modal.SetBorder(true)
	d.modal.SetBorderColor(styles.InfoColor)

	d.modal.AddButtons([]string{"OK"})
	d.modal.SetDoneFunc(func(_ int, _ string) {
		if d.onCancel != nil {
			d.onCancel()
		}
	})

	d.modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyEnter {
			if d.onCancel != nil {
				d.onCancel()
			}
			return nil
		}
		return event
	})

	// Center the modal
	d.SetDirection(tview.FlexRow)
	d.AddItem(nil, 0, 1, false)
	d.AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(d.modal, 50, 0, true).
		AddItem(nil, 0, 1, false), 10, 0, true)
	d.AddItem(nil, 0, 1, false)

	return d
}

// NewErrorDialog creates an error dialog with OK button.
func NewErrorDialog(app *App, title, message string, onClose func()) *Dialog {
	d := &Dialog{
		Flex:     tview.NewFlex(),
		app:      app,
		onCancel: onClose,
	}

	styles := app.styles

	d.modal = tview.NewModal()
	d.modal.SetText(message)
	d.modal.SetTitle(fmt.Sprintf(" %s ", title))
	d.modal.SetTitleColor(styles.ErrorColor)
	d.modal.SetBackgroundColor(styles.BgColor)
	d.modal.SetTextColor(styles.ErrorColor)
	d.modal.SetButtonBackgroundColor(styles.ErrorColor)
	d.modal.SetButtonTextColor(styles.BgColor)
	d.modal.SetBorder(true)
	d.modal.SetBorderColor(styles.ErrorColor)

	d.modal.AddButtons([]string{"OK"})
	d.modal.SetDoneFunc(func(_ int, _ string) {
		if d.onCancel != nil {
			d.onCancel()
		}
	})

	d.modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyEnter {
			if d.onCancel != nil {
				d.onCancel()
			}
			return nil
		}
		return event
	})

	// Center the modal
	d.SetDirection(tview.FlexRow)
	d.AddItem(nil, 0, 1, false)
	d.AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(d.modal, 50, 0, true).
		AddItem(nil, 0, 1, false), 10, 0, true)
	d.AddItem(nil, 0, 1, false)

	return d
}

// Focus sets focus to the modal.
func (d *Dialog) Focus(delegate func(p tview.Primitive)) {
	delegate(d.modal)
}

// ShowConfirmDialog displays a confirm dialog and returns.
func (a *App) ShowConfirmDialog(title, message string, onConfirm func()) {
	onClose := func() {
		a.content.Pop()
		if view := a.getCurrentView(); view != nil {
			a.SetFocus(view.Primitive())
		}
	}

	dialog := NewConfirmDialog(a, title, message, func() {
		onClose()
		if onConfirm != nil {
			onConfirm()
		}
	}, onClose)

	a.content.Push("dialog", dialog)
	a.SetFocus(dialog)
}

// ShowInfoDialog displays an info dialog.
func (a *App) ShowInfoDialog(title, message string) {
	onClose := func() {
		a.content.Pop()
		if view := a.getCurrentView(); view != nil {
			a.SetFocus(view.Primitive())
		}
	}

	dialog := NewInfoDialog(a, title, message, onClose)
	a.content.Push("dialog", dialog)
	a.SetFocus(dialog)
}

// ShowErrorDialog displays an error dialog.
func (a *App) ShowErrorDialog(title, message string) {
	onClose := func() {
		a.content.Pop()
		if view := a.getCurrentView(); view != nil {
			a.SetFocus(view.Primitive())
		}
	}

	dialog := NewErrorDialog(a, title, message, onClose)
	a.content.Push("dialog", dialog)
	a.SetFocus(dialog)
}
