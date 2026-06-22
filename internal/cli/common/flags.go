package common

import "github.com/spf13/cobra"

// AddLimitFlag adds a --limit/-n flag with the given default.
func AddLimitFlag(cmd *cobra.Command, target *int, defaultValue int) {
	cmd.Flags().IntVarP(target, "limit", "n", defaultValue, "Maximum number of items to show")
}

// AddPageTokenFlag adds a --page-token flag for pagination.
func AddPageTokenFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, "page-token", "", "Page token for pagination")
}

// AddYesFlag adds a --yes/-y flag for skipping confirmations.
func AddYesFlag(cmd *cobra.Command, target *bool) {
	cmd.Flags().BoolVarP(target, "yes", "y", false, "Skip confirmation prompts")
}
