package common

import "github.com/fatih/color"

// Common color definitions used across CLI commands.
// Import these instead of defining package-local color vars.
var (
	// Basic colors
	Cyan   = color.New(color.FgCyan)
	Green  = color.New(color.FgGreen)
	Yellow = color.New(color.FgYellow)
	Red    = color.New(color.FgRed)
	Blue   = color.New(color.FgBlue)

	// Styles
	Bold = color.New(color.Bold)
	Dim  = color.New(color.Faint)

	// Bold colors
	BoldWhite  = color.New(color.FgWhite, color.Bold)
	BoldCyan   = color.New(color.FgCyan, color.Bold)
	BoldGreen  = color.New(color.FgGreen, color.Bold)
	BoldBlue   = color.New(color.FgBlue, color.Bold)
	BoldYellow = color.New(color.FgYellow, color.Bold)
	BoldRed    = color.New(color.FgRed, color.Bold)

	// High intensity
	HiBlack = color.New(color.FgHiBlack)

	// Reset (no formatting)
	Reset = color.New(color.Reset)
)
