// Package aws exposes the `okctl aws` command tree for cross-account
// onboarding: connect a customer AWS account, poll its status, verify,
// reconnect, resync, and disconnect. Every subcommand is a thin wrapper over a
// single client method; the onboarding logic lives server-side.
package aws

import "github.com/spf13/cobra"

// New returns the `okctl aws` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "aws",
		Short: "Connect and manage AWS accounts",
		Long: `Connect an AWS account to Ownkube and manage its access.

Connecting deploys a small CloudFormation stack in your account that grants
Ownkube a scoped cross-account role. Ownkube never receives your AWS keys; the
stack phones home once it is created and access is verified automatically.`,
	}
	root.AddCommand(
		connectCmd(),
		listCmd(),
		getCmd(),
		verifyCmd(),
		reconnectCmd(),
		resyncCmd(),
		deleteCmd(),
	)
	return root
}
