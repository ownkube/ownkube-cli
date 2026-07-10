package aws

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/prompt"
	"github.com/spf13/cobra"
)

func deleteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "delete <account-id>",
		Aliases: []string{"disconnect", "rm"},
		Short:   "Disconnect an AWS account from Ownkube",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			yes, _ := cmd.Flags().GetBool("yes")
			if !yes && !ux.IsStructured() {
				ok, err := prompt.Confirm(fmt.Sprintf("Disconnect AWS account %s?", args[0]))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}

			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			if err := api.DeleteAwsAccount(cmd.Context(), args[0]); err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), map[string]bool{"success": true})
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Disconnected.")
			return nil
		},
	}
	c.Flags().BoolP("yes", "y", false, "Skip the confirmation prompt")
	return c
}
