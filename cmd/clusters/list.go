package clusters

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List clusters",
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			clusters, err := api.ListClusters(cmd.Context())
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), clusters)
			}
			if len(clusters) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No clusters found.")
				return nil
			}

			rows := [][]string{{"ID", "NAME", "PROVIDER", "TYPE", "STATUS", "REGION"}}
			for _, c := range clusters {
				rows = append(rows, []string{
					c.Id, c.Name, c.Provider, c.ClusterType, c.Status, ux.Deref(c.Region),
				})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}
