package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func resyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resync <deployment-id>",
		Short: "Re-apply a deployment's current settings to recover a stalled rollout",
		Long: "Re-apply the deployment's current settings with no config change and " +
			"no new revision. Use it to recover a rollout that stalled on a " +
			"transient error (see the message in `okctl deploy status`), including " +
			"databases and caches. Not valid for functions.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := cl.Resync(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Resync started for %s. Check progress with `okctl deploy status %s`.\n", args[0], args[0])
			return nil
		},
	}
}
