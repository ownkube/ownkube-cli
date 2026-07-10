// Package config exposes the `okctl config` command tree (get, set, view).
package config

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

// New returns the `okctl config` command with get/set/view attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
	}
	root.AddCommand(getCmd(), setCmd(), viewCmd())
	return root
}

func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long:  "Get a CLI configuration value.\n\nValid keys: api_url, output_format, organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := ux.Config().LoadConfig()
			if err != nil {
				return err
			}
			val, err := cfg.Get(args[0])
			if err != nil {
				return err
			}
			if val == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "(not set)")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), val)
			}
			return nil
		},
	}
}

func setCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long:  "Set a CLI configuration value.\n\nValid keys: api_url, output_format, organization",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := ux.Config().LoadConfig()
			if err != nil {
				return err
			}
			if err := cfg.Set(args[0], args[1]); err != nil {
				return err
			}
			if err := ux.Config().SaveConfig(cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %s\n", args[0], args[1])
			return nil
		},
	}
}

func viewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "View all configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := ux.Config().LoadConfig()
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), cfg)
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"KEY", "VALUE"},
				{"api_url", displayVal(cfg.APIURL)},
				{"output_format", displayVal(cfg.OutputFormat)},
				{"organization", displayVal(cfg.Organization)},
			})
		},
	}
}

func displayVal(s string) string {
	if s == "" {
		return "(default)"
	}
	return s
}
