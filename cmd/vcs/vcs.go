// Package vcs exposes the `okctl vcs` command tree: the GitHub and GitLab
// accounts connected to the organization, and the repositories and branches
// they can see. Connecting an account happens in the dashboard; `connections
// --open` opens that page. Use the results with `okctl deploy source`.
package vcs

import "github.com/spf13/cobra"

// New returns the `okctl vcs` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "vcs",
		Short: "List connected GitHub/GitLab accounts, repositories and branches",
	}
	root.AddCommand(connectionsCmd(), reposCmd(), branchesCmd())
	return root
}
