// Package marketplace exposes the `okctl marketplace` command tree: ready-made
// open-source apps deployed in one step with their database and cache wired in.
package marketplace

import (
	"fmt"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

// New returns the `okctl marketplace` command.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:     "marketplace",
		Aliases: []string{"mp"},
		Short:   "Deploy ready-made open-source apps",
		Long: "Browse and deploy ready-made open-source apps. One deploy creates " +
			"every piece the app needs (app, database, cache) wired together.",
	}
	root.AddCommand(listCmd(), getCmd(), checkCmd(), deployCmd(), deleteCmd())
	return root
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List marketplace apps",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			apps, err := cl.ListMarketplaceApps(cmd.Context())
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), apps)
			}
			if len(apps) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No marketplace apps.")
				return nil
			}
			rows := [][]string{{"SLUG", "NAME", "CATEGORY", "CREATES", "DEPLOYABLE"}}
			for _, a := range apps {
				rows = append(rows, []string{
					a.Slug, a.Name, a.Category,
					strings.Join(a.Provisions, ", "),
					deployable(a.Deployable, a.UnavailableReason),
				})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}

func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <slug>",
		Short: "Show an app, what it creates, and the inputs it needs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			app, err := cl.GetMarketplaceApp(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), app)
			}
			return renderDetail(cmd, app)
		},
	}
}

func renderDetail(cmd *cobra.Command, app *api.MarketplaceAppDetail) error {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "%s (%s)\n", app.Name, app.Slug)
	fmt.Fprintln(w, app.Blurb)
	fmt.Fprintf(w, "Deployable: %s\n", deployable(app.Deployable, app.UnavailableReason))
	if app.SourceUrl != "" {
		fmt.Fprintf(w, "Source:     %s\n", app.SourceUrl)
	}

	fmt.Fprintln(w, "\nCreates:")
	rows := [][]string{{"REF", "KIND", "DETAIL", "IMAGE"}}
	for _, c := range app.Components {
		rows = append(rows, []string{c.Ref, c.Kind, c.Detail, c.Image})
	}
	if err := ux.Print(w, rows); err != nil {
		return err
	}

	if len(app.Inputs) > 0 {
		fmt.Fprintln(w, "\nInputs (pass with --input KEY=VALUE):")
		rows = [][]string{{"KEY", "LABEL", "TYPE", "REQUIRED", "DEFAULT"}}
		for _, in := range app.Inputs {
			req := "no"
			if in.Required {
				req = "yes"
			}
			rows = append(rows, []string{in.Key, in.Label, inputType(in), req, inputDefault(in)})
		}
		if err := ux.Print(w, rows); err != nil {
			return err
		}
	}

	if len(app.Connections) > 0 {
		fmt.Fprintln(w, "\nSet for you:")
		rows = [][]string{{"APP", "VARIABLE", "FROM"}}
		for _, c := range app.Connections {
			rows = append(rows, []string{c.AppRef, c.EnvName, c.Label})
		}
		if err := ux.Print(w, rows); err != nil {
			return err
		}
	}
	return nil
}

func deployable(ok bool, reason string) string {
	if ok {
		return "yes"
	}
	if reason == "" {
		return "no"
	}
	return "no: " + reason
}

func inputType(in api.MarketplaceInput) string {
	if in.Options == nil || len(*in.Options) == 0 {
		return string(in.Type)
	}
	vals := make([]string, 0, len(*in.Options))
	for _, o := range *in.Options {
		vals = append(vals, o.Value)
	}
	return "one of " + strings.Join(vals, "|")
}

// inputDefault renders a string|number|bool default as plain text. Secret
// inputs never show one.
func inputDefault(in api.MarketplaceInput) string {
	if in.Default == nil || (in.Secret != nil && *in.Secret) {
		return ""
	}
	raw, err := in.Default.MarshalJSON()
	if err != nil || string(raw) == "null" {
		return ""
	}
	return strings.Trim(string(raw), `"`)
}
