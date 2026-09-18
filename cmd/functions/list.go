package functions

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Compute functions in the current organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			fns, err := api.ListFunctions(cmd.Context())
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), fns)
			}
			if len(fns) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No functions found.")
				return nil
			}

			rows := [][]string{{"ID", "NAME", "STATUS", "ENVIRONMENT", "HOSTNAME"}}
			for _, f := range fns {
				rows = append(rows, []string{
					f.Id, f.Name, f.Status, f.EnvironmentId, f.PublicHostname,
				})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}
