// Package environments exposes the `okctl environments` command tree.
package environments

import "github.com/spf13/cobra"

// New returns the `okctl environments` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:     "environments",
		Aliases: []string{"envs", "env"},
		Short:   "Manage environments",
	}
	root.AddCommand(listCmd(), getCmd())
	return root
}
