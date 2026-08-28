// Package billing exposes the `okctl billing` command tree — the prepaid
// wallet, signup credit, spend controls, and browser handoffs for
// subscriptions, top-ups, and the billing portal.
package billing

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

// New returns the `okctl billing` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "billing",
		Short: "Manage the Ownkube Compute wallet, credit, and spend controls",
	}
	root.AddCommand(
		walletCmd(),
		creditCmd(),
		spendControlsCmd(),
		subscribeCmd(),
		topUpCmd(),
		portalCmd(),
	)
	return root
}

func usd(v float32) string { return fmt.Sprintf("$%.2f", v) }

// openCheckoutURL prints a browser handoff URL and opens it unless --no-browser
// is set. Structured callers receive the raw payload instead and never trigger
// a browser launch.
func openCheckoutURL(cmd *cobra.Command, url string) error {
	out := cmd.OutOrStdout()
	noBrowser, _ := cmd.Flags().GetBool("no-browser")
	fmt.Fprintf(out, "Open this URL to continue:\n%s\n", url)
	if noBrowser {
		return nil
	}
	if err := ux.OpenBrowser(url); err != nil {
		fmt.Fprintf(out, "Could not open browser automatically: %v\n", err)
	}
	return nil
}
