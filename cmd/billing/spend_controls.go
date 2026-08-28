package billing

import (
	"fmt"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

func spendControlsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "spend-controls",
		Aliases: []string{"budget"},
		Short:   "View and set spend controls (budget cap + alerts)",
	}
	c.AddCommand(spendControlsGetCmd(), spendControlsSetCmd())
	return c
}

func renderSpendControls(cmd *cobra.Command, sc *api.SpendControls) error {
	thresholds := make([]string, 0, len(sc.AlertThresholdsPct))
	for _, t := range sc.AlertThresholdsPct {
		thresholds = append(thresholds, fmt.Sprintf("%g%%", t))
	}
	return ux.Print(cmd.OutOrStdout(), [][]string{
		{"FIELD", "VALUE"},
		{"Monthly Budget", usd(sc.MonthlyBudgetUsd)},
		{"Budget Action", string(sc.BudgetAction)},
		{"Alerts Enabled", fmt.Sprintf("%t", sc.AlertsEnabled)},
		{"Alert Thresholds", strings.Join(thresholds, ", ")},
	})
}

func spendControlsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Show the current spend controls",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := api.GetSpendControls(cmd.Context())
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			if !res.Enabled {
				fmt.Fprintln(cmd.OutOrStdout(), "Ownkube Compute is not enabled for this organization.")
				return nil
			}
			return renderSpendControls(cmd, &res.Controls)
		},
	}
}

func spendControlsSetCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "set",
		Short: "Update spend controls (only the flags you pass change)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}

			var body api.UpdateSpendControlsBody
			flags := cmd.Flags()
			if flags.Changed("budget") {
				v, _ := flags.GetFloat32("budget")
				body.MonthlyBudgetUsd = &v
			}
			if flags.Changed("budget-action") {
				v, _ := flags.GetString("budget-action")
				action := api.UpdateSpendControlsBodyBudgetAction(v)
				if !action.Valid() {
					return fmt.Errorf("invalid --budget-action %q: use notify or pause", v)
				}
				body.BudgetAction = &action
			}
			if flags.Changed("alerts") {
				v, _ := flags.GetBool("alerts")
				body.AlertsEnabled = &v
			}
			if flags.Changed("alert-thresholds") {
				v, _ := flags.GetIntSlice("alert-thresholds")
				body.AlertThresholdsPct = &v
			}
			if body.MonthlyBudgetUsd == nil && body.BudgetAction == nil &&
				body.AlertsEnabled == nil && body.AlertThresholdsPct == nil {
				return fmt.Errorf("pass at least one of --budget, --budget-action, --alerts, --alert-thresholds")
			}

			res, err := cl.UpdateSpendControls(cmd.Context(), body)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			return renderSpendControls(cmd, res)
		},
	}
	c.Flags().Float32("budget", 0, "Monthly spend cap in USD")
	c.Flags().String("budget-action", "", "What happens at the cap: notify or pause")
	c.Flags().Bool("alerts", false, "Enable or disable spend alerts")
	c.Flags().IntSlice("alert-thresholds", nil, "Percent-of-budget thresholds that fire an alert (e.g. 50,80,100)")
	return c
}
