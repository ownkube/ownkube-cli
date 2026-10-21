package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/client"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List deployments",
		Long: `List deployments in one cluster (--cluster), environment (--environment)
or project (--project); environments and projects take a name or an id. With
none of them, a linked directory lists its environment, and an unlinked one
lists every project's deployments.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cluster, _ := cmd.Flags().GetString("cluster")
			envRef, _ := cmd.Flags().GetString("environment")
			projectRef, _ := cmd.Flags().GetString("project")
			if cluster != "" && (envRef != "" || projectRef != "") {
				return fmt.Errorf("--cluster can't be combined with --environment or --project")
			}

			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			var deps []api.Deployment
			if cluster != "" {
				deps, err = cl.ListDeployments(ctx, client.ListDeploymentsFilter{ClusterID: cluster})
			} else {
				var scope ux.Scope
				if scope, err = ux.ResolveScope(ctx, cl, projectRef, envRef); err != nil {
					return err
				}
				if scope == (ux.Scope{}) {
					scope = linkedScope()
				}
				if scope == (ux.Scope{}) {
					if b, ok, _ := ux.Linked("."); ok && b.ClusterID != "" {
						deps, err = cl.ListDeployments(ctx, client.ListDeploymentsFilter{ClusterID: b.ClusterID})
					} else {
						deps, err = ux.ListAllDeployments(ctx, cl)
					}
				} else {
					deps, err = ux.ListScopedDeployments(ctx, cl, scope)
				}
			}
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
	c.Flags().StringP("environment", "e", "", "Filter by environment (name or id)")
	c.Flags().StringP("project", "p", "", "Filter by project (name or id)")
	return c
}

// linkedScope is this directory's linked environment, or an empty Scope.
func linkedScope() ux.Scope {
	if b, ok, _ := ux.Linked("."); ok && b.EnvironmentID != "" {
		return ux.Scope{EnvironmentID: b.EnvironmentID}
	}
	return ux.Scope{}
}
