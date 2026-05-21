package organizations

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List organizations the current user belongs to",
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			orgs, err := api.ListOrganizations(cmd.Context())
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), orgs)
			}
			if len(orgs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No organizations found.")
				return nil
			}

			rows := [][]string{{"ID", "NAME", "SLUG", "ROLE"}}
			for _, o := range orgs {
				rows = append(rows, []string{o.Id, o.Name, ux.Deref(o.Slug), o.Role})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}
