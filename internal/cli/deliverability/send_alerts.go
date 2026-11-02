package deliverability

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

type alertRuleFlags struct {
	metric      string
	threshold   string
	windowHours int64
	minVolume   int64
}

func addAlertRuleFlags(cmd *cobra.Command, f *alertRuleFlags) {
	cmd.Flags().StringVar(&f.metric, "metric", "", "Metric: "+strings.Join(domain.SendAlertMetrics, ", "))
	cmd.Flags().StringVar(&f.threshold, "threshold", "",
		"Threshold percent, up to 2 decimals; bounce/complaint fire above it (0 to <100), delivery fires below it (>0 to 100)")
	cmd.Flags().Int64Var(&f.windowHours, "window-hours", 0, "Look-back window in hours, 1-24")
	cmd.Flags().Int64Var(&f.minVolume, "min-volume", 0, "Minimum sends in the window before the rule can fire (default 0)")
}

// buildAlertRuleRequest includes only the flags the user set, so update leaves the rest unchanged.
func buildAlertRuleRequest(cmd *cobra.Command, f *alertRuleFlags) (*domain.SendAlertRuleRequest, error) {
	req := &domain.SendAlertRuleRequest{}
	if cmd.Flags().Changed("metric") {
		req.Metric = &f.metric
	}
	if cmd.Flags().Changed("threshold") {
		if _, err := strconv.ParseFloat(f.threshold, 64); err != nil {
			return nil, common.NewUserError(fmt.Sprintf("invalid --threshold value %q", f.threshold), "Use a percentage such as 5 or 0.25")
		}
		n := json.Number(strings.TrimSpace(f.threshold))
		req.Threshold = &n
	}
	if cmd.Flags().Changed("window-hours") {
		req.WindowHours = &f.windowHours
	}
	if cmd.Flags().Changed("min-volume") {
		req.MinVolume = &f.minVolume
	}
	return req, nil
}

func newSendAlertsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List send alert rules",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withClient(func(ctx context.Context, client ports.NylasClient) error {
				rules, err := client.ListSendAlertRules(ctx)
				if err != nil {
					return common.WrapListError("send alert rules", err)
				}
				if done, err := writeStructuredList(cmd, rules, rules); done {
					return err
				}
				if len(rules) == 0 {
					common.PrintEmptyStateWithHint("send alert rules",
						"Create one with: nylas deliverability send-alerts create --metric bounce_rate --threshold 5 --window-hours 1")
					return nil
				}
				table := newTable(cmd.OutOrStdout(), "ID", "FIRES WHEN", "WINDOW", "MIN VOLUME", "UPDATED")
				for _, r := range rules {
					table.AddRow(r.ID, ruleCondition(r), fmt.Sprintf("%dh", r.WindowHours), formatCount(r.MinVolume), formatUnix(r.UpdatedAt))
				}
				table.Render()
				return nil
			})
		},
	}
}

func newSendAlertsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <rule-id>",
		Short: "Show a send alert rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(func(ctx context.Context, client ports.NylasClient) error {
				rule, err := client.GetSendAlertRule(ctx, args[0])
				if err != nil {
					return common.WrapGetError("send alert rule", err)
				}
				return writeRule(cmd, rule)
			})
		},
	}
}

func newSendAlertsCreateCmd() *cobra.Command {
	var flags alertRuleFlags
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a send alert rule",
		Example: `  nylas deliverability send-alerts create --metric bounce_rate --threshold 5 --window-hours 1 --min-volume 50
  nylas deliverability send-alerts create --metric delivery_rate --threshold 95 --window-hours 24`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			req, err := buildAlertRuleRequest(cmd, &flags)
			if err != nil {
				return err
			}
			return withClient(func(ctx context.Context, client ports.NylasClient) error {
				rule, err := client.CreateSendAlertRule(ctx, req)
				if err != nil {
					return common.WrapCreateError("send alert rule", err)
				}
				if !common.IsStructuredOutput(cmd) {
					common.PrintSuccess("Created send alert rule %s", rule.ID)
				}
				return writeRule(cmd, rule)
			})
		},
	}
	addAlertRuleFlags(cmd, &flags)
	_ = cmd.MarkFlagRequired("metric")
	_ = cmd.MarkFlagRequired("threshold")
	_ = cmd.MarkFlagRequired("window-hours")
	return cmd
}

func newSendAlertsUpdateCmd() *cobra.Command {
	var flags alertRuleFlags
	cmd := &cobra.Command{
		Use:     "update <rule-id>",
		Short:   "Change a send alert rule; flags you leave out keep their value",
		Example: `  nylas deliverability send-alerts update <rule-id> --threshold 3`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := buildAlertRuleRequest(cmd, &flags)
			if err != nil {
				return err
			}
			if *req == (domain.SendAlertRuleRequest{}) {
				return common.NewUserError("nothing to update", "Set at least one of --metric, --threshold, --window-hours, --min-volume")
			}
			return withClient(func(ctx context.Context, client ports.NylasClient) error {
				rule, err := client.UpdateSendAlertRule(ctx, args[0], req)
				if err != nil {
					return common.WrapUpdateError("send alert rule", err)
				}
				if !common.IsStructuredOutput(cmd) {
					common.PrintSuccess("Updated send alert rule %s", rule.ID)
				}
				return writeRule(cmd, rule)
			})
		},
	}
	addAlertRuleFlags(cmd, &flags)
	return cmd
}

func newSendAlertsDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "delete <rule-id>",
		Short:   "Delete a send alert rule",
		Example: `  nylas deliverability send-alerts delete <rule-id> --yes`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return common.NewUserError("deletion requires confirmation", "Re-run with --yes to delete the rule")
			}
			return withClient(func(ctx context.Context, client ports.NylasClient) error {
				if err := client.DeleteSendAlertRule(ctx, args[0]); err != nil {
					return common.WrapDeleteError("send alert rule", err)
				}
				common.PrintSuccess("Deleted send alert rule %s", args[0])
				return nil
			})
		},
	}
	common.AddYesFlag(cmd, &yes)
	return cmd
}

func writeRule(cmd *cobra.Command, r *domain.SendAlertRule) error {
	if done, err := writeStructured(cmd, r); done {
		return err
	}
	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(w, "ID:          %s\n", r.ID)
	_, _ = fmt.Fprintf(w, "Fires when:  %s\n", ruleCondition(*r))
	_, _ = fmt.Fprintf(w, "Window:      %d hour(s)\n", r.WindowHours)
	_, _ = fmt.Fprintf(w, "Min volume:  %d\n", r.MinVolume)
	_, _ = fmt.Fprintf(w, "Created:     %s\n", formatUnix(r.CreatedAt))
	_, _ = fmt.Fprintf(w, "Updated:     %s\n", formatUnix(r.UpdatedAt))
	return nil
}

// ruleCondition reads like "bounce_rate above 5%".
func ruleCondition(r domain.SendAlertRule) string {
	return fmt.Sprintf("%s %s %s", r.Metric, r.Direction, formatThreshold(r.Threshold))
}
