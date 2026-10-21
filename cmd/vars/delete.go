package vars

import (
	"fmt"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/prompt"
	"github.com/spf13/cobra"
)

func deleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "delete [app] NAME...",
		Aliases: []string{"unset", "rm"},
		Short:   "Remove variables from an app",
		Long: "Remove the app's own variables in one change. A variable shared by the " +
			"environment can't be removed here; change it with 'okctl environments " +
			"set-env'. Asks first unless --yes.\n\n" +
			"With more than one NAME, give the app too (or --service): the first " +
			"argument is otherwise taken as the app.",
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			depID, names := args[0], args[1:]
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			vars, err := cl.ListVariables(cmd.Context(), depID)
			if err != nil {
				return err
			}
			own, shared := map[string]bool{}, map[string]bool{}
			for _, v := range vars {
				if v.Source == "app" {
					own[v.Name] = true
				} else {
					shared[v.Name] = true
				}
			}
			seen := map[string]bool{}
			for _, n := range names {
				switch {
				case seen[n]:
					return fmt.Errorf("%s is given twice", n)
				case own[n]:
				case shared[n]:
					return fmt.Errorf("%s is shared by the app's environment; change it with 'okctl environments set-env'", n)
				default:
					return fmt.Errorf("the app has no variable %s", n)
				}
				seen[n] = true
			}

			if !yes {
				ok, err := prompt.Confirm(fmt.Sprintf("Remove %s? The app redeploys without them.", strings.Join(names, ", ")))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}
			return update(cmd, cl, depID, api.UpdateAppVariablesBody{Unset: &names})
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")
	return cmd
}
