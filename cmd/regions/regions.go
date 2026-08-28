// Package regions exposes the `okctl regions` command tree — the Ownkube
// Compute region catalog used when deploying without a cluster.
package regions

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

// New returns the `okctl regions` command.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "regions",
		Short: "List Ownkube Compute regions you can deploy to",
	}
	root.AddCommand(listCmd())
	return root
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available deploy regions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			res, err := api.ListRegions(cmd.Context())
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}

			if !res.Available {
				fmt.Fprintln(cmd.OutOrStdout(), "Ownkube Compute is not currently open for new deployments.")
				return nil
			}
			if len(res.Regions) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No regions available.")
				return nil
			}
			rows := [][]string{{"ID", "NAME"}}
			for _, r := range res.Regions {
				rows = append(rows, []string{r.Id, r.Name})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}
