package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/prompt"
	"github.com/spf13/cobra"
)

func resetPasswordCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "reset-password <deployment-id>",
		Short: "Reset a database deployment's password",
		Long: "Rotate the password for a database deployment and print the new " +
			"credential. The current password stops working immediately — update " +
			"any connected applications.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			if !yes {
				ok, err := prompt.Confirm(fmt.Sprintf(
					"Reset the password for database %s? The current one stops working.", args[0]))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}

			result, err := api.ResetDatabasePassword(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), result)
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"Password", result.Password},
			})
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")
	return cmd
}
