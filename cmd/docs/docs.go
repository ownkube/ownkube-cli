// Package docs implements `okctl docs`: open the Ownkube documentation.
package docs

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

// URL is the documentation home.
const URL = "https://ownkube.io/docs"

// New returns the `okctl docs` command.
func New() *cobra.Command {
	var printOnly bool
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Open the Ownkube documentation in your browser",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if printOnly || ux.IsStructured() {
				fmt.Fprintln(out, URL)
				return nil
			}
			if err := ux.OpenBrowser(URL); err != nil {
				fmt.Fprintf(out, "Could not open a browser (%v). Visit:\n", err)
			}
			fmt.Fprintln(out, URL)
			return nil
		},
	}
	cmd.Flags().BoolVar(&printOnly, "print", false, "Print the URL instead of opening it")
	return cmd
}
