// Package cmd builds the root cobra command and wires in each resource
// subpackage (auth, config, deploy). Subpackages depend on cmd/internal/ux
// for shared state and helpers — they never reach back into cmd directly.
package cmd

import (
	"fmt"
	"os"

	"github.com/ownkube/okctl/cmd/alerts"
	"github.com/ownkube/okctl/cmd/auth"
	awscmd "github.com/ownkube/okctl/cmd/aws"
	"github.com/ownkube/okctl/cmd/billing"
	"github.com/ownkube/okctl/cmd/clusters"
	"github.com/ownkube/okctl/cmd/config"
	"github.com/ownkube/okctl/cmd/deploy"
	"github.com/ownkube/okctl/cmd/domains"
	"github.com/ownkube/okctl/cmd/environments"
	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/cmd/organizations"
	"github.com/ownkube/okctl/cmd/regions"
	"github.com/ownkube/okctl/cmd/registries"
	"github.com/ownkube/okctl/cmd/usage"
	cfgpkg "github.com/ownkube/okctl/internal/config"
	"github.com/spf13/cobra"
)

var (
	flagAPIURL string
	flagOutput string
	flagOrg    string
)

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
		auth.Status(),
		config.New(),
		deploy.New(),
		awscmd.New(),
		clusters.New(),
		environments.New(),
		organizations.New(),
		registries.New(),
		regions.New(),
		usage.New(),
		billing.New(),
		alerts.New(),
		domains.New(),
	)
}

// Execute runs the root command. Called by main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
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
