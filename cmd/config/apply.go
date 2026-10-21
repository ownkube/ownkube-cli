package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/client"
	"github.com/ownkube/okctl/internal/manifest"
	"github.com/ownkube/okctl/internal/prompt"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type applyResult struct {
	DeploymentID       string `json:"deploymentId" yaml:"deploymentId"`
	Plan               string `json:"plan" yaml:"plan"`
	SettingsChanged    bool   `json:"settingsChanged" yaml:"settingsChanged"`
	VariablesChanged   bool   `json:"variablesChanged" yaml:"variablesChanged"`
	VariableRevisionID string `json:"variableRevisionId,omitempty" yaml:"variableRevisionId,omitempty"`
}

func applyCmd() *cobra.Command {
	var (
		planID string
		yes    bool
		note   string
	)
	cmd := &cobra.Command{
		Use:   "apply [app]",
		Short: "Change the app to match ownkube.yaml",
		Long: "Apply the changes 'okctl config plan' shows. With --plan <id>, apply runs " +
			"only if the plan is still exactly the one reviewed, and refuses if the app " +
			"or ownkube.yaml has changed since. Without --plan it shows the plan and asks " +
			"first (--yes skips the question).\n\n" +
			"Settings and variables are applied as separate changes, so an apply that " +
			"touches both rolls out twice. A new secret can't be created from ownkube.yaml, " +
			"which holds names only: set it with 'okctl vars set NAME=<value> --secret'.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, out := cmd.Context(), cmd.OutOrStdout()
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			m, err := manifest.Load(".")
			if err != nil {
				return err
			}
			p, err := makePlan(ctx, cl, args[0], m)
			if err != nil {
				return err
			}

			if planID != "" && planID != p.ID {
				return fmt.Errorf("the plan is now %s, not %s: the app or %s changed since it was reviewed. Run 'okctl config plan' again", p.ID, planID, manifest.FileName)
			}
			if p.InSync {
				if ux.IsStructured() {
					return ux.Print(out, applyResult{DeploymentID: p.DeploymentID, Plan: p.ID})
				}
				fmt.Fprintf(out, "No changes: %s matches %s.\n", p.Name, manifest.FileName)
				return nil
			}
			if probs := p.problems(); len(probs) > 0 {
				return fmt.Errorf("can't apply:\n  %s", strings.Join(probs, "\n  "))
			}
			if planID == "" && !yes {
				if ux.IsStructured() || !term.IsTerminal(int(os.Stdin.Fd())) {
					return fmt.Errorf("pass --plan <id> from 'okctl config plan', or --yes")
				}
				if err := printPlan(os.Stderr, p); err != nil {
					return err
				}
				ok, err := prompt.Confirm(fmt.Sprintf("Apply these changes to %s?", p.Name))
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("aborted")
				}
			}

			res := applyResult{DeploymentID: p.DeploymentID, Plan: p.ID}
			if len(p.Changes) > 0 {
				body := map[string]any{p.key: p.block}
				if note != "" {
					body["note"] = note
				}
				raw, err := json.Marshal(body)
				if err != nil {
					return err
				}
				if _, err := cl.UpdateDeployment(ctx, p.DeploymentID, raw); err != nil {
					if client.IsAPICode(err, "PRECONDITION_FAILED") {
						return fmt.Errorf("%s's settings unlock after its first build finishes", p.Name)
					}
					return err
				}
				res.SettingsChanged = true
			}
			if len(p.Variables) > 0 {
				r, err := cl.UpdateVariables(ctx, p.DeploymentID, varBody(p.Variables))
				if err != nil {
					if res.SettingsChanged {
						return fmt.Errorf("settings applied, but variables failed: %w", err)
					}
					return err
				}
				res.VariablesChanged = r.Changed
				res.VariableRevisionID = r.RevisionId
			}

			if ux.IsStructured() {
				return ux.Print(out, res)
			}
			if res.SettingsChanged {
				fmt.Fprintf(out, "Applied %d setting change(s) to %s.\n", len(p.Changes), p.Name)
			}
			if res.VariablesChanged {
				fmt.Fprintf(out, "Applied %d variable change(s) to %s.\n", len(p.Variables), p.Name)
			}
			fmt.Fprintf(out, "Follow the rollout with: okctl deploy watch %s\n", p.DeploymentID)
			return nil
		},
	}
	cmd.Flags().StringVar(&planID, "plan", "", "Apply only if the plan id from 'okctl config plan' still matches")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Apply without asking")
	cmd.Flags().StringVar(&note, "note", "", "Note recorded on the settings revision")
	return cmd
}

// varBody turns the plan's variable changes into one variables update.
func varBody(changes []varChange) api.UpdateAppVariablesBody {
	var set []api.AppVariableInput
	var unset []string
	for _, v := range changes {
		if v.Op == "unset" {
			unset = append(unset, v.Name)
			continue
		}
		in := api.AppVariableInput{Name: v.Name, Value: v.Value}
		if v.Op == "change" {
			// Pin the flag so a plain/secret switch goes through.
			secret := v.Secret
			in.Secret = &secret
		}
		set = append(set, in)
	}
	var body api.UpdateAppVariablesBody
	if len(set) > 0 {
		body.Set = &set
	}
	if len(unset) > 0 {
		body.Unset = &unset
	}
	return body
}
