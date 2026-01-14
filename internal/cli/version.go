package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "nylas version %s\n", Version)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Commit:     %s\n", Commit)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Built:      %s\n", BuildDate)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Go version: %s\n", runtime.Version())
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}
}
