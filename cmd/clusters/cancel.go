package clusters

import (
	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func cancelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <cluster-id>",
		Short: "Cancel an in-flight cluster creation",
		Long: "Cancel a cluster that is still being created. The teardown of any " +
			"partially provisioned resources is handled for you.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			res, err := api.CancelCluster(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"Cluster ID", res.ClusterId},
				{"Status", res.Status},
			})
		},
	}
}
