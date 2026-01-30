// Package version provides version information for the CLI.
// This package exists to avoid import cycles between cli and adapters packages.
package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the CLI version, set via ldflags during build.
	Version = "dev"

	// Commit is the git commit hash, set via ldflags during build.
	Commit = "none"

	// BuildDate is the build timestamp, set via ldflags during build.
	BuildDate = "unknown"
)

// UserAgent returns the User-Agent string for API requests.
func UserAgent() string {
	return fmt.Sprintf("nylas-cli/%s (%s/%s)", Version, runtime.GOOS, runtime.GOARCH)
}
