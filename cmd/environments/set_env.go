package environments

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

func setEnvCmd() *cobra.Command {
	var (
		file    string
		envVars []string
		secrets []string
	)

	cmd := &cobra.Command{
		Use:   "set-env <environment-id>",
		Short: "Replace the shared environment variables for an environment",
		Long: "Replace the full shared env-var set for an environment. This is a " +
			"REPLACE, not a merge: any variable not included is removed. Every " +
			"cluster-bound app in the environment is redeployed to apply the change.\n\n" +
			"Provide variables with --env KEY=VALUE / --secret KEY=VALUE (repeatable) " +
			"or a JSON file (-f) containing an array of {name, value, secret} objects. " +
			"Use '-f -' to read from stdin.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vars, err := collectEnvVars(file, envVars, secrets)
			if err != nil {
				return err
			}

			client, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := client.SetEnvVars(cmd.Context(), args[0], vars)
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"Variables", fmt.Sprintf("%d", len(res.Env))},
				{"Redeployed", fmt.Sprintf("%g", res.Redeployed)},
				{"Failed", fmt.Sprintf("%g", res.Failed)},
			})
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "JSON file with an array of {name,value,secret}; '-' for stdin")
	cmd.Flags().StringArrayVar(&envVars, "env", nil, "Plain variable KEY=VALUE (repeatable)")
	cmd.Flags().StringArrayVar(&secrets, "secret", nil, "Secret variable KEY=VALUE (repeatable)")
	return cmd
}

// collectEnvVars builds the replacement set from either a JSON file or the
// --env/--secret flags. The two sources are mutually exclusive to avoid an
// ambiguous partial merge.
func collectEnvVars(file string, envVars, secrets []string) ([]api.EnvVarInput, error) {
	if file != "" {
		if len(envVars) > 0 || len(secrets) > 0 {
			return nil, fmt.Errorf("pass either -f or --env/--secret, not both")
		}
		raw, err := ux.ReadFileOrStdin(file)
		if err != nil {
			return nil, fmt.Errorf("reading env file: %w", err)
		}
		var vars []api.EnvVarInput
		if err := json.Unmarshal(raw, &vars); err != nil {
			return nil, fmt.Errorf("parsing env file: expected a JSON array of {name,value,secret}: %w", err)
		}
		return vars, nil
	}

	if len(envVars) == 0 && len(secrets) == 0 {
		return nil, fmt.Errorf("provide variables with --env/--secret or a file (-f); to clear all, pass -f with an empty array []")
	}

	out := make([]api.EnvVarInput, 0, len(envVars)+len(secrets))
	plain, err := parseKeyValues(envVars, false)
	if err != nil {
		return nil, err
	}
	secret, err := parseKeyValues(secrets, true)
	if err != nil {
		return nil, err
	}
	out = append(out, plain...)
	out = append(out, secret...)
	return out, nil
}

func parseKeyValues(pairs []string, secret bool) ([]api.EnvVarInput, error) {
	out := make([]api.EnvVarInput, 0, len(pairs))
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid KEY=VALUE %q", p)
		}
		item := api.EnvVarInput{Name: k, Value: v}
		if secret {
			s := true
			item.Secret = &s
		}
		out = append(out, item)
	}
	return out, nil
}
