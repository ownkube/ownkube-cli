package aws

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func reconnectCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "reconnect <account-id>",
		Short: "Mint a fresh stack link to re-establish access for an account",
		Long: `Re-launch onboarding for an existing account.

Returns a fresh external ID and a browser quick-create URL. Use this after a
failed connection or when access has drifted. Pass --open to launch the URL in
your browser.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			open, _ := cmd.Flags().GetBool("open")

			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			res, err := api.ReconnectAwsAccount(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "External ID: %s\n", res.ExternalId)
			fmt.Fprintf(out, "Open this URL and click Create stack:\n  %s\n", res.QuickCreateUrl)
			if open {
				if err := ux.OpenBrowser(res.QuickCreateUrl); err != nil {
					fmt.Fprintf(out, "\nCould not open browser automatically: %v\n", err)
				}
			}
			return nil
		},
	}
	c.Flags().Bool("open", false, "Open the quick-create URL in your browser")
	return c
}
