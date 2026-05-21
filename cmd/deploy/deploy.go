// Package deploy exposes the `okctl deploy` command tree. Each subcommand is
// a thin wrapper over a single client method.
package deploy

import "github.com/spf13/cobra"

// New returns the `okctl deploy` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "deploy",
		Short: "Manage deployments",
	}
	root.AddCommand(
		listCmd(),
		getCmd(),
		statusCmd(),
		logsCmd(),
		revisionsCmd(),
		connectionCmd(),
	)
	return root
}
