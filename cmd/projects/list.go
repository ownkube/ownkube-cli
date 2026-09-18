package projects

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List projects in the current organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			projects, err := api.ListProjects(cmd.Context())
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), projects)
			}
			if len(projects) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No projects found.")
				return nil
			}

			rows := [][]string{{"ID", "NAME", "SLUG", "DEFAULT", "DEPLOYMENTS", "ENVIRONMENTS"}}
			for _, p := range projects {
				isDefault := ""
				if p.IsDefault != nil {
					isDefault = fmt.Sprintf("%t", *p.IsDefault)
				}
				rows = append(rows, []string{
					p.Id, p.Name, p.Slug, isDefault,
					floatCount(p.DeploymentCount), floatCount(p.EnvironmentCount),
				})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}
