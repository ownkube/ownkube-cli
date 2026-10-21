// Package skills exposes `okctl skills`: installs the okctl agent skill, the
// copy embedded in this binary, into Claude Code, Cursor and Codex.
package skills

import (
	"fmt"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/agents"
	embedded "github.com/ownkube/okctl/skills"
	"github.com/spf13/cobra"
)

// New returns the `okctl skills` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "skills",
		Short: "Teach AI agents to use okctl",
		Long: "The okctl skill tells an AI agent (Claude Code, Cursor, Codex) how to deploy " +
			"and operate apps with okctl. It ships inside this binary, so it always " +
			"matches the commands you have; reinstall it after 'okctl upgrade'.",
	}
	root.AddCommand(installCmd(), showCmd())
	return root
}

// Install writes the okctl skill into each agent's user skills directory.
func Install(targets []agents.Agent) ([]agents.Result, error) {
	var out []agents.Result
	for _, a := range targets {
		res, err := a.InstallSkill(embedded.Okctl)
		if err != nil {
			return out, fmt.Errorf("%s: %w", a.Name, err)
		}
		out = append(out, res)
	}
	return out, nil
}

func installCmd() *cobra.Command {
	var targets []string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the okctl skill into Claude Code, Cursor or Codex",
		Long: "Write the okctl skill to each agent's user skills directory " +
			"(<agent config>/skills/" + agents.SkillName + "/SKILL.md), replacing an earlier " +
			"copy. With no --agent, every agent found on this machine gets it.",
		Example: "  okctl skills install\n" +
			"  okctl skills install --agent claude",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			selected, err := agents.Select(targets)
			if err != nil {
				return err
			}
			results, err := Install(selected)
			out := cmd.OutOrStdout()
			if ux.IsStructured() {
				if perr := ux.Print(out, results); perr != nil {
					return perr
				}
				return err
			}
			for _, r := range results {
				fmt.Fprintln(out, Describe(r))
			}
			return err
		},
	}
	cmd.Flags().StringSliceVar(&targets, "agent", nil, "Agent to set up: "+strings.Join(agents.IDs(), ", ")+" (repeatable; default: every one found)")
	return cmd
}

// Describe is one line saying where a skill install went.
func Describe(r agents.Result) string {
	if r.Note != "" {
		return fmt.Sprintf("%s: okctl skill %s (%s).", r.Agent, r.Note, r.Target)
	}
	return fmt.Sprintf("%s: installed the okctl skill (%s).", r.Agent, r.Target)
}

func showCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the okctl skill",
		Long:  "Print the okctl skill's SKILL.md, to add it to an agent okctl doesn't set up itself.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := cmd.OutOrStdout().Write(embedded.Okctl)
			return err
		},
	}
}
