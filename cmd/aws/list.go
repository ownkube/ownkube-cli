package aws

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "status"},
		Short:   "List connected AWS accounts and their status",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			accounts, err := api.ListAwsAccounts(cmd.Context())
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), accounts)
			}
			if len(accounts) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No AWS accounts connected. Run 'okctl aws connect' to add one.")
				return nil
			}
			return printAccounts(cmd.OutOrStdout(), accounts)
		},
	}
}
