package vcs

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func branchesCmd() *cobra.Command {
	var connection string

	cmd := &cobra.Command{
		Use:   "branches <owner/repo>",
		Short: "List a repository's branches, most recently committed first",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := cl.ListVcsBranches(cmd.Context(), args[0], connection)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			if len(res.Branches) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No branches found.")
				return nil
			}
			rows := [][]string{{"BRANCH", "LAST COMMIT"}}
			for _, b := range res.Branches {
				rows = append(rows, []string{b.Name, b.LastCommitAt})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
	cmd.Flags().StringVar(&connection, "connection", "", "Connection to read through (default: any that can see the repository)")
	return cmd
}
