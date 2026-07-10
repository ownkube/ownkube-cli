package deploy

import (
	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <deployment-id>",
		Short: "Get deployment details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			d, err := api.GetDeployment(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), d)
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"ID", d.Id},
				{"Name", d.Name},
				{"Type", string(d.ResourceType)},
				{"Status", d.Status},
				{"Status Message", ux.Deref(d.StatusMessage)},
				{"Cluster", ux.Deref(d.ClusterId)},
				{"Environment", ux.Deref(d.EnvironmentId)},
				{"Public Hostname", ux.Deref(d.PublicHostname)},
				{"Chart Version", ux.Deref(d.ChartVersion)},
			})
		},
	}
}
