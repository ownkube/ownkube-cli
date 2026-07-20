// Package clusters exposes the `okctl clusters` command tree.
package clusters

import "github.com/spf13/cobra"

// New returns the `okctl clusters` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "clusters",
		Short: "Manage clusters",
	}
	root.AddCommand(listCmd(), getCmd(), createCmd(), deleteCmd(), cancelCmd(), statusCmd())
	return root
}
