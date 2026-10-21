package config

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/manifest"
	"github.com/spf13/cobra"
)

func pullCmd() *cobra.Command {
	var force, printOnly bool
	cmd := &cobra.Command{
		Use:   "pull [app]",
		Short: "Write an app's live settings to ownkube.yaml",
		Long: "Write the app's live settings and variables to ownkube.yaml in the current " +
			"directory, ready to commit and change with 'okctl config plan' and " +
			"'okctl config apply'. Secret variables are written by name only, and " +
			"credentials, the password-protection settings and maintenance mode are " +
			"left out. --print writes to stdout instead of the file.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			d, err := cl.GetDeployment(ctx, args[0])
			if err != nil {
				return err
			}
			sp, live, err := specFor(d)
			if err != nil {
				return err
			}
			var vars []api.AppVariable
			if sp.hasVars {
				if vars, err = cl.ListVariables(ctx, d.Id); err != nil {
					return err
				}
			}
			m := toManifest(d, sp, live, vars)

			out := cmd.OutOrStdout()
			if printOnly {
				if ux.IsStructured() {
					return ux.Print(out, m)
				}
				data, err := m.YAML()
				if err != nil {
					return err
				}
				_, err = out.Write(data)
				return err
			}
			if !force && manifest.Exists(".") {
				return fmt.Errorf("%s already exists; pass --force to overwrite it, or --print to compare", manifest.FileName)
			}
			if err := manifest.Write(".", m, true); err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(out, map[string]string{"deploymentId": d.Id, "file": manifest.FileName})
			}
			fmt.Fprintf(out, "Wrote %s for %s (%s).\n", manifest.FileName, d.Name, sp.resourceType)
			for _, e := range m.Env {
				if e.Secret {
					fmt.Fprintln(out, "Secret variables are listed by name only; their values stay on Ownkube.")
					break
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite an existing "+manifest.FileName)
	cmd.Flags().BoolVar(&printOnly, "print", false, "Print the settings instead of writing "+manifest.FileName)
	return cmd
}
