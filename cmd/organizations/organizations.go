// Package organizations exposes the `okctl organizations` command tree.
package organizations

import "github.com/spf13/cobra"

// New returns the `okctl organizations` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:     "organizations",
		Aliases: []string{"orgs", "org"},
		Short:   "Manage organizations",
	}
	root.AddCommand(listCmd())
	return root
}
