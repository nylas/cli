// Package common provides shared CLI utilities.
package common

import (
	"os"

	"github.com/charmbracelet/huh"
	"golang.org/x/term"
)

// SelectOption represents a labeled option for Select prompts.
type SelectOption[T comparable] struct {
	Label string
	Value T
}

// Select presents an interactive select menu with arrow-key navigation.
// Falls back to the first option if stdin is not a TTY.
func Select[T comparable](title string, options []SelectOption[T]) (T, error) {
	if len(options) == 0 {
		var zero T
		return zero, nil
	}

	// Non-interactive fallback
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return options[0].Value, nil
	}

	var result T
	huhOpts := make([]huh.Option[T], len(options))
	for i, opt := range options {
		huhOpts[i] = huh.NewOption(opt.Label, opt.Value)
	}

	err := huh.NewSelect[T]().
		Title(title).
		Options(huhOpts...).
		Value(&result).
		Run()

	return result, err
}

// ConfirmPrompt presents an interactive yes/no confirmation with arrow-key navigation.
func ConfirmPrompt(title string, defaultYes bool) (bool, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return defaultYes, nil
	}

	result := defaultYes
	err := huh.NewConfirm().
		Title(title).
		Affirmative("Yes").
		Negative("No").
		Value(&result).
		Run()

	return result, err
}

// InputPrompt presents an interactive text input.
func InputPrompt(title, placeholder string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return placeholder, nil
	}

	var result string
	field := huh.NewInput().
		Title(title).
		Value(&result)

	if placeholder != "" {
		field.Placeholder(placeholder)
	}

	err := field.Run()
	if err != nil {
		return "", err
	}
	if result == "" && placeholder != "" {
		return placeholder, nil
	}
	return result, nil
}

// PasswordPrompt presents an interactive masked password input.
func PasswordPrompt(title string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", nil
	}

	var result string
	err := huh.NewInput().
		Title(title).
		EchoMode(huh.EchoModePassword).
		Value(&result).
		Run()

	return result, err
}
