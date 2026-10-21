// Package functions exposes the `okctl functions` command tree. Functions are
// cluster-less Ownkube Compute deployments — they attach to a cloud account +
// region rather than a cluster, so they list separately from cluster-hosted
// workloads under `okctl deploy`.
package functions

import "github.com/spf13/cobra"

// New returns the `okctl functions` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:     "functions",
		Aliases: []string{"function", "fn"},
		Short:   "Manage Compute functions",
	}
	root.AddCommand(listCmd(), sourceCmd(), deployCmd())
	return root
}
