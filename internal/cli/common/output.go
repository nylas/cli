package common

import (
	"io"

	"github.com/nylas/cli/internal/adapters/output"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

// AddOutputFlags adds common output flags to a command
// These flags are inherited by all subcommands when added to a parent
func AddOutputFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("format", "", "Output format: table, json, yaml")
	cmd.PersistentFlags().Bool("json", false, "Output in JSON format")
	cmd.PersistentFlags().BoolP("quiet", "q", false, "Quiet mode - only output essential data (IDs)")
	cmd.PersistentFlags().Bool("no-color", false, "Disable colored output")
	cmd.PersistentFlags().BoolP("wide", "w", false, "Wide output - show full IDs without truncation")
}

// GetOutputWriter creates an output writer based on command flags
func GetOutputWriter(cmd *cobra.Command) ports.OutputWriter {
	return GetOutputWriterTo(cmd, cmd.OutOrStdout())
}

// GetOutputWriterTo creates an output writer to a specific destination
func GetOutputWriterTo(cmd *cobra.Command, w io.Writer) ports.OutputWriter {
	opts := GetOutputOptions(cmd, w)
	return output.NewWriter(w, opts)
}

// GetOutputOptions extracts output options from command flags
func GetOutputOptions(cmd *cobra.Command, w io.Writer) ports.OutputOptions {
	format := getOutputFormat(cmd)
	noColor, _ := cmd.Flags().GetBool("no-color")

	return ports.OutputOptions{
		Format:  format,
		NoColor: noColor,
		Writer:  w,
	}
}

// getOutputFormat determines the output format from flags
func getOutputFormat(cmd *cobra.Command) ports.OutputFormat {
	// --quiet/-q takes precedence
	quiet, _ := cmd.Flags().GetBool("quiet")
	if quiet {
		return ports.FormatQuiet
	}

	// --json takes precedence over --format
	jsonFlag, _ := cmd.Flags().GetBool("json")
	if jsonFlag {
		return ports.FormatJSON
	}

	// Check --format flag
	format, _ := cmd.Flags().GetString("format")
	switch format {
	case "json":
		return ports.FormatJSON
	case "yaml":
		return ports.FormatYAML
	case "quiet":
		return ports.FormatQuiet
	default:
		return ports.FormatTable
	}
}

// IsJSON returns true if JSON output is enabled
func IsJSON(cmd *cobra.Command) bool {
	jsonFlag, _ := cmd.Flags().GetBool("json")
	if jsonFlag {
		return true
	}
	format, _ := cmd.Flags().GetString("format")
	return format == "json"
}

// IsStructuredOutput returns true if non-table output is requested (JSON, YAML, or quiet).
func IsStructuredOutput(cmd *cobra.Command) bool {
	if IsJSON(cmd) {
		return true
	}
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	return format == "yaml" || format == "quiet" || quiet
}

// IsWide returns true if wide output mode is enabled
func IsWide(cmd *cobra.Command) bool {
	wide, _ := cmd.Flags().GetBool("wide")
	return wide
}

// WriteListWithColumns writes list output with appropriate columns based on flags
// This is a convenience function that handles the common pattern of outputting data
func WriteListWithColumns(cmd *cobra.Command, data any, columns []ports.Column) error {
	out := GetOutputWriter(cmd)
	return out.WriteList(data, columns)
}

// WriteListWithWideColumns writes list output with different columns for normal vs wide mode
// This is useful for tables where you want to show full IDs in wide mode
func WriteListWithWideColumns(cmd *cobra.Command, data any, normalCols, wideCols []ports.Column) error {
	cols := normalCols
	if IsWide(cmd) {
		cols = wideCols
	}
	out := GetOutputWriter(cmd)
	return out.WriteList(data, cols)
}
