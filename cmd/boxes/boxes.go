// Package boxes exposes the `okctl boxes` command tree — the Ownkube Compute
// box catalog (reserved sizes for web apps, databases and caches).
package boxes

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

// New returns the `okctl boxes` command.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "boxes",
		Short: "List Ownkube Compute box sizes and prices",
	}
	root.AddCommand(listCmd())
	return root
}

func listCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List box sizes for web apps, databases and caches",
		Long: "List the reserved box sizes on Ownkube Compute with their monthly " +
			"price. A database or cache needs a box: pass its ID as skuId (with " +
			"billingMode \"reserved\") when creating one.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, _ := cmd.Flags().GetString("kind")
			switch kind {
			case "", "web", "database", "cache":
			default:
				return fmt.Errorf("invalid --kind %q: must be web, database, or cache", kind)
			}

			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			cat, err := cl.ListBoxes(cmd.Context())
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), cat)
			}

			rows := [][]string{{"KIND", "ID", "LABEL", "VCPU", "MEMORY", "CPU", "MONTHLY"}}
			add := func(k string, boxes []api.ComputeBox) {
				if kind != "" && kind != k {
					return
				}
				for _, b := range boxes {
					rows = append(rows, []string{
						k, b.Id, b.Label,
						fmt.Sprintf("%g", b.Vcpu),
						fmt.Sprintf("%g GiB", b.MemoryGiB),
						string(b.CpuClass),
						fmt.Sprintf("$%.2f", b.MonthlyUsd),
					})
				}
			}
			add("web", cat.Web)
			add("database", cat.Database)
			add("cache", cat.Cache)
			if err := ux.Print(cmd.OutOrStdout(), rows); err != nil {
				return err
			}
			if kind == "" || kind == "database" {
				fmt.Fprintf(cmd.OutOrStdout(), "\nDatabase storage: $%.2f per GiB per month.\n", cat.DatabaseStorageGibMonthUsd)
			}
			return nil
		},
	}
	c.Flags().String("kind", "", "Only show one kind: web, database, or cache")
	return c
}
