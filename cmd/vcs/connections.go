package vcs

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func connectionsCmd() *cobra.Command {
	var open bool

	cmd := &cobra.Command{
		Use:   "connections",
		Short: "List the GitHub and GitLab accounts connected to the organization",
		Long: "List the GitHub and GitLab accounts connected to the organization. " +
			"Connecting an account, or granting it access to more repositories, " +
			"happens in the dashboard; --open opens that page.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := cl.ListVcsConnections(cmd.Context())
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()

			if open {
				if err := ux.OpenBrowser(res.ConnectUrl); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Could not open a browser (%v); visit %s\n", err, res.ConnectUrl)
				}
			}
			if ux.IsStructured() {
				return ux.Print(out, res)
			}

			if len(res.Connections) == 0 {
				fmt.Fprintf(out, "No GitHub or GitLab account is connected. Connect one at:\n  %s\n", res.ConnectUrl)
				return nil
			}
			rows := [][]string{{"ID", "PROVIDER", "ACCOUNT", "TYPE", "STATUS"}}
			for _, c := range res.Connections {
				rows = append(rows, []string{c.Id, string(c.Provider), c.Account, c.AccountType, c.Status})
			}
			if err := ux.Print(out, rows); err != nil {
				return err
			}
			if !open {
				fmt.Fprintf(out, "\nConnect another account or grant access to more repositories: %s\n", res.ConnectUrl)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&open, "open", false, "Open the dashboard page for connecting an account")
	return cmd
}
