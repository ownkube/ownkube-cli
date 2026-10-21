package vars

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	var kv bool
	cmd := &cobra.Command{
		Use:     "list [app]",
		Aliases: []string{"ls"},
		Short:   "List an app's variables",
		Long: "List the app's variables and the ones its environment shares. Secret " +
			"values are never shown. --kv prints NAME=VALUE lines for the plain " +
			"variables only, ready for a .env file.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			vars, err := cl.ListVariables(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			sortVars(vars)
			out := cmd.OutOrStdout()

			if kv {
				// The app's own value wins over a shared one of the same name.
				eff := byName(vars)
				for _, v := range vars {
					if eff[v.Name] == v && !v.Secret {
						fmt.Fprintf(out, "%s=%s\n", v.Name, dotenvValue(v.Value))
					}
				}
				return nil
			}
			if ux.IsStructured() {
				return ux.Print(out, vars)
			}
			if len(vars) == 0 {
				fmt.Fprintln(out, "No variables.")
				return nil
			}
			rows := [][]string{{"NAME", "VALUE", "SOURCE"}}
			for _, v := range vars {
				value := v.Value
				if v.Secret {
					value = "(secret)"
				}
				source := string(v.Source)
				if v.OverridesEnvironment {
					source += " (overrides environment)"
				}
				rows = append(rows, []string{v.Name, value, source})
			}
			return ux.Print(out, rows)
		},
	}
	cmd.Flags().BoolVar(&kv, "kv", false, "Print NAME=VALUE lines (secrets left out)")
	return cmd
}
