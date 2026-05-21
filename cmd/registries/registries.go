// Package registries exposes the `okctl registries` command tree.
package registries

import "github.com/spf13/cobra"

// New returns the `okctl registries` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "registries",
		Short: "Manage container registries",
	}
	root.AddCommand(listCmd(), getCmd())
	return root
}
