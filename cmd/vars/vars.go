// Package vars exposes the `okctl vars` command tree: an app's variables,
// its own and the ones its environment shares.
package vars

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/client"
	"github.com/spf13/cobra"
)

// New returns the `okctl vars` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:     "vars",
		Aliases: []string{"variables", "var"},
		Short:   "Manage an app's variables",
		Long: "List, set and remove an app's variables. Secret values are write-only: " +
			"okctl never reads them back. Variables shared by the app's environment are " +
			"listed too; setting one on the app overrides it for that app only, and the " +
			"shared value itself is changed with 'okctl environments set-env'.\n\n" +
			"Every change rolls out a new revision of the app.",
	}
	root.AddCommand(listCmd(), setCmd(), deleteCmd(), editCmd())
	return root
}

var namePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func checkName(name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("%q isn't a valid variable name: use letters, numbers and underscores, not starting with a number", name)
	}
	return nil
}

// update sends one variables change and reports it.
func update(cmd *cobra.Command, cl *client.Client, depID string, body api.UpdateAppVariablesBody) error {
	res, err := cl.UpdateVariables(cmd.Context(), depID, body)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	if ux.IsStructured() {
		return ux.Print(out, res)
	}
	if !res.Changed {
		fmt.Fprintln(out, "No changes: the app already has these values.")
		return nil
	}
	if len(res.Set) > 0 {
		fmt.Fprintf(out, "Set %s.\n", strings.Join(res.Set, ", "))
	}
	if len(res.Unset) > 0 {
		fmt.Fprintf(out, "Removed %s.\n", strings.Join(res.Unset, ", "))
	}
	if res.RevisionId != "" {
		fmt.Fprintf(out, "Rolling out revision %s. Follow it with: okctl deploy watch %s\n", res.RevisionId, depID)
	}
	return nil
}

// byName indexes variables, the app's own winning over shared ones.
func byName(vars []api.AppVariable) map[string]api.AppVariable {
	out := make(map[string]api.AppVariable, len(vars))
	for _, v := range vars {
		if _, seen := out[v.Name]; !seen || v.Source == "app" {
			out[v.Name] = v
		}
	}
	return out
}

func sortVars(vars []api.AppVariable) {
	sort.SliceStable(vars, func(i, j int) bool { return vars[i].Name < vars[j].Name })
}
