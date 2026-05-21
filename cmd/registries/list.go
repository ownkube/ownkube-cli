package registries

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List container registries",
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			regs, err := api.ListRegistries(cmd.Context())
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), regs)
			}
			if len(regs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No registries found.")
				return nil
			}

			rows := [][]string{{"ID", "NAME", "PROVIDER", "REGION", "STATUS"}}
			for _, r := range regs {
				rows = append(rows, []string{
					r.Id, r.Name, r.Provider, ux.Deref(r.Region), ux.Deref(r.Status),
				})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}
