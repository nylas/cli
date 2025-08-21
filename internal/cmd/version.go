package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = &cobra.Command{
	Use:     "version",
	Aliases: []string{"-v", "--version"},
	Short:   "Display the CLI's version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Nylas CLI version \x1B[1m1.0.0\x1B[22m")
	},
}

func init() {
	root.AddCommand(version)
}
