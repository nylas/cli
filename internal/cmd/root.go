package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// CLI root
var root = &cobra.Command{
	Use:   "nylas",
	Short: "The official CLI for Nylas",
	Long:  "The official CLI for Nylas.\n\nBefore using the CLI, you'll need to set your Nylas API Key:\n    $ nylas init",
}

func Execute() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
