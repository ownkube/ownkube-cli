// Package open implements `okctl open`: open an app (or this directory's
// linked app) in the Ownkube dashboard.
package open

import (
	"fmt"
	"net/url"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

// New returns the `okctl open` command.
func New() *cobra.Command {
	var (
		serviceFlag string
		printOnly   bool
	)
	cmd := &cobra.Command{
		Use:   "open [app]",
		Short: "Open an app in the Ownkube dashboard",
		Long: "Open the dashboard page for an app: the name or id argument, --service, " +
			"or this directory's linked app. With no app and no link, opens the " +
			"dashboard home. --print prints the URL instead.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref := serviceFlag
			if len(args) == 1 {
				if ref != "" {
					return fmt.Errorf("pass the app as an argument or with --service, not both")
				}
				ref = args[0]
			}
			origin, err := dashboardOrigin()
			if err != nil {
				return err
			}

			binding, linked, _ := ux.Linked(".")
			target := "/dashboard"
			if ref != "" || (linked && binding.DeploymentID != "") {
				cl, err := ux.RequireClient()
				if err != nil {
					return err
				}
				id, err := ux.ResolveApp(cmd.Context(), cl, ref, ux.Scope{}, ".")
				if err != nil {
					return err
				}
				target = "/dashboard?deployment=" + url.QueryEscape(id)
			}

			link := origin + target
			org := ux.Organization()
			if org == "" && linked {
				org = binding.OrganizationID
			}
			if org != "" {
				// Land in the app's organization, whichever one the browser
				// session last had open.
				link = origin + "/api/switch-org?org=" + url.QueryEscape(org) + "&to=" + url.QueryEscape(target)
			}

			out := cmd.OutOrStdout()
			if printOnly || ux.IsStructured() {
				fmt.Fprintln(out, link)
				return nil
			}
			if err := ux.OpenBrowser(link); err != nil {
				fmt.Fprintf(out, "Could not open a browser (%v). Visit:\n", err)
			}
			fmt.Fprintln(out, link)
			return nil
		},
	}
	cmd.Flags().StringVarP(&serviceFlag, "service", "s", "", "App name or id (default: this directory's link)")
	cmd.Flags().BoolVar(&printOnly, "print", false, "Print the URL instead of opening it")
	return cmd
}

// dashboardOrigin is the scheme and host of the configured API URL, which
// also serves the dashboard.
func dashboardOrigin() (string, error) {
	u, err := url.Parse(ux.APIURL())
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("can't read a dashboard address from the API URL %q", ux.APIURL())
	}
	return u.Scheme + "://" + u.Host, nil
}
