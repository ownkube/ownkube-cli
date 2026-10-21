// Package cmd builds the root cobra command and wires in each resource
// subpackage (auth, config, deploy). Subpackages depend on cmd/internal/ux
// for shared state and helpers — they never reach back into cmd directly.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ownkube/okctl/cmd/alerts"
	"github.com/ownkube/okctl/cmd/apicall"
	"github.com/ownkube/okctl/cmd/auth"
	awscmd "github.com/ownkube/okctl/cmd/aws"
	"github.com/ownkube/okctl/cmd/billing"
	"github.com/ownkube/okctl/cmd/boxes"
	"github.com/ownkube/okctl/cmd/clusters"
	"github.com/ownkube/okctl/cmd/config"
	"github.com/ownkube/okctl/cmd/connect"
	"github.com/ownkube/okctl/cmd/deploy"
	"github.com/ownkube/okctl/cmd/docs"
	"github.com/ownkube/okctl/cmd/domains"
	"github.com/ownkube/okctl/cmd/environments"
	"github.com/ownkube/okctl/cmd/functions"
	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/cmd/link"
	"github.com/ownkube/okctl/cmd/marketplace"
	"github.com/ownkube/okctl/cmd/mcp"
	"github.com/ownkube/okctl/cmd/open"
	"github.com/ownkube/okctl/cmd/organizations"
	"github.com/ownkube/okctl/cmd/projects"
	"github.com/ownkube/okctl/cmd/regions"
	"github.com/ownkube/okctl/cmd/registries"
	"github.com/ownkube/okctl/cmd/run"
	"github.com/ownkube/okctl/cmd/setup"
	"github.com/ownkube/okctl/cmd/skills"
	sshcmd "github.com/ownkube/okctl/cmd/ssh"
	"github.com/ownkube/okctl/cmd/status"
	"github.com/ownkube/okctl/cmd/up"
	"github.com/ownkube/okctl/cmd/upgrade"
	"github.com/ownkube/okctl/cmd/usage"
	"github.com/ownkube/okctl/cmd/vars"
	"github.com/ownkube/okctl/cmd/vcs"
	cfgpkg "github.com/ownkube/okctl/internal/config"
	"github.com/ownkube/okctl/internal/update"
	"github.com/ownkube/okctl/internal/version"
	"github.com/spf13/cobra"
)

var (
	flagAPIURL string
	flagOutput string
	flagOrg    string
)

// updateNotice yields a newer release version ("" for none) when the passive
// update check ran for this invocation.
var updateNotice <-chan string

// quietCommands never print the update notice: they're about the binary
// itself, or their output is read by another program.
var quietCommands = map[string]bool{"upgrade": true, "version": true, "completion": true, "__complete": true, "serve": true}

var rootCmd = &cobra.Command{
	Use:   "okctl",
	Short: "Ownkube CLI",
	Long:  "okctl is the command-line interface for the Ownkube developer platform.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := cfgpkg.NewManager("")
		if err != nil {
			return err
		}

		apiURL, err := resolveAPIURL(mgr)
		if err != nil {
			return err
		}
		out, err := resolveOutputFormat(mgr)
		if err != nil {
			return err
		}
		org, err := resolveOrganization(mgr)
		if err != nil {
			return err
		}

		ux.Set(ux.Globals{
			APIURL:       apiURL,
			OutputFormat: out,
			Organization: org,
			Config:       mgr,
		})
		if !quietCommands[cmd.Name()] && update.NoticeEnabled(version.Version) {
			updateNotice = update.Check(mgr.Dir(), version.Version)
		}
		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagAPIURL, "api-url", "", "Ownkube API URL (env: OKCTL_API_URL)")
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "", "Output format: table, json, yaml")
	rootCmd.PersistentFlags().StringVar(&flagOrg, "organization", "", "Organization ID for org-scoped commands (env: OKCTL_ORGANIZATION)")

	rootCmd.AddCommand(
		auth.Login(),
		auth.Logout(),
		auth.Whoami(),
		status.New(),
		config.New(),
		deploy.New(),
		awscmd.New(),
		clusters.New(),
		environments.New(),
		projects.New(),
		functions.New(),
		link.Link(),
		link.Unlink(),
		up.New(),
		organizations.New(),
		registries.New(),
		regions.New(),
		boxes.New(),
		marketplace.New(),
		usage.New(),
		billing.New(),
		alerts.New(),
		domains.New(),
		sshcmd.New(),
		connect.New(),
		vars.New(),
		run.New(),
		mcp.New(),
		skills.New(),
		setup.New(),
		vcs.New(),
		apicall.New(),
		open.New(),
		docs.New(),
		upgrade.New(),
	)
	ux.WireAppTargets(rootCmd)
}

// Execute runs the root command. Called by main.
func Execute() {
	err := rootCmd.Execute()
	code := 0
	if err != nil {
		code = 1
		var exitErr *ux.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.Code
		}
		if exitErr == nil || exitErr.Err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
	}
	printUpdateNotice()
	if code != 0 {
		os.Exit(code)
	}
}

// printUpdateNotice tells the user, on stderr, about a newer release. It waits
// briefly for a background check so a slow network never holds a command up.
func printUpdateNotice() {
	if updateNotice == nil {
		return
	}
	select {
	case latest := <-updateNotice:
		if latest == "" {
			return
		}
		hint := "okctl upgrade"
		if exe, err := os.Executable(); err == nil {
			hint = update.UpgradeHint(exe)
		}
		fmt.Fprintf(os.Stderr, "\nA new okctl is available: %s -> %s. Upgrade with: %s\n", version.Version, latest, hint)
	case <-time.After(time.Second):
	}
}

// resolveAPIURL applies the priority chain: flag > env > config > default.
func resolveAPIURL(mgr *cfgpkg.Manager) (string, error) {
	if flagAPIURL != "" {
		return flagAPIURL, nil
	}
	if v := os.Getenv("OKCTL_API_URL"); v != "" {
		return v, nil
	}
	cfg, err := mgr.LoadConfig()
	if err != nil {
		return "", err
	}
	if cfg.APIURL != "" {
		return cfg.APIURL, nil
	}
	return cfgpkg.DefaultAPIURL, nil
}

// resolveOrganization applies the priority chain: flag > env > config.
// An empty result is valid — single-org accounts don't need one, and the
// server resolves the sole membership automatically.
func resolveOrganization(mgr *cfgpkg.Manager) (string, error) {
	if flagOrg != "" {
		return flagOrg, nil
	}
	if v := os.Getenv("OKCTL_ORGANIZATION"); v != "" {
		return v, nil
	}
	cfg, err := mgr.LoadConfig()
	if err != nil {
		return "", err
	}
	return cfg.Organization, nil
}

// resolveOutputFormat applies the priority chain: flag > config > default.
func resolveOutputFormat(mgr *cfgpkg.Manager) (string, error) {
	if flagOutput != "" {
		return flagOutput, nil
	}
	cfg, err := mgr.LoadConfig()
	if err != nil {
		return "", err
	}
	if cfg.OutputFormat != "" {
		return cfg.OutputFormat, nil
	}
	return cfgpkg.DefaultOutputFormat, nil
}
