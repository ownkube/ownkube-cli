package config

import (
	"fmt"
	"io"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/manifest"
	"github.com/spf13/cobra"
)

// driftExit is the exit status of `plan --detect-changes` when the app
// differs from ownkube.yaml (1 stays for errors).
const driftExit = 2

func planCmd() *cobra.Command {
	var detect bool
	cmd := &cobra.Command{
		Use:   "plan [app]",
		Short: "Show what 'config apply' would change",
		Long: "Compare ownkube.yaml in the current directory with the app's live settings " +
			"and variables, and print each difference. Settings the file leaves out stay " +
			"as they are. When the file lists env, it is the app's full set of variables: " +
			"one missing from the file is removed. A secret is listed by name, so only " +
			"its presence is compared.\n\n" +
			"The plan ends with an id; 'okctl config apply --plan <id>' applies exactly " +
			"that plan and refuses if anything has changed since. --detect-changes exits " +
			"with status 2 when there are differences, for CI drift checks.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			m, err := manifest.Load(".")
			if err != nil {
				return err
			}
			p, err := makePlan(cmd.Context(), cl, args[0], m)
			if err != nil {
				return err
			}
			if err := printPlan(cmd.OutOrStdout(), p); err != nil {
				return err
			}
			if detect && !p.InSync {
				return &ux.ExitError{Code: driftExit}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&detect, "detect-changes", false, "Exit with status 2 when the app differs from "+manifest.FileName)
	return cmd
}

func printPlan(w io.Writer, p *plan) error {
	if ux.IsStructured() {
		return ux.Print(w, p)
	}
	if p.InSync {
		fmt.Fprintf(w, "No changes: %s matches %s.\n", p.Name, manifest.FileName)
		return nil
	}
	fmt.Fprintf(w, "Changes to %s (%s):\n", p.Name, p.DeploymentID)
	for _, l := range p.lines() {
		fmt.Fprintln(w, l)
	}
	fmt.Fprintf(w, "\n%d setting(s), %d variable(s).\n", len(p.Changes), len(p.Variables))
	if len(p.problems()) > 0 {
		fmt.Fprintln(w, "Fix the entries marked can't apply before applying.")
		return nil
	}
	fmt.Fprintf(w, "Plan %s. Apply exactly this with:\n  okctl config apply %s --plan %s\n", p.ID, p.DeploymentID, p.ID)
	return nil
}
