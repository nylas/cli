package cli

import (
	"fmt"
	"runtime"

	"github.com/nylas/cli/internal/version"
	"github.com/spf13/cobra"
)

// Version aliases for backwards compatibility
var (
	Version   = version.Version
	Commit    = version.Commit
	BuildDate = version.BuildDate
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "nylas version %s\n", version.Version)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Commit:     %s\n", version.Commit)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Built:      %s\n", version.BuildDate)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Go version: %s\n", runtime.Version())
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}
}
