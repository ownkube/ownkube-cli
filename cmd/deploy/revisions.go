package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func revisionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revisions <deployment-id>",
		Short: "List revisions for a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			revs, err := api.ListRevisions(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), revs)
			}
			if len(revs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No revisions found.")
				return nil
			}

			rows := [][]string{{"ID", "STATUS", "IMAGE", "TAG", "SOURCE", "NOTE"}}
			for _, r := range revs {
				rows = append(rows, []string{r.Id, r.Status, r.Image, r.Tag, ux.Deref(r.Source), ux.Deref(r.Note)})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}
