package common

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Nylas brand color palette — used consistently across all CLI output.
const (
	ColorPrimary = lipgloss.Color("#00BCD4") // Cyan — brand accent
	ColorSuccess = lipgloss.Color("#4CAF50") // Green
	ColorWarning = lipgloss.Color("#FFC107") // Amber
	ColorError   = lipgloss.Color("#F44336") // Red
	ColorMuted   = lipgloss.Color("#6B7280") // Gray
	ColorText    = lipgloss.Color("#E0E0E0") // Light gray
	ColorDim     = lipgloss.Color("#4A4A4A") // Dark gray
)

// NylasTheme returns the huh theme used for all interactive prompts.
func NylasTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Focused field styles
	t.Focused.Base = lipgloss.NewStyle().
		PaddingLeft(1).
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderForeground(ColorPrimary)

	t.Focused.Title = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	t.Focused.Description = lipgloss.NewStyle().
		Foreground(ColorMuted)

	t.Focused.ErrorIndicator = lipgloss.NewStyle().
		Foreground(ColorError).
		SetString(" *")

	t.Focused.ErrorMessage = lipgloss.NewStyle().
		Foreground(ColorError)

	// Select
	t.Focused.SelectSelector = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		SetString("❯ ")

	t.Focused.Option = lipgloss.NewStyle().
		Foreground(ColorText)

	t.Focused.NextIndicator = lipgloss.NewStyle().
		Foreground(ColorMuted).
		SetString(" →")

	t.Focused.PrevIndicator = lipgloss.NewStyle().
		Foreground(ColorMuted).
		SetString("← ")

	// MultiSelect
	t.Focused.MultiSelectSelector = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		SetString("❯ ")

	t.Focused.SelectedOption = lipgloss.NewStyle().
		Foreground(ColorSuccess)

	t.Focused.SelectedPrefix = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		SetString("✓ ")

	t.Focused.UnselectedOption = lipgloss.NewStyle().
		Foreground(ColorText)

	t.Focused.UnselectedPrefix = lipgloss.NewStyle().
		Foreground(ColorMuted).
		SetString("○ ")

	// Confirm buttons
	t.Focused.FocusedButton = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(ColorPrimary).
		Padding(0, 2).
		Bold(true)

	t.Focused.BlurredButton = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorDim).
		Padding(0, 2)

	// Text input
	t.Focused.TextInput.Cursor = lipgloss.NewStyle().
		Foreground(ColorPrimary)

	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().
		Foreground(ColorMuted)

	t.Focused.TextInput.Prompt = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		SetString("❯ ")

	t.Focused.TextInput.Text = lipgloss.NewStyle().
		Foreground(ColorText)

	// Card / Note
	t.Focused.Card = t.Focused.Base
	t.Focused.NoteTitle = t.Focused.Title
	t.Focused.Next = t.Focused.FocusedButton

	// Blurred state — same styles but hidden border
	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	return t
}
