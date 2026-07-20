package clusters

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <cluster-id>",
		Short: "Get cluster details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			c, err := api.GetCluster(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), c)
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"ID", c.Id},
				{"Name", c.Name},
				{"Provider", c.Provider},
				{"Type", c.ClusterType},
				{"Region", ux.Deref(c.Region)},
				{"Status", c.Status},
				{"Status Message", ux.Deref(c.StatusMessage)},
				{"Kubernetes Version", ux.Deref(c.KubernetesVersion)},
				{"Platform Version", ux.Deref(c.BootstrapChartVersion)},
				{"Latest Platform Version", ux.Deref(c.LatestBootstrapChartVersion)},
				{"Active Deployments", fmt32(c.ActiveDeploymentCount)},
				{"Upgrade Available", fmtBool(c.UpgradeAvailable)},
			})
		},
	}
}

func fmt32(v *float32) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%g", *v)
}

func fmtBool(v *bool) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%t", *v)
}
