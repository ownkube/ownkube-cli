package clusters

import (
	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <cluster-id>",
		Short: "Show a cluster's live provisioning status",
		Long: "Lightweight poll target during create/destroy: current status and " +
			"message without the full cluster row.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			s, err := api.ClusterStatus(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), s)
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"ID", s.Id},
				{"Status", s.Status},
				{"Status Message", ux.Deref(s.StatusMessage)},
			})
		},
	}
}
