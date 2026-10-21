// Package setup exposes `okctl setup`: one-step setup of things okctl works
// with, starting with AI agents.
package setup

import (
	"fmt"
	"strings"

	"github.com/ownkube/okctl/cmd/auth"
	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/cmd/mcp"
	"github.com/ownkube/okctl/cmd/skills"
	"github.com/ownkube/okctl/internal/agents"
	"github.com/spf13/cobra"
)

// New returns the `okctl setup` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "setup",
		Short: "Set up okctl integrations",
	}
	root.AddCommand(agentCmd())
	return root
}

func agentCmd() *cobra.Command {
	var (
		targets         []string
		noMCP, noSkills bool
	)
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Set up Claude Code, Cursor or Codex to work with Ownkube",
		Long: "Get an AI agent ready for Ownkube in one step: sign in if you aren't already " +
			"('okctl login'), add the Ownkube MCP server ('okctl mcp install'), and install " +
			"the okctl skill ('okctl skills install'). With no --agent, every agent found " +
			"on this machine is set up. Safe to run again, e.g. after 'okctl upgrade'.",
		Example: "  okctl setup agent\n" +
			"  okctl setup agent --agent claude\n" +
			"  okctl setup agent --agent cursor --no-skills",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if noMCP && noSkills {
				return fmt.Errorf("--no-mcp and --no-skills together leave nothing to set up")
			}
			selected, err := agents.Select(targets)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()

			if !noMCP {
				creds, err := ux.Config().LoadCredentials()
				if err != nil {
					return err
				}
				if creds == nil {
					if ux.IsStructured() {
						return fmt.Errorf("not logged in: run 'okctl login' first")
					}
					if err := auth.SignIn(cmd); err != nil {
						return err
					}
					fmt.Fprintln(out)
				}
			}

			var report struct {
				MCP    []agents.Result `json:"mcp,omitempty"`
				Skills []agents.Result `json:"skills,omitempty"`
			}
			if !noMCP {
				report.MCP, err = mcp.Install(cmd, selected)
				if err == nil && !noSkills {
					report.Skills, err = skills.Install(selected)
				}
			} else {
				report.Skills, err = skills.Install(selected)
			}

			if ux.IsStructured() {
				if perr := ux.Print(out, report); perr != nil {
					return perr
				}
				return err
			}
			for _, r := range report.MCP {
				fmt.Fprintf(out, "%s: added the %s MCP server (%s).\n", r.Agent, agents.ServerName, r.Target)
			}
			for _, r := range report.Skills {
				fmt.Fprintln(out, skills.Describe(r))
			}
			if err != nil {
				return err
			}
			names := make([]string, len(selected))
			for i, a := range selected {
				names[i] = a.Name
			}
			fmt.Fprintf(out, "Done. Restart %s to load it.\n", strings.Join(names, ", "))
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&targets, "agent", nil, "Agent to set up: "+strings.Join(agents.IDs(), ", ")+" (repeatable; default: every one found)")
	cmd.Flags().BoolVar(&noMCP, "no-mcp", false, "Skip the MCP server")
	cmd.Flags().BoolVar(&noSkills, "no-skills", false, "Skip the okctl skill")
	return cmd
}
