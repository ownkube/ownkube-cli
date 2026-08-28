// Package alerts exposes the `okctl alerts` command tree — per-deployment
// alert rules (list/create/update/delete) and the recent firing history.
package alerts

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/client"
	"github.com/spf13/cobra"
)

// New returns the `okctl alerts` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "alerts",
		Short: "Manage workload alert rules and view firings",
	}
	root.AddCommand(listCmd(), createCmd(), updateCmd(), deleteCmd(), firingsCmd())
	return root
}

func renderRule(cmd *cobra.Command, r *api.AlertRule) error {
	return ux.Print(cmd.OutOrStdout(), [][]string{
		{"FIELD", "VALUE"},
		{"ID", r.Id},
		{"Name", r.Name},
		{"Metric", string(r.Metric)},
		{"Comparator", string(r.Comparator)},
		{"Threshold", r.Threshold},
		{"Window (s)", fmt.Sprintf("%g", r.WindowSeconds)},
		{"For (s)", fmt.Sprintf("%g", r.ForDurationSeconds)},
		{"Severity", string(r.Severity)},
		{"Enabled", fmt.Sprintf("%t", r.Enabled)},
	})
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <deployment-id>",
		Short: "List alert rules for a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			rules, err := cl.ListAlertRules(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), rules)
			}
			if len(rules) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No alert rules.")
				return nil
			}
			rows := [][]string{{"ID", "NAME", "METRIC", "COMP", "THRESHOLD", "SEVERITY", "ENABLED"}}
			for i := range rules {
				r := &rules[i]
				rows = append(rows, []string{
					r.Id, r.Name, string(r.Metric), string(r.Comparator),
					r.Threshold, string(r.Severity), fmt.Sprintf("%t", r.Enabled),
				})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}

func firingsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "firings",
		Short: "Show recent alert firings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			deploymentID, _ := cmd.Flags().GetString("deployment")
			limit, _ := cmd.Flags().GetInt("limit")
			firings, err := cl.ListAlertFirings(cmd.Context(), client.AlertFiringsFilter{
				DeploymentID: deploymentID,
				Limit:        limit,
			})
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), firings)
			}
			if len(firings) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No firings.")
				return nil
			}
			rows := [][]string{{"ID", "RULE", "METRIC", "SEVERITY", "STATUS", "VALUE"}}
			for i := range firings {
				f := &firings[i]
				rows = append(rows, []string{
					f.Id, f.RuleName, string(f.Metric), string(f.Severity), string(f.Status), f.Value,
				})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
	c.Flags().String("deployment", "", "Filter firings by deployment ID")
	c.Flags().Int("limit", 0, "Maximum number of firings to return")
	return c
}
