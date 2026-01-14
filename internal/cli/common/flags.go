package common

import "github.com/spf13/cobra"

// AddLimitFlag adds a --limit/-n flag with the given default.
func AddLimitFlag(cmd *cobra.Command, target *int, defaultValue int) {
	cmd.Flags().IntVarP(target, "limit", "n", defaultValue, "Maximum number of items to show")
}

// AddFormatFlag adds a --format/-f flag for output format.
func AddFormatFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVarP(target, "format", "f", "table", "Output format (table, json, yaml, csv)")
}

// AddIDFlag adds a --id flag to show resource IDs.
func AddIDFlag(cmd *cobra.Command, target *bool) {
	cmd.Flags().BoolVar(target, "id", false, "Show IDs in output")
}

// AddPageTokenFlag adds a --page-token flag for pagination.
func AddPageTokenFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, "page-token", "", "Page token for pagination")
}

// AddForceFlag adds a --force/-f flag for skipping confirmations.
func AddForceFlag(cmd *cobra.Command, target *bool) {
	cmd.Flags().BoolVarP(target, "force", "f", false, "Skip confirmation prompts")
}

// AddYesFlag adds a --yes/-y flag for skipping confirmations.
func AddYesFlag(cmd *cobra.Command, target *bool) {
	cmd.Flags().BoolVarP(target, "yes", "y", false, "Skip confirmation prompts")
}

// AddVerboseFlag adds a --verbose/-v flag for verbose output.
func AddVerboseFlag(cmd *cobra.Command, target *bool) {
	cmd.Flags().BoolVarP(target, "verbose", "v", false, "Show verbose output")
}
