// Package common provides shared utilities for CLI commands.
package common

import "sync/atomic"

// quietMode suppresses decorative output (success messages, tables, spinners,
// progress) process-wide. It is set from the --quiet flag at command startup
// (see the root PersistentPreRunE); the OutputWriter handles structured data
// separately. atomic.Bool because it is written at startup while spinner/
// progress goroutines read it concurrently.
var quietMode atomic.Bool

// SetQuiet enables or disables quiet mode.
func SetQuiet(quiet bool) {
	quietMode.Store(quiet)
}

// IsQuiet returns true if quiet mode is enabled.
func IsQuiet() bool {
	return quietMode.Load()
}
