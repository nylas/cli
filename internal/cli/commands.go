package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
)

type commandFlagSpec struct {
	Name      string `json:"name" yaml:"name"`
	Shorthand string `json:"shorthand,omitempty" yaml:"shorthand,omitempty"`
	Type      string `json:"type,omitempty" yaml:"type,omitempty"`
	Default   string `json:"default,omitempty" yaml:"default,omitempty"`
	Usage     string `json:"usage,omitempty" yaml:"usage,omitempty"`
	Required  bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Hidden    bool   `json:"hidden,omitempty" yaml:"hidden,omitempty"`
}

type commandSpec struct {
	Name           string            `json:"name" yaml:"name"`
	Path           string            `json:"path" yaml:"path"`
	Use            string            `json:"use" yaml:"use"`
	Short          string            `json:"short,omitempty" yaml:"short,omitempty"`
	Long           string            `json:"long,omitempty" yaml:"long,omitempty"`
	Example        string            `json:"example,omitempty" yaml:"example,omitempty"`
	Aliases        []string          `json:"aliases,omitempty" yaml:"aliases,omitempty"`
	Deprecated     string            `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Hidden         bool              `json:"hidden,omitempty" yaml:"hidden,omitempty"`
	Runnable       bool              `json:"runnable" yaml:"runnable"`
	HasSubcommands bool              `json:"has_subcommands" yaml:"has_subcommands"`
	Flags          []commandFlagSpec `json:"flags,omitempty" yaml:"flags,omitempty"`
	InheritedFlags []commandFlagSpec `json:"inherited_flags,omitempty" yaml:"inherited_flags,omitempty"`
	Subcommands    []commandSpec     `json:"subcommands,omitempty" yaml:"subcommands,omitempty"`
}

type commandRow struct {
	Path    string
	Short   string
	Aliases string
}

func (r commandRow) QuietField() string {
	return r.Path
}

func newCommandsCmd() *cobra.Command {
	var includeHidden bool

	cmd := &cobra.Command{
		Use:   "commands [command-path...]",
		Short: "Inspect command metadata",
		Long: `Show machine-readable command and flag metadata for the CLI.

By default, this prints a flat list of commands for quick browsing.
Use --json or --format yaml to inspect a structured schema that agents and
automation can consume without scraping prose help output.`,
		Example: `  nylas commands
  nylas commands --json
  nylas commands email send --json
  nylas commands --all --format yaml`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveCommandTarget(cmd.Root(), args)
			if err != nil {
				return err
			}

			spec := buildCommandSpec(target, includeHidden)
			if spec == nil {
				return nil
			}

			quiet, _ := cmd.Flags().GetBool("quiet")
			if quiet {
				rows := flattenCommandSpec(*spec)
				return common.GetOutputWriter(cmd).WriteList(rows, []ports.Column{
					{Header: "Command", Field: "Path", Width: -1},
				})
			}

			if common.IsStructuredOutput(cmd) {
				return common.GetOutputWriter(cmd).Write(spec)
			}

			rows := flattenCommandSpec(*spec)
			if target == cmd.Root() && len(rows) > 0 {
				rows = rows[1:]
			}
			return common.GetOutputWriter(cmd).WriteList(rows, []ports.Column{
				{Header: "Command", Field: "Path", Width: -1},
				{Header: "Summary", Field: "Short", Width: -1},
				{Header: "Aliases", Field: "Aliases", Width: -1},
			})
		},
	}

	cmd.Flags().BoolVar(&includeHidden, "all", false, "Include hidden commands and flags")

	return cmd
}

func resolveCommandTarget(root *cobra.Command, args []string) (*cobra.Command, error) {
	if len(args) == 0 {
		return root, nil
	}

	target, _, err := root.Find(args)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, fmt.Errorf("command not found: %s", strings.Join(args, " "))
	}

	return target, nil
}

func buildCommandSpec(cmd *cobra.Command, includeHidden bool) *commandSpec {
	if cmd == nil {
		return nil
	}
	if cmd.Hidden && !includeHidden {
		return nil
	}

	spec := &commandSpec{
		Name:           cmd.Name(),
		Path:           cmd.CommandPath(),
		Use:            cmd.Use,
		Short:          cmd.Short,
		Long:           cmd.Long,
		Example:        cmd.Example,
		Aliases:        append([]string(nil), cmd.Aliases...),
		Deprecated:     cmd.Deprecated,
		Hidden:         cmd.Hidden,
		Runnable:       cmd.Runnable(),
		HasSubcommands: cmd.HasAvailableSubCommands(),
	}

	localFlags, localNames := collectCommandFlagSpecs(cmd, includeHidden)
	spec.Flags = localFlags
	spec.InheritedFlags = collectInheritedCommandFlagSpecs(cmd, includeHidden, localNames)

	children := append([]*cobra.Command(nil), cmd.Commands()...)
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name() < children[j].Name()
	})

	for _, child := range children {
		childSpec := buildCommandSpec(child, includeHidden)
		if childSpec == nil {
			continue
		}
		spec.Subcommands = append(spec.Subcommands, *childSpec)
	}

	return spec
}

func collectFlagSpecs(flags *pflag.FlagSet, includeHidden bool) []commandFlagSpec {
	if flags == nil {
		return nil
	}

	var specs []commandFlagSpec
	flags.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden && !includeHidden {
			return
		}

		specs = append(specs, commandFlagSpec{
			Name:      flag.Name,
			Shorthand: flag.Shorthand,
			Type:      flag.Value.Type(),
			Default:   flag.DefValue,
			Usage:     flag.Usage,
			Required:  isRequiredFlag(flag),
			Hidden:    flag.Hidden,
		})
	})

	return specs
}

func collectCommandFlagSpecs(cmd *cobra.Command, includeHidden bool) ([]commandFlagSpec, map[string]struct{}) {
	specs := make([]commandFlagSpec, 0)
	names := make(map[string]struct{})

	appendFlags := func(flags *pflag.FlagSet) {
		for _, spec := range collectFlagSpecs(flags, includeHidden) {
			if _, exists := names[spec.Name]; exists {
				continue
			}
			names[spec.Name] = struct{}{}
			specs = append(specs, spec)
		}
	}

	appendFlags(cmd.Flags())
	appendFlags(cmd.PersistentFlags())

	sort.Slice(specs, func(i, j int) bool {
		return specs[i].Name < specs[j].Name
	})

	return specs, names
}

func collectInheritedCommandFlagSpecs(cmd *cobra.Command, includeHidden bool, localNames map[string]struct{}) []commandFlagSpec {
	specs := make([]commandFlagSpec, 0)
	seen := make(map[string]struct{})

	for parent := cmd.Parent(); parent != nil; parent = parent.Parent() {
		for _, spec := range collectFlagSpecs(parent.PersistentFlags(), includeHidden) {
			if _, exists := localNames[spec.Name]; exists {
				continue
			}
			if _, exists := seen[spec.Name]; exists {
				continue
			}
			seen[spec.Name] = struct{}{}
			specs = append(specs, spec)
		}
	}

	sort.Slice(specs, func(i, j int) bool {
		return specs[i].Name < specs[j].Name
	})

	return specs
}

func isRequiredFlag(flag *pflag.Flag) bool {
	if flag == nil || flag.Annotations == nil {
		return false
	}
	_, ok := flag.Annotations[cobra.BashCompOneRequiredFlag]
	return ok
}

func flattenCommandSpec(spec commandSpec) []commandRow {
	rows := []commandRow{{
		Path:    spec.Path,
		Short:   spec.Short,
		Aliases: strings.Join(spec.Aliases, ", "),
	}}

	for _, child := range spec.Subcommands {
		rows = append(rows, flattenCommandSpec(child)...)
	}

	return rows
}
