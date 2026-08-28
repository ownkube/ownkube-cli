package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/client"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List deployments (scoped to one cluster or environment)",
		Long: `List deployments. Exactly one of --cluster or --environment is required —
the API does not support listing across all clusters at once.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cluster, _ := cmd.Flags().GetString("cluster")
			env, _ := cmd.Flags().GetString("environment")
			if (cluster == "") == (env == "") {
				return fmt.Errorf("specify exactly one of --cluster or --environment")
			}

			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			deps, err := api.ListDeployments(cmd.Context(), client.ListDeploymentsFilter{
				ClusterID:     cluster,
				EnvironmentID: env,
			})
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), deps)
			}
			if len(deps) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No deployments found.")
				return nil
			}

			rows := [][]string{{"ID", "NAME", "TYPE", "STATUS", "CLUSTER", "HOSTNAME"}}
			for _, d := range deps {
				rows = append(rows, []string{
					d.Id, d.Name, string(d.ResourceType), d.Status, d.ClusterId, d.PublicHostname,
				})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
	c.Flags().String("cluster", "", "Filter by cluster ID")
	c.Flags().String("environment", "", "Filter by environment ID")
	return c
}
