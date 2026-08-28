// Package usage exposes the `okctl usage` command tree — live burn rate,
// month-to-date draw, month projection, and historical points.
package usage

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

// New returns the `okctl usage` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "usage",
		Short: "Show Ownkube Compute usage and cost",
	}
	root.AddCommand(currentCmd(), monthToDateCmd(), projectedCmd(), historyCmd())
	return root
}

func usd(v float32) string { return fmt.Sprintf("$%.2f", v) }

func currentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show the live hourly burn rate",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := api.GetUsageCurrent(cmd.Context())
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"Hourly Cost", usd(res.HourlyCostUsd)},
				{"vCPU", fmt.Sprintf("%g", res.Vcpu)},
				{"Memory (GiB)", fmt.Sprintf("%g", res.MemoryGiB)},
				{"Active Clusters", fmt.Sprintf("%g", res.ActiveClusters)},
				{"Starter Clusters", fmt.Sprintf("%g", res.StarterClusters)},
			})
		},
	}
}

func monthToDateCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "month-to-date",
		Aliases: []string{"mtd"},
		Short:   "Show month-to-date usage drawn from the wallet",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := api.GetUsageMonthToDate(cmd.Context())
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"Tier", res.Tier},
				{"Cost", usd(res.CostUsd)},
				{"vCPU-hours", fmt.Sprintf("%g", res.VcpuHours)},
				{"Memory GiB-hours", fmt.Sprintf("%g", res.MemoryGibHours)},
				{"Egress (GiB)", fmt.Sprintf("%g", res.EgressGib)},
				{"Starter Clusters", fmt.Sprintf("%g", res.StarterClusters)},
			})
		},
	}
}

func projectedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "projected",
		Short: "Show the straight-line projected cost for the month",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := api.GetUsageProjected(cmd.Context())
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"Tier", res.Tier},
				{"Month-to-date Cost", usd(res.MtdCostUsd)},
				{"Projected Cost", usd(res.ProjectedCostUsd)},
				{"Starter Clusters", fmt.Sprintf("%g", res.StarterClusters)},
			})
		},
	}
}

func historyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "Show historical usage points (use -o json for full detail)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := api.GetUsageHistory(cmd.Context())
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d usage points. Use --output json for full detail.\n", len(res.Points))
			return nil
		},
	}
}
