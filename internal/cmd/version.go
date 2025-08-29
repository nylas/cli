package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Get build info
func SetBuildInfo(v string, c string, d string) {
    if v != "" {
        version = v
    }
    if c != "" {
        commit = c
    }
    if d != "" {
        date = d
    }
}

var version = &cobra.Command{
	Use:     "version",
	Aliases: []string{"-v", "--version"},
	Short:   "Display the CLI's version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Nylas CLI version \x1B[1m%s\x1B[22m (commit %s, built %s)\n", cliVersion, cliCommit, cliDate)
	},
}

func init() {
	root.AddCommand(version)
}
