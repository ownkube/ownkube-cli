package mcp

import (
	"fmt"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/agents"
	"github.com/spf13/cobra"
)

func installCmd() *cobra.Command {
	var (
		targets   []string
		printOnly bool
	)
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Add the Ownkube MCP server to Claude Code, Cursor or Codex",
		Long: "Add an '" + agents.ServerName + "' MCP server to each agent's user config that " +
			"runs 'okctl mcp serve'. The agent then uses okctl's sign-in; no API key goes " +
			"into its config. With no --agent, every agent found on this machine is set " +
			"up. Running it again replaces okctl's entry and leaves other servers alone. " +
			"Restart the agent afterwards.\n\n" +
			"A non-default --api-url, and an --organization or OKCTL_ORGANIZATION given to " +
			"this command, are pinned in the entry.",
		Example: "  okctl mcp install\n" +
			"  okctl mcp install --agent claude --agent cursor\n" +
			"  okctl mcp install --agent codex --print",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			selected, err := agents.Select(targets)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if printOnly {
				srv := agents.OkctlServer(serverFlags(cmd)...)
				for i, a := range selected {
					if i > 0 {
						fmt.Fprintln(out)
					}
					fmt.Fprintf(out, "# %s\n%s\n", a.Name, strings.TrimRight(a.Snippet(srv), "\n"))
				}
				return nil
			}

			results, err := Install(cmd, selected)
			if ux.IsStructured() {
				if perr := ux.Print(out, results); perr != nil {
					return perr
				}
				return err
			}
			for _, r := range results {
				fmt.Fprintf(out, "%s: added the %s MCP server (%s).\n", r.Agent, agents.ServerName, r.Target)
			}
			if err != nil {
				return err
			}
			fmt.Fprintln(out, "Restart the agent to load it.")
			if creds, _ := ux.Config().LoadCredentials(); creds == nil {
				fmt.Fprintln(out, "You're not signed in yet: run 'okctl login' before the agent starts the server.")
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&targets, "agent", nil, "Agent to set up: "+strings.Join(agents.IDs(), ", ")+" (repeatable; default: every one found)")
	cmd.Flags().BoolVar(&printOnly, "print", false, "Print the config to add instead of writing it")
	return cmd
}
