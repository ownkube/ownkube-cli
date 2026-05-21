package environments

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			envs, err := api.ListEnvironments(cmd.Context())
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), envs)
			}
			if len(envs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No environments found.")
				return nil
			}

			rows := [][]string{{"ID", "NAME", "SLUG", "DEPLOYMENTS"}}
			for _, e := range envs {
				count := ""
				if e.DeploymentCount != nil {
					count = fmt.Sprintf("%g", *e.DeploymentCount)
				}
				rows = append(rows, []string{e.Id, e.Name, e.Slug, count})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}
