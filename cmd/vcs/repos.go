package vcs

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func reposCmd() *cobra.Command {
	var connection string

	cmd := &cobra.Command{
		Use:   "repos",
		Short: "List repositories the connected accounts can see",
		Long: "List repositories visible to the organization's connected GitHub and " +
			"GitLab accounts, or to one of them with --connection. A repository " +
			"missing here needs access granted in the dashboard (see 'okctl vcs " +
			"connections --open').",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			repos, err := cl.ListVcsRepositories(cmd.Context(), connection)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), repos)
			}
			if len(repos) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No repositories found. See 'okctl vcs connections' to connect an account or grant access.")
				return nil
			}
			rows := [][]string{{"REPOSITORY", "DEFAULT BRANCH", "VISIBILITY", "ACCOUNT", "CONNECTION"}}
			for _, r := range repos {
				visibility := "public"
				if r.Private {
					visibility = "private"
				}
				rows = append(rows, []string{r.FullName, r.DefaultBranch, visibility, r.Account, r.ConnectionId})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
	cmd.Flags().StringVar(&connection, "connection", "", "Only list this connection's repositories (see 'okctl vcs connections')")
	return cmd
}
