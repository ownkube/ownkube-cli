package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func moveCmd() *cobra.Command {
	var project string

	cmd := &cobra.Command{
		Use:   "move <deployment-id>",
		Short: "Re-file a deployment under a different project",
		Long: "Move a deployment to another project (--project). This is a " +
			"control-plane-only change: it cuts no revision, doesn't re-sync, and " +
			"keeps the environment, address, and running workloads exactly where they " +
			"are. A marketplace app and its managed database/cache move together.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				return fmt.Errorf("--project is required")
			}

			client, err := ux.RequireClient()
			if err != nil {
				return err
			}

			result, err := client.MoveToProject(cmd.Context(), args[0], project)
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), result)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Moved %g deployment(s) to project %s.\n", result.Moved, result.ProjectId)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Target project ID (required)")
	return cmd
}
