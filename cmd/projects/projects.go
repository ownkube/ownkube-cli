// Package projects exposes the `okctl projects` command tree.
package projects

import "github.com/spf13/cobra"

// New returns the `okctl projects` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:     "projects",
		Aliases: []string{"project", "proj"},
		Short:   "Manage projects",
	}
	root.AddCommand(listCmd(), getCmd(), createCmd(), updateCmd(), deleteCmd())
	return root
}
