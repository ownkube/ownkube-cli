package aws

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func resyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resync <account-id>",
		Short: "Re-verify a connected account and refresh its status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			if err := api.ResyncAwsAccount(cmd.Context(), args[0]); err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), map[string]bool{"success": true})
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Resync started.")
			return nil
		},
	}
}
