// Package mcp exposes `okctl mcp`: a stdio bridge to the Ownkube MCP server
// and an installer that points AI agents at it.
package mcp

import (
	"os"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/agents"
	cfgpkg "github.com/ownkube/okctl/internal/config"
	"github.com/spf13/cobra"
)

// New returns the `okctl mcp` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "mcp",
		Short: "Connect AI agents to the Ownkube MCP server",
		Long: "The Ownkube MCP server gives an AI agent (Claude Code, Cursor, Codex) tools " +
			"for your apps, databases, logs and deploys. 'okctl mcp install' adds it to an " +
			"agent's config as a local command, 'okctl mcp serve', which signs the agent's " +
			"requests with the credentials from 'okctl login'. No API key is written into " +
			"the agent's config.",
	}
	root.AddCommand(serveCmd(), installCmd())
	return root
}

// Install adds okctl's MCP server entry to each agent, pinning the global
// flags this invocation was given so the agent talks to the same API and
// organization.
func Install(cmd *cobra.Command, targets []agents.Agent) ([]agents.Result, error) {
	srv := agents.OkctlServer(serverFlags(cmd)...)
	var out []agents.Result
	for _, a := range targets {
		res, err := a.InstallMCP(srv)
		if err != nil {
			return out, err
		}
		out = append(out, res)
	}
	return out, nil
}

// serverFlags are the global flags `okctl mcp serve` needs to reach what this
// invocation reaches. Values from okctl's config file need no flag: serve
// reads the same file.
func serverFlags(cmd *cobra.Command) []string {
	var flags []string
	if f := cmd.Flags().Lookup("api-url"); f != nil && f.Changed || envSet("OKCTL_API_URL") {
		if u := ux.APIURL(); u != cfgpkg.DefaultAPIURL {
			flags = append(flags, "--api-url", u)
		}
	}
	if f := cmd.Flags().Lookup("organization"); f != nil && f.Changed || envSet("OKCTL_ORGANIZATION") {
		if org := ux.Organization(); org != "" {
			flags = append(flags, "--organization", org)
		}
	}
	return flags
}

func envSet(key string) bool { return os.Getenv(key) != "" }
