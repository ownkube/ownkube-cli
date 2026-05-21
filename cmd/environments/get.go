package environments

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <environment-id>",
		Short: "Get environment details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			e, err := api.GetEnvironment(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), e)
			}
			count := ""
			if e.DeploymentCount != nil {
				count = fmt.Sprintf("%g", *e.DeploymentCount)
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"ID", e.Id},
				{"Name", e.Name},
				{"Slug", e.Slug},
				{"Description", ux.Deref(e.Description)},
				{"Color", ux.Deref(e.Color)},
				{"Deployments", count},
			})
		},
	}
}
