package clusters

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/prompt"
	"github.com/spf13/cobra"
)

func deleteCmd() *cobra.Command {
	var (
		yes   bool
		force bool
	)

	cmd := &cobra.Command{
		Use:   "delete <cluster-id>",
		Short: "Destroy a cluster",
		Long: "Tear down a cluster and its infrastructure. Blocked when the cluster " +
			"still has active deployments unless --force is passed, which tears the " +
			"deployments down as part of the destroy.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			if !yes {
				msg := fmt.Sprintf("Destroy cluster %s?", args[0])
				if force {
					msg = fmt.Sprintf("Destroy cluster %s and every deployment on it?", args[0])
				}
				ok, err := prompt.Confirm(msg)
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}

			res, err := api.DestroyCluster(cmd.Context(), args[0], force)
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

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")
	cmd.Flags().BoolVar(&force, "force", false, "Destroy even when the cluster has active deployments")
	return cmd
}
