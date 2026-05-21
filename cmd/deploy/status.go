package deploy

import (
	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <deployment-id>",
		Short: "Show live status (sync, health, gateway address) for a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			s, err := api.GetDeploymentStatus(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), s)
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"Status", s.Status},
				{"Sync", s.Sync},
				{"Health", s.Health},
				{"Gateway Address", ux.Deref(s.GatewayAddress)},
			})
		},
	}
}
