package aws

import (
	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <account-id>",
		Short: "Show details and status for one connected AWS account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			a, err := api.GetAwsAccount(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), a)
			}
			return printAccount(cmd.OutOrStdout(), a)
		},
	}
}
